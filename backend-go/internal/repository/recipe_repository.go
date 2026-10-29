package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wanli/backend-go/internal/model"
)

var ErrDraftNotMutable = errors.New("source draft not found or already public")

// SourceDraftParams 是保存 SOURCE 草稿版本的参数。
type SourceDraftParams struct {
	VideoSource model.VideoSource // 不含 ID，按 BVID upsert
	RecipeTitle string            // 菜名（取 AI 解析结果）
	CreatorID   uuid.UUID         // 提交解析的用户
	Parsed      *model.ParsedRecipe
}

// VersionDetail 是菜谱版本的完整聚合（版本 + 菜谱 + 视频来源 + 作者昵称）。
type VersionDetail struct {
	Version        *model.RecipeVersion // 已预加载 Ingredients 与 Steps
	Recipe         *model.Recipe
	Source         *model.VideoSource // 视频来源可能已被置空，为 nil
	AuthorNickname string             // 作者已注销时为空串
}

// DraftUpdateParams 是校对阶段更新 SOURCE 草稿的参数（全量替换食材与步骤）。
type DraftUpdateParams struct {
	VersionID uuid.UUID
	RecipeID  uuid.UUID
	Title     string // 更新 recipes.title
	Parsed    *model.ParsedRecipe
	Publish   bool // true 时在同一事务中完成校对内容写入与公开
}

// RecipeListItem 是公开菜谱列表的单条摘要。
type RecipeListItem struct {
	RecipeID           uuid.UUID
	Title              string
	CoverURL           *string
	UpNickname         string
	VideoStatus        string
	SourceVersionID    uuid.UUID
	Servings           *string
	EstimatedMinutes   *int
	AuthorNickname     string
	PublicVersionCount int64
	CookingRecordCount int64
	CreatedAt          time.Time
}

// VersionTreeNode 是版本派生树的节点。
type VersionTreeNode struct {
	ID             uuid.UUID
	Title          string
	VersionType    string
	ParentID       *uuid.UUID
	AuthorNickname string
	ChangeLog      *string
	CreatedAt      time.Time
}

// UserSourceDraft 是用户自己导入、尚未发布的 SOURCE 草稿（「我的菜谱」列表项）。
type UserSourceDraft struct {
	VersionID   uuid.UUID
	Title       string
	BVID        string
	CoverURL    *string
	CreatedAt   time.Time
	RecipeID    uuid.UUID
	VideoStatus string // 视频来源失效时可用于提示；来源记录缺失时为 EXPIRED
}

// 公开菜谱列表排序方式。
const (
	RecipeSortLatest  = "latest"
	RecipeSortHottest = "hottest"
)

// RecipeRepository 菜谱聚合（video_sources / recipes / recipe_versions 等）的数据访问接口。
type RecipeRepository interface {
	// FindVideoSourceByBVID 按 BVID 查询视频来源；不存在时返回 (nil, nil)。
	FindVideoSourceByBVID(ctx context.Context, bvid string) (*model.VideoSource, error)
	// FindRecipeIDByVideoSourceID 查询视频来源关联的未下架菜谱 ID；不存在时返回 (nil, nil)。
	FindRecipeIDByVideoSourceID(ctx context.Context, videoSourceID uuid.UUID) (*uuid.UUID, error)
	// ListUserSourceDrafts 查询指定用户导入、尚未发布的 SOURCE 草稿（按创建时间倒序）。
	ListUserSourceDrafts(ctx context.Context, authorID uuid.UUID) ([]UserSourceDraft, error)
	// SaveSourceDraft 在一个事务中保存视频来源、菜谱与 SOURCE 草稿版本（含食材与步骤）。
	SaveSourceDraft(ctx context.Context, params SourceDraftParams) (*model.RecipeVersion, error)
	// FindVersionDetail 查询版本完整聚合；不存在时返回 (nil, nil)。
	FindVersionDetail(ctx context.Context, versionID uuid.UUID) (*VersionDetail, error)
	// SetVersionPublic 更新版本公开状态。
	SetVersionPublic(ctx context.Context, versionID uuid.UUID, isPublic bool) error
	// UpdateSourceDraft 在一个事务中更新 SOURCE 草稿（标题、版本字段、全量替换食材与步骤）。
	UpdateSourceDraft(ctx context.Context, params DraftUpdateParams) error
	// ListPublicRecipes 查询公开菜谱列表（存在公开 SOURCE 版本的菜谱），返回列表与总数。
	ListPublicRecipes(ctx context.Context, sort string, offset, limit int) ([]RecipeListItem, int64, error)
	// FindRecipeByID 查询菜谱；不存在时返回 (nil, nil)。
	FindRecipeByID(ctx context.Context, recipeID uuid.UUID) (*model.Recipe, error)
	// ListPublicVersionsByRecipe 查询菜谱下所有公开版本（派生树节点，按创建时间升序）。
	ListPublicVersionsByRecipe(ctx context.Context, recipeID uuid.UUID) ([]VersionTreeNode, error)
	// CreateDerivedVersion 创建派生版本（纠错合入的新 SOURCE 版或改编的 ADAPTED 版），公开可见。
	CreateDerivedVersion(ctx context.Context, params DerivedVersionParams) (*model.RecipeVersion, error)
}

