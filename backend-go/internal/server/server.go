package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/auth"
	"github.com/wanli/backend-go/internal/config"
	"github.com/wanli/backend-go/internal/handler"
	"github.com/wanli/backend-go/internal/infrastructure/ai"
	"github.com/wanli/backend-go/internal/infrastructure/bilibili"
	"github.com/wanli/backend-go/internal/infrastructure/turnstile"
	"github.com/wanli/backend-go/internal/middleware"
	"github.com/wanli/backend-go/internal/repository"
	"github.com/wanli/backend-go/internal/service"
)

// Server 聚合 HTTP 服务依赖。
type Server struct {
	cfg        *config.Config
	logger     *slog.Logger
	router     *gin.Engine
	httpServer *http.Server
}

// New 创建并配置 HTTP Server。
func New(
	cfg *config.Config,
	logger *slog.Logger,
	db *gorm.DB,
	rdb *redis.Client,
	s3Presigner *minio.Client,
) *Server {
	if cfg.Server.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, time.Duration(cfg.JWT.ExpiryHours)*time.Hour)

	// 全局中间件
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestLogger(logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.Auth(jwtManager))

	// 依赖组装
	var userRepo repository.UserRepository
	var kitchenRepo repository.KitchenRepository
	if db != nil {
		userRepo = repository.NewGormUserRepository(db)
		kitchenRepo = repository.NewGormKitchenRepository(db)
	} else {
		// 阶段 0 可能无数据库连接，提供 panic 保护
		logger.Warn("database not available, repositories disabled")
	}

	var turnstileVerifier turnstile.Verifier
	switch {
	case !cfg.Turnstile.Enabled:
		logger.Info("Turnstile 已关闭（TURNSTILE_ENABLED=false），跳过人机验证")
		turnstileVerifier = turnstile.Noop{}
	case cfg.Turnstile.SecretKey == "":
		logger.Warn("TURNSTILE_SECRET_KEY 未配置，人机验证将恒失败（闭锁）")
		turnstileVerifier = turnstile.Stub{}
	default:
		turnstileVerifier = turnstile.New(cfg.Turnstile.SecretKey)
	}

	var loginLimiter service.LoginLimiter
	lockout := time.Duration(cfg.Auth.LockoutMinutes) * time.Minute
	if rdb != nil {
		loginLimiter = service.NewRedisLoginLimiter(rdb, cfg.Auth.MaxFailures, lockout)
	} else {
		logger.Warn("redis not available, using in-memory login limiter")
		loginLimiter = service.NewMemoryLoginLimiter(cfg.Auth.MaxFailures, lockout)
	}

	var registerLimiter service.RegisterLimiter
	if cfg.Auth.RegisterMaxPerHour > 0 {
		if rdb != nil {
			registerLimiter = service.NewRedisRegisterLimiter(rdb, cfg.Auth.RegisterMaxPerHour, time.Hour)
		} else {
			logger.Warn("redis not available, using in-memory register limiter")
			registerLimiter = service.NewMemoryRegisterLimiter(cfg.Auth.RegisterMaxPerHour, time.Hour)
		}
	}

	authService := service.NewAuthService(userRepo, jwtManager, turnstileVerifier, loginLimiter)
	kitchenService := service.NewKitchenService(kitchenRepo)

	// AI 提供商池：多端点轮询 + 失败冷却切换；视频导入与厨房适配共享同一个池，
	// 让冷却状态全局生效（某端点挂了对两条链路同时跳过）。
	aiPool := newAIPool(cfg.AI, logger)

	authHandler := handler.NewAuthHandler(authService, cfg.Turnstile.SiteKey, registerLimiter)
	kitchenHandler := handler.NewKitchenHandler(kitchenService)
	healthHandler := handler.NewHealthHandler()

	// 菜谱阅读、校对与发布（阶段 3）+ 纠错与派生改编（阶段 4）
	var recipeHandler *handler.RecipeHandler
	var correctionHandler *handler.CorrectionHandler
	var adaptationHandler *handler.AdaptationHandler
	var communityHandler *handler.CommunityHandler
	var messageHandler *handler.MessageHandler
	if db != nil {
		recipeRepo := repository.NewGormRecipeRepository(db)
		recipeService := service.NewRecipeService(recipeRepo)
		recipeHandler = handler.NewRecipeHandler(recipeService)
		correctionRepo := repository.NewGormCorrectionRepository(db)
		correctionHandler = handler.NewCorrectionHandler(
			service.NewCorrectionService(recipeRepo, correctionRepo),
		)
		adaptationHandler = handler.NewAdaptationHandler(
			service.NewAdaptationService(
				recipeRepo,
				kitchenRepo,
				aiPool,
			),
		)
		communityRepo := repository.NewGormCommunityRepository(db)
		communityHandler = handler.NewCommunityHandler(
			service.NewCommunityService(
				communityRepo,
				service.CookingPhotoPublicPrefix(cfg.Storage),
			),
		)
		messageHandler = handler.NewMessageHandler(service.NewMessageService(communityRepo, correctionRepo))
	} else {
		logger.Warn("database not available, recipe endpoints disabled")
	}

	var uploadHandler *handler.UploadHandler
	if s3Presigner != nil {
		uploadHandler = handler.NewUploadHandler(service.NewUploadService(s3Presigner, cfg.Storage))
	} else {
		logger.Warn("storage not available, upload endpoint disabled")
	}

	// 视频导入（阶段 2）：需要 Redis（任务状态机）与数据库（草稿落库）同时可用
	var videoImportHandler *handler.VideoImportHandler
	if db != nil && rdb != nil {
		importTaskRepo := repository.NewImportTaskRepository(rdb, 24*time.Hour)
		importTaskService := service.NewImportTaskService(importTaskRepo, logger)
		executor := service.NewImportExecutor(5, 3*time.Minute, logger)

		// 服务重启后恢复卡死的解析任务
		if recovered, err := importTaskService.RecoverStale(context.Background(), 10*time.Minute); err != nil {
			logger.Error("恢复卡死解析任务失败", "error", err)
		} else if recovered > 0 {
			logger.Info("已恢复卡死的解析任务", "count", recovered)
		}

		videoImportService := service.NewVideoImportService(
			importTaskService,
			executor,
			bilibili.NewWithCookie(bilibili.DefaultBaseURL, cfg.Bilibili.Cookie, logger),
			aiPool,
			repository.NewGormRecipeRepository(db),
			logger,
		)
		videoImportHandler = handler.NewVideoImportHandler(videoImportService)
	} else {
		logger.Warn("redis or database not available, video import disabled")
	}

	// 路由注册
	registerRoutes(
		router,
		healthHandler,
		authHandler,
		kitchenHandler,
		videoImportHandler,
		recipeHandler,
		correctionHandler,
		adaptationHandler,
		communityHandler,
		uploadHandler,
		messageHandler,
	)

	return &Server{
		cfg:    cfg,
		logger: logger,
		router: router,
		httpServer: &http.Server{
			Addr:    ":" + cfg.Server.Port,
			Handler: router,
		},
	}
}

