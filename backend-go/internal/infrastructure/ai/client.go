// Package ai 封装与 LLM（OpenAI 兼容接口）的交互：视频文本 → 结构化菜谱。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wanli/backend-go/internal/model"
)

// SubtitleInput 是一条带时间轴的字幕输入（时间单位：秒）。
type SubtitleInput struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// ParseInput 是 AI 解析的输入（对应 docs/ai-parsing.md §2.1）。
type ParseInput struct {
	BVID        string          `json:"bvid"`
	Title       string          `json:"title,omitempty"`
	Subtitles   []SubtitleInput `json:"subtitles"`
	Description string          `json:"description,omitempty"`
	Comments    []string        `json:"comments,omitempty"`
}

// AdaptationInput 是厨房适配输入（对应 docs/ai-parsing.md §3.1）。
type AdaptationInput struct {
	RecipeVersionID     string                   `json:"recipe_version_id"`
	OriginalRecipe      *model.ParsedRecipe      `json:"original_recipe"`
	KitchenProfile      AdaptationKitchenProfile `json:"kitchen_profile"`
	DietaryRestrictions []AdaptationRestriction  `json:"dietary_restrictions"`
}

type AdaptationKitchenProfile struct {
	OwnedIngredients []string `json:"owned_ingredients"`
	OwnedTools       []string `json:"owned_tools"`
}

type AdaptationRestriction struct {
	RestrictionType string `json:"restriction_type"`
	IngredientName  string `json:"ingredient_name"`
}

// HasContent 判断是否有任何可用文本来源。
func (in ParseInput) HasContent() bool {
	return len(in.Subtitles) > 0 || in.Description != "" || len(in.Comments) > 0
}

// Client 是 OpenAI 兼容协议的 AI 客户端。
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// New 创建 AI 客户端。baseURL 形如 https://api.deepseek.com/v1。
func New(apiKey, baseURL, model string) *Client {
	return NewWithHTTPClient(apiKey, baseURL, model, &http.Client{Timeout: 120 * time.Second})
}

// NewWithHTTPClient 创建使用自定义 http.Client 的 AI 客户端（便于测试注入）。
func NewWithHTTPClient(apiKey, baseURL, model string, httpClient *http.Client) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: httpClient,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ExtractRecipe 调用 AI 将视频文本解析为结构化菜谱。
// 返回错误统一以 "AI解析失败" 前缀包装，由上层映射为 AI_PARSE_FAILED。
func (c *Client) ExtractRecipe(ctx context.Context, input ParseInput) (*model.ParsedRecipe, error) {
	if !input.HasContent() {
		return nil, fmt.Errorf("AI解析失败: 无任何可用文本来源")
	}

	userContent, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("AI解析失败: 序列化输入: %w", err)
	}

	reqBody := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: parseSystemPrompt},
			{Role: "user", Content: string(userContent)},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      4096,
		Temperature:    0.1, // 低温度，减少编造
	}

	resp, err := c.callAPI(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("AI解析失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AI解析失败: AI 返回空结果")
	}

	content := stripCodeFence(resp.Choices[0].Message.Content)
	var recipe model.ParsedRecipe
	if err := json.Unmarshal([]byte(content), &recipe); err != nil {
		return nil, fmt.Errorf("AI解析失败: 输出 JSON 无法解析: %w", err)
	}
	if err := recipe.Validate(); err != nil {
		return nil, fmt.Errorf("AI解析失败: 输出未通过 schema 校验: %w", err)
	}
	return &recipe, nil
}

// AdaptRecipe 根据厨房配置和饮食限制生成结构化个人适配版。
func (c *Client) AdaptRecipe(ctx context.Context, input AdaptationInput) (*model.AdaptationResult, error) {
	userContent, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("AI适配失败: 序列化输入: %w", err)
	}
	reqBody := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: adaptationSystemPrompt},
			{Role: "user", Content: string(userContent)},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      4096,
		Temperature:    0.1,
	}
	resp, err := c.callAPI(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("AI适配失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AI适配失败: AI 返回空结果")
	}

	content := stripCodeFence(resp.Choices[0].Message.Content)
	var result model.AdaptationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("AI适配失败: 输出 JSON 无法解析: %w", err)
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("AI适配失败: 输出未通过 schema 校验: %w", err)
	}
	return &result, nil
}