// DerivedVersionParams 是创建派生版本的参数。
type DerivedVersionParams struct {
	RecipeID    uuid.UUID
	ParentID    *uuid.UUID
	AuthorID    uuid.UUID
	VersionType string // SOURCE（纠错合入）或 ADAPTED（派生改编）
	ChangeLog   string
	Parsed      *model.ParsedRecipe // 新的食材与步骤内容
}

// GormRecipeRepository 基于 GORM 的实现。
type GormRecipeRepository struct {
	db *gorm.DB
}

// NewGormRecipeRepository 创建菜谱仓库。
func NewGormRecipeRepository(db *gorm.DB) *GormRecipeRepository {
	return &GormRecipeRepository{db: db}
}

func (r *GormRecipeRepository) FindVideoSourceByBVID(ctx context.Context, bvid string) (*model.VideoSource, error) {
	var source model.VideoSource
	err := r.db.WithContext(ctx).First(&source, "bvid = ?", bvid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find video source: %w", err)
	}
	return &source, nil
}

// FindRecipeIDByVideoSourceID 只返回未下架（ACTIVE）菜谱的 ID，避免已下架的菜谱长期占用 BVID。
func (r *GormRecipeRepository) FindRecipeIDByVideoSourceID(ctx context.Context, videoSourceID uuid.UUID) (*uuid.UUID, error) {
	var recipe model.Recipe
	err := r.db.WithContext(ctx).Select("id").
		First(&recipe, "video_source_id = ? AND status = ?", videoSourceID, model.ContentStatusActive).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find recipe by video source: %w", err)
	}
	return &recipe.ID, nil
}

func (r *GormRecipeRepository) ListUserSourceDrafts(ctx context.Context, authorID uuid.UUID) ([]UserSourceDraft, error) {
	var rows []UserSourceDraft
	err := r.db.WithContext(ctx).
		Table("recipe_versions AS v").
		Select(`v.id AS version_id, v.title, v.recipe_id, v.created_at,
			COALESCE(vs.bvid, '') AS bvid,
			vs.cover_url,
			COALESCE(vs.status, 'EXPIRED') AS video_status`).
		Joins("JOIN recipes r ON r.id = v.recipe_id").
		Joins("LEFT JOIN video_sources vs ON vs.id = r.video_source_id").
		Where("v.author_id = ? AND v.version_type = ? AND v.is_public = FALSE AND v.status = ? AND r.status = ?",
			authorID, model.VersionTypeSource, model.ContentStatusActive, model.ContentStatusActive).
		Order("v.created_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list user source drafts: %w", err)
	}
	return rows, nil
}

