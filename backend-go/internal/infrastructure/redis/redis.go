package redis

import (
	"github.com/redis/go-redis/v9"

	"github.com/wanli/backend-go/internal/config"
)

// New 创建 Redis 客户端。
func New(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       0,
	})
}
