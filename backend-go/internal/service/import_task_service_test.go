package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

func newTestTaskService(t *testing.T) (*ImportTaskService, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := repository.NewImportTaskRepository(rdb, 24*time.Hour)
	svc := NewImportTaskService(repo, slog.Default())
	return svc, mr
}

func TestImportTask_CreateAndGet(t *testing.T) {
	svc, _ := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, "BV1xx411c7xx", "https://www.bilibili.com/video/BV1xx411c7xx/")
	require.NoError(t, err)
	assert.NotEmpty(t, task.TaskID)
	assert.Equal(t, model.ImportTaskPending, task.Status)

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, task.TaskID, got.TaskID)
	assert.Equal(t, "BV1xx411c7xx", got.BVID)
	assert.Equal(t, model.ImportTaskPending, got.Status)
	assert.Equal(t, task.CreatedAt.Unix(), got.CreatedAt.Unix())
}

func TestImportTask_GetNotFound(t *testing.T) {
	svc, _ := newTestTaskService(t)

	_, err := svc.Get(context.Background(), "not-exist")
	assert.ErrorIs(t, err, repository.ErrImportTaskNotFound)
}

func TestImportTask_TransitionFlow(t *testing.T) {
	svc, _ := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	require.NoError(t, svc.Transition(ctx, task.TaskID, model.ImportTaskFetching, "正在获取视频信息"))
	require.NoError(t, svc.Transition(ctx, task.TaskID, model.ImportTaskParsing, "正在用AI拆解菜谱"))

	require.NoError(t, svc.Complete(ctx, task.TaskID, "version-uuid"))

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskCompleted, got.Status)
	assert.Equal(t, "version-uuid", got.RecipeVersionID)

	// 进入终态后 inflight 应已清空
	ids, err := svc.repo.ListInflight(ctx)
	require.NoError(t, err)
	assert.NotContains(t, ids, task.TaskID)
}

func TestImportTask_IllegalTransition(t *testing.T) {
	svc, _ := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	assert.Error(t, svc.Transition(ctx, task.TaskID, model.ImportTaskParsing, "跳级"),
		"PENDING 不能直接跳到 PARSING")
	assert.Error(t, svc.Complete(ctx, task.TaskID, "v"),
		"PENDING 不能直接完成")

	// 终态后不可再流转
	require.NoError(t, svc.Transition(ctx, task.TaskID, model.ImportTaskFetching, ""))
	require.NoError(t, svc.Fail(ctx, task.TaskID, "SUBTITLE_NOT_AVAILABLE", "无可用文本"))
	assert.Error(t, svc.Transition(ctx, task.TaskID, model.ImportTaskParsing, ""))
	assert.Error(t, svc.Fail(ctx, task.TaskID, "X", "重复失败"))
}

func TestImportTask_Fail(t *testing.T) {
	svc, _ := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	require.NoError(t, svc.Transition(ctx, task.TaskID, model.ImportTaskFetching, ""))
	require.NoError(t, svc.Fail(ctx, task.TaskID, "SUBTITLE_NOT_AVAILABLE", "该视频无可用文本来源"))

	got, err := svc.Get(ctx, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskFailed, got.Status)
	assert.Equal(t, "SUBTITLE_NOT_AVAILABLE", got.ErrorCode)

	ids, err := svc.repo.ListInflight(ctx)
	require.NoError(t, err)
	assert.NotContains(t, ids, task.TaskID)
}

func TestImportTask_RecoverStale(t *testing.T) {
	svc, _ := newTestTaskService(t)
	ctx := context.Background()

	base := time.Now()
	svc.now = func() time.Time { return base }

	stale, err := svc.Create(ctx, "BV1xx411c7xx", "url-stale")
	require.NoError(t, err)
	fresh, err := svc.Create(ctx, "BV1xx411c7yy", "url-fresh")
	require.NoError(t, err)
	require.NoError(t, svc.Transition(ctx, stale.TaskID, model.ImportTaskFetching, ""))

	// 20 分钟后：stale 任务心跳停在创建时，fresh 任务刚更新过
	svc.now = func() time.Time { return base.Add(20 * time.Minute) }
	require.NoError(t, svc.Transition(ctx, fresh.TaskID, model.ImportTaskFetching, ""))

	recovered, err := svc.RecoverStale(ctx, 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)

	got, err := svc.Get(ctx, stale.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskFailed, got.Status)

	got, err = svc.Get(ctx, fresh.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportTaskFetching, got.Status, "新鲜任务不应被恢复")
}

func TestImportTask_RecoverStale_ExpiredRecord(t *testing.T) {
	svc, mr := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, "BV1xx411c7xx", "url")
	require.NoError(t, err)

	// 任务记录过 TTL 已消失，但 inflight 登记还留着
	mr.FastForward(25 * time.Hour)

	recovered, err := svc.RecoverStale(ctx, 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered, "已过期的记录只清理登记，不计入恢复数")

	ids, err := svc.repo.ListInflight(ctx)
	require.NoError(t, err)
	assert.NotContains(t, ids, task.TaskID)
}
