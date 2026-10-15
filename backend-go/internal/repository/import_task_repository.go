package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wanli/backend-go/internal/model"
)

const (
	importTaskKeyPrefix = "video_import:task:"
	importInflightKey   = "video_import:inflight"
)

// ErrImportTaskNotFound 表示任务不存在（或已过期）。
var ErrImportTaskNotFound = errors.New("import task not found")

// ImportTaskRepository 基于 Redis 的解析任务存储。
// 任务本体是带 TTL 的 hash；进行中的任务 ID 同时登记在 inflight 集合中，
// 用于服务重启后的卡死任务恢复。
type ImportTaskRepository struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewImportTaskRepository 创建解析任务存储。ttl 为任务记录保留时长（如 24h）。
func NewImportTaskRepository(rdb *redis.Client, ttl time.Duration) *ImportTaskRepository {
	return &ImportTaskRepository{rdb: rdb, ttl: ttl}
}

func (r *ImportTaskRepository) key(taskID string) string {
	return importTaskKeyPrefix + taskID
}

// Create 写入新任务并登记到 inflight 集合。
func (r *ImportTaskRepository) Create(ctx context.Context, task *model.VideoImportTask) error {
	pipe := r.rdb.TxPipeline()
	pipe.HSet(ctx, r.key(task.TaskID), taskToMap(task))
	pipe.Expire(ctx, r.key(task.TaskID), r.ttl)
	pipe.SAdd(ctx, importInflightKey, task.TaskID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("create import task: %w", err)
	}
	return nil
}

// Get 读取任务；不存在时返回 ErrImportTaskNotFound。
func (r *ImportTaskRepository) Get(ctx context.Context, taskID string) (*model.VideoImportTask, error) {
	values, err := r.rdb.HGetAll(ctx, r.key(taskID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get import task: %w", err)
	}
	if len(values) == 0 {
		return nil, ErrImportTaskNotFound
	}
	return taskFromMap(values)
}

// Update 覆盖写入任务字段并刷新 TTL。
func (r *ImportTaskRepository) Update(ctx context.Context, task *model.VideoImportTask) error {
	pipe := r.rdb.TxPipeline()
	pipe.HSet(ctx, r.key(task.TaskID), taskToMap(task))
	pipe.Expire(ctx, r.key(task.TaskID), r.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("update import task: %w", err)
	}
	return nil
}

// RemoveInflight 将任务从 inflight 集合移除（进入终态时调用）。
func (r *ImportTaskRepository) RemoveInflight(ctx context.Context, taskID string) error {
	if err := r.rdb.SRem(ctx, importInflightKey, taskID).Err(); err != nil {
		return fmt.Errorf("remove inflight task: %w", err)
	}
	return nil
}

// ListInflight 返回所有进行中（未进终态）的任务 ID。
func (r *ImportTaskRepository) ListInflight(ctx context.Context) ([]string, error) {
	ids, err := r.rdb.SMembers(ctx, importInflightKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list inflight tasks: %w", err)
	}
	return ids, nil
}

func taskToMap(t *model.VideoImportTask) map[string]interface{} {
	return map[string]interface{}{
		"task_id":           t.TaskID,
		"bvid":              t.BVID,
		"url":               t.URL,
		"status":            string(t.Status),
		"message":           t.Message,
		"error_code":        t.ErrorCode,
		"recipe_version_id": t.RecipeVersionID,
		"created_at":        t.CreatedAt.Unix(),
		"updated_at":        t.UpdatedAt.Unix(),
	}
}

func taskFromMap(values map[string]string) (*model.VideoImportTask, error) {
	createdAt, err := strconv.ParseInt(values["created_at"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := strconv.ParseInt(values["updated_at"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &model.VideoImportTask{
		TaskID:          values["task_id"],
		BVID:            values["bvid"],
		URL:             values["url"],
		Status:          model.ImportTaskStatus(values["status"]),
		Message:         values["message"],
		ErrorCode:       values["error_code"],
		RecipeVersionID: values["recipe_version_id"],
		CreatedAt:       time.Unix(createdAt, 0),
		UpdatedAt:       time.Unix(updatedAt, 0),
	}, nil
}
