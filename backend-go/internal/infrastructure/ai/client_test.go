package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/model"
)

func validRecipeJSON() string {
	return `{
		"title": "家常红烧肉",
		"servings": "2-3人份",
		"estimated_minutes": 45,
		"ingredients": [
			{"name": "五花肉", "amount": "500克", "category": "MAIN", "is_explicit": true},
			{"name": "盐", "amount": null, "category": "SEASONING", "is_explicit": false}
		],
		"steps": [
			{"step_order": 1, "description": "五花肉切块焯水", "video_timestamp": 45, "duration_seconds": 300, "heat": null},
			{"step_order": 2, "description": "小火慢炖40分钟", "video_timestamp": 120, "duration_seconds": null, "heat": "小火"}
		],
		"instructions": "焯水要冷水下锅"
	}`
}

// newFakeAIServer 返回模拟 OpenAI 兼容接口的服务器，content 为 AI 输出的 message.content。
func newFakeAIServer(t *testing.T, content string, assertReq func(*testing.T, chatCompletionRequest)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req chatCompletionRequest
		require.NoError(t, json.Unmarshal(body, &req))
		if assertReq != nil {
			assertReq(t, req)
		}

		resp := fmt.Sprintf(`{"choices": [{"message": {"content": %s}}]}`, mustJSONString(t, content))
		fmt.Fprint(w, resp)
	}))
	t.Cleanup(server.Close)
	return server
}

func mustJSONString(t *testing.T, s string) string {
	t.Helper()
	data, err := json.Marshal(s)
	require.NoError(t, err)
	return string(data)
}

func sampleInput() ParseInput {
	return ParseInput{
		BVID:  "BV1xx411c7xx",
		Title: "家常红烧肉",
		Subtitles: []SubtitleInput{
			{Start: 0, End: 5, Text: "今天做一道家常红烧肉"},
			{Start: 6, End: 12, Text: "准备五花肉500克"},
		},
		Description: "简介：五花肉500克",
		Comments:    []string{"课代表：焯水后小火炖40分钟"},
	}
}

func TestExtractRecipe_Success(t *testing.T) {
	server := newFakeAIServer(t, validRecipeJSON(), func(t *testing.T, req chatCompletionRequest) {
		// 必须请求结构化 JSON 输出
		require.NotNil(t, req.ResponseFormat)
		assert.Equal(t, "json_object", req.ResponseFormat.Type)
		assert.Equal(t, "deepseek-chat", req.Model)
		require.Len(t, req.Messages, 2)
		assert.Equal(t, "system", req.Messages[0].Role)
		// 输入文本应原样进入 user 消息
		assert.Contains(t, req.Messages[1].Content, "准备五花肉500克")
		assert.Contains(t, req.Messages[1].Content, "课代表")
	})

	client := New("test-key", server.URL, "deepseek-chat")
	recipe, err := client.ExtractRecipe(context.Background(), sampleInput())
	require.NoError(t, err)

	assert.Equal(t, "家常红烧肉", recipe.Title)
	assert.Equal(t, "2-3人份", *recipe.Servings)
	require.Len(t, recipe.Ingredients, 2)
	assert.Nil(t, recipe.Ingredients[1].Amount, "AI 输出的 null 必须保留")
	assert.False(t, recipe.Ingredients[1].IsExplicit)
	require.Len(t, recipe.Steps, 2)
	assert.Equal(t, 45, *recipe.Steps[0].VideoTimestamp)
	assert.Nil(t, recipe.Steps[1].DurationSeconds)
}

func TestExtractRecipe_CodeFenceOutput(t *testing.T) {
	fenced := "```json\n" + validRecipeJSON() + "\n```"
	server := newFakeAIServer(t, fenced, nil)

	client := New("test-key", server.URL, "deepseek-chat")
	recipe, err := client.ExtractRecipe(context.Background(), sampleInput())
	require.NoError(t, err)
	assert.Equal(t, "家常红烧肉", recipe.Title)
}

func TestExtractRecipe_InvalidJSON(t *testing.T) {
	server := newFakeAIServer(t, "这不是 JSON", nil)

	client := New("test-key", server.URL, "deepseek-chat")
	_, err := client.ExtractRecipe(context.Background(), sampleInput())
	assert.ErrorContains(t, err, "AI解析失败")
}

