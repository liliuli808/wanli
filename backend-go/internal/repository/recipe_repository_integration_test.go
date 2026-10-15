package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

// TestGormRecipeRepository_SaveSourceDraft_Integration 需要真实 PostgreSQL（docker compose）。
// 设置 WANLI_TEST_DB_DSN 环境变量后运行；测试结束自动清理数据。
func TestGormRecipeRepository_SaveSourceDraft_Integration(t *testing.T) {
	dsn := os.Getenv("WANLI_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("未设置 WANLI_TEST_DB_DSN，跳过集成测试")
	}

	db, err := gorm.Open(postgres.Open(dsn))
	require.NoError(t, err)
	repo := NewGormRecipeRepository(db)
	ctx := context.Background()

	bvid := "BV1integration0"
	amount := "500克"
	ts := 45

	// creator_id 有外键约束，先创建临时用户
	creator := model.User{Email: "integration-test@example.com", HashedPassword: "x", Nickname: "集成测试"}
	require.NoError(t, db.Create(&creator).Error)
	creatorID := creator.ID
	parsed := &model.ParsedRecipe{
		Title: "集成测试红烧肉",
		Ingredients: []model.ParsedIngredient{
			{Name: "五花肉", Amount: &amount, Category: model.IngredientCategoryMain, IsExplicit: true},
			{Name: "盐", Amount: nil, Category: model.IngredientCategorySeasoning, IsExplicit: false},
		},
		Steps: []model.ParsedStep{
			{StepOrder: 1, Description: "五花肉切块焯水", VideoTimestamp: &ts},
		},
	}
	params := SourceDraftParams{
		VideoSource: model.VideoSource{
			BVID: bvid, Title: "测试视频", UpNickname: "测试UP",
			VideoURL: "https://www.bilibili.com/video/" + bvid + "/", Status: model.VideoStatusActive,
		},
		RecipeTitle: parsed.Title,
		CreatorID:   creatorID,
		Parsed:      parsed,
	}

	// 测试结束后清理（外键级联删除版本、食材、步骤）
	var recipeID uuid.UUID
	t.Cleanup(func() {
		db.Exec("DELETE FROM video_sources WHERE bvid = ?", bvid)
		if recipeID != uuid.Nil {
			db.Exec("DELETE FROM recipes WHERE id = ?", recipeID)
		}
		db.Exec("DELETE FROM users WHERE id = ?", creatorID)
	})

	// 首次保存
	version, err := repo.SaveSourceDraft(ctx, params)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, version.ID)

	var ingCount, stepCount int64
	db.Model(&model.RecipeIngredient{}).Where("recipe_version_id = ?", version.ID).Count(&ingCount)
	db.Model(&model.RecipeStep{}).Where("recipe_version_id = ?", version.ID).Count(&stepCount)
	assert.Equal(t, int64(2), ingCount)
	assert.Equal(t, int64(1), stepCount)

	var recipe model.Recipe
	require.NoError(t, db.First(&recipe, "video_source_id IS NOT NULL AND title = ?", "集成测试红烧肉").Error)
	recipeID = recipe.ID

	// 重复 BVID 再次保存：video_source 应 upsert 而非报错
	_, err = repo.SaveSourceDraft(ctx, params)
	require.NoError(t, err, "重复 BVID 应走 upsert")

	var sourceCount int64
	db.Model(&model.VideoSource{}).Where("bvid = ?", bvid).Count(&sourceCount)
	assert.Equal(t, int64(1), sourceCount)

	// 去重查询
	source, err := repo.FindVideoSourceByBVID(ctx, bvid)
	require.NoError(t, err)
	require.NotNil(t, source)
	foundRecipeID, err := repo.FindRecipeIDByVideoSourceID(ctx, source.ID)
	require.NoError(t, err)
	require.NotNil(t, foundRecipeID)
}

