package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginLimiter 记录登录失败并判定账号是否被临时锁定。
type LoginLimiter interface {
	IsLocked(ctx context.Context, email string) (bool, error)
	RecordFailure(ctx context.Context, email string) error
	Reset(ctx context.Context, email string) error
}

// RedisLoginLimiter 基于 Redis 的实现（多实例共享）。
type RedisLoginLimiter struct {
	rdb            *redis.Client
	maxFailures    int64
	lockoutTimeout time.Duration
}

// NewRedisLoginLimiter 创建 Redis 限流器。
func NewRedisLoginLimiter(rdb *redis.Client, maxFailures int, lockout time.Duration) *RedisLoginLimiter {
	return &RedisLoginLimiter{
		rdb:            rdb,
		maxFailures:    int64(maxFailures),
		lockoutTimeout: lockout,
	}
}

func (l *RedisLoginLimiter) key(email string) string {
	return "auth:fail:" + email
}

// IsLocked 返回账号当前是否处于锁定状态。
func (l *RedisLoginLimiter) IsLocked(ctx context.Context, email string) (bool, error) {
	n, err := l.rdb.Get(ctx, l.key(email)).Int64()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get login failures: %w", err)
	}
	return n >= l.maxFailures, nil
}

// RecordFailure 累加一次失败并刷新过期时间。
func (l *RedisLoginLimiter) RecordFailure(ctx context.Context, email string) error {
	key := l.key(email)
	pipe := l.rdb.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, l.lockoutTimeout)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	return nil
}

// Reset 清除失败计数（登录成功后调用）。
func (l *RedisLoginLimiter) Reset(ctx context.Context, email string) error {
	if err := l.rdb.Del(ctx, l.key(email)).Err(); err != nil {
		return fmt.Errorf("reset login failures: %w", err)
	}
	return nil
}

type memEntry struct {
	count int
	until time.Time
}

// MemoryLoginLimiter 进程内实现（开发环境无 Redis 时兜底，重启即重置）。
type MemoryLoginLimiter struct {
	mu             sync.Mutex
	failures       map[string]memEntry
	maxFailures    int
	lockoutTimeout time.Duration
	now            func() time.Time
}

// NewMemoryLoginLimiter 创建内存限流器。
func NewMemoryLoginLimiter(maxFailures int, lockout time.Duration) *MemoryLoginLimiter {
	return newMemoryLoginLimiterWithClock(maxFailures, lockout, time.Now)
}

// newMemoryLoginLimiterWithClock 创建带可注入时钟的内存限流器（仅供测试使用）。
func newMemoryLoginLimiterWithClock(maxFailures int, lockout time.Duration, now func() time.Time) *MemoryLoginLimiter {
	return &MemoryLoginLimiter{
		failures:       make(map[string]memEntry),
		maxFailures:    maxFailures,
		lockoutTimeout: lockout,
		now:            now,
	}
}

// IsLocked 返回账号是否处于锁定状态。
func (l *MemoryLoginLimiter) IsLocked(ctx context.Context, email string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.failures[email]
	if !ok {
		return false, nil
	}
	if l.now().After(e.until) {
		delete(l.failures, email)
		return false, nil
	}
	return e.count >= l.maxFailures, nil
}

// RecordFailure 累加一次失败。
func (l *MemoryLoginLimiter) RecordFailure(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, ok := l.failures[email]
	if !ok || now.After(e.until) {
		l.failures[email] = memEntry{count: 1, until: now.Add(l.lockoutTimeout)}
		return nil
	}
	e.count++
	e.until = now.Add(l.lockoutTimeout)
	l.failures[email] = e
	return nil
}

// Reset 清除失败计数。
func (l *MemoryLoginLimiter) Reset(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, email)
	return nil
}
