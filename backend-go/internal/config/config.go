package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/viper"
)

// Config 聚合所有环境变量配置。
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Storage   StorageConfig
	JWT       JWTConfig
	AI        AIConfig
	Bilibili  BilibiliConfig
	Turnstile TurnstileConfig
	Auth      AuthConfig
}

type ServerConfig struct {
	Port     string
	LogLevel string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

type StorageConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Region         string
	UseSSL         bool
	// PublicUseSSL 决定对外 URL（预签名上传地址、成品图公开地址）使用 https。
	// 与 UseSSL 分离：容器内连 MinIO 走 http，经 Cloudflare Tunnel 暴露的公网地址必须是 https。
	PublicUseSSL bool
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type AIConfig struct {
	// 单提供商配置（向后兼容 AI_API_KEY/AI_BASE_URL/AI_MODEL）。
	APIKey  string
	BaseURL string
	Model   string
	// Providers 是 AI 池的端点列表；由 AI_PROVIDERS 解析，未设置时回落到上面的单提供商配置。
	Providers []AIProviderConfig
	// CooldownSeconds 是池内单个端点失败后的冷却秒数。
	CooldownSeconds int
}

// AIProviderConfig 是一个第三方 OpenAI 兼容端点的配置。
type AIProviderConfig struct {
	Name    string
	BaseURL string
	APIKey  string
	Model   string
}

// parseAIProviders 解析 AI_PROVIDERS 环境变量，支持两种格式：
//
//  1. JSON 数组（推荐，字段名清晰）：
//     [{"base_url":"https://a.com/v1","api_key":"sk-a","model":"gpt-4o","name":"站点A"}, ...]
//     name 可省略；model 可省略并回落 defaultModel。
//  2. 行内分隔格式（兼容旧配置）：
//     base_url|api_key|model;base_url2|api_key2|model2
//
// 以 "[" 开头时按 JSON 解析；JSON 解析失败返回空（避免静默用上错误的配置）。
func parseAIProviders(raw, defaultModel string) []AIProviderConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var items []struct {
			Name    string `json:"name"`
			BaseURL string `json:"base_url"`
			APIKey  string `json:"api_key"`
			Model   string `json:"model"`
		}
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return nil
		}
		out := make([]AIProviderConfig, 0, len(items))
		for _, it := range items {
			if it.BaseURL == "" || it.APIKey == "" {
				continue // 缺 base_url 或 api_key 的条目直接忽略
			}
			model := it.Model
			if model == "" {
				model = defaultModel
			}
			out = append(out, AIProviderConfig{
				Name:    it.Name,
				BaseURL: it.BaseURL,
				APIKey:  it.APIKey,
				Model:   model,
			})
		}
		return out
	}

	var out []AIProviderConfig
	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "|", 3)
		if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			continue // 缺 base_url 或 api_key 的条目直接忽略
		}
		model := defaultModel
		if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
			model = strings.TrimSpace(parts[2])
		}
		out = append(out, AIProviderConfig{
			BaseURL: strings.TrimSpace(parts[0]),
			APIKey:  strings.TrimSpace(parts[1]),
			Model:   model,
		})
	}
	return out
}

// buildAIProviders 汇总 AI 池端点：AI_PROVIDERS 列表在前，旧的单提供商
// 配置（AI_API_KEY 非空时）追加在末尾作为兜底。
func buildAIProviders(raw, singleKey, singleBaseURL, singleModel string) []AIProviderConfig {
	providers := parseAIProviders(raw, singleModel)
	if singleKey != "" {
		providers = append(providers, AIProviderConfig{
			BaseURL: singleBaseURL,
			APIKey:  singleKey,
			Model:   singleModel,
		})
	}
	return providers
}

type BilibiliConfig struct {
	Cookie string
}

type TurnstileConfig struct {
	// Enabled 为 false 时跳过人机验证（国内网络下 Turnstile 挑战域名可用性差）。
	Enabled   bool
	SiteKey   string
	SecretKey string
}

type AuthConfig struct {
	MaxFailures    int
	LockoutMinutes int
	// RegisterMaxPerHour 是同一 IP 每小时允许的注册次数，0 表示不限制。
	RegisterMaxPerHour int
}

