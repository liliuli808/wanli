package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// 合法的状态流转：PENDING → FETCHING → PARSING → COMPLETED，任何进行中的状态都可 → FAILED。
var importTaskTransitions = map[model.ImportTaskStatus][]model.ImportTaskStatus{
	model.ImportTaskPending:  {model.ImportTaskFetching, model.ImportTaskFailed},
	model.ImportTaskFetching: {model.ImportTaskParsing, model.ImportTaskFailed},
	model.ImportTaskParsing:  {model.ImportTaskCompleted, model.ImportTaskFailed},
}

// ImportTaskService 管理视频解析任务的状态机。
type ImportTaskService struct {
	repo   *repository.ImportTaskRepository
	logger *slog.Logger
	now    func() time.Time // 便于测试注入
}

// NewImportTaskService 创建任务服务。
func NewImportTaskService(repo *repository.ImportTaskRepository, logger *slog.Logger) *ImportTaskService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ImportTaskService{repo: repo, logger: logger, now: time.Now}
}

// Create 创建 PENDING 状态的新任务。
func (s *ImportTaskService) Create(ctx context.Context, bvid, rawURL string) (*model.VideoImportTask, error) {
	now := s.now()
	task := &model.VideoImportTask{
		TaskID:    uuid.NewString(),
		BVID:      bvid,
		URL:       rawURL,
		Status:    model.ImportTaskPending,
		Message:   "任务已创建，等待解析",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// Get 查询任务状态；不存在返回 repository.ErrImportTaskNotFound。
func (s *ImportTaskService) Get(ctx context.Context, taskID string) (*model.VideoImportTask, error) {
	return s.repo.Get(ctx, taskID)
}

// Transition 推进状态机并更新进度文案；非法流转返回错误。
func (s *ImportTaskService) Transition(ctx context.Context, taskID string, to model.ImportTaskStatus, message string) error {
	task, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if !canTransition(task.Status, to) {
		return fmt.Errorf("illegal status transition: %s -> %s", task.Status, to)
	}
	task.Status = to
	task.Message = message
	task.UpdatedAt = s.now()
	return s.repo.Update(ctx, task)
}

// Complete 标记任务完成并关联生成的草稿菜谱版本。
func (s *ImportTaskService) Complete(ctx context.Context, taskID, recipeVersionID string) error {
	task, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if !canTransition(task.Status, model.ImportTaskCompleted) {
		return fmt.Errorf("illegal status transition: %s -> %s", task.Status, model.ImportTaskCompleted)
	}
	task.Status = model.ImportTaskCompleted
	task.Message = "解析完成"
	task.RecipeVersionID = recipeVersionID
	task.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, task); err != nil {
		return err
	}
	return s.repo.RemoveInflight(ctx, taskID)
}

// Fail 标记任务失败（允许从任何进行中状态进入，包括重复失败）。
func (s *ImportTaskService) Fail(ctx context.Context, taskID, errorCode, message string) error {
	task, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status.IsTerminal() {
		return fmt.Errorf("task already in terminal status: %s", task.Status)
	}
	task.Status = model.ImportTaskFailed
	task.ErrorCode = errorCode
	task.Message = message
	task.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, task); err != nil {
		return err
	}
	return s.repo.RemoveInflight(ctx, taskID)
}

// RecoverStale 把长时间无心跳的进行中任务标记为失败（服务重启后调用），返回恢复数量。
func (s *ImportTaskService) RecoverStale(ctx context.Context, staleAfter time.Duration) (int, error) {
	ids, err := s.repo.ListInflight(ctx)
	if err != nil {
		return 0, err
	}

	recovered := 0
	for _, id := range ids {
		task, err := s.repo.Get(ctx, id)
		if errors.Is(err, repository.ErrImportTaskNotFound) {
			// 记录已过期，清理登记即可
			if err := s.repo.RemoveInflight(ctx, id); err != nil {
				return recovered, err
			}
			continue
		}
		if err != nil {
			return recovered, err
		}
		if task.Status.IsTerminal() || s.now().Sub(task.UpdatedAt) < staleAfter {
			continue
		}
		task.Status = model.ImportTaskFailed
		task.ErrorCode = "INTERNAL_ERROR"
		task.Message = "服务中断导致解析终止，请重新提交"
		task.UpdatedAt = s.now()
		if err := s.repo.Update(ctx, task); err != nil {
			return recovered, err
		}
		if err := s.repo.RemoveInflight(ctx, id); err != nil {
			return recovered, err
		}
		recovered++
		s.logger.Info("恢复卡死的解析任务", slog.String("task_id", id))
	}
	return recovered, nil
}

func canTransition(from, to model.ImportTaskStatus) bool {
	for _, allowed := range importTaskTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
