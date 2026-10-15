# Bilibili Cookie Environment Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the Go backend authenticate Bilibili requests with an optional `BILIBILI_COOKIE` environment variable so login-only subtitle tracks can be discovered.

**Architecture:** Add a server-only Bilibili configuration value, inject it into the Bilibili HTTP client during dependency assembly, and set the standard HTTP `Cookie` header from the client's unified GET method. Preserve existing unauthenticated constructors and behavior for tests and installations without a Cookie.

**Tech Stack:** Go 1.22, Viper dotenv configuration, `net/http`, `httptest`, Testify, Markdown documentation

## Global Constraints

- `BILIBILI_COOKIE` contains the complete browser `Cookie` request-header value.
- Cookie data must remain server-side and must never appear in logs, API responses, Android code, database records, Redis, or checked-in examples.
- Empty Cookie configuration must preserve the current guest request behavior.
- Keep the strict subtitle gate: no visible subtitle after authenticated lookup still returns `SUBTITLE_NOT_AVAILABLE`.
- Do not add QR login, automatic Cookie refresh, Android-side Bilibili login, database changes, or API contract changes.
- Do not modify the user-owned `backend-go/.env` file.

---

### Task 1: Load Bilibili Cookie from server configuration

**Files:**
- Modify: `backend-go/internal/config/config_test.go:20-76`
- Modify: `backend-go/internal/config/config.go:11-149`

**Interfaces:**
- Produces: `config.Config.Bilibili.Cookie string`
- Consumes: `.env` or operating-system variable `BILIBILI_COOKIE`

- [x] **Step 1: Add failing configuration expectations**

In `TestLoad_Defaults`, add:

```go
assert.Empty(t, cfg.Bilibili.Cookie)
```

Change the `TestLoad_EnvFile` fixture to:

```go
"DB_PORT=5433\nREDIS_PORT=6380\nAI_MODEL=gpt-4o\nBILIBILI_COOKIE=SESSDATA=test-session; bili_jct=test-csrf\n"
```

and add:

```go
assert.Equal(t, "SESSDATA=test-session; bili_jct=test-csrf", cfg.Bilibili.Cookie)
```

In `TestLoad_OSEnv`, add:

```go
t.Setenv("BILIBILI_COOKIE", "SESSDATA=os-session")
```

and:

```go
assert.Equal(t, "SESSDATA=os-session", cfg.Bilibili.Cookie)
```

- [x] **Step 2: Run the focused test and verify it fails**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go test ./internal/config -count=1
```

Expected: compilation fails because `Config` does not yet contain `Bilibili`.

- [x] **Step 3: Add the minimal configuration type and mapping**

Add the field to `Config`:

```go
Bilibili BilibiliConfig
```

Add the type:

```go
type BilibiliConfig struct {
	Cookie string
}
```

Add this mapping to the returned `Config`:

```go
Bilibili: BilibiliConfig{
	Cookie: v.GetString("BILIBILI_COOKIE"),
},
```

No default call is needed because an unset Viper string is already empty.

- [x] **Step 4: Run the focused configuration tests**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go test ./internal/config -count=1
```

Expected: PASS.

---

### Task 2: Send the configured Cookie on Bilibili requests

**Files:**
- Modify: `backend-go/internal/infrastructure/bilibili/client_test.go:91-124`
- Modify: `backend-go/internal/infrastructure/bilibili/client.go:74-110`
- Modify: `backend-go/internal/server/server.go:125-132`

**Interfaces:**
- Consumes: `config.Config.Bilibili.Cookie`
- Produces: `bilibili.NewWithCookie(baseURL, cookie string, logger *slog.Logger) *Client`
- Preserves: `bilibili.New(baseURL string, logger *slog.Logger) *Client`
- Preserves: `bilibili.NewWithHTTPClient(baseURL string, httpClient *http.Client, logger *slog.Logger) *Client`

- [x] **Step 1: Add a failing Cookie-header test**

Add this test after `TestGetVideoInfo`:

```go
func TestGetVideoInfo_CookieHeader(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
	}{
		{name: "authenticated", cookie: "SESSDATA=test-session; bili_jct=test-csrf"},
		{name: "guest", cookie: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedCookie string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedCookie = r.Header.Get("Cookie")
				fmt.Fprint(w, `{"code":0,"message":"0","data":{"bvid":"BV1xx411c7xx","pages":[],"subtitle":{"list":[]}}}`)
			}))
			t.Cleanup(server.Close)

			client := NewWithCookie(server.URL, tt.cookie, nil)
			_, err := client.GetVideoInfo(context.Background(), "BV1xx411c7xx")
			require.NoError(t, err)
			assert.Equal(t, tt.cookie, receivedCookie)
		})
	}
}
```

- [x] **Step 2: Run the focused test and verify it fails**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go test ./internal/infrastructure/bilibili -run TestGetVideoInfo_CookieHeader -count=1
```

Expected: compilation fails because `NewWithCookie` does not yet exist.

- [x] **Step 3: Implement Cookie-aware client construction**

Add `cookie string` to `Client`:

```go
type Client struct {
	httpClient *http.Client
	baseURL    string
	cookie     string
	logger     *slog.Logger
}
```

Keep existing constructors and add the authenticated constructor through one private initializer:

```go
func New(baseURL string, logger *slog.Logger) *Client {
	return NewWithCookie(baseURL, "", logger)
}

