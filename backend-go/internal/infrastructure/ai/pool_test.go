package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// chatOKResponse 返回一个合法的 chat completion 响应体。
func chatOKResponse(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("编码响应失败: %v", err)
	}
}

const poolValidRecipeJSON = `{
  "title": "番茄炒蛋",
  "servings": "2人份",
  "estimated_minutes": 15,
  "ingredients": [{"name": "番茄", "amount": "2个", "category": "MAIN", "is_explicit": true}],
  "steps": [{"step_order": 1, "description": "番茄切块", "video_timestamp": 10, "duration_seconds": 60, "heat": "大火"}],
  "instructions": null
}`

func parseInput() ParseInput {
	return ParseInput{BVID: "BV1xx", Description: "番茄炒蛋的做法"}
}

func TestPoolFailoverOnServerError(t *testing.T) {
	var calls1, calls2 atomic.Int32
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls1.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls2.Add(1)
		chatOKResponse(t, w, poolValidRecipeJSON)
	}))
	defer good.Close()

	pool := NewPool([]Provider{
		{Name: "bad", BaseURL: bad.URL + "/v1", APIKey: "k1", Model: "m"},
		{Name: "good", BaseURL: good.URL + "/v1", APIKey: "k2", Model: "m"},
	}, time.Minute, nil)

	recipe, err := pool.ExtractRecipe(context.Background(), parseInput())
	if err != nil {
		t.Fatalf("期望故障切换成功，得到错误: %v", err)
	}
	if recipe.Title != "番茄炒蛋" {
		t.Fatalf("标题不符: %s", recipe.Title)
	}
	if calls1.Load() != 1 || calls2.Load() != 1 {
		t.Fatalf("调用次数: bad=%d good=%d，期望各 1 次", calls1.Load(), calls2.Load())
	}

	// 第二次调用：bad 在冷却中应被直接跳过
	if _, err := pool.ExtractRecipe(context.Background(), parseInput()); err != nil {
		t.Fatalf("第二次调用失败: %v", err)
	}
	if calls1.Load() != 1 {
		t.Fatalf("冷却中的 bad 被再次调用（共 %d 次）", calls1.Load())
	}
	if calls2.Load() != 2 {
		t.Fatalf("good 应被调用 2 次，实际 %d", calls2.Load())
	}
}

func TestPoolFailoverOnInvalidJSON(t *testing.T) {
	garbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatOKResponse(t, w, "这不是 JSON")
	}))
	defer garbage.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatOKResponse(t, w, poolValidRecipeJSON)
	}))
	defer good.Close()

	pool := NewPool([]Provider{
		{Name: "garbage", BaseURL: garbage.URL + "/v1", APIKey: "k", Model: "m"},
		{Name: "good", BaseURL: good.URL + "/v1", APIKey: "k", Model: "m"},
	}, time.Minute, nil)

	if _, err := pool.ExtractRecipe(context.Background(), parseInput()); err != nil {
		t.Fatalf("非法 JSON 应触发切换: %v", err)
	}
}

func TestPoolAllDownReturnsAggregatedError(t *testing.T) {
	bad1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer bad1.Close()
	bad2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer bad2.Close()

	pool := NewPool([]Provider{
		{Name: "bad1", BaseURL: bad1.URL + "/v1", APIKey: "k", Model: "m"},
		{Name: "bad2", BaseURL: bad2.URL + "/v1", APIKey: "k", Model: "m"},
	}, time.Minute, nil)

	_, err := pool.ExtractRecipe(context.Background(), parseInput())
	if err == nil {
		t.Fatal("全部失败时应返回错误")
	}
	msg := err.Error()
	if !strings.Contains(msg, "bad1") || !strings.Contains(msg, "bad2") {
		t.Fatalf("错误应汇总各端点信息: %s", msg)
	}
}

