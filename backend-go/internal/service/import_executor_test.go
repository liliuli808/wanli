package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportExecutor_RespectsConcurrencyLimit(t *testing.T) {
	exec := NewImportExecutor(2, time.Minute, slog.Default())

	var running int32
	var maxRunning int32
	var wg sync.WaitGroup

	for i := 0; i < 6; i++ {
		wg.Add(1)
		exec.Submit("task", func(ctx context.Context) error {
			defer wg.Done()
			cur := atomic.AddInt32(&running, 1)
			for {
				old := atomic.LoadInt32(&maxRunning)
				if cur <= old || atomic.CompareAndSwapInt32(&maxRunning, old, cur) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt32(&running, -1)
			return nil
		})
	}
	wg.Wait()

	assert.Equal(t, int32(2), atomic.LoadInt32(&maxRunning), "并发峰值不应超过信号量上限")
}

func TestImportExecutor_TaskTimeout(t *testing.T) {
	exec := NewImportExecutor(1, 50*time.Millisecond, slog.Default())

	done := make(chan error, 1)
	exec.Submit("task", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			done <- ctx.Err()
			return ctx.Err()
		case <-time.After(5 * time.Second):
			done <- nil
			return nil
		}
	})

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded, "超时后任务 ctx 应被取消")
	case <-time.After(2 * time.Second):
		t.Fatal("任务未在超时时间内被取消")
	}
}

func TestImportExecutor_SerializesWhenConcurrencyIsOne(t *testing.T) {
	exec := NewImportExecutor(1, time.Minute, slog.Default())

	var running int32
	var maxRunning int32
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		exec.Submit("task", func(ctx context.Context) error {
			defer wg.Done()
			cur := atomic.AddInt32(&running, 1)
			for {
				old := atomic.LoadInt32(&maxRunning)
				if cur <= old || atomic.CompareAndSwapInt32(&maxRunning, old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&running, -1)
			return nil
		})
	}
	wg.Wait()

	// goroutine 的启动顺序本就不保证 FIFO，这里只验证严格串行
	assert.Equal(t, int32(1), atomic.LoadInt32(&maxRunning), "并发为 1 时不应有任何并行执行")
}
