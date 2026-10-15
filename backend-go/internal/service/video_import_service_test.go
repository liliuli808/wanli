package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/infrastructure/ai"
	"github.com/wanli/backend-go/internal/infrastructure/bilibili"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// fakeRecipeRepo 是 RecipeRepository 的内存实现。
type fakeRecipeRepo struct {
	mu           sync.Mutex
	videoSources map[string]*model.VideoSource
	recipesByVS  map[uuid.UUID]uuid.UUID
	saved        []repository.SourceDraftParams
	nextVersion  uuid.UUID
}

func newFakeRecipeRepo() *fakeRecipeRepo {
	return &fakeRecipeRepo{
		videoSources: map[string]*model.VideoSource{},
		recipesByVS:  map[uuid.UUID]uuid.UUID{},
		nextVersion:  uuid.New(),
	}
}

func (r *fakeRecipeRepo) FindVideoSourceByBVID(_ context.Context, bvid string) (*model.VideoSource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.videoSources[bvid], nil
}

func (r *fakeRecipeRepo) FindRecipeIDByVideoSourceID(_ context.Context, vsID uuid.UUID) (*uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.recipesByVS[vsID]; ok {
		return &id, nil
	}
	return nil, nil
}

func (r *fakeRecipeRepo) SaveSourceDraft(_ context.Context, params repository.SourceDraftParams) (*model.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, params)
	return &model.RecipeVersion{ID: r.nextVersion}, nil
}

// 阶段 3 扩展的接口方法：视频导入流程不会调用，提供占位实现。
func (r *fakeRecipeRepo) FindVersionDetail(context.Context, uuid.UUID) (*repository.VersionDetail, error) {
	return nil, nil
}

func (r *fakeRecipeRepo) SetVersionPublic(context.Context, uuid.UUID, bool) error {
	return nil
}

func (r *fakeRecipeRepo) UpdateSourceDraft(context.Context, repository.DraftUpdateParams) error {
	return nil
}

func (r *fakeRecipeRepo) ListPublicRecipes(context.Context, string, int, int) ([]repository.RecipeListItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeRecipeRepo) FindRecipeByID(context.Context, uuid.UUID) (*model.Recipe, error) {
	return nil, nil
}

func (r *fakeRecipeRepo) ListPublicVersionsByRecipe(context.Context, uuid.UUID) ([]repository.VersionTreeNode, error) {
	return nil, nil
}

func (r *fakeRecipeRepo) CreateDerivedVersion(context.Context, repository.DerivedVersionParams) (*model.RecipeVersion, error) {
	return nil, nil
}

// noopSubmitter 不真正执行后台任务，仅记录。
type noopSubmitter struct {
	submitted []string
}

func (n *noopSubmitter) Submit(taskID string, fn func(ctx context.Context) error) {
	n.submitted = append(n.submitted, taskID)
}

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

// fakeAIServer 返回模拟 OpenAI 兼容接口的服务器。
func fakeAIServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	server, _ := fakeAIServerWithCalls(t, content)
	return server
}

func newImportService(t *testing.T, biliURL, aiURL string, repo repository.RecipeRepository, submitter taskSubmitter) *VideoImportService {
	t.Helper()
	taskSvc, _ := newTestTaskService(t)
	return NewVideoImportService(
		taskSvc,
		submitter,
		bilibili.New(biliURL, slog.Default()),
		ai.New("test-key", aiURL, "deepseek-chat"),
		repo,
		slog.Default(),
	)
}

const validAIOutput = `{
	"title": "家常红烧肉",
	"servings": "2-3人份",
	"estimated_minutes": 45,
	"ingredients": [{"name": "五花肉", "amount": "500克", "category": "MAIN", "is_explicit": true}],
	"steps": [{"step_order": 1, "description": "五花肉切块焯水", "video_timestamp": 0, "duration_seconds": null, "heat": null}],
	"instructions": null
}`

