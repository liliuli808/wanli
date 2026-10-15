package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// KitchenService 处理厨房配置业务。
type KitchenService struct {
	repo repository.KitchenRepository
}

// NewKitchenService 创建 KitchenService。
func NewKitchenService(repo repository.KitchenRepository) *KitchenService {
	return &KitchenService{repo: repo}
}

// GetProfile 获取用户厨房配置。
func (s *KitchenService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.KitchenProfile, error) {
	return s.repo.GetProfile(ctx, userID)
}

// UpdateProfile 更新用户厨房配置。
func (s *KitchenService) UpdateProfile(ctx context.Context, userID uuid.UUID, ingredients, tools []string) (*model.KitchenProfile, error) {
	profile := &model.KitchenProfile{
		UserID:           userID,
		OwnedIngredients: ingredients,
		OwnedTools:       tools,
	}
	if err := s.repo.SaveProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// GetRestrictions 获取用户忌口。
func (s *KitchenService) GetRestrictions(ctx context.Context, userID uuid.UUID) ([]model.DietaryRestriction, error) {
	return s.repo.GetRestrictions(ctx, userID)
}

// UpdateRestrictions 更新用户忌口（全量替换）。
func (s *KitchenService) UpdateRestrictions(ctx context.Context, userID uuid.UUID, items []model.DietaryRestriction) ([]model.DietaryRestriction, error) {
	for i := range items {
		items[i].UserID = userID
	}
	if err := s.repo.SaveRestrictions(ctx, userID, items); err != nil {
		return nil, err
	}
	return s.repo.GetRestrictions(ctx, userID)
}
