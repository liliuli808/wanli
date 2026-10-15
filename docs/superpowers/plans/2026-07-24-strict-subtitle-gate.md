# Strict Subtitle Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop video-import tasks before any AI call when no usable subtitle was obtained, while retaining descriptions and recipe-like comments as supplements when subtitles exist.

**Architecture:** Enforce the source requirement at the orchestration boundary in `VideoImportService.Process`, immediately after `FetchBestSubtitle`. Keep the existing API error code, synchronize user-facing wording across Go, Android, and documentation, and prove with a call-counting fake AI server that fallback text cannot bypass the subtitle gate.

**Tech Stack:** Go 1.22, Gin service layer, Testify, `httptest`, Kotlin, Jetpack ViewModel tests, Markdown documentation

## Global Constraints

- A video import requires at least one usable CC or AI subtitle segment.
- Missing, empty, or failed subtitle retrieval must produce `SUBTITLE_NOT_AVAILABLE`.
- Descriptions and recipe-like comments may supplement existing subtitles but may not replace them.
- Do not add ASR, webpage scraping, AI retry behavior, database changes, or API response-shape changes.
- Do not expose provider errors or API keys to the Android client.
- Update `docs/api-contract.md` and `docs/ai-parsing.md` with the behavior change.

---

### Task 1: Enforce the backend subtitle gate

**Files:**
- Modify: `backend-go/internal/service/video_import_service_test.go:7-224`
- Modify: `backend-go/internal/service/video_import_service.go:155-187`
- Modify: `backend-go/pkg/apperror/apperror.go:42`

**Interfaces:**
- Consumes: `bilibili.Client.FetchBestSubtitle(ctx, videoInfo) ([]SubtitleSegment, bool, error)`
- Produces: `FAILED` import tasks with `error_code=SUBTITLE_NOT_AVAILABLE` before `ai.Client.ExtractRecipe` is called

- [x] **Step 1: Add a failing backend regression test**

Add `sync/atomic` to the test imports. Change `fakeBiliServer` so `noSubtitle=true`
removes only the subtitle track while retaining a non-empty description and a recipe-like
top comment:

```go
// fakeBiliServer 返回模拟 B 站 API；noSubtitle=true 时保留简介/评论但不提供字幕。
func fakeBiliServer(t *testing.T, noSubtitle bool) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/x/web-interface/view", func(w http.ResponseWriter, r *http.Request) {
		subtitleSection := fmt.Sprintf(`"list": [{"id": 1, "lan": "zh-CN", "lan_doc": "简体中文", "subtitle_url": "%s/subtitle.json", "ai_status": 0}]`, server.URL)
		if noSubtitle {
			subtitleSection = `"list": []`
		}
		fmt.Fprintf(w, `{
			"code": 0, "message": "0",
			"data": {
				"aid": 123456, "bvid": "BV1xx411c7xx", "title": "家常红烧肉",
				"desc": "简介：五花肉500克", "duration": 300, "pic": "",
				"owner": {"name": "美食作家"},
				"pages": [{"cid": 111, "page": 1, "part": "正片", "duration": 300}],
				"subtitle": {%s}
			}
		}`, subtitleSection)
	})
	mux.HandleFunc("/subtitle.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"body": [{"from": 0, "to": 5, "content": "今天做红烧肉"}]}`)
	})
	mux.HandleFunc("/x/v2/reply/main", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code": 0, "message": "0", "data": {"top": {"upper": {"rpid": 1, "like": 10, "content": {"message": "配方：五花肉500克，生抽2勺，小火炖30分钟"}, "member": {"uname": "作者"}}}, "replies": []}}`)
	})

	server = httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}
```

Add a call-counting AI server without changing the existing helper's callers:

```go
func fakeAIServerWithCalls(t *testing.T, content string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := json.Marshal(content)
		fmt.Fprintf(w, `{"choices": [{"message": {"content": %s}}]}`, body)
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func fakeAIServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	server, _ := fakeAIServerWithCalls(t, content)
	return server
}
```

Replace `TestVideoImport_Process_NoTextSource` with:

```go
func TestVideoImport_Process_NoSubtitleRejectsFallbackTextBeforeAI(t *testing.T) {
	repo := newFakeRecipeRepo()
	aiServer, aiCalls := fakeAIServerWithCalls(t, validAIOutput)
	svc := newImportService(t, fakeBiliServer(t, true).URL, aiServer.URL, repo, &noopSubmitter{})
	ctx := context.Background()

	task, err := svc.tasks.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	err = svc.Process(ctx, task.TaskID, uuid.New())
	require.Error(t, err)

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskFailed, got.Status)
	assert.Equal(t, "SUBTITLE_NOT_AVAILABLE", got.ErrorCode)
	assert.Zero(t, aiCalls.Load(), "无字幕时不应调用 AI")
	assert.Empty(t, repo.saved, "无字幕时不应落库")
}
```

- [x] **Step 2: Run the regression test and verify it fails**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-strict-subtitle-go-cache go test ./internal/service -run TestVideoImport_Process_NoSubtitleRejectsFallbackTextBeforeAI -count=1
```