func TestPoolAllCoolingDownResetsAndRetries(t *testing.T) {
	var calls atomic.Int32
	flaky := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError) // 第一次失败进入冷却
			return
		}
		chatOKResponse(t, w, poolValidRecipeJSON)
	}))
	defer flaky.Close()

	pool := NewPool([]Provider{
		{Name: "flaky", BaseURL: flaky.URL + "/v1", APIKey: "k", Model: "m"},
	}, time.Minute, nil)

	if _, err := pool.ExtractRecipe(context.Background(), parseInput()); err == nil {
		t.Fatal("第一次调用应失败")
	}
	// 端点处于冷却中，但池应重置冷却兜底重试（而不是直接报"全部冷却"）
	if _, err := pool.ExtractRecipe(context.Background(), parseInput()); err != nil {
		t.Fatalf("冷却兜底重试应成功: %v", err)
	}
}

func TestPoolRoundRobinStart(t *testing.T) {
	var calls1, calls2 atomic.Int32
	mk := func(counter *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter.Add(1)
			chatOKResponse(t, w, poolValidRecipeJSON)
		}))
	}
	s1 := mk(&calls1)
	defer s1.Close()
	s2 := mk(&calls2)
	defer s2.Close()

	pool := NewPool([]Provider{
		{BaseURL: s1.URL + "/v1", APIKey: "k", Model: "m"},
		{BaseURL: s2.URL + "/v1", APIKey: "k", Model: "m"},
	}, time.Minute, nil)

	for i := 0; i < 4; i++ {
		if _, err := pool.ExtractRecipe(context.Background(), parseInput()); err != nil {
			t.Fatalf("第 %d 次调用失败: %v", i+1, err)
		}
	}
	if calls1.Load() != 2 || calls2.Load() != 2 {
		t.Fatalf("轮询应均匀分配（2/2），实际 %d/%d", calls1.Load(), calls2.Load())
	}
}

func TestPoolEmptyReturnsError(t *testing.T) {
	pool := NewPool(nil, time.Minute, nil)
	if pool.Size() != 0 {
		t.Fatalf("Size 应为 0，实际 %d", pool.Size())
	}
	if _, err := pool.ExtractRecipe(context.Background(), parseInput()); err == nil {
		t.Fatal("空池应返回错误")
	}
}

func TestPoolExtractRejectsEmptyInput(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		chatOKResponse(t, w, poolValidRecipeJSON)
	}))
	defer s.Close()

	pool := NewPool([]Provider{{BaseURL: s.URL + "/v1", APIKey: "k", Model: "m"}}, time.Minute, nil)
	if _, err := pool.ExtractRecipe(context.Background(), ParseInput{BVID: "BV1xx"}); err == nil {
		t.Fatal("空输入应报错")
	}
	if calls.Load() != 0 {
		t.Fatalf("空输入不应产生任何 API 调用，实际 %d 次", calls.Load())
	}
}

func TestPoolAdaptRecipeFailover(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatOKResponse(t, w, `{
		  "safe": true, "block_reason": null, "title": "适配版番茄炒蛋",
		  "ingredients": [{"name":"番茄","amount":null,"category":"MAIN","is_explicit":false,"substitute_for":null}],
		  "steps": [{"step_order":1,"description":"炒","video_timestamp":null,"duration_seconds":null,"heat":null}],
		  "adaptation_notes": {"missing_ingredients": [], "missing_tools": [], "substitutions": [], "flavor_changes": []}
		}`)
	}))
	defer good.Close()

	pool := NewPool([]Provider{
		{Name: "bad", BaseURL: bad.URL + "/v1", APIKey: "k", Model: "m"},
		{Name: "good", BaseURL: good.URL + "/v1", APIKey: "k", Model: "m"},
	}, time.Minute, nil)

	result, err := pool.AdaptRecipe(context.Background(), AdaptationInput{RecipeVersionID: "v1"})
	if err != nil {
		t.Fatalf("适配应故障切换成功: %v", err)
	}
	if result.Title != "适配版番茄炒蛋" {
		t.Fatalf("标题不符: %s", result.Title)
	}
}