func TestExtractRecipe_SchemaViolation(t *testing.T) {
	// title 缺失 + ingredients 为空 → 未通过 schema 校验
	server := newFakeAIServer(t, `{"title": "", "ingredients": [], "steps": []}`, nil)

	client := New("test-key", server.URL, "deepseek-chat")
	_, err := client.ExtractRecipe(context.Background(), sampleInput())
	assert.ErrorContains(t, err, "AI解析失败")
	assert.ErrorContains(t, err, "schema")
}

func TestExtractRecipe_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error": {"message": "rate limited"}}`)
	}))
	defer server.Close()

	client := New("test-key", server.URL, "deepseek-chat")
	_, err := client.ExtractRecipe(context.Background(), sampleInput())
	assert.ErrorContains(t, err, "429")
}

func TestExtractRecipe_EmptyInput(t *testing.T) {
	client := New("test-key", "http://unused", "deepseek-chat")
	_, err := client.ExtractRecipe(context.Background(), ParseInput{BVID: "BV1xx411c7xx"})
	assert.ErrorContains(t, err, "无任何可用文本来源")
}

func TestParseInput_HasContent(t *testing.T) {
	assert.False(t, ParseInput{}.HasContent())
	assert.True(t, ParseInput{Description: "简介"}.HasContent())
	assert.True(t, ParseInput{Comments: []string{"评论"}}.HasContent())
	assert.True(t, ParseInput{Subtitles: []SubtitleInput{{Text: "字幕"}}}.HasContent())
}

func TestStripCodeFence(t *testing.T) {
	assert.Equal(t, `{"a":1}`, stripCodeFence("```json\n{\"a\":1}\n```"))
	assert.Equal(t, `{"a":1}`, stripCodeFence("```\n{\"a\":1}\n```"))
	assert.Equal(t, `{"a":1}`, stripCodeFence(`{"a":1}`))
	assert.Equal(t, `{"a":1}`, stripCodeFence("  {\"a\":1}  "))
}

func TestAdaptRecipe_Success(t *testing.T) {
	content := `{
		"safe": true,
		"block_reason": null,
		"title": "芝麻酱拌面",
		"servings": "1人份",
		"estimated_minutes": 15,
		"ingredients": [
			{"name": "芝麻酱", "amount": "2勺", "category": "SEASONING", "is_explicit": true, "substitute_for": "花生酱"}
		],
		"steps": [
			{"step_order": 1, "description": "加入芝麻酱拌匀", "video_timestamp": null, "duration_seconds": null, "heat": null}
		],
		"instructions": null,
		"adaptation_notes": {
			"missing_ingredients": [],
			"missing_tools": [],
			"substitutions": [{"original":"花生酱","replacement":"芝麻酱","reason":"规避过敏源"}],
			"flavor_changes": ["香气会有变化"]
		}
	}`
	server := newFakeAIServer(t, content, func(t *testing.T, req chatCompletionRequest) {
		require.NotNil(t, req.ResponseFormat)
		assert.Equal(t, "json_object", req.ResponseFormat.Type)
		assert.Contains(t, req.Messages[0].Content, "过敏")
		assert.Contains(t, req.Messages[1].Content, `"ingredient_name":"花生"`)
	})

	client := New("test-key", server.URL, "deepseek-chat")
	result, err := client.AdaptRecipe(context.Background(), AdaptationInput{
		RecipeVersionID: "version-id",
		OriginalRecipe: &model.ParsedRecipe{
			Title: "麻酱拌面",
			Ingredients: []model.ParsedIngredient{{
				Name: "花生酱", Category: model.IngredientCategorySeasoning, IsExplicit: true,
			}},
			Steps: []model.ParsedStep{{StepOrder: 1, Description: "拌匀"}},
		},
		KitchenProfile: AdaptationKitchenProfile{OwnedIngredients: []string{"芝麻酱"}},
		DietaryRestrictions: []AdaptationRestriction{{
			RestrictionType: model.RestrictionTypeAllergy,
			IngredientName:  "花生",
		}},
	})
	require.NoError(t, err)
	assert.True(t, result.Safe)
	assert.Equal(t, "芝麻酱", result.Ingredients[0].Name)
	assert.Equal(t, "花生酱", *result.Ingredients[0].SubstituteFor)
	require.Len(t, result.AdaptationNotes.Substitutions, 1)
}

func TestAdaptRecipe_UnsafeRequiresBlockReason(t *testing.T) {
	server := newFakeAIServer(t, `{"safe":false,"block_reason":null}`, nil)
	client := New("test-key", server.URL, "deepseek-chat")

	_, err := client.AdaptRecipe(context.Background(), AdaptationInput{})
	assert.ErrorContains(t, err, "schema")
	assert.ErrorContains(t, err, "block_reason")
}
