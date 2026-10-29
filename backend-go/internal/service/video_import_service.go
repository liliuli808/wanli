package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/infrastructure/ai"
	"github.com/wanli/backend-go/internal/infrastructure/bilibili"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/parser"
)

// ErrInvalidVideoURL 表示输入不是有效的 B 站视频链接。
var ErrInvalidVideoURL = errors.New("invalid bilibili video url")

// DuplicateVideoError 表示该 BVID 已被解析过，携带可跳转的已有菜谱 ID。
type DuplicateVideoError struct {
	RecipeID  uuid.UUID
	VersionID *uuid.UUID // 已公开的 SOURCE 版本（可能尚未发布，为 nil）
}

func (e *DuplicateVideoError) Error() string {
	return fmt.Sprintf("bvid already parsed, recipe_id=%s", e.RecipeID)
}

// taskSubmitter 抽象任务异步投递，便于测试替换为同步/no-op 实现。
type taskSubmitter interface {
	Submit(taskID string, fn func(ctx context.Context) error)
}

// recipeExtractor 抽象 AI 菜谱解析，*ai.Client 与 *ai.Pool（多端点池）均满足。
type recipeExtractor interface {
	ExtractRecipe(ctx context.Context, input ai.ParseInput) (*model.ParsedRecipe, error)
}

// BilibiliCredentialProvider 提供用户授权的 B 站 SESSDATA；未授权时返回空串。
type BilibiliCredentialProvider interface {
	Sessdata(ctx context.Context, userID uuid.UUID) string
}

// VideoImportService 串起“提交 → 拉取文本 → AI 解析 → 保存草稿”的完整流水线。
type VideoImportService struct {
	tasks       *ImportTaskService
	executor    taskSubmitter
	bili        *bilibili.Client
	ai          recipeExtractor
	recipes     repository.RecipeRepository
	credentials BilibiliCredentialProvider
	logger      *slog.Logger
}

// NewVideoImportService 创建视频导入服务。
func NewVideoImportService(
	tasks *ImportTaskService,
	executor taskSubmitter,
	bili *bilibili.Client,
	aiClient recipeExtractor,
	recipes repository.RecipeRepository,
	logger *slog.Logger,
) *VideoImportService {
	if logger == nil {
		logger = slog.Default()
	}
	return &VideoImportService{
		tasks:    tasks,
		executor: executor,
		bili:     bili,
		ai:       aiClient,
		recipes:  recipes,
		logger:   logger,
	}
}

// WithCredentials 注入用户级 B 站凭证提供者；为 nil 时全部按匿名/服务端兜底 Cookie 访问。
func (s *VideoImportService) WithCredentials(p BilibiliCredentialProvider) *VideoImportService {
	s.credentials = p
	return s
}

// clientFor 返回带用户授权 Cookie 的 B 站客户端；用户未授权时沿用默认客户端。
func (s *VideoImportService) clientFor(ctx context.Context, userID uuid.UUID) *bilibili.Client {
	if s.credentials == nil {
		return s.bili
	}
	sessdata := s.credentials.Sessdata(ctx, userID)
	if sessdata == "" {
		return s.bili
	}
	return s.bili.WithCookie("SESSDATA=" + sessdata)
}

// Submit 校验链接、按 BVID 去重，然后创建任务并异步执行解析。
// 返回 ErrInvalidVideoURL 或 *DuplicateVideoError 时，调用方应映射为对应业务错误码。
func (s *VideoImportService) Submit(ctx context.Context, userID uuid.UUID, rawURL string) (*model.VideoImportTask, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || !parser.IsValidBilibiliURL(rawURL) {
		return nil, ErrInvalidVideoURL
	}

	bvid, err := parser.ParseBVID(rawURL)
	if err != nil {
		return nil, ErrInvalidVideoURL
	}

	// AC1：重复 BVID 直接跳转已有菜谱
	existing, err := s.recipes.FindVideoSourceByBVID(ctx, bvid)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		recipeID, err := s.recipes.FindRecipeIDByVideoSourceID(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		if recipeID != nil {
			dup := &DuplicateVideoError{RecipeID: *recipeID}
			// 附带已公开的 SOURCE 版本 ID，客户端可直接跳转菜谱详情页
			if nodes, err := s.recipes.ListPublicVersionsByRecipe(ctx, *recipeID); err == nil {
				for _, node := range nodes {
					if node.VersionType == model.VersionTypeSource {
						versionID := node.ID
						dup.VersionID = &versionID
						break
					}
				}
			}
			return nil, dup
		}
	}

	task, err := s.tasks.Create(ctx, bvid, rawURL)
	if err != nil {
		return nil, err
	}

	taskID := task.TaskID
	s.executor.Submit(taskID, func(taskCtx context.Context) error {
		return s.Process(taskCtx, taskID, userID)
	})

	s.logger.Info("视频解析任务已提交",
		slog.String("task_id", taskID),
		slog.String("bvid", bvid),
		slog.String("user_id", userID.String()))
	return task, nil
}