Expected: FAIL because the current service uses the description/comment fallback and calls AI.

- [x] **Step 3: Implement the minimal service gate**

Replace the text-source collection block in `VideoImportService.Process` with:

```go
// Step 2: 获取字幕；无可用字幕时立即终止，简介/评论只能作为字幕补充。
subtitleSegments, hasSubtitle, err := s.bili.FetchBestSubtitle(ctx, videoInfo)
if err != nil {
	s.logger.Warn("字幕获取失败，终止解析",
		slog.String("task_id", taskID), slog.String("error", err.Error()))
	return fail(apperror.ErrSubtitleUnavailable, fmt.Errorf("fetch subtitle: %w", err))
}
if !hasSubtitle || len(subtitleSegments) == 0 {
	return fail(apperror.ErrSubtitleUnavailable, errors.New("no usable subtitle"))
}

comments, _ := s.bili.GetRecipeLikeComments(ctx, videoInfo.AID, 5)
input := ai.ParseInput{
	BVID:        videoInfo.BVID,
	Title:       videoInfo.Title,
	Description: videoInfo.Description,
}
for _, seg := range subtitleSegments {
	input.Subtitles = append(input.Subtitles, ai.SubtitleInput{
		Start: seg.From,
		End:   seg.To,
		Text:  seg.Content,
	})
}
for _, c := range comments {
	input.Comments = append(input.Comments, c.Message)
}
```

Remove the later `if !input.HasContent()` branch because the subtitle gate guarantees valid
input for this workflow.

Change the predefined error message to:

```go
ErrSubtitleUnavailable = New("SUBTITLE_NOT_AVAILABLE", "该视频没有可用字幕，无法解析", http.StatusUnprocessableEntity)
```

- [x] **Step 4: Run focused backend tests**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-strict-subtitle-go-cache go test ./internal/service ./pkg/apperror -count=1
```

Expected: PASS.

---

### Task 2: Synchronize Android messaging and documentation

**Files:**
- Modify: `frontend-android/app/src/test/kotlin/com/wanli/android/presentation/home/ImportViewModelTest.kt:150-158`
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/presentation/home/ImportViewModel.kt:120-129`
- Modify: `docs/api-contract.md:37`
- Modify: `docs/ai-parsing.md:32-64,112-125,271-275`
- Modify: `docs/user-flows.md:18-25`
- Modify: `docs/development-tasks.md:61,74`

**Interfaces:**
- Consumes: backend error code `SUBTITLE_NOT_AVAILABLE`
- Produces: the fallback text `该视频没有可用字幕，无法解析` and documentation that treats descriptions/comments as supplements only

- [x] **Step 1: Change the Android expectation first**

Update the expected text in `ImportViewModelTest`:

```kotlin
assertEquals(
    "该视频没有可用字幕，无法解析",
    ImportViewModel.failureMessage(null, "SUBTITLE_NOT_AVAILABLE")
)
```

