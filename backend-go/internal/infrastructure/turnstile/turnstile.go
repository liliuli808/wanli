package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Verifier 校验前端提交的 Turnstile token。
type Verifier interface {
	Verify(ctx context.Context, token, remoteIP string) (bool, error)
}

// CloudflareTurnstile 调用 challenges.cloudflare.com siteverify 的客户端。
type CloudflareTurnstile struct {
	secretKey string
	endpoint  string
	client    *http.Client
}

// New 创建指向 Cloudflare 官方 siteverify 端点的客户端。
func New(secretKey string) *CloudflareTurnstile {
	return &CloudflareTurnstile{
		secretKey: secretKey,
		endpoint:  "https://challenges.cloudflare.com/turnstile/v0/siteverify",
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// NewWithEndpoint 指定 endpoint（测试用）。
func NewWithEndpoint(secretKey, endpoint string, client *http.Client) *CloudflareTurnstile {
	return &CloudflareTurnstile{secretKey: secretKey, endpoint: endpoint, client: client}
}

type siteVerifyResponse struct {
	Success bool `json:"success"`
}

// Verify 返回 token 是否通过人机验证。token 为空直接返回 (false, nil)。
func (c *CloudflareTurnstile) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if token == "" {
		return false, nil
	}
	form := url.Values{}
	form.Set("secret", c.secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, fmt.Errorf("build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("siteverify request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("siteverify unexpected status: %d", resp.StatusCode)
	}
	var out siteVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("decode siteverify response: %w", err)
	}
	return out.Success, nil
}

// Stub 是未配置 secret key 时的闭锁实现，恒返回未通过，防止误放行。
type Stub struct{}

// Verify 恒返回 false。
func (Stub) Verify(context.Context, string, string) (bool, error) {
	return false, nil
}
