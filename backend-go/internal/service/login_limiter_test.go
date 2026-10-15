package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRedisLimiter(t *testing.T) (*RedisLoginLimiter, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisLoginLimiter(rdb, 5, 15*time.Minute), s
}

func TestRedisLoginLimiter_LocksAfterMaxFailures(t *testing.T) {
	limiter, _ := newRedisLimiter(t)
	ctx := context.Background()

	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}

	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestRedisLoginLimiter_ResetUnlocks(t *testing.T) {
	limiter, _ := newRedisLimiter(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}
	require.NoError(t, limiter.Reset(ctx, "a@b.c"))

	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestMemoryLoginLimiter_BehavesSame(t *testing.T) {
	limiter := NewMemoryLoginLimiter(5, 15*time.Minute)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}
	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked)

	require.NoError(t, limiter.Reset(ctx, "a@b.c"))
	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestMemoryLoginLimiter_LockoutExpires(t *testing.T) {
	now := time.Now()
	limiter := newMemoryLoginLimiterWithClock(5, 15*time.Minute, func() time.Time { return now })
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}
	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked)

	now = now.Add(15*time.Minute + time.Second)
	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestMemoryLoginLimiter_LocksAtMaxFailuresBoundary(t *testing.T) {
	limiter := NewMemoryLoginLimiter(5, 15*time.Minute)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}
	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked, "maxFailures-1 failures must NOT lock")

	require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked, "maxFailures failures must lock")
}

func TestMemoryLoginLimiter_RefreshesWindowOnEachFailure(t *testing.T) {
	now := time.Now()
	limiter := newMemoryLoginLimiterWithClock(5, 15*time.Minute, func() time.Time { return now })
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}

	now = now.Add(10 * time.Minute)
	require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked, "rolling window: recent failure must keep lock")

	now = now.Add(5*time.Minute + time.Second)
	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked, "last failure must extend the lockout window")

	now = now.Add(15*time.Minute + time.Second)
	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked, "lockout must expire after full window from last failure")
}
