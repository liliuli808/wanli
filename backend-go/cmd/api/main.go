package main

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"

	"github.com/wanli/backend-go/internal/config"
	"github.com/wanli/backend-go/internal/infrastructure/database"
	"github.com/wanli/backend-go/internal/infrastructure/redis"
	"github.com/wanli/backend-go/internal/infrastructure/storage"
	"github.com/wanli/backend-go/internal/server"
	"github.com/wanli/backend-go/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log := logger.New(cfg.Server.LogLevel)

	// 初始化基础设施连接（阶段 0：验证连通性，不强制要求业务使用）
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Error("database connection failed", "error", err)
		// 阶段 0 允许继续启动，方便独立测试 HTTP 层
	} else {
		log.Info("database connected")
		_ = db
	}

	ctx := context.Background()
	rdb := redis.New(cfg.Redis)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("redis connection failed", "error", err)
	} else {
		log.Info("redis connected")
	}

	var s3Presigner *minio.Client
	s3Client, err := storage.New(cfg.Storage)
	if err != nil {
		log.Error("storage connection failed", "error", err)
	} else {
		if err := storage.EnsureBucket(ctx, s3Client, cfg.Storage.Bucket); err != nil {
			log.Error("ensure bucket failed", "error", err)
		} else {
			log.Info("storage connected", "bucket", cfg.Storage.Bucket)
			presignConfig := cfg.Storage
			if presignConfig.PublicEndpoint != "" {
				presignConfig.Endpoint = presignConfig.PublicEndpoint
				// 对外地址的协议独立于内网连接（如经 Tunnel 暴露时需 https）
				presignConfig.UseSSL = presignConfig.PublicUseSSL
			}
			s3Presigner, err = storage.New(presignConfig)
			if err != nil {
				log.Error("storage presigner initialization failed", "error", err)
			}
		}
	}

	srv := server.New(cfg, log, db, rdb, s3Presigner)
	if err := srv.Start(); err != nil {
		log.Error("server shutdown error", "error", err)
	}
}
