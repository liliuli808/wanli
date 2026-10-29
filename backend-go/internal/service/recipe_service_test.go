package service

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// memoryRecipeRepo 是 RecipeRepository 的全功能内存实现，供阶段 3 服务测试使用。
type memoryRecipeRepo struct {
	mu       sync.Mutex
	versions map[uuid.UUID]*model.RecipeVersion
	recipes  map[uuid.UUID]*model.Recipe
	sources  map[uuid.UUID]*model.VideoSource
	users    map[uuid.UUID]string
}

func newMemoryRecipeRepo() *memoryRecipeRepo {
	return &memoryRecipeRepo{
		versions: map[uuid.UUID]*model.RecipeVersion{},
		recipes:  map[uuid.UUID]*model.Recipe{},
		sources:  map[uuid.UUID]*model.VideoSource{},
		users:    map[uuid.UUID]string{},
	}
}

// seed 写入一条完整的 菜谱 + 版本 + 视频来源 数据并返回各 ID。
func (r *memoryRecipeRepo) seed(authorID uuid.UUID, versionType string, isPublic bool) (recipeID, versionID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sourceID := uuid.New()
	r.sources[sourceID] = &model.VideoSource{
		ID:         sourceID,
		BVID:       "BV1xx411c7xx",
		Title:      "红烧肉教学",
		UpNickname: "美食作家王刚",
		VideoURL:   "https://www.bilibili.com/video/BV1xx411c7xx/",
		Status:     model.VideoStatusActive,
	}

	recipeID = uuid.New()
	r.recipes[recipeID] = &model.Recipe{
		ID:            recipeID,
		Title:         "家常红烧肉",
		VideoSourceID: &sourceID,
		CreatorID:     &authorID,
		Status:        model.ContentStatusActive,
		CreatedAt:     time.Now(),
	}

	versionID = uuid.New()
	amount := "500克"
	ts := 45
	r.versions[versionID] = &model.RecipeVersion{
		ID:          versionID,
		RecipeID:    recipeID,
		AuthorID:    &authorID,
		VersionType: versionType,
		Title:       "家常红烧肉",
		IsPublic:    isPublic,
		Status:      model.ContentStatusActive,
		Ingredients: []model.RecipeIngredient{
			{RecipeVersionID: versionID, Name: "五花肉", Amount: &amount, Category: model.IngredientCategoryMain, IsExplicit: true},
		},
		Steps: []model.RecipeStep{
			{RecipeVersionID: versionID, StepOrder: 1, Description: "切块", VideoTimestamp: &ts},
		},
	}
	r.users[authorID] = "王刚粉丝甲"
	return recipeID, versionID
}

func (r *memoryRecipeRepo) FindVersionDetail(_ context.Context, versionID uuid.UUID) (*repository.VersionDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	version, ok := r.versions[versionID]
	if !ok {
		return nil, nil
	}
	recipe := r.recipes[version.RecipeID]
	detail := &repository.VersionDetail{Version: version, Recipe: recipe}
	if recipe.VideoSourceID != nil {
		detail.Source = r.sources[*recipe.VideoSourceID]
	}
	if version.AuthorID != nil {
		detail.AuthorNickname = r.users[*version.AuthorID]
	}
	return detail, nil
}

func (r *memoryRecipeRepo) SetVersionPublic(_ context.Context, versionID uuid.UUID, isPublic bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	version, ok := r.versions[versionID]
	if !ok {
		return nil
	}
	version.IsPublic = isPublic
	return nil
}

func (r *memoryRecipeRepo) UpdateSourceDraft(_ context.Context, params repository.DraftUpdateParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	version := r.versions[params.VersionID]
	if version == nil || version.IsPublic {
		return assert.AnError
	}
	r.recipes[params.RecipeID].Title = params.Title
	version.Title = params.Title
	version.IsPublic = params.Publish
	version.Servings = params.Parsed.Servings
	version.EstimatedMinutes = params.Parsed.EstimatedMinutes
	version.Instructions = params.Parsed.Instructions

	version.Ingredients = version.Ingredients[:0]
	for _, ing := range params.Parsed.Ingredients {
		version.Ingredients = append(version.Ingredients, model.RecipeIngredient{
			RecipeVersionID: params.VersionID,
			Name:            ing.Name,
			Amount:          ing.Amount,
			Category:        ing.Category,
			IsExplicit:      ing.IsExplicit,
		})
	}
	version.Steps = version.Steps[:0]
	for _, step := range params.Parsed.Steps {
		version.Steps = append(version.Steps, model.RecipeStep{
			RecipeVersionID: params.VersionID,
			StepOrder:       step.StepOrder,
			Description:     step.Description,
			VideoTimestamp:  step.VideoTimestamp,
		})
	}
	return nil
}