- [x] **Step 2: Run the Android unit test and verify it fails**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest --tests 'com.wanli.android.presentation.home.ImportViewModelTest'
```

Expected: FAIL because `ImportViewModel.failureMessage` still mentions descriptions and comments.

- [x] **Step 3: Update the Android fallback message**

Change the `SUBTITLE_NOT_AVAILABLE` branch in `ImportViewModel.failureMessage`:

```kotlin
"SUBTITLE_NOT_AVAILABLE" -> "该视频没有可用字幕，无法解析"
```

- [x] **Step 4: Update the API and AI contracts**

In `docs/api-contract.md`, replace the error-code row with:

```markdown
| `SUBTITLE_NOT_AVAILABLE` | 无可用字幕 | 未找到 CC/AI 字幕或字幕拉取失败 |
```

In `docs/ai-parsing.md`, make these rules explicit:

```markdown
1. **字幕是强制来源**：AI 解析必须取得 B 站 CC/AI 字幕（原始文本 + 时间戳）。没有字幕轨、字幕内容为空或字幕拉取失败时，任务直接失败（`SUBTITLE_NOT_AVAILABLE`）。**禁止使用简介/评论替代字幕，也禁止回退到语音转写（ASR）**。
2. **补充文本来源**：已取得字幕时，可附加视频简介、置顶/高赞菜谱型评论作为补充；与字幕冲突时以字幕为准。
```

Replace the input-field descriptions and the two bullets below the input example with:

```markdown
  "description": "视频简介（可选，仅在已有字幕时作为补充）",
  "comments": ["置顶/高赞评论（可选，仅在已有字幕时作为补充）"]
```

```markdown
- `subtitles` 必须包含至少一条可用字幕；没有字幕轨、字幕内容为空或字幕拉取失败时，任务直接失败。
- 简介与评论仅作字幕的补充：与字幕冲突时以字幕为准；缺乏字幕支撑的补充内容应体现不确定性（如将相关 `amount` 置为 `null`）。
```

Add this changelog row:

```markdown
| 2026-07-24 | AI 助手 | 恢复严格字幕拦截：无可用字幕时立即失败；简介/评论仅在已有字幕时作为补充 |
```

Keep the documented system-prompt block and JSON schema unchanged: the service gate guarantees
that this existing prompt is only called with subtitles, and its description/comment rules still
apply to supplemental facts.

- [x] **Step 5: Update workflow and completion documentation**

Replace the no-subtitle branch in `docs/user-flows.md` with:

```text
  │  2. 字幕拉取与强制校验  ├─────────► [ 无可用字幕/拉取失败 ] ──► 终止任务：显示“该视频没有可用字幕，无法解析”
  └──────────┬─────────────┘
             │ [ 获取到字幕；简介/评论可作为补充 ]
```

In `docs/development-tasks.md`, update the completed backend item and AC2 to:

```markdown
- [x] 实现 B 站 CC/AI 字幕获取服务；无可用字幕或字幕拉取失败时立即终止，简介/评论仅作补充。
- [x] AC2：无字幕的视频 15 秒内进入 FAILED 状态并提示，不调用 AI。
```

- [x] **Step 6: Run the focused Android test and documentation consistency check**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest --tests 'com.wanli.android.presentation.home.ImportViewModelTest'
```

Expected: PASS.

Run from the repository root:

```bash
! rg -n "简介/评论兜底|字幕、简介、评论均不可用|没有可用的字幕、简介或评论|无任何可用文本来源" \
  docs/api-contract.md docs/user-flows.md docs/development-tasks.md \
  backend-go/pkg/apperror backend-go/internal/service frontend-android --glob '!**/build/**'
! sed -n '1,270p' docs/ai-parsing.md |
  rg "简介/评论兜底|字幕、简介、评论均不可用|无任何可用文本来源"
```

Expected: both searches return no matches. The historical changelog entry below line 270 in
`docs/ai-parsing.md` remains intact and is followed by the new strict-gate entry. The generic
`ai.Client` validation message is internal-only; the import service now prevents that path from
receiving subtitle-free input.

---

### Task 3: Run complete verification

**Files:**
- Verify only; no additional source changes

**Interfaces:**
- Consumes: all changes from Tasks 1 and 2
- Produces: fresh evidence that the strict subtitle gate passes project quality checks

- [x] **Step 1: Format and vet Go**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-strict-subtitle-go-cache go fmt ./...
GOCACHE=/tmp/wanli-strict-subtitle-go-cache go vet ./...
```

Expected: both commands exit `0`.

- [x] **Step 2: Run all Go tests and build**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-strict-subtitle-go-cache go test ./...
GOCACHE=/tmp/wanli-strict-subtitle-go-cache go build -buildvcs=false ./...
```

Expected: tests and build exit `0`.

- [x] **Step 3: Run Android tests, Release build, and lint**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest assembleRelease lintDebug
```

Expected: `BUILD SUCCESSFUL`.

- [x] **Step 4: Confirm the user-owned backend remains untouched**

Run:

```bash
ss -ltnp
```

Expected: verification commands do not start or stop the user's backend process.

No commit step is available because the supplied workspace is not recognized as a Git repository.
