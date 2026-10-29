package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newReleaseServer 模拟 GitHub Releases 的 latest 目录。
func newReleaseServer(t *testing.T, versionJSON, apkBody string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var versionHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version.json":
			versionHits.Add(1)
			_, _ = io.WriteString(w, versionJSON)
		case "/wanli-latest.apk":
			_, _ = io.WriteString(w, apkBody)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &versionHits
}

const validVersionJSON = `{
  "version_code": 42,
  "version_name": "0.0.42",
  "asset": "wanli-latest.apk",
  "sha256": "abc123",
  "size": 11,
  "commit": "d830e28",
  "published_at": "2026-10-29T00:00:00Z"
}`

func TestAppUpdateLatest(t *testing.T) {
	srv, hits := newReleaseServer(t, validVersionJSON, "fake-apk-hex")

	svc := NewAppUpdateService(srv.URL, time.Hour)
	release, err := svc.Latest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 42, release.VersionCode)
	assert.Equal(t, "0.0.42", release.VersionName)
	assert.Equal(t, "wanli-latest.apk", release.Asset)

	// 缓存生效：第二次不再请求上游
	_, err = svc.Latest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(1), hits.Load())
}

func TestAppUpdateLatest_CacheExpires(t *testing.T) {
	srv, hits := newReleaseServer(t, validVersionJSON, "fake-apk-hex")

	now := time.Now()
	svc := NewAppUpdateService(srv.URL, time.Minute)
	svc.now = func() time.Time { return now }

	_, err := svc.Latest(context.Background())
	require.NoError(t, err)

	now = now.Add(2 * time.Minute)
	_, err = svc.Latest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(2), hits.Load())
}

func TestAppUpdateLatest_RejectsIllegalAsset(t *testing.T) {
	srv, _ := newReleaseServer(t, `{"version_code":1,"version_name":"1.0","asset":"../../etc/passwd"}`, "")

	svc := NewAppUpdateService(srv.URL, time.Hour)
	_, err := svc.Latest(context.Background())
	require.Error(t, err)
}

func TestAppUpdateLatest_RejectsIncompleteRelease(t *testing.T) {
	srv, _ := newReleaseServer(t, `{"version_code":0,"version_name":""}`, "")

	svc := NewAppUpdateService(srv.URL, time.Hour)
	_, err := svc.Latest(context.Background())
	require.Error(t, err)
}

func TestAppUpdateLatest_FallsBackToCacheOnUpstreamFailure(t *testing.T) {
	srv, hits := newReleaseServer(t, validVersionJSON, "")

	now := time.Now()
	svc := NewAppUpdateService(srv.URL, time.Minute)
	svc.now = func() time.Time { return now }
	_, err := svc.Latest(context.Background())
	require.NoError(t, err)

	// 上游挂掉：返回上次成功的缓存
	srv.Close()
	now = now.Add(2 * time.Minute)
	release, err := svc.Latest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 42, release.VersionCode)
	assert.Equal(t, int32(1), hits.Load())
}

func TestAppUpdateLatest_Disabled(t *testing.T) {
	svc := NewAppUpdateService("", time.Hour)
	assert.False(t, svc.Enabled())
	_, err := svc.Latest(context.Background())
	require.Error(t, err)
}

func TestAppUpdateOpen(t *testing.T) {
	srv, _ := newReleaseServer(t, validVersionJSON, "fake-apk-hex")

	svc := NewAppUpdateService(strings.TrimRight(srv.URL, "/")+"/", time.Hour)
	body, size, err := svc.Open(context.Background())
	require.NoError(t, err)
	defer body.Close()

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "fake-apk-hex", string(data))
	assert.Equal(t, int64(len("fake-apk-hex")), size)
}

func TestAppUpdateOpen_UpstreamMissing(t *testing.T) {
	srv, _ := newReleaseServer(t, `{"version_code":1,"version_name":"1.0","asset":"missing.apk"}`, "")

	svc := NewAppUpdateService(srv.URL, time.Hour)
	_, _, err := svc.Open(context.Background())
	require.Error(t, err)
}