func (c *Client) callAPI(ctx context.Context, reqBody chatCompletionRequest) (*chatCompletionResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回状态码 %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API 错误: %s", result.Error.Message)
	}
	return &result, nil
}

// stripCodeFence 去掉模型可能输出的 Markdown 代码块包裹。
func stripCodeFence(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	if i := strings.Index(content, "\n"); i >= 0 {
		content = content[i+1:]
	}
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// parseSystemPrompt 与 docs/ai-parsing.md §2.3 的 Prompt 核心约束保持一致。
const parseSystemPrompt = `你是一位严谨的家庭菜谱结构化助手。请根据用户提供的视频字幕（可选含视频简介与评论），生成一份结构化菜谱 JSON。

来源优先级：
1. 字幕内容最可信，包含操作步骤和顺序。
2. 视频简介中的材料和配方可信，常包含文字版菜谱。
3. 置顶/高赞评论只作补充参考；若只是用户评价，不要当成菜谱步骤。
4. 不同来源冲突时，以字幕为准。

规则：
1. 必须完全忠实于来源文本，不得编造未提及的用量、火候或步骤。
2. 当来源没有给出具体用量时，amount 字段必须设为 null；若来源使用“少许、适量、一点点”等模糊词，请原样保留。
3. 若你补充了来源未明确提及但菜品必需的食材/步骤，必须设置 is_explicit = false。
4. 每个步骤的 video_timestamp 必须是该步骤在字幕中首次出现的时间（秒）；无字幕来源或无法对应时设为 null。
5. 仅来源于简介/评论、缺乏来源支撑的内容，相关 amount 置为 null，并在 instructions 中说明不确定性。
6. 如果无法从来源中提取有效菜谱，返回 ingredients 和 steps 为空数组的 JSON。
7. 不要输出任何解释性文字，只输出符合 schema 的 JSON。

JSON 格式如下：
{
  "title": "菜名",
  "servings": "2-3人份 或 null",
  "estimated_minutes": 45,
  "ingredients": [
    { "name": "五花肉", "amount": "500克 或 null 或 适量", "category": "MAIN | AUXILIARY | SEASONING", "is_explicit": true }
  ],
  "steps": [
    { "step_order": 1, "description": "五花肉切块焯水", "video_timestamp": 45, "duration_seconds": 300, "heat": "大火 或 null" }
  ],
  "instructions": "提示与注意事项 或 null"
}`

const adaptationSystemPrompt = `你是一位注重安全的家庭菜谱适配助手。用户会提供原菜谱、厨房调料/厨具和忌口信息。只输出 JSON。

安全规则（最高优先级）：
1. 原菜谱含有用户过敏或医疗限制食材且无法安全替代时，safe=false 并填写 block_reason。
2. safe=true 的菜谱标题、实际配料、步骤和提示中绝不能包含用户过敏食材。
3. 只能使用用户已有调料/厨具或明确安全的替代品，禁止发明与原菜谱无关的烹饪步骤。
4. 不得估算来源没有给出的用量；不确定时 amount=null。
5. 每次替换必须填写 substitute_for，并在 adaptation_notes.substitutions 中解释。

JSON 格式：
{
  "safe": true,
  "block_reason": null,
  "title": "适配后的菜名",
  "servings": "2人份 或 null",
  "estimated_minutes": 30,
  "ingredients": [
    {"name":"替代食材","amount":null,"category":"MAIN | AUXILIARY | SEASONING","is_explicit":false,"substitute_for":"原食材 或 null"}
  ],
  "steps": [
    {"step_order":1,"description":"步骤","video_timestamp":null,"duration_seconds":null,"heat":null}
  ],
  "instructions": "提示 或 null",
  "adaptation_notes": {
    "missing_ingredients": [],
    "missing_tools": [],
    "substitutions": [{"original":"原食材","replacement":"替代食材","reason":"原因"}],
    "flavor_changes": []
  }
}`
