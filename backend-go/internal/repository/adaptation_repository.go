package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

var ErrKitchenFitNotOwned = errors.New("kitchen fit version not found or not owned")

type KitchenFitCreateParams struct {
	RecipeID uuid.UUID
	ParentID uuid.UUID
	AuthorID uuid.UUID
	Result   *model.AdaptationResult
}

type KitchenFitListItem struct {
	ID        uuid.UUID
	RecipeID  uuid.UUID
	ParentID  *uuid.UUID
	Title     string
	IsPublic  bool
	CreatedAt time.Time
}

// AdaptationRepository 是阶段5使用的窄仓储接口。
type AdaptationRepository interface {
	FindVersionDetail(ctx context.Context, versionID uuid.UUID) (*VersionDetail, error)
	CreateKitchenFitVersion(ctx context.Context, params KitchenFitCreateParams) (*model.RecipeVersion, error)
	SetKitchenFitVisibility(ctx context.Context, userID, versionID uuid.UUID, isPublic bool) error
	ListKitchenFitsByAuthor(ctx context.Context, userID uuid.UUID) ([]KitchenFitListItem, error)
}

func (r *GormRecipeRepository) CreateKitchenFitVersion(ctx context.Context, params KitchenFitCreateParams) (*model.RecipeVersion, error) {
	notes, err := json.Marshal(params.Result.AdaptationNotes)
	if err != nil {
		return nil, fmt.Errorf("marshal adaptation notes: %w", err)
	}

	var version *model.RecipeVersion
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		parentID := params.ParentID
		version = &model.RecipeVersion{
			RecipeID:         params.RecipeID,
			ParentID:         &parentID,
			AuthorID:         &params.AuthorID,
			VersionType:      model.VersionTypeKitchenFit,
			Title:            params.Result.Title,
			IsPublic:         false,
			Status:           model.ContentStatusActive,
			Servings:         params.Result.Servings,
			EstimatedMinutes: params.Result.EstimatedMinutes,
			Instructions:     params.Result.Instructions,
			AdaptationNotes:  notes,
		}
		if err := tx.Create(version).Error; err != nil {
			return fmt.Errorf("create kitchen fit version: %w", err)
		}

		ingredients := make([]model.RecipeIngredient, 0, len(params.Result.Ingredients))
		for _, ingredient := range params.Result.Ingredients {
			ingredients = append(ingredients, model.RecipeIngredient{
				RecipeVersionID: version.ID,
				Name:            ingredient.Name,
				Amount:          ingredient.Amount,
				Category:        ingredient.Category,
				IsExplicit:      ingredient.IsExplicit,
				SubstituteFor:   ingredient.SubstituteFor,
			})
		}
		if err := tx.Create(&ingredients).Error; err != nil {
			return fmt.Errorf("create kitchen fit ingredients: %w", err)
		}

		steps := make([]model.RecipeStep, 0, len(params.Result.Steps))
		for _, step := range params.Result.Steps {
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
			return fmt.Errorf("create kitchen fit steps: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (r *GormRecipeRepository) SetKitchenFitVisibility(ctx context.Context, userID, versionID uuid.UUID, isPublic bool) error {
	result := r.db.WithContext(ctx).Model(&model.RecipeVersion{}).
		Where(
			"id = ? AND author_id = ? AND version_type = ? AND status = ?",
			versionID, userID, model.VersionTypeKitchenFit, model.ContentStatusActive,
		).
		Update("is_public", isPublic)
	if result.Error != nil {
		return fmt.Errorf("set kitchen fit visibility: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrKitchenFitNotOwned
	}
	return nil
}

func (r *GormRecipeRepository) ListKitchenFitsByAuthor(ctx context.Context, userID uuid.UUID) ([]KitchenFitListItem, error) {
	var items []KitchenFitListItem
	err := r.db.WithContext(ctx).
		Table("recipe_versions").
		Select("id, recipe_id, parent_id, title, is_public, created_at").
		Where(
			"author_id = ? AND version_type = ? AND status = ?",
			userID, model.VersionTypeKitchenFit, model.ContentStatusActive,
		).
		Order("created_at DESC").
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list kitchen fit versions: %w", err)
	}
	return items, nil
}
