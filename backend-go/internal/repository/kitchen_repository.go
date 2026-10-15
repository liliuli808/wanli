package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

// KitchenRepository 厨房配置数据访问接口。
type KitchenRepository interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*model.KitchenProfile, error)
	SaveProfile(ctx context.Context, profile *model.KitchenProfile) error
	GetRestrictions(ctx context.Context, userID uuid.UUID) ([]model.DietaryRestriction, error)
	SaveRestrictions(ctx context.Context, userID uuid.UUID, restrictions []model.DietaryRestriction) error
}

// GormKitchenRepository 基于 GORM 的实现。
type GormKitchenRepository struct {
	db *gorm.DB
}

func NewGormKitchenRepository(db *gorm.DB) *GormKitchenRepository {
	return &GormKitchenRepository{db: db}
}

func (r *GormKitchenRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*model.KitchenProfile, error) {
	var profile model.KitchenProfile
	if err := r.db.WithContext(ctx).First(&profile, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &model.KitchenProfile{
				UserID:           userID,
				OwnedIngredients: []string{},
				OwnedTools:       []string{},
			}, nil
		}
		return nil, fmt.Errorf("get kitchen profile: %w", err)
	}
	return &profile, nil
}

func (r *GormKitchenRepository) SaveProfile(ctx context.Context, profile *model.KitchenProfile) error {
	if err := r.db.WithContext(ctx).Save(profile).Error; err != nil {
		return fmt.Errorf("save kitchen profile: %w", err)
	}
	return nil
}

func (r *GormKitchenRepository) GetRestrictions(ctx context.Context, userID uuid.UUID) ([]model.DietaryRestriction, error) {
	var restrictions []model.DietaryRestriction
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&restrictions).Error; err != nil {
		return nil, fmt.Errorf("get restrictions: %w", err)
	}
	return restrictions, nil
}

func (r *GormKitchenRepository) SaveRestrictions(ctx context.Context, userID uuid.UUID, restrictions []model.DietaryRestriction) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("begin transaction: %w", tx.Error)
	}

	if err := tx.Where("user_id = ?", userID).Delete(&model.DietaryRestriction{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("delete old restrictions: %w", err)
	}

	if len(restrictions) > 0 {
		if err := tx.Create(restrictions).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("create restrictions: %w", err)
		}
	}

	return tx.Commit().Error
}