func (r *GormRecipeRepository) SaveSourceDraft(ctx context.Context, params SourceDraftParams) (*model.RecipeVersion, error) {
	var version *model.RecipeVersion

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 同一 BVID 只保留一条来源记录，重复解析时刷新元信息
		source := params.VideoSource
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "bvid"}},
			DoUpdates: clause.AssignmentColumns([]string{"title", "up_nickname", "video_url", "cover_url", "updated_at"}),
		}).Create(&source).Error; err != nil {
			return fmt.Errorf("upsert video source: %w", err)
		}

		recipe := model.Recipe{
			Title:         params.RecipeTitle,
			VideoSourceID: &source.ID,
			CreatorID:     &params.CreatorID,
			Status:        model.ContentStatusActive,
		}
		if err := tx.Create(&recipe).Error; err != nil {
			return fmt.Errorf("create recipe: %w", err)
		}

		version = &model.RecipeVersion{
			RecipeID:         recipe.ID,
			AuthorID:         &params.CreatorID,
			VersionType:      model.VersionTypeSource,
			Title:            params.Parsed.Title,
			IsPublic:         false, // 草稿待校对，发布（阶段 3）时置为 true
			Status:           model.ContentStatusActive,
			Servings:         params.Parsed.Servings,
			EstimatedMinutes: params.Parsed.EstimatedMinutes,
			Instructions:     params.Parsed.Instructions,
		}
		if err := tx.Create(version).Error; err != nil {
			return fmt.Errorf("create recipe version: %w", err)
		}

		ingredients := make([]model.RecipeIngredient, 0, len(params.Parsed.Ingredients))
		for _, ing := range params.Parsed.Ingredients {
			ingredients = append(ingredients, model.RecipeIngredient{
				RecipeVersionID: version.ID,
				Name:            ing.Name,
				Amount:          ing.Amount,
				Category:        ing.Category,
				IsExplicit:      ing.IsExplicit,
				SubstituteFor:   ing.SubstituteFor,
			})
		}
		if err := tx.Create(&ingredients).Error; err != nil {
			return fmt.Errorf("create ingredients: %w", err)
		}

		steps := make([]model.RecipeStep, 0, len(params.Parsed.Steps))
		for _, step := range params.Parsed.Steps {
			steps = append(steps, model.RecipeStep{
				RecipeVersionID: version.ID,
				StepOrder:       step.StepOrder,
				Description:     step.Description,
				VideoTimestamp:  step.VideoTimestamp,
				DurationSeconds: step.DurationSeconds,
				Heat:            step.Heat,
			})
		}
		if err := tx.Create(&steps).Error; err != nil {
			return fmt.Errorf("create steps: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (r *GormRecipeRepository) FindVersionDetail(ctx context.Context, versionID uuid.UUID) (*VersionDetail, error) {
	var version model.RecipeVersion
	err := r.db.WithContext(ctx).
		Preload("Ingredients").
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("step_order ASC") }).
		First(&version, "id = ?", versionID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find recipe version: %w", err)
	}

	detail := &VersionDetail{Version: &version}

	var recipe model.Recipe
	if err := r.db.WithContext(ctx).First(&recipe, "id = ?", version.RecipeID).Error; err != nil {
		return nil, fmt.Errorf("find recipe of version: %w", err)
	}
	detail.Recipe = &recipe

	if recipe.VideoSourceID != nil {
		var source model.VideoSource
		err := r.db.WithContext(ctx).First(&source, "id = ?", *recipe.VideoSourceID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find video source of recipe: %w", err)
		}
		if err == nil {
			detail.Source = &source
		}
	}

	if version.AuthorID != nil {
		var nickname string
		err := r.db.WithContext(ctx).Model(&model.User{}).
			Select("nickname").Where("id = ?", *version.AuthorID).
			Limit(1).Find(&nickname).Error
		if err != nil {
			return nil, fmt.Errorf("find author nickname: %w", err)
		}
		detail.AuthorNickname = nickname
	}

	return detail, nil
}

