package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRegisterLimiter_AllowsUpToMaxWithinWindow(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	l := newMemoryRegisterLimiterWithClock(2, time.Hour, func() time.Time { return now })
	ctx := context.Background()

	ok, err := l.Allow(ctx, "1.2.3.4")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = l.Allow(ctx, "1.2.3.4")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = l.Allow(ctx, "1.2.3.4")
	require.NoError(t, err)
	assert.False(t, ok, "超过上限应拒绝")

	// 其他 IP 不受影响
	ok, err = l.Allow(ctx, "5.6.7.8")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestMemoryRegisterLimiter_ResetsAfterWindow(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	l := newMemoryRegisterLimiterWithClock(1, time.Hour, func() time.Time { return now })
	ctx := context.Background()

	ok, _ := l.Allow(ctx, "1.2.3.4")
	assert.True(t, ok)
	ok, _ = l.Allow(ctx, "1.2.3.4")
	assert.False(t, ok)

	now = now.Add(time.Hour + time.Second)
	ok, _ = l.Allow(ctx, "1.2.3.4")
	assert.True(t, ok, "窗口过期后应重新放行")
}

func TestRegisterLimiter_EmptyIPIsUnlimited(t *testing.T) {
	l := NewMemoryRegisterLimiter(1, time.Hour)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		ok, err := l.Allow(ctx, "")
		require.NoError(t, err)
		assert.True(t, ok)
	}
}
