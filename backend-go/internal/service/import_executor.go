package service

import (
	"context"
	"log/slog"
	"time"
)

// ImportExecutor 以 goroutine 池方式异步执行解析任务：
// 信号量限制并发数（超出的任务排队等待），每个任务带独立超时。
type ImportExecutor struct {
	sem     chan struct{}
	timeout time.Duration
	logger  *slog.Logger
}

// NewImportExecutor 创建执行器。concurrency 为最大并行解析数，timeout 为单任务总时限。
func NewImportExecutor(concurrency int, timeout time.Duration, logger *slog.Logger) *ImportExecutor {
	if concurrency < 1 {
		concurrency = 1
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ImportExecutor{
		sem:     make(chan struct{}, concurrency),
		timeout: timeout,
		logger:  logger,
	}
}

// Submit 立即返回；任务排队获得信号量后在后台 goroutine 中执行 fn。
// fn 负责自身的状态推进与失败标记；ctx 取消（超时）时 fn 应尽快返回。
func (e *ImportExecutor) Submit(taskID string, fn func(ctx context.Context) error) {
	go func() {
		e.sem <- struct{}{}
		defer func() { <-e.sem }()

		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		defer cancel()

		if err := fn(ctx); err != nil {
			e.logger.Warn("解析任务执行结束（含错误）",
				slog.String("task_id", taskID),
				slog.String("error", err.Error()))
		}
	}()
}