func (r *GormRecipeRepository) SetVersionPublic(ctx context.Context, versionID uuid.UUID, isPublic bool) error {
	result := r.db.WithContext(ctx).Model(&model.RecipeVersion{}).
		Where("id = ?", versionID).
		Update("is_public", isPublic)
	if result.Error != nil {
		return fmt.Errorf("set version public: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("set version public: version %s not found", versionID)
	}
	return nil
}

func (r *GormRecipeRepository) UpdateSourceDraft(ctx context.Context, params DraftUpdateParams) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Recipe{}).
			Where("id = ?", params.RecipeID).
			Update("title", params.Title).Error; err != nil {
			return fmt.Errorf("update recipe title: %w", err)
		}

		// 防御性 WHERE：已公开的版本禁止静默覆盖，仅草稿可整体改写
		result := tx.Model(&model.RecipeVersion{}).
			Where("id = ? AND is_public = FALSE", params.VersionID).
			Updates(map[string]any{
				"title":             params.Title,
				"servings":          params.Parsed.Servings,
				"estimated_minutes": params.Parsed.EstimatedMinutes,
				"instructions":      params.Parsed.Instructions,
				"is_public":         params.Publish,
			})
		if result.Error != nil {
			return fmt.Errorf("update source draft: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("%w: version %s", ErrDraftNotMutable, params.VersionID)
		}

		if err := tx.Where("recipe_version_id = ?", params.VersionID).
			Delete(&model.RecipeIngredient{}).Error; err != nil {
			return fmt.Errorf("clear draft ingredients: %w", err)
		}
		if err := tx.Where("recipe_version_id = ?", params.VersionID).
			Delete(&model.RecipeStep{}).Error; err != nil {
			return fmt.Errorf("clear draft steps: %w", err)
		}

		ingredients := make([]model.RecipeIngredient, 0, len(params.Parsed.Ingredients))
		for _, ing := range params.Parsed.Ingredients {
			ingredients = append(ingredients, model.RecipeIngredient{
				RecipeVersionID: params.VersionID,
				Name:            ing.Name,
				Amount:          ing.Amount,
				Category:        ing.Category,
				IsExplicit:      ing.IsExplicit,
				SubstituteFor:   ing.SubstituteFor,
			})
		}
		if err := tx.Create(&ingredients).Error; err != nil {
			return fmt.Errorf("create draft ingredients: %w", err)
		}

		steps := make([]model.RecipeStep, 0, len(params.Parsed.Steps))
		for _, step := range params.Parsed.Steps {
			steps = append(steps, model.RecipeStep{
				RecipeVersionID: params.VersionID,
				StepOrder:       step.StepOrder,
				Description:     step.Description,
				VideoTimestamp:  step.VideoTimestamp,
				DurationSeconds: step.DurationSeconds,
				Heat:            step.Heat,
			})
		}
		if err := tx.Create(&steps).Error; err != nil {
			return fmt.Errorf("create draft steps: %w", err)
		}

		return nil
	})
}

