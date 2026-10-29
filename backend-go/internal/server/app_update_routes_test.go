package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/config"
)

func newAppUpdateTestServer(t *testing.T, upstreamURL string) *Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{LogLevel: "info"},
		JWT:    config.JWTConfig{Secret: "test-secret", ExpiryHours: 24},
		Auth:   config.AuthConfig{MaxFailures: 5, LockoutMinutes: 15},
		AI:     config.AIConfig{CooldownSeconds: 60},
		AppUpdate: config.AppUpdateConfig{
			BaseURL:      upstreamURL,
			CacheSeconds: 600,
		},
	}
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
}

func TestAppUpdateRoute_Check(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version.json":
			_, _ = io.WriteString(w, `{"version_code":7,"version_name":"0.0.7","asset":"wanli-latest.apk","sha256":"deadbeef","size":3,"commit":"abc1234","published_at":"2026-10-29T00:00:00Z"}`)
		case "/wanli-latest.apk":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = io.WriteString(w, "apk")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(upstream.Close)

	srv := newAppUpdateTestServer(t, upstream.URL)

	w := performJSONRequest(srv, http.MethodGet, "/api/v1/app/update", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			VersionCode  int    `json:"version_code"`
			VersionName  string `json:"version_name"`
			SHA256       string `json:"sha256"`
			DownloadPath string `json:"download_path"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 7, resp.Data.VersionCode)
	assert.Equal(t, "0.0.7", resp.Data.VersionName)
	assert.Equal(t, "deadbeef", resp.Data.SHA256)
	assert.Equal(t, "/api/v1/app/download", resp.Data.DownloadPath)

	// 下载代理返回 APK 字节与安装包 MIME
	dw := performJSONRequest(srv, http.MethodGet, "/api/v1/app/download", "")
	require.Equal(t, http.StatusOK, dw.Code, dw.Body.String())
	assert.Equal(t, "application/vnd.android.package-archive", dw.Header().Get("Content-Type"))
	assert.Equal(t, "apk", dw.Body.String())
}

func TestAppUpdateRoute_UnavailableWhenUpstreamDown(t *testing.T) {
	srv := newAppUpdateTestServer(t, "http://127.0.0.1:1")

	w := performJSONRequest(srv, http.MethodGet, "/api/v1/app/update", "")
	assertErrorCode(t, w, "APP_UPDATE_UNAVAILABLE")
}
