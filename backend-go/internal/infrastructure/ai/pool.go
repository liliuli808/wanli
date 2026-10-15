package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanli/backend-go/internal/model"
)

// Provider 是一个 OpenAI 兼容端点（第三方 API 站）的配置。
type Provider struct {
	Name    string // 日志标识，缺省时用 BaseURL
	BaseURL string
	APIKey  string
	Model   string
}

// Pool 是多提供商 AI 客户端池：轮询分配起点，调用失败（网络错误/超时/5xx/429/
// 返回非法 JSON 等）后自动切换下一个端点，失败端点进入冷却期暂时跳过。
// 同时实现 ExtractRecipe 与 AdaptRecipe，可直接替代 *Client 注入各服务。
type Pool struct {
	entries  []*poolEntry
	cooldown time.Duration
	next     atomic.Uint64
	logger   *slog.Logger
}

type poolEntry struct {
	client *Client
	name   string

	mu            sync.Mutex
	cooldownUntil time.Time
}

// NewPool 创建 AI 池。cooldown 是单个端点失败后的冷却时长；providers 为空时
// 调用直接返回错误（等价于未配置 AI）。
func NewPool(providers []Provider, cooldown time.Duration, logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	entries := make([]*poolEntry, 0, len(providers))
	for i, p := range providers {
		name := p.Name
		if name == "" {
			name = p.BaseURL
		}
		if name == "" {
			name = fmt.Sprintf("provider-%d", i+1)
		}
		entries = append(entries, &poolEntry{
			client: New(p.APIKey, p.BaseURL, p.Model),
			name:   name,
		})
	}
	return &Pool{entries: entries, cooldown: cooldown, logger: logger}
}

// Size 返回池内端点数量。
func (p *Pool) Size() int { return len(p.entries) }

// ExtractRecipe 将视频文本解析为结构化菜谱，失败时自动切换端点重试。
func (p *Pool) ExtractRecipe(ctx context.Context, input ParseInput) (*model.ParsedRecipe, error) {
	// 输入本身不可用时不值得轮询所有端点
	if !input.HasContent() {
		return nil, fmt.Errorf("AI解析失败: 无任何可用文本来源")
	}
	return poolCall(ctx, p, "解析", func(c *Client) (*model.ParsedRecipe, error) {
		return c.ExtractRecipe(ctx, input)
	})
}

// AdaptRecipe 生成厨房适配版，失败时自动切换端点重试。
func (p *Pool) AdaptRecipe(ctx context.Context, input AdaptationInput) (*model.AdaptationResult, error) {
	return poolCall(ctx, p, "适配", func(c *Client) (*model.AdaptationResult, error) {
		return c.AdaptRecipe(ctx, input)
	})
}

// poolCall 从轮询起点开始依次尝试各端点：跳过冷却中的端点，失败则记录冷却并
// 换下一个；全部处于冷却时清空冷却兜底重试一轮。
// 注意：各端点是顺序尝试的，最坏耗时 = 端点数 × 单端点超时（120s），调用方
// 的 ctx 取消会中断当前请求并终止整个轮询。
func poolCall[T any](ctx context.Context, p *Pool, op string, fn func(*Client) (T, error)) (T, error) {
	var zero T
	if len(p.entries) == 0 {
		return zero, fmt.Errorf("AI%s失败: 未配置任何 AI 提供商", op)
	}

	result, tried, errs := tryEntries(ctx, p, p.ordered(), true, fn)
	if tried == 0 {
		// 全部冷却中：重置后兜底再试一轮，避免长时间整体不可用
		p.logger.Warn("所有 AI 提供商均在冷却期，重置冷却重试", "op", op)
		for _, e := range p.entries {
			e.resetCooldown()
		}
		result, _, errs = tryEntries(ctx, p, p.ordered(), false, fn)
	}
	if errs != nil {
		return zero, fmt.Errorf("所有 AI 提供商均失败: %s", strings.Join(errs, "; "))
	}
	return result, nil
}

// tryEntries 按顺序尝试端点。skipCooldown 为 true 时跳过冷却中的端点。
// 返回成功结果、实际尝试的端点数、以及各端点错误列表（全部失败时非 nil）。
// （Go 方法不支持类型参数，故泛型逻辑放在包级函数。）
func tryEntries[T any](ctx context.Context, p *Pool, entries []*poolEntry, skipCooldown bool, fn func(*Client) (T, error)) (T, int, []string) {
	var zero T
	now := time.Now()
	tried := 0
	var errs []string
	for _, e := range entries {
		if ctx.Err() != nil {
			break
		}
		if skipCooldown && e.inCooldown(now) {
			continue
		}
		tried++
		result, err := fn(e.client)
		if err != nil {
			e.markFailed(p.cooldown)
			p.logger.Warn("AI 提供商调用失败，切换下一个",
				"provider", e.name, "error", err, "cooldown", p.cooldown)
			errs = append(errs, fmt.Sprintf("[%s] %v", e.name, err))
			continue
		}
		e.resetCooldown()
		return result, tried, nil
	}
	return zero, tried, errs
}

// ordered 返回从轮询起点开始的端点快照（起点随调用轮转，首次从 0 号端点开始）。
func (p *Pool) ordered() []*poolEntry {
	n := uint64(len(p.entries))
	start := (p.next.Add(1) - 1) % n
	out := make([]*poolEntry, 0, n)
	for i := uint64(0); i < n; i++ {
		out = append(out, p.entries[(start+i)%n])
	}
	return out
}

func (e *poolEntry) inCooldown(now time.Time) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return now.Before(e.cooldownUntil)
}

func (e *poolEntry) markFailed(cooldown time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cooldownUntil = time.Now().Add(cooldown)
}

func (e *poolEntry) resetCooldown() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cooldownUntil = time.Time{}
}