// recipeListRow 是公开菜谱列表查询的扫描行。
type recipeListRow struct {
	RecipeID           uuid.UUID `gorm:"column:recipe_id"`
	Title              string    `gorm:"column:title"`
	CoverURL           *string   `gorm:"column:cover_url"`
	UpNickname         string    `gorm:"column:up_nickname"`
	VideoStatus        string    `gorm:"column:video_status"`
	SourceVersionID    uuid.UUID `gorm:"column:source_version_id"`
	Servings           *string   `gorm:"column:servings"`
	EstimatedMinutes   *int      `gorm:"column:estimated_minutes"`
	AuthorNickname     string    `gorm:"column:author_nickname"`
	PublicVersionCount int64     `gorm:"column:public_version_count"`
	CookingRecordCount int64     `gorm:"column:cooking_record_count"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

const publicRecipeListSelect = `
	SELECT r.id AS recipe_id, v.title, v.created_at,
		COALESCE(vs.cover_url, NULL) AS cover_url,
		COALESCE(vs.up_nickname, '') AS up_nickname,
		COALESCE(vs.status, 'EXPIRED') AS video_status,
		v.id AS source_version_id, v.servings, v.estimated_minutes,
		COALESCE(u.nickname, '') AS author_nickname,
		(SELECT COUNT(*) FROM recipe_versions pv
			WHERE pv.recipe_id = r.id AND pv.is_public = TRUE AND pv.status = 'ACTIVE') AS public_version_count,
		(SELECT COUNT(*) FROM cooking_records cr
			JOIN recipe_versions pv2 ON pv2.id = cr.recipe_version_id
			WHERE pv2.recipe_id = r.id AND pv2.is_public = TRUE AND pv2.status = 'ACTIVE'
				AND cr.status = 'ACTIVE') AS cooking_record_count
	FROM recipes r
	JOIN LATERAL (
		SELECT source_version.id, source_version.title, source_version.servings,
			source_version.estimated_minutes, source_version.created_at
		FROM recipe_versions AS source_version
		WHERE source_version.recipe_id = r.id
			AND source_version.version_type = 'SOURCE'
			AND source_version.is_public = TRUE
			AND source_version.status = 'ACTIVE'
		ORDER BY source_version.created_at DESC, source_version.id DESC
		LIMIT 1
	) AS v ON TRUE
	LEFT JOIN video_sources vs ON vs.id = r.video_source_id
	LEFT JOIN users u ON u.id = r.creator_id
	WHERE r.status = 'ACTIVE'`

func (r *GormRecipeRepository) ListPublicRecipes(ctx context.Context, sort string, offset, limit int) ([]RecipeListItem, int64, error) {
	var total int64
	countSQL := `SELECT COUNT(*) FROM (` + publicRecipeListSelect + `) AS listed`
	if err := r.db.WithContext(ctx).Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count public recipes: %w", err)
	}

	orderBy := "v.created_at DESC"
	if sort == RecipeSortHottest {
		orderBy = "public_version_count DESC, v.created_at DESC"
	}

	var rows []recipeListRow
	listSQL := publicRecipeListSelect + " ORDER BY " + orderBy + " LIMIT ? OFFSET ?"
	if err := r.db.WithContext(ctx).Raw(listSQL, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list public recipes: %w", err)
	}

	items := make([]RecipeListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, RecipeListItem(row))
	}
	return items, total, nil
}

func (r *GormRecipeRepository) FindRecipeByID(ctx context.Context, recipeID uuid.UUID) (*model.Recipe, error) {
	var recipe model.Recipe
	err := r.db.WithContext(ctx).First(&recipe, "id = ?", recipeID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find recipe: %w", err)
	}
	return &recipe, nil
}

func (r *GormRecipeRepository) ListPublicVersionsByRecipe(ctx context.Context, recipeID uuid.UUID) ([]VersionTreeNode, error) {
	var nodes []VersionTreeNode
	err := r.db.WithContext(ctx).
		Table("recipe_versions AS v").
		Select("v.id, v.title, v.version_type, v.parent_id, v.change_log, v.created_at, COALESCE(u.nickname, '') AS author_nickname").
		Joins("LEFT JOIN users u ON u.id = v.author_id").
		Where("v.recipe_id = ? AND v.is_public = TRUE AND v.status = 'ACTIVE'", recipeID).
		Order("v.created_at ASC").
		Scan(&nodes).Error
	if err != nil {
		return nil, fmt.Errorf("list public versions: %w", err)
	}
	return nodes, nil
}

func (r *GormRecipeRepository) CreateDerivedVersion(ctx context.Context, params DerivedVersionParams) (*model.RecipeVersion, error) {
	var version *model.RecipeVersion
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		version, err = createDerivedVersionTx(tx, params)
		return err
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

// createDerivedVersionTx 在事务中创建派生版本（含食材与步骤），供创建与纠错合入复用。
func createDerivedVersionTx(tx *gorm.DB, params DerivedVersionParams) (*model.RecipeVersion, error) {
	changeLog := params.ChangeLog
	version := &model.RecipeVersion{
		RecipeID:         params.RecipeID,
		ParentID:         params.ParentID,
		AuthorID:         &params.AuthorID,
		VersionType:      params.VersionType,
		Title:            params.Parsed.Title,
		IsPublic:         true, // 纠错合入与改编版发布即公开
		Status:           model.ContentStatusActive,
		Servings:         params.Parsed.Servings,
		EstimatedMinutes: params.Parsed.EstimatedMinutes,
		Instructions:     params.Parsed.Instructions,
		ChangeLog:        &changeLog,
	}
	if err := tx.Create(version).Error; err != nil {
		return nil, fmt.Errorf("create derived version: %w", err)
	}

	ingredients := make([]model.RecipeIngredient, 0, len(params.Parsed.Ingredients))
	for _, ing := range params.Parsed.Ingredients {
		ingredients = append(ingredients, model.RecipeIngredient{
			RecipeVersionID: version.ID,
			Name:            ing.Name,
			Amount:          ing.Amount,
			Category:        ing.Category,
			IsExplicit:      ing.IsExplicit,
			SubstituteFor:   ing.SubstituteFor,
		})
	}
	if err := tx.Create(&ingredients).Error; err != nil {
		return nil, fmt.Errorf("create derived ingredients: %w", err)
	}

	steps := make([]model.RecipeStep, 0, len(params.Parsed.Steps))
	for _, step := range params.Parsed.Steps {
		steps = append(steps, model.RecipeStep{
			RecipeVersionID: version.ID,
			StepOrder:       step.StepOrder,
			Description:     step.Description,
			VideoTimestamp:  step.VideoTimestamp,
			DurationSeconds: step.DurationSeconds,
			Heat:            step.Heat,
		})
	}
	if err := tx.Create(&steps).Error; err != nil {
		return nil, fmt.Errorf("create derived steps: %w", err)
	}

	return version, nil
}