func registerRoutes(
	r *gin.Engine,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	kitchenHandler *handler.KitchenHandler,
	videoImportHandler *handler.VideoImportHandler,
	recipeHandler *handler.RecipeHandler,
	correctionHandler *handler.CorrectionHandler,
	adaptationHandler *handler.AdaptationHandler,
	communityHandler *handler.CommunityHandler,
	uploadHandler *handler.UploadHandler,
	messageHandler *handler.MessageHandler,
) {
	// 人机验证（Turnstile）组件页面，供 Android WebView 加载
	r.GET("/captcha", authHandler.CaptchaWidget)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Health)

		// 认证（允许游客）
		v1.POST("/auth/register", authHandler.Register)
		v1.POST("/auth/login", authHandler.Login)

		// 菜谱公开浏览（允许游客）
		if recipeHandler != nil {
			v1.GET("/recipes", recipeHandler.ListRecipes)
			v1.GET("/recipes/:id/version-tree", recipeHandler.GetVersionTree)
			v1.GET("/recipe-versions/:id", recipeHandler.GetVersion)
		}
		if communityHandler != nil {
			v1.GET("/recipe-versions/:id/comments", communityHandler.ListComments)
			v1.GET("/recipe-versions/:id/cooking-records", communityHandler.ListCookingRecords)
			v1.GET("/feed/cooking-records", communityHandler.ListCookingFeed)
		}

		// 需要登录
		authorized := v1.Group("", middleware.RequireAuth())
		{
			authorized.GET("/me", authHandler.Me)
			authorized.PUT("/me/profile", authHandler.UpdateProfile)
			authorized.GET("/me/kitchen", kitchenHandler.GetKitchen)
			authorized.PUT("/me/kitchen", kitchenHandler.UpdateKitchen)
			authorized.GET("/me/dietary-restrictions", kitchenHandler.GetRestrictions)
			authorized.PUT("/me/dietary-restrictions", kitchenHandler.UpdateRestrictions)

			if messageHandler != nil {
				authorized.GET("/me/messages", messageHandler.List)
			}

			if videoImportHandler != nil {
				authorized.POST("/video-imports", videoImportHandler.Create)
				authorized.GET("/video-imports/:task_id", videoImportHandler.Get)
			}

			if recipeHandler != nil {
				authorized.POST("/recipe-versions/:id/publish", recipeHandler.PublishVersion)
				authorized.PUT("/recipe-versions/:id/draft", recipeHandler.UpdateDraft)
				authorized.POST("/recipe-versions/:id/forks", recipeHandler.ForkVersion)
			}

			if correctionHandler != nil {
				authorized.POST("/recipe-versions/:id/corrections", correctionHandler.SubmitCorrection)
				authorized.POST("/correction-proposals/:id/review", correctionHandler.ReviewCorrection)
				authorized.GET("/me/correction-proposals", correctionHandler.ListPending)
			}

			if adaptationHandler != nil {
				authorized.POST("/recipe-versions/:id/adaptations", adaptationHandler.AdaptVersion)
				authorized.PATCH("/recipe-versions/:id/visibility", adaptationHandler.SetVisibility)
				authorized.GET("/me/adaptations", adaptationHandler.ListMine)
			}

			if communityHandler != nil {
				authorized.POST("/recipe-versions/:id/comments", communityHandler.CreateComment)
				authorized.DELETE("/comments/:id", communityHandler.DeleteComment)
				authorized.POST("/recipe-versions/:id/cooking-records", communityHandler.CreateCookingRecord)
				authorized.DELETE("/cooking-records/:id", communityHandler.DeleteCookingRecord)
				authorized.GET("/me/cooking-records", communityHandler.ListMyCookingRecords)
			}

			if uploadHandler != nil {
				authorized.POST("/upload/presigned-url", uploadHandler.Presign)
			}
		}
	}

	// 404 统一处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "RESOURCE_NOT_FOUND", "message": "路由不存在"}})
	})
}

// Start 启动服务并监听退出信号。
func (s *Server) Start() error {
	s.logger.Info("starting server", "addr", s.httpServer.Addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("server listen error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// Router 返回 gin.Engine（用于测试）。
func (s *Server) Router() *gin.Engine {
	return s.router
}

// newAIPool 根据配置构建 AI 提供商池。providers 为空（未配置任何 key）时池仍可
// 创建，调用时返回“未配置”错误，保持服务可启动、仅 AI 功能不可用。
func newAIPool(cfg config.AIConfig, logger *slog.Logger) *ai.Pool {
	providers := make([]ai.Provider, 0, len(cfg.Providers))
	for _, p := range cfg.Providers {
		providers = append(providers, ai.Provider{
			Name:    p.Name,
			BaseURL: p.BaseURL,
			APIKey:  p.APIKey,
			Model:   p.Model,
		})
	}
	pool := ai.NewPool(providers, time.Duration(cfg.CooldownSeconds)*time.Second, logger)
	logger.Info("AI 提供商池已初始化", "providers", pool.Size(), "cooldown_seconds", cfg.CooldownSeconds)
	return pool
}
