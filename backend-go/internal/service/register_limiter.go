package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RegisterLimiter 按 IP 限制注册频率（固定窗口计数）。
type RegisterLimiter interface {
	Allow(ctx context.Context, ip string) (bool, error)
}

// RedisRegisterLimiter 基于 Redis 的实现（多实例共享）。
type RedisRegisterLimiter struct {
	rdb    *redis.Client
	max    int64
	window time.Duration
}

// NewRedisRegisterLimiter 创建 Redis 注册限流器。
func NewRedisRegisterLimiter(rdb *redis.Client, max int, window time.Duration) *RedisRegisterLimiter {
	return &RedisRegisterLimiter{rdb: rdb, max: int64(max), window: window}
}

func (l *RedisRegisterLimiter) key(ip string) string {
	return "auth:register:" + ip
}

// Allow 记录一次注册尝试并返回是否放行。ip 为空时不限制。
func (l *RedisRegisterLimiter) Allow(ctx context.Context, ip string) (bool, error) {
	if ip == "" {
		return true, nil
	}
	key := l.key(ip)
	n, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("count registrations: %w", err)
	}
	if n == 1 {
		if err := l.rdb.Expire(ctx, key, l.window).Err(); err != nil {
			return false, fmt.Errorf("expire registration counter: %w", err)
		}
	}
	return n <= l.max, nil
}

type registerEntry struct {
	count int
	reset time.Time
}

// MemoryRegisterLimiter 进程内实现（开发环境无 Redis 时兜底，重启即重置）。
type MemoryRegisterLimiter struct {
	mu     sync.Mutex
	counts map[string]registerEntry
	max    int
	window time.Duration
	now    func() time.Time
}

// NewMemoryRegisterLimiter 创建内存注册限流器。
func NewMemoryRegisterLimiter(max int, window time.Duration) *MemoryRegisterLimiter {
	return newMemoryRegisterLimiterWithClock(max, window, time.Now)
}

// newMemoryRegisterLimiterWithClock 创建带可注入时钟的内存限流器（仅供测试使用）。
func newMemoryRegisterLimiterWithClock(max int, window time.Duration, now func() time.Time) *MemoryRegisterLimiter {
	return &MemoryRegisterLimiter{
		counts: make(map[string]registerEntry),
		max:    max,
		window: window,
		now:    now,
	}
}

// Allow 记录一次注册尝试并返回是否放行。ip 为空时不限制。
func (l *MemoryRegisterLimiter) Allow(_ context.Context, ip string) (bool, error) {
	if ip == "" {
		return true, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, ok := l.counts[ip]
	if !ok || now.After(e.reset) {
		e = registerEntry{reset: now.Add(l.window)}
	}
	e.count++
	l.counts[ip] = e
	return e.count <= l.max, nil
}