func (r *memoryRecipeRepo) ListPublicRecipes(_ context.Context, sortBy string, offset, limit int) ([]repository.RecipeListItem, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var items []repository.RecipeListItem
	for _, recipe := range r.recipes {
		if recipe.Status != model.ContentStatusActive {
			continue
		}
		var sourceVersion *model.RecipeVersion
		var publicCount int64
		for _, v := range r.versions {
			if v.RecipeID != recipe.ID || !v.IsPublic || v.Status != model.ContentStatusActive {
				continue
			}
			publicCount++
			if v.VersionType == model.VersionTypeSource {
				sourceVersion = v
			}
		}
		if sourceVersion == nil {
			continue
		}
		item := repository.RecipeListItem{
			RecipeID:           recipe.ID,
			Title:              sourceVersion.Title,
			SourceVersionID:    sourceVersion.ID,
			PublicVersionCount: publicCount,
			CreatedAt:          recipe.CreatedAt,
		}
		if recipe.VideoSourceID != nil {
			if source := r.sources[*recipe.VideoSourceID]; source != nil {
				item.UpNickname = source.UpNickname
				item.VideoStatus = source.Status
			}
		}
		if recipe.CreatorID != nil {
			item.AuthorNickname = r.users[*recipe.CreatorID]
		}
		items = append(items, item)
	}

	if sortBy == repository.RecipeSortHottest {
		sort.Slice(items, func(i, j int) bool { return items[i].PublicVersionCount > items[j].PublicVersionCount })
	} else {
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	}

	total := int64(len(items))
	if offset >= len(items) {
		return []repository.RecipeListItem{}, total, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], total, nil
}

func (r *memoryRecipeRepo) FindRecipeByID(_ context.Context, recipeID uuid.UUID) (*model.Recipe, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recipes[recipeID], nil
}

func (r *memoryRecipeRepo) ListPublicVersionsByRecipe(_ context.Context, recipeID uuid.UUID) ([]repository.VersionTreeNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var nodes []repository.VersionTreeNode
	for _, v := range r.versions {
		if v.RecipeID != recipeID || !v.IsPublic || v.Status != model.ContentStatusActive {
			continue
		}
		node := repository.VersionTreeNode{
			ID:          v.ID,
			Title:       v.Title,
			VersionType: v.VersionType,
			ParentID:    v.ParentID,
			ChangeLog:   v.ChangeLog,
			CreatedAt:   v.CreatedAt,
		}
		if v.AuthorID != nil {
			node.AuthorNickname = r.users[*v.AuthorID]
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].CreatedAt.Before(nodes[j].CreatedAt) })
	return nodes, nil
}

// 阶段 2 的接口方法：阶段 3 服务不调用，提供占位实现。
func (r *memoryRecipeRepo) FindVideoSourceByBVID(context.Context, string) (*model.VideoSource, error) {
	return nil, nil
}

func (r *memoryRecipeRepo) FindRecipeIDByVideoSourceID(context.Context, uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}

func (r *memoryRecipeRepo) ListUserSourceDrafts(_ context.Context, authorID uuid.UUID) ([]repository.UserSourceDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var drafts []repository.UserSourceDraft
	for _, v := range r.versions {
		if v.AuthorID == nil || *v.AuthorID != authorID || v.IsPublic || v.VersionType != model.VersionTypeSource {
			continue
		}
		recipe := r.recipes[v.RecipeID]
		if recipe == nil || recipe.Status != model.ContentStatusActive {
			continue
		}
		d := repository.UserSourceDraft{
			VersionID:   v.ID,
			Title:       v.Title,
			RecipeID:    v.RecipeID,
			CreatedAt:   v.CreatedAt,
			VideoStatus: model.VideoStatusExpired,
		}
		if recipe.VideoSourceID != nil {
			if src := r.sources[*recipe.VideoSourceID]; src != nil {
				d.BVID = src.BVID
				d.CoverURL = src.CoverURL
				d.VideoStatus = src.Status
			}
		}
		drafts = append(drafts, d)
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].CreatedAt.After(drafts[j].CreatedAt) })
	return drafts, nil
}

