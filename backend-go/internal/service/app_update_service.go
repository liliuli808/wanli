package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// assetNameRegex 限制下载文件名，避免把上游返回的任意字符串拼进 URL。
var assetNameRegex = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// AppRelease 是 CI 发布的 version.json 内容。
type AppRelease struct {
	VersionCode int    `json:"version_code"`
	VersionName string `json:"version_name"`
	Asset       string `json:"asset"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Commit      string `json:"commit"`
	PublishedAt string `json:"published_at"`
}

// AppUpdateService 读取 CI 发布到 GitHub Releases 的最新版本信息。
//
// 客户端不直接访问 GitHub（国内经常不可达），由后端代理 version.json 与 APK。
type AppUpdateService struct {
	baseURL string
	client  *http.Client
	// downloadClient 不设总超时：APK 体积较大，靠请求上下文取消。
	downloadClient *http.Client
	ttl            time.Duration
	now            func() time.Time

	mu       sync.Mutex
	cached   *AppRelease
	cachedAt time.Time
}

// NewAppUpdateService 创建服务。baseURL 指向 latest release 的下载目录，
// 例如 https://github.com/liliuli808/wanli/releases/download/latest。
func NewAppUpdateService(baseURL string, ttl time.Duration) *AppUpdateService {
	return &AppUpdateService{
		baseURL:        trimTrailingSlash(baseURL),
		client:         &http.Client{Timeout: 10 * time.Second},
		downloadClient: &http.Client{},
		ttl:            ttl,
		now:            time.Now,
	}
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// Enabled 表示是否配置了上游地址。
func (s *AppUpdateService) Enabled() bool {
	return s != nil && s.baseURL != ""
}

// Latest 返回最新发布信息，命中缓存时不再请求上游。
func (s *AppUpdateService) Latest(ctx context.Context) (*AppRelease, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("未配置 APP_UPDATE_BASE_URL")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil && s.now().Sub(s.cachedAt) < s.ttl {
		return s.cached, nil
	}

	release, err := s.fetch(ctx)
	if err != nil {
		// 上游短暂不可用时继续用旧缓存，避免 App 因检查更新失败而报错
		if s.cached != nil {
			return s.cached, nil
		}
		return nil, err
	}
	s.cached = release
	s.cachedAt = s.now()
	return release, nil
}

func (s *AppUpdateService) fetch(ctx context.Context) (*AppRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/version.json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求版本信息失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("版本信息返回状态码 %d", resp.StatusCode)
	}

	var release AppRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %w", err)
	}
	if release.VersionCode <= 0 || release.VersionName == "" {
		return nil, fmt.Errorf("版本信息不完整")
	}
	if !assetNameRegex.MatchString(release.Asset) {
		return nil, fmt.Errorf("版本信息中的文件名非法: %q", release.Asset)
	}
	return &release, nil
}

// Open 打开最新 APK 的读取流，返回内容、大小与内容类型。
func (s *AppUpdateService) Open(ctx context.Context) (io.ReadCloser, int64, error) {
	release, err := s.Latest(ctx)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/"+release.Asset, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.downloadClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("下载安装包失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("下载安装包返回状态码 %d", resp.StatusCode)
	}
	return resp.Body, resp.ContentLength, nil
}
