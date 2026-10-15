package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/config"
	"github.com/wanli/backend-go/pkg/response"
)

// newAuthTestServer 构建无 DB / Redis / 存储的测试服务器。
// TURNSTILE_SECRET_KEY 为空 → 使用闭锁 Stub，人机验证恒失败（fail-closed）。
func newAuthTestServer(t *testing.T, siteKey string) *Server {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{
			LogLevel: "info",
		},
		JWT: config.JWTConfig{
			Secret:      "test-secret",
			ExpiryHours: 24,
		},
		Turnstile: config.TurnstileConfig{
			SiteKey:   siteKey,
			SecretKey: "", // 空 secret key → turnstile.Stub，恒返回未通过
		},
		Auth: config.AuthConfig{
			MaxFailures:    5,
			LockoutMinutes: 15,
		},
		AI: config.AIConfig{
			CooldownSeconds: 60,
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(cfg, logger, nil, nil, nil)
}

// performJSONRequest 以 application/json 请求体发起请求，返回 recorder。
func performJSONRequest(srv *Server, method, path, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w
}

// assertErrorCode 断言响应为失败 JSON 且错误码匹配。
func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "响应不是合法 JSON: %s", w.Body.String())
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, wantCode, resp.Error.Code)
}

func TestAuthRoutes_Register_CaptchaFailed(t *testing.T) {
	srv := newAuthTestServer(t, "1x00000000000000000000AA")
	body := `{"email":"cook@example.com","password":"password123","turnstile_token":"dummy-token"}`

	w := performJSONRequest(srv, http.MethodPost, "/api/v1/auth/register", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorCode(t, w, "CAPTCHA_FAILED")
}

func TestAuthRoutes_Login_CaptchaFailed(t *testing.T) {
	srv := newAuthTestServer(t, "1x00000000000000000000AA")
	body := `{"email":"cook@example.com","password":"password123","turnstile_token":"dummy-token"}`

	w := performJSONRequest(srv, http.MethodPost, "/api/v1/auth/login", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorCode(t, w, "CAPTCHA_FAILED")
}

func TestAuthRoutes_Register_ValidationError(t *testing.T) {
	srv := newAuthTestServer(t, "1x00000000000000000000AA")
	// 缺少必填的 turnstile_token
	body := `{"email":"cook@example.com","password":"password123"}`

	w := performJSONRequest(srv, http.MethodPost, "/api/v1/auth/register", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorCode(t, w, "VALIDATION_ERROR")
}

func TestAuthRoutes_CaptchaWidget_ContainsSiteKey(t *testing.T) {
	const siteKey = "1x00000000000000000000AA"
	srv := newAuthTestServer(t, siteKey)

	w := performJSONRequest(srv, http.MethodGet, "/captcha", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "text/html"))
	assert.Contains(t, w.Body.String(), siteKey)
}