// Get 查询任务状态。
func (s *VideoImportService) Get(ctx context.Context, taskID string) (*model.VideoImportTask, error) {
	return s.tasks.Get(ctx, taskID)
}

// Process 执行解析流水线（由执行器异步调用）。
// 任何一步失败都会把任务标记为 FAILED 并携带业务错误码。
func (s *VideoImportService) Process(ctx context.Context, taskID string, userID uuid.UUID) error {
	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return fmt.Errorf("load task: %w", err)
	}

	fail := func(appErr *apperror.AppError, cause error) error {
		if err := s.tasks.Fail(ctx, taskID, appErr.Code, appErr.Message); err != nil {
			s.logger.Error("标记任务失败时出错", slog.String("task_id", taskID), slog.String("error", err.Error()))
		}
		return cause
	}

	// Step 1: 获取视频信息
	if err := s.tasks.Transition(ctx, taskID, model.ImportTaskFetching, "正在获取视频信息"); err != nil {
		return err
	}
	bili := s.clientFor(ctx, userID)
	videoInfo, err := bili.GetVideoInfo(ctx, task.BVID)
	if err != nil {
		return fail(apperror.ErrVideoFetchFailed, err)
	}

	// Step 2: 获取字幕；无可用字幕时立即终止，简介/评论只能作为字幕补充。
	subtitleSegments, hasSubtitle, err := bili.FetchBestSubtitle(ctx, videoInfo)
	if err != nil {
		s.logger.Warn("字幕获取失败，终止解析",
			slog.String("task_id", taskID), slog.String("error", err.Error()))
		if errors.Is(err, bilibili.ErrLoginRequired) {
			return fail(apperror.ErrBilibiliLoginRequired, err)
		}
		return fail(apperror.ErrSubtitleUnavailable, fmt.Errorf("fetch subtitle: %w", err))
	}
	if !hasSubtitle || len(subtitleSegments) == 0 {
		return fail(apperror.ErrSubtitleUnavailable, errors.New("no usable subtitle"))
	}

	comments, _ := bili.GetRecipeLikeComments(ctx, videoInfo.AID, 5)
	input := ai.ParseInput{
		BVID:        videoInfo.BVID,
		Title:       videoInfo.Title,
		Description: videoInfo.Description,
	}
	for _, seg := range subtitleSegments {
		input.Subtitles = append(input.Subtitles, ai.SubtitleInput{
			Start: seg.From,
			End:   seg.To,
			Text:  seg.Content,
		})
	}
	for _, c := range comments {
		input.Comments = append(input.Comments, c.Message)
	}

	// Step 3: AI 解析
	if err := s.tasks.Transition(ctx, taskID, model.ImportTaskParsing, "正在用AI拆解菜谱"); err != nil {
		return err
	}
	parsed, err := s.ai.ExtractRecipe(ctx, input)
	if err != nil {
		return fail(apperror.ErrAIParseFailed, err)
	}

	// Step 4: 保存为 SOURCE 草稿版本
	videoURL := fmt.Sprintf("https://www.bilibili.com/video/%s/", videoInfo.BVID)
	version, err := s.recipes.SaveSourceDraft(ctx, repository.SourceDraftParams{
		VideoSource: model.VideoSource{
			BVID:       videoInfo.BVID,
			Title:      videoInfo.Title,
			UpNickname: videoInfo.OwnerName,
			VideoURL:   videoURL,
			CoverURL:   stringPtrIfNotEmpty(videoInfo.CoverURL),
			Status:     model.VideoStatusActive,
		},
		RecipeTitle: parsed.Title,
		CreatorID:   userID,
		Parsed:      parsed,
	})
	if err != nil {
		return fail(apperror.ErrInternal, err)
	}

	if err := s.tasks.Complete(ctx, taskID, version.ID.String()); err != nil {
		return err
	}

	s.logger.Info("视频解析完成",
		slog.String("task_id", taskID),
		slog.String("bvid", videoInfo.BVID),
		slog.Bool("has_subtitle", hasSubtitle),
		slog.String("recipe_version_id", version.ID.String()))
	return nil
}

func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
