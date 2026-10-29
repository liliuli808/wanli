package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// 阶段 3：校对发布与菜谱阅读的业务错误。
var (
	ErrVersionNotFound  = errors.New("菜谱版本不存在")
	ErrNotVersionAuthor = errors.New("仅版本作者可以执行此操作")
	ErrNotSourceDraft   = errors.New("仅待发布的来源解析草稿可以执行此操作")
	ErrInvalidSort      = errors.New("不支持的排序方式")
	ErrDraftInvalid     = errors.New("草稿内容校验失败")
)

// VideoExpiredWarning 是视频失效时随菜谱详情返回的警示文案（AC4）。
const VideoExpiredWarning = "原视频已下架或失效，菜谱内容仍可正常浏览"

// 公开菜谱列表分页约束。
const (
	defaultPageSize = 20
	maxPageSize     = 50
)

// RecipeService 菜谱阅读、校对与发布服务。
type RecipeService struct {
	repo repository.RecipeRepository
}

// NewRecipeService 创建菜谱服务。
func NewRecipeService(repo repository.RecipeRepository) *RecipeService {
	return &RecipeService{repo: repo}
}

// VersionDetailResult 是版本详情的查询结果，含视频失效警示。
type VersionDetailResult struct {
	Detail       *repository.VersionDetail
	VideoWarning string // 视频失效时非空，否则为空串
}

// GetVersionDetail 查询菜谱版本详情。
// 公开版本允许游客浏览；草稿仅作者可见（对外表现为不存在，避免泄露草稿存在性）。
func (s *RecipeService) GetVersionDetail(ctx context.Context, userID *uuid.UUID, versionID uuid.UUID) (*VersionDetailResult, error) {
	detail, err := s.repo.FindVersionDetail(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if detail == nil || !isReadable(detail) {
		return nil, ErrVersionNotFound
	}
	if !detail.Version.IsPublic && (userID == nil || detail.Version.AuthorID == nil || *detail.Version.AuthorID != *userID) {
		return nil, ErrVersionNotFound
	}

	result := &VersionDetailResult{Detail: detail}
	if videoExpired(detail) {
		result.VideoWarning = VideoExpiredWarning
	}
	return result, nil
}

// PublishSource 原子写入最终校对内容并发布来源解析草稿。
func (s *RecipeService) PublishSource(ctx context.Context, userID, versionID uuid.UUID, title string, parsed *model.ParsedRecipe) (*VersionDetailResult, error) {
	return s.saveSourceDraft(ctx, userID, versionID, title, parsed, true)
}

// UpdateSourceDraft 校对阶段更新 SOURCE 草稿（全量替换食材与步骤），返回更新后的详情。
func (s *RecipeService) UpdateSourceDraft(ctx context.Context, userID, versionID uuid.UUID, title string, parsed *model.ParsedRecipe) (*VersionDetailResult, error) {
	return s.saveSourceDraft(ctx, userID, versionID, title, parsed, false)
}

func (s *RecipeService) saveSourceDraft(ctx context.Context, userID, versionID uuid.UUID, title string, parsed *model.ParsedRecipe, publish bool) (*VersionDetailResult, error) {
	detail, err := s.repo.FindVersionDetail(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if err := checkSourceDraftAuthor(detail, userID); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, fmt.Errorf("%w: 菜名不能为空", ErrDraftInvalid)
	}
	parsed.Title = title
	if err := parsed.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDraftInvalid, err)
	}

	err = s.repo.UpdateSourceDraft(ctx, repository.DraftUpdateParams{
		VersionID: versionID,
		RecipeID:  detail.Recipe.ID,
		Title:     title,
		Parsed:    parsed,
		Publish:   publish,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDraftNotMutable) {
			return nil, ErrNotSourceDraft
		}
		return nil, err
	}
	return s.GetVersionDetail(ctx, &userID, versionID)
}

// ListPublicRecipes 查询公开菜谱列表。
func (s *RecipeService) ListPublicRecipes(ctx context.Context, sort string, page, pageSize int) ([]repository.RecipeListItem, int64, error) {
	if sort == "" {
		sort = repository.RecipeSortLatest
	}
	if sort != repository.RecipeSortLatest && sort != repository.RecipeSortHottest {
		return nil, 0, ErrInvalidSort
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return s.repo.ListPublicRecipes(ctx, sort, (page-1)*pageSize, pageSize)
}

// ListMySourceDrafts 查询当前用户导入、尚未发布的 SOURCE 草稿。
// 解析成功即写库，草稿不公开、也不会出现在发现页；没有这个入口用户就再也找不回
// 自己解析的结果，因此单列一个「我的菜谱（待校对）」列表。
func (s *RecipeService) ListMySourceDrafts(ctx context.Context, userID uuid.UUID) ([]repository.UserSourceDraft, error) {
	return s.repo.ListUserSourceDrafts(ctx, userID)
}

// GetVersionTree 查询菜谱的公开版本派生树（游客可浏览）。
func (s *RecipeService) GetVersionTree(ctx context.Context, recipeID uuid.UUID) ([]repository.VersionTreeNode, error) {
	recipe, err := s.repo.FindRecipeByID(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if recipe == nil || recipe.Status != model.ContentStatusActive {
		return nil, ErrVersionNotFound
	}
	return s.repo.ListPublicVersionsByRecipe(ctx, recipeID)
}

// isReadable 判断版本所属内容是否处于可浏览状态（未被治理下架）。
func isReadable(detail *repository.VersionDetail) bool {
	return detail.Version.Status == model.ContentStatusActive &&
		detail.Recipe.Status == model.ContentStatusActive
}

// videoExpired 判断视频来源是否失效（被标记 EXPIRED 或来源记录已删除）。
func videoExpired(detail *repository.VersionDetail) bool {
	return detail.Source == nil || detail.Source.Status == model.VideoStatusExpired
}

// checkSourceDraftAuthor 校验版本是「当前用户的待发布 SOURCE 草稿」。
func checkSourceDraftAuthor(detail *repository.VersionDetail, userID uuid.UUID) error {
	if detail == nil || !isReadable(detail) {
		return ErrVersionNotFound
	}
	if detail.Version.AuthorID == nil || *detail.Version.AuthorID != userID {
		return ErrNotVersionAuthor
	}
	if detail.Version.VersionType != model.VersionTypeSource || detail.Version.IsPublic {
		return ErrNotSourceDraft
	}
	return nil
}
