package turnstile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient 起一个假 siteverify 服务，断言请求带 secret/response，返回固定 success。
func newTestClient(t *testing.T, success bool) *CloudflareTurnstile {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "test-secret", r.Form.Get("secret"))
		assert.Equal(t, "test-token", r.Form.Get("response"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": success})
	}))
	t.Cleanup(srv.Close)
	return NewWithEndpoint("test-secret", srv.URL, srv.Client())
}

func TestVerify_Success(t *testing.T) {
	c := newTestClient(t, true)
	ok, err := c.Verify(context.Background(), "test-token", "")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerify_Rejected(t *testing.T) {
	c := newTestClient(t, false)
	ok, err := c.Verify(context.Background(), "test-token", "")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerify_EmptyToken_ReturnsFalseWithoutRequest(t *testing.T) {
	c := &CloudflareTurnstile{} // 无 endpoint，若发请求会 panic，可证明未走 HTTP
	ok, err := c.Verify(context.Background(), "", "")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := NewWithEndpoint("test-secret", srv.URL, srv.Client())

	_, err := c.Verify(context.Background(), "test-token", "")
	require.Error(t, err)
}

func TestVerify_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	addr := srv.URL
	srv.Close() // 立刻关掉，制造连接错误

	c := NewWithEndpoint("test-secret", addr, srv.Client())
	_, err := c.Verify(context.Background(), "test-token", "")
	require.Error(t, err)
}