// TestGormRecipeRepository_Stage3_Integration 覆盖阶段 3 新增的仓储方法。
// 需要真实 PostgreSQL，设置 WANLI_TEST_DB_DSN 后运行；测试结束自动清理数据。
func TestGormRecipeRepository_Stage3_Integration(t *testing.T) {
	dsn := os.Getenv("WANLI_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("未设置 WANLI_TEST_DB_DSN，跳过集成测试")
	}

	db, err := gorm.Open(postgres.Open(dsn))
	require.NoError(t, err)
	repo := NewGormRecipeRepository(db)
	ctx := context.Background()

	creator := model.User{Email: "stage3-integration@example.com", HashedPassword: "x", Nickname: "阶段三测试员"}
	require.NoError(t, db.Create(&creator).Error)
	creatorID := creator.ID

	bvid := "BV1stage3test0"
	amount := "500克"
	version, err := repo.SaveSourceDraft(ctx, SourceDraftParams{
		VideoSource: model.VideoSource{
			BVID: bvid, Title: "阶段三视频", UpNickname: "阶段三UP",
			VideoURL: "https://www.bilibili.com/video/" + bvid + "/", Status: model.VideoStatusActive,
		},
		RecipeTitle: "阶段三测试菜",
		CreatorID:   creatorID,
		Parsed: &model.ParsedRecipe{
			Title:       "阶段三测试菜",
			Ingredients: []model.ParsedIngredient{{Name: "五花肉", Amount: &amount, Category: model.IngredientCategoryMain, IsExplicit: true}},
			Steps:       []model.ParsedStep{{StepOrder: 1, Description: "切块"}},
		},
	})
	require.NoError(t, err)

	recipeID := version.RecipeID
	t.Cleanup(func() {
		db.Exec("DELETE FROM video_sources WHERE bvid = ?", bvid)
		db.Exec("DELETE FROM recipes WHERE id = ?", recipeID)
		db.Exec("DELETE FROM users WHERE id = ?", creatorID)
	})

	// FindVersionDetail：完整聚合
	detail, err := repo.FindVersionDetail(ctx, version.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "阶段三测试菜", detail.Recipe.Title)
	assert.Equal(t, "阶段三UP", detail.Source.UpNickname)
	assert.Equal(t, "阶段三测试员", detail.AuthorNickname)
	require.Len(t, detail.Version.Ingredients, 1)
	require.Len(t, detail.Version.Steps, 1)

	missing, err := repo.FindVersionDetail(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)

	// UpdateSourceDraft：草稿可整体改写
	newAmount := "600克"
	err = repo.UpdateSourceDraft(ctx, DraftUpdateParams{
		VersionID: version.ID,
		RecipeID:  recipeID,
		Title:     "阶段三测试菜（校对）",
		Parsed: &model.ParsedRecipe{
			Title: "阶段三测试菜（校对）",
			Ingredients: []model.ParsedIngredient{
				{Name: "五花肉", Amount: &newAmount, Category: model.IngredientCategoryMain, IsExplicit: true},
				{Name: "冰糖", Category: model.IngredientCategorySeasoning, IsExplicit: false},
			},
			Steps: []model.ParsedStep{{StepOrder: 1, Description: "焯水"}, {StepOrder: 2, Description: "炒糖色"}},
		},
	})
	require.NoError(t, err)

	detail, err = repo.FindVersionDetail(ctx, version.ID)
	require.NoError(t, err)
	assert.Equal(t, "阶段三测试菜（校对）", detail.Recipe.Title)
	require.Len(t, detail.Version.Ingredients, 2)
	require.Len(t, detail.Version.Steps, 2)
	assert.Equal(t, "焯水", detail.Version.Steps[0].Description)

	// 草稿不出现在公开列表与版本树
	items, total, err := repo.ListPublicRecipes(ctx, RecipeSortLatest, 0, 20)
	require.NoError(t, err)
	for _, item := range items {
		assert.NotEqual(t, recipeID, item.RecipeID)
	}
	_ = total

	// 发布
	require.NoError(t, repo.SetVersionPublic(ctx, version.ID, true))

	// 公开列表出现该菜谱
	items, total, err = repo.ListPublicRecipes(ctx, RecipeSortLatest, 0, 20)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	var found *RecipeListItem
	for i := range items {
		if items[i].RecipeID == recipeID {
			found = &items[i]
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "阶段三测试菜（校对）", found.Title)
	assert.Equal(t, "阶段三UP", found.UpNickname)
	assert.Equal(t, model.VideoStatusActive, found.VideoStatus)
	assert.Equal(t, int64(1), found.PublicVersionCount)

	// 已公开版本禁止整体改写（禁止静默覆盖）
	err = repo.UpdateSourceDraft(ctx, DraftUpdateParams{
		VersionID: version.ID,
		RecipeID:  recipeID,
		Title:     "不应生效",
		Parsed: &model.ParsedRecipe{
			Title:       "不应生效",
			Ingredients: []model.ParsedIngredient{{Name: "五花肉", Category: model.IngredientCategoryMain}},
			Steps:       []model.ParsedStep{{StepOrder: 1, Description: "切块"}},
		},
	})
	require.Error(t, err)

	// 版本树
	nodes, err := repo.ListPublicVersionsByRecipe(ctx, recipeID)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, model.VersionTypeSource, nodes[0].VersionType)
	assert.Nil(t, nodes[0].ParentID)
	assert.Equal(t, "阶段三测试员", nodes[0].AuthorNickname)
}
