package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chdir 切换工作目录并在测试结束时还原（t.Chdir 需要 Go 1.24，本项目为 1.22）。
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestLoad_Defaults(t *testing.T) {
	// 在无 .env 的临时目录中加载，验证默认值
	chdir(t, t.TempDir())

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "wanli_db", cfg.Database.Name)
	assert.Equal(t, "127.0.0.1:6379", cfg.Redis.Addr())
	assert.Equal(t, "deepseek-chat", cfg.AI.Model)
	assert.Equal(t, "https://api.deepseek.com/v1", cfg.AI.BaseURL)
	assert.Empty(t, cfg.Bilibili.Cookie)
	assert.Empty(t, cfg.Turnstile.SiteKey)
	assert.Empty(t, cfg.Turnstile.SecretKey)
	assert.Equal(t, 5, cfg.Auth.MaxFailures)
	assert.Equal(t, 15, cfg.Auth.LockoutMinutes)
	assert.Equal(t, 168, cfg.JWT.ExpiryHours)
}

func TestLoad_EnvFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(
		"DB_PORT=5433\nREDIS_PORT=6380\nAI_MODEL=gpt-4o\nBILIBILI_COOKIE=SESSDATA=test-session; bili_jct=test-csrf\n",
	), 0o600))
	chdir(t, dir)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "127.0.0.1:6380", cfg.Redis.Addr())
	assert.Equal(t, "gpt-4o", cfg.AI.Model)
	assert.Equal(t, "SESSDATA=test-session; bili_jct=test-csrf", cfg.Bilibili.Cookie)
	// 未覆盖的项仍为默认值
	assert.Equal(t, "postgres", cfg.Database.User)
}

func TestLoad_OSEnvOverridesEnvFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_PORT=5433\n"), 0o600))
	chdir(t, dir)
	t.Setenv("DB_PORT", "5444")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 5444, cfg.Database.Port, "操作系统环境变量应优先于 .env 文件")
}

func TestLoad_OSEnv(t *testing.T) {
	chdir(t, t.TempDir())
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("TURNSTILE_SECRET_KEY", "test-secret")
	t.Setenv("BILIBILI_COOKIE", "SESSDATA=os-session")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "db.internal", cfg.Database.Host)
	assert.Equal(t, "test-secret", cfg.Turnstile.SecretKey)
	assert.Equal(t, "SESSDATA=os-session", cfg.Bilibili.Cookie)
}

func TestBuildAIProviders(t *testing.T) {
	// 仅旧版单提供商配置
	single := buildAIProviders("", "sk-single", "https://api.deepseek.com/v1", "deepseek-chat")
	require.Len(t, single, 1)
	assert.Equal(t, "sk-single", single[0].APIKey)
	assert.Equal(t, "deepseek-chat", single[0].Model)

	// AI_PROVIDERS 列表在前，旧配置兜底在后；缺省 model 回落 AI_MODEL
	multi := buildAIProviders(
		" https://a.com/v1 | sk-a | gpt-4o ; https://b.com/v1|sk-b ;; badentry ",
		"sk-single", "https://api.deepseek.com/v1", "deepseek-chat",
	)
	require.Len(t, multi, 3)
	assert.Equal(t, AIProviderConfig{BaseURL: "https://a.com/v1", APIKey: "sk-a", Model: "gpt-4o"}, multi[0])
	assert.Equal(t, AIProviderConfig{BaseURL: "https://b.com/v1", APIKey: "sk-b", Model: "deepseek-chat"}, multi[1])
	assert.Equal(t, "sk-single", multi[2].APIKey)

	// 什么都没有 → 空池
	assert.Empty(t, buildAIProviders("", "", "https://api.deepseek.com/v1", "deepseek-chat"))
}

func TestBuildAIProvidersJSONArray(t *testing.T) {
	raw := `[
		{"name":"站点A","base_url":"https://a.com/v1","api_key":"sk-a","model":"gpt-4o"},
		{"base_url":"https://b.com/v1","api_key":"sk-b"},
		{"base_url":"","api_key":"skip-me"}
	]`
	providers := buildAIProviders(raw, "", "https://api.deepseek.com/v1", "deepseek-chat")
	require.Len(t, providers, 2)
	assert.Equal(t, AIProviderConfig{Name: "站点A", BaseURL: "https://a.com/v1", APIKey: "sk-a", Model: "gpt-4o"}, providers[0])
	// model 省略时回落默认模型
	assert.Equal(t, "deepseek-chat", providers[1].Model)

	// JSON 非法 → 空（不会误用旧格式解析）
	assert.Empty(t, buildAIProviders(`[{bad json`, "", "https://x/v1", "m"))
}