func (r *memoryRecipeRepo) SaveSourceDraft(context.Context, repository.SourceDraftParams) (*model.RecipeVersion, error) {
	return nil, nil
}

func (r *memoryRecipeRepo) CreateDerivedVersion(_ context.Context, params repository.DerivedVersionParams) (*model.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	version := &model.RecipeVersion{
		ID:               uuid.New(),
		RecipeID:         params.RecipeID,
		ParentID:         params.ParentID,
		AuthorID:         &params.AuthorID,
		VersionType:      params.VersionType,
		Title:            params.Parsed.Title,
		IsPublic:         true,
		Status:           model.ContentStatusActive,
		Servings:         params.Parsed.Servings,
		EstimatedMinutes: params.Parsed.EstimatedMinutes,
		Instructions:     params.Parsed.Instructions,
		ChangeLog:        &params.ChangeLog,
		CreatedAt:        time.Now(),
	}
	for _, ing := range params.Parsed.Ingredients {
		version.Ingredients = append(version.Ingredients, model.RecipeIngredient{
			RecipeVersionID: version.ID,
			Name:            ing.Name,
			Amount:          ing.Amount,
			Category:        ing.Category,
			IsExplicit:      ing.IsExplicit,
		})
	}
	for _, step := range params.Parsed.Steps {
		version.Steps = append(version.Steps, model.RecipeStep{
			RecipeVersionID: version.ID,
			StepOrder:       step.StepOrder,
			Description:     step.Description,
			VideoTimestamp:  step.VideoTimestamp,
		})
	}
	r.versions[version.ID] = version
	return version, nil
}

func TestListMySourceDrafts(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	repo.seed(authorID, model.VersionTypeSource, false) // 未发布的 SOURCE 草稿
	repo.seed(authorID, model.VersionTypeSource, true)  // 已发布，不应出现
	repo.seed(uuid.New(), model.VersionTypeSource, false)

	svc := NewRecipeService(repo)
	drafts, err := svc.ListMySourceDrafts(context.Background(), authorID)
	require.NoError(t, err)
	require.Len(t, drafts, 1, "只返回自己的未发布草稿")
	assert.Equal(t, "家常红烧肉", drafts[0].Title)
	assert.Equal(t, "BV1xx411c7xx", drafts[0].BVID, "带上 BVID 便于客户端展示来源")
}

func TestGetVersionDetail_PublicVersionAllowsGuest(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, versionID := repo.seed(authorID, model.VersionTypeSource, true)
	svc := NewRecipeService(repo)

	result, err := svc.GetVersionDetail(context.Background(), nil, versionID)
	require.NoError(t, err)
	assert.Equal(t, "家常红烧肉", result.Detail.Recipe.Title)
	assert.Equal(t, "美食作家王刚", result.Detail.Source.UpNickname)
	assert.Equal(t, "王刚粉丝甲", result.Detail.AuthorNickname)
	assert.Empty(t, result.VideoWarning)
	require.Len(t, result.Detail.Version.Ingredients, 1)
	assert.Equal(t, "五花肉", result.Detail.Version.Ingredients[0].Name)
}

func TestGetVersionDetail_ExpiredVideoReturnsWarning(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	recipeID, versionID := repo.seed(authorID, model.VersionTypeSource, true)
	repo.sources[*repo.recipes[recipeID].VideoSourceID].Status = model.VideoStatusExpired
	svc := NewRecipeService(repo)

	// AC4：视频失效后菜谱仍可浏览并返回警示
	result, err := svc.GetVersionDetail(context.Background(), nil, versionID)
	require.NoError(t, err)
	assert.Equal(t, VideoExpiredWarning, result.VideoWarning)
	assert.Equal(t, model.VideoStatusExpired, result.Detail.Source.Status)
}