func TestVideoImport_Process_Success(t *testing.T) {
	repo := newFakeRecipeRepo()
	svc := newImportService(t, fakeBiliServer(t, false).URL, fakeAIServer(t, validAIOutput).URL, repo, &noopSubmitter{})
	ctx := context.Background()

	task, err := svc.tasks.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	require.NoError(t, svc.Process(ctx, task.TaskID, uuid.New()))

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskCompleted, got.Status)
	assert.Equal(t, repo.nextVersion.String(), got.RecipeVersionID)

	require.Len(t, repo.saved, 1)
	assert.Equal(t, "家常红烧肉", repo.saved[0].RecipeTitle)
	assert.Equal(t, "BV1xx411c7xx", repo.saved[0].VideoSource.BVID)
	assert.Equal(t, "美食作家", repo.saved[0].VideoSource.UpNickname)
}

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

func TestVideoImport_Process_AIFailure(t *testing.T) {
	repo := newFakeRecipeRepo()
	svc := newImportService(t, fakeBiliServer(t, false).URL, fakeAIServer(t, "垃圾输出").URL, repo, &noopSubmitter{})
	ctx := context.Background()

	task, err := svc.tasks.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	err = svc.Process(ctx, task.TaskID, uuid.New())
	require.Error(t, err)

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskFailed, got.Status)
	assert.Equal(t, "AI_PARSE_FAILED", got.ErrorCode)
}

func TestVideoImport_Process_VideoFetchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code": -404, "message": "啥都木有"}`)
	}))
	t.Cleanup(server.Close)

	repo := newFakeRecipeRepo()
	svc := newImportService(t, server.URL, "http://unused", repo, &noopSubmitter{})
	ctx := context.Background()

	task, err := svc.tasks.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	require.Error(t, svc.Process(ctx, task.TaskID, uuid.New()))

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskFailed, got.Status)
	assert.Equal(t, "VIDEO_FETCH_FAILED", got.ErrorCode)
}

func TestVideoImport_Submit_InvalidURL(t *testing.T) {
	repo := newFakeRecipeRepo()
	svc := newImportService(t, "http://unused", "http://unused", repo, &noopSubmitter{})

	_, err := svc.Submit(context.Background(), uuid.New(), "https://www.youtube.com/watch?v=abc")
	assert.ErrorIs(t, err, ErrInvalidVideoURL)
}

func TestVideoImport_Submit_Duplicate(t *testing.T) {
	repo := newFakeRecipeRepo()
	vsID := uuid.New()
	recipeID := uuid.New()
	repo.videoSources["BV1xx411c7xx"] = &model.VideoSource{ID: vsID, BVID: "BV1xx411c7xx"}
	repo.recipesByVS[vsID] = recipeID

	svc := newImportService(t, "http://unused", "http://unused", repo, &noopSubmitter{})

	_, err := svc.Submit(context.Background(), uuid.New(), "https://www.bilibili.com/video/BV1xx411c7xx/")
	var dup *DuplicateVideoError
	require.True(t, errors.As(err, &dup), "应返回重复错误")
	assert.Equal(t, recipeID, dup.RecipeID)
}

func TestVideoImport_Submit_Success(t *testing.T) {
	repo := newFakeRecipeRepo()
	submitter := &noopSubmitter{}
	svc := newImportService(t, "http://unused", "http://unused", repo, submitter)
	ctx := context.Background()

	task, err := svc.Submit(ctx, uuid.New(), "https://www.bilibili.com/video/BV1xx411c7xx/")
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskPending, task.Status)
	assert.Equal(t, "BV1xx411c7xx", task.BVID)
	assert.Equal(t, []string{task.TaskID}, submitter.submitted, "任务应被异步投递")
}

func TestVideoImport_Submit_VideoSourceExistsButNoRecipe(t *testing.T) {
	// video_source 有记录但没有关联菜谱（异常残留），应允许重新解析
	repo := newFakeRecipeRepo()
	repo.videoSources["BV1xx411c7xx"] = &model.VideoSource{ID: uuid.New(), BVID: "BV1xx411c7xx"}

	submitter := &noopSubmitter{}
	svc := newImportService(t, "http://unused", "http://unused", repo, submitter)

	task, err := svc.Submit(context.Background(), uuid.New(), "BV1xx411c7xx")
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.Len(t, submitter.submitted, 1)
}