func NewWithCookie(baseURL, cookie string, logger *slog.Logger) *Client {
	return newClient(baseURL, cookie, &http.Client{Timeout: 15 * time.Second}, logger)
}

func NewWithHTTPClient(baseURL string, httpClient *http.Client, logger *slog.Logger) *Client {
	return newClient(baseURL, "", httpClient, logger)
}

func newClient(baseURL, cookie string, httpClient *http.Client, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		cookie:     strings.TrimSpace(cookie),
		logger:     logger,
	}
}
```

Update the unified GET request method:

```go
req.Header.Set("User-Agent", userAgent)
req.Header.Set("Referer", "https://www.bilibili.com")
if c.cookie != "" {
	req.Header.Set("Cookie", c.cookie)
}
```

- [x] **Step 4: Inject the Cookie in server assembly**

Replace the production Bilibili client construction with:

```go
bilibili.NewWithCookie(bilibili.DefaultBaseURL, cfg.Bilibili.Cookie, logger),
```

Do not log `cfg.Bilibili.Cookie`.

- [x] **Step 5: Run focused Bilibili and server tests**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go test ./internal/infrastructure/bilibili ./internal/server -count=1
```

Expected: PASS.

---

### Task 3: Document secure Cookie configuration

**Files:**
- Modify: `backend-go/.env.example:28-37`
- Modify: `README.md:65-82,225-260`
- Modify: `docs/ai-parsing.md:5-30`

**Interfaces:**
- Consumes: `BILIBILI_COOKIE`
- Produces: operator documentation without any real credential value

- [x] **Step 1: Add the empty environment example**

Add before the AI section in `backend-go/.env.example`:

```env
# Bilibili (optional; enables subtitle tracks visible only after login)
# Sensitive: paste the complete browser Cookie header only into your local .env.
BILIBILI_COOKIE=
```

- [x] **Step 2: Update README configuration guidance**

Add this line to the complete `.env` example:

```env
BILIBILI_COOKIE=
```

Add the following operator guidance immediately after that example:

```markdown
`BILIBILI_COOKIE` 可选，用于读取仅登录用户可见的 B 站字幕轨。填写浏览器请求中的
完整 `Cookie` 请求头，只能保存在后端本地 `.env` 或部署平台密钥中；禁止提交、
写入日志或放入 Android。Cookie 更新后需要重启后端，过期后需重新获取。
```

Also add one sentence below the initial `cp .env.example .env` startup block:

```markdown
如需解析仅登录可见的字幕，请在本地 `.env` 中配置 `BILIBILI_COOKIE` 后再启动服务。
```

- [x] **Step 3: Update the AI parsing source documentation**

Add this subsection before “AI 服务配置” in `docs/ai-parsing.md`:

````markdown
## B 站字幕登录态

后端可通过 `BILIBILI_COOKIE` 携带 B 站登录态，以读取仅登录用户可见的字幕轨。
该值是完整的浏览器 `Cookie` 请求头，只能保存在服务端环境变量或未纳入版本控制的
本地 `.env` 中，禁止写入日志、API 响应或 Android 客户端。未配置时使用游客模式；
Cookie 过期或被撤销后需要人工更新并重启后端。

```env
BILIBILI_COOKIE=
```

登录态只影响字幕可见性，不改变严格字幕拦截：携带 Cookie 后仍没有可用字幕时，
任务返回 `SUBTITLE_NOT_AVAILABLE`，不会调用 AI。
````

- [x] **Step 4: Verify documentation contains no Cookie value**

Run:

```bash
test "$(rg -l '^BILIBILI_COOKIE=$' backend-go/.env.example README.md docs/ai-parsing.md | wc -l)" -eq 3
! rg -n 'BILIBILI_COOKIE=.+|SESSDATA=|bili_jct=' backend-go/.env.example README.md docs/ai-parsing.md
```

Expected: PASS; all examples are empty and no credential-like values appear.

---

### Task 4: Run complete verification

**Files:**
- Verify only; no additional source changes

**Interfaces:**
- Consumes: all changes from Tasks 1-3
- Produces: fresh evidence that Cookie configuration is secure, compatible, and buildable

- [x] **Step 1: Format and vet Go**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go fmt ./...
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go vet ./...
```

Expected: both commands exit `0`.

- [x] **Step 2: Run all Go tests and build**

Run:

```bash
cd backend-go
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go test ./...
GOCACHE=/tmp/wanli-bilibili-cookie-go-cache go build -buildvcs=false ./...
```

Expected: tests and build exit `0`.

- [x] **Step 3: Run Android tests, Release build, and lint**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest assembleRelease lintDebug
```

Expected: `BUILD SUCCESSFUL`.

- [x] **Step 4: Confirm the backend process was not restarted**

Run:

```bash
ss -ltnp
```

Expected: verification does not start or stop the user-owned backend process.

No commit step is available because the supplied workspace is not recognized as a Git repository.