func TestGetVersionDetail_DraftVisibility(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, versionID := repo.seed(authorID, model.VersionTypeSource, false)
	svc := NewRecipeService(repo)

	// 游客不可见草稿
	_, err := svc.GetVersionDetail(context.Background(), nil, versionID)
	assert.ErrorIs(t, err, ErrVersionNotFound)

	// 其他用户不可见草稿
	otherID := uuid.New()
	_, err = svc.GetVersionDetail(context.Background(), &otherID, versionID)
	assert.ErrorIs(t, err, ErrVersionNotFound)

	// 作者可见自己的草稿
	result, err := svc.GetVersionDetail(context.Background(), &authorID, versionID)
	require.NoError(t, err)
	assert.False(t, result.Detail.Version.IsPublic)
}

func TestGetVersionDetail_NotFoundAndBlocked(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	recipeID, versionID := repo.seed(authorID, model.VersionTypeSource, true)
	svc := NewRecipeService(repo)

	_, err := svc.GetVersionDetail(context.Background(), nil, uuid.New())
	assert.ErrorIs(t, err, ErrVersionNotFound)

	// 被治理下架的菜谱对所有人不可见
	repo.recipes[recipeID].Status = model.ContentStatusBlocked
	_, err = svc.GetVersionDetail(context.Background(), nil, versionID)
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestPublishSource_Success(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, versionID := repo.seed(authorID, model.VersionTypeSource, false)
	svc := NewRecipeService(repo)

	result, err := svc.PublishSource(context.Background(), authorID, versionID, "秘制红烧肉", validParsedRecipe())
	require.NoError(t, err)
	assert.True(t, result.Detail.Version.IsPublic)
	assert.True(t, repo.versions[versionID].IsPublic)
	assert.Equal(t, "秘制红烧肉", repo.versions[versionID].Title)
}

func TestPublishSource_PermissionAndState(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, draftID := repo.seed(authorID, model.VersionTypeSource, false)
	_, publicID := repo.seed(authorID, model.VersionTypeSource, true)
	_, adaptedDraftID := repo.seed(authorID, model.VersionTypeAdapted, false)
	svc := NewRecipeService(repo)

	// 非作者发布 → FORBIDDEN
	_, err := svc.PublishSource(context.Background(), uuid.New(), draftID, "红烧肉", validParsedRecipe())
	assert.ErrorIs(t, err, ErrNotVersionAuthor)

	// 已公开的版本不能重复发布
	_, err = svc.PublishSource(context.Background(), authorID, publicID, "红烧肉", validParsedRecipe())
	assert.ErrorIs(t, err, ErrNotSourceDraft)

	// 非 SOURCE 类型不能走来源发布
	_, err = svc.PublishSource(context.Background(), authorID, adaptedDraftID, "红烧肉", validParsedRecipe())
	assert.ErrorIs(t, err, ErrNotSourceDraft)

	// 不存在的版本
	_, err = svc.PublishSource(context.Background(), authorID, uuid.New(), "红烧肉", validParsedRecipe())
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func validParsedRecipe() *model.ParsedRecipe {
	return &model.ParsedRecipe{
		Ingredients: []model.ParsedIngredient{
			{Name: "五花肉", Category: model.IngredientCategoryMain, IsExplicit: true},
		},
		Steps: []model.ParsedStep{{StepOrder: 1, Description: "切块"}},
	}
}

func TestUpdateSourceDraft_Success(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, versionID := repo.seed(authorID, model.VersionTypeSource, false)
	svc := NewRecipeService(repo)

	newAmount := "600克"
	parsed := &model.ParsedRecipe{
		Ingredients: []model.ParsedIngredient{
			{Name: "五花肉", Amount: &newAmount, Category: model.IngredientCategoryMain, IsExplicit: true},
			{Name: "冰糖", Category: model.IngredientCategorySeasoning, IsExplicit: false},
		},
		Steps: []model.ParsedStep{
			{StepOrder: 1, Description: "焯水去腥"},
			{StepOrder: 2, Description: "炒糖色"},
		},
	}

	result, err := svc.UpdateSourceDraft(context.Background(), authorID, versionID, "秘制红烧肉", parsed)
	require.NoError(t, err)
	assert.Equal(t, "秘制红烧肉", result.Detail.Recipe.Title)
	require.Len(t, result.Detail.Version.Ingredients, 2)
	assert.Equal(t, "冰糖", result.Detail.Version.Ingredients[1].Name)
	require.Len(t, result.Detail.Version.Steps, 2)
	assert.Equal(t, "焯水去腥", result.Detail.Version.Steps[0].Description)
}

func TestUpdateSourceDraft_RejectsInvalidAndPublic(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, draftID := repo.seed(authorID, model.VersionTypeSource, false)
	_, publicID := repo.seed(authorID, model.VersionTypeSource, true)
	svc := NewRecipeService(repo)

	validParsed := &model.ParsedRecipe{
		Ingredients: []model.ParsedIngredient{{Name: "五花肉", Category: model.IngredientCategoryMain, IsExplicit: true}},
		Steps:       []model.ParsedStep{{StepOrder: 1, Description: "切块"}},
	}

	// 空标题
	_, err := svc.UpdateSourceDraft(context.Background(), authorID, draftID, "", validParsed)
	assert.ErrorIs(t, err, ErrDraftInvalid)

	// 空食材
	_, err = svc.UpdateSourceDraft(context.Background(), authorID, draftID, "红烧肉", &model.ParsedRecipe{
		Steps: []model.ParsedStep{{StepOrder: 1, Description: "切块"}},
	})
	assert.ErrorIs(t, err, ErrDraftInvalid)

	// 非法分类
	_, err = svc.UpdateSourceDraft(context.Background(), authorID, draftID, "红烧肉", &model.ParsedRecipe{
		Ingredients: []model.ParsedIngredient{{Name: "五花肉", Category: "OTHER"}},
		Steps:       []model.ParsedStep{{StepOrder: 1, Description: "切块"}},
	})
	assert.ErrorIs(t, err, ErrDraftInvalid)

	// 已公开版本禁止改写（禁止静默覆盖公开菜谱版本）
	_, err = svc.UpdateSourceDraft(context.Background(), authorID, publicID, "红烧肉", validParsed)
	assert.ErrorIs(t, err, ErrNotSourceDraft)

	// 非作者
	_, err = svc.UpdateSourceDraft(context.Background(), uuid.New(), draftID, "红烧肉", validParsed)
	assert.ErrorIs(t, err, ErrNotVersionAuthor)
}

func TestListPublicRecipes_SortAndPagination(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	repo.seed(authorID, model.VersionTypeSource, true)
	repo.seed(authorID, model.VersionTypeSource, true)
	repo.seed(authorID, model.VersionTypeSource, false) // 草稿不进入公开列表
	svc := NewRecipeService(repo)

	items, total, err := svc.ListPublicRecipes(context.Background(), "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)

	// 分页
	items, total, err = svc.ListPublicRecipes(context.Background(), "latest", 2, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 1)

	// 非法排序
	_, _, err = svc.ListPublicRecipes(context.Background(), "random", 1, 20)
	assert.ErrorIs(t, err, ErrInvalidSort)
}

func TestGetVersionTree(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	recipeID, sourceID := repo.seed(authorID, model.VersionTypeSource, true)

	// 追加一个公开改编版挂在来源版之下
	repo.versions[uuid.New()] = &model.RecipeVersion{
		ID:          uuid.New(),
		RecipeID:    recipeID,
		ParentID:    &sourceID,
		AuthorID:    &authorID,
		VersionType: model.VersionTypeAdapted,
		IsPublic:    true,
		Status:      model.ContentStatusActive,
		CreatedAt:   time.Now().Add(time.Hour),
	}
	svc := NewRecipeService(repo)

	nodes, err := svc.GetVersionTree(context.Background(), recipeID)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	assert.Equal(t, model.VersionTypeSource, nodes[0].VersionType)
	assert.Nil(t, nodes[0].ParentID)
	assert.Equal(t, model.VersionTypeAdapted, nodes[1].VersionType)
	assert.Equal(t, &sourceID, nodes[1].ParentID)

	// 菜谱不存在
	_, err = svc.GetVersionTree(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrVersionNotFound)
}