// Load 加载配置：默认值 < .env 文件（存在时） < 操作系统环境变量。
// 注意：不使用 viper.Unmarshal（其扁平 key 无法绑定嵌套结构体），改为显式读取。
func Load() (*Config, error) {
	v := viper.New()

	// .env 文件可选；不存在时静默跳过（如生产环境直接注入环境变量）
	v.SetConfigFile(".env")
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		var pathErr *fs.PathError
		if !errors.As(err, &notFound) && !errors.As(err, &pathErr) {
			return nil, fmt.Errorf("read .env: %w", err)
		}
	}
	v.AutomaticEnv()

	v.SetDefault("PORT", "8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DB_HOST", "127.0.0.1")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "wanli_db")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("REDIS_HOST", "127.0.0.1")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("S3_REGION", "us-east-1")
	v.SetDefault("S3_USE_SSL", false)
	v.SetDefault("S3_PUBLIC_USE_SSL", false)
	v.SetDefault("JWT_EXPIRY_HOURS", 168)
	v.SetDefault("AUTH_MAX_FAILURES", 5)
	v.SetDefault("AUTH_LOCKOUT_MINUTES", 15)
	v.SetDefault("AUTH_REGISTER_MAX_PER_HOUR", 10)
	v.SetDefault("TURNSTILE_ENABLED", false)
	v.SetDefault("AI_BASE_URL", "https://api.deepseek.com/v1")
	v.SetDefault("AI_MODEL", "deepseek-chat")
	v.SetDefault("AI_POOL_COOLDOWN_SECONDS", 60)

	return &Config{
		Server: ServerConfig{
			Port:     v.GetString("PORT"),
			LogLevel: v.GetString("LOG_LEVEL"),
		},
		Database: DatabaseConfig{
			Host:     v.GetString("DB_HOST"),
			Port:     v.GetInt("DB_PORT"),
			User:     v.GetString("DB_USER"),
			Password: v.GetString("DB_PASSWORD"),
			Name:     v.GetString("DB_NAME"),
			SSLMode:  v.GetString("DB_SSLMODE"),
		},
		Redis: RedisConfig{
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetInt("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
		},
		Storage: StorageConfig{
			Endpoint:       v.GetString("S3_ENDPOINT"),
			PublicEndpoint: v.GetString("S3_PUBLIC_ENDPOINT"),
			AccessKey:      v.GetString("S3_ACCESS_KEY"),
			SecretKey:      v.GetString("S3_SECRET_KEY"),
			Bucket:         v.GetString("S3_BUCKET"),
			Region:         v.GetString("S3_REGION"),
			UseSSL:         v.GetBool("S3_USE_SSL"),
			PublicUseSSL:   v.GetBool("S3_PUBLIC_USE_SSL"),
		},
		JWT: JWTConfig{
			Secret:      v.GetString("JWT_SECRET"),
			ExpiryHours: v.GetInt("JWT_EXPIRY_HOURS"),
		},
		AI: AIConfig{
			APIKey:          v.GetString("AI_API_KEY"),
			BaseURL:         v.GetString("AI_BASE_URL"),
			Model:           v.GetString("AI_MODEL"),
			CooldownSeconds: v.GetInt("AI_POOL_COOLDOWN_SECONDS"),
			Providers: buildAIProviders(
				v.GetString("AI_PROVIDERS"),
				v.GetString("AI_API_KEY"),
				v.GetString("AI_BASE_URL"),
				v.GetString("AI_MODEL"),
			),
		},
		Bilibili: BilibiliConfig{
			Cookie: v.GetString("BILIBILI_COOKIE"),
		},
		Turnstile: TurnstileConfig{
			Enabled:   v.GetBool("TURNSTILE_ENABLED"),
			SiteKey:   v.GetString("TURNSTILE_SITE_KEY"),
			SecretKey: v.GetString("TURNSTILE_SECRET_KEY"),
		},
		Auth: AuthConfig{
			MaxFailures:        v.GetInt("AUTH_MAX_FAILURES"),
			LockoutMinutes:     v.GetInt("AUTH_LOCKOUT_MINUTES"),
			RegisterMaxPerHour: v.GetInt("AUTH_REGISTER_MAX_PER_HOUR"),
		},
	}, nil
}

// DSN 返回 PostgreSQL 连接字符串。
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// Addr 返回 Redis 地址。
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
