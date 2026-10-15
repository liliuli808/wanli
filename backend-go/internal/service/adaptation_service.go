package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/infrastructure/ai"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

var (
	ErrAIAdaptationFailed = errors.New("AI 厨房适配失败")
	ErrDietaryBlocked     = errors.New("饮食限制无法安全适配")
	ErrAllergyBlocked     = errors.New("过敏源无法安全适配")
	ErrKitchenFitNotOwned = errors.New("个人适配版不存在、类型不符或不属于当前用户")
)

type AdaptationAI interface {
	AdaptRecipe(ctx context.Context, input ai.AdaptationInput) (*model.AdaptationResult, error)
}

type AdaptationService struct {
	recipes repository.AdaptationRepository
	kitchen repository.KitchenRepository
	ai      AdaptationAI
}

func NewAdaptationService(
	recipes repository.AdaptationRepository,
	kitchen repository.KitchenRepository,
	aiClient AdaptationAI,
) *AdaptationService {
	return &AdaptationService{recipes: recipes, kitchen: kitchen, ai: aiClient}
}

func (s *AdaptationService) AdaptVersion(ctx context.Context, userID, versionID uuid.UUID) (*VersionDetailResult, error) {
	detail, err := s.recipes.FindVersionDetail(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if detail == nil || !isReadable(detail) || !detail.Version.IsPublic {
		return nil, ErrVersionNotFound
	}
	if detail.Version.VersionType == model.VersionTypeKitchenFit {
		return nil, ErrKitchenFitCannotFork
	}

	profile, err := s.kitchen.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	restrictions, err := s.kitchen.GetRestrictions(ctx, userID)
	if err != nil {
		return nil, err
	}

	input := ai.AdaptationInput{
		RecipeVersionID: versionID.String(),
		OriginalRecipe:  parsedFromVersion(detail.Version),
		KitchenProfile: ai.AdaptationKitchenProfile{
			OwnedIngredients: profile.OwnedIngredients,
			OwnedTools:       profile.OwnedTools,
		},
		DietaryRestrictions: make([]ai.AdaptationRestriction, 0, len(restrictions)),
	}
	for _, restriction := range restrictions {
		input.DietaryRestrictions = append(input.DietaryRestrictions, ai.AdaptationRestriction{
			RestrictionType: restriction.RestrictionType,
			IngredientName:  restriction.IngredientName,
		})
	}

	result, err := s.ai.AdaptRecipe(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIAdaptationFailed, err)
	}
	if !result.Safe {
		if hasAllergyRestriction(restrictions) {
			return nil, fmt.Errorf("%w: %s", ErrAllergyBlocked, blockReason(result))
		}
		return nil, fmt.Errorf("%w: %s", ErrDietaryBlocked, blockReason(result))
	}
	if allergen := findAllergenInResult(result, restrictions); allergen != "" {
		return nil, fmt.Errorf("%w: 适配结果仍包含过敏源“%s”", ErrAllergyBlocked, allergen)
	}

	version, err := s.recipes.CreateKitchenFitVersion(ctx, repository.KitchenFitCreateParams{
		RecipeID: detail.Recipe.ID,
		ParentID: versionID,
		AuthorID: userID,
		Result:   result,
	})
	if err != nil {
		return nil, err
	}
	created, err := s.recipes.FindVersionDetail(ctx, version.ID)
	if err != nil {
		return nil, err
	}
	return &VersionDetailResult{Detail: created}, nil
}

func (s *AdaptationService) SetVisibility(
	ctx context.Context,
	userID, versionID uuid.UUID,
	isPublic bool,
) (*VersionDetailResult, error) {
	detail, err := s.recipes.FindVersionDetail(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if detail == nil || !isReadable(detail) {
		return nil, ErrVersionNotFound
	}
	if detail.Version.AuthorID == nil || *detail.Version.AuthorID != userID {
		return nil, ErrNotVersionAuthor
	}
	if detail.Version.VersionType != model.VersionTypeKitchenFit {
		return nil, ErrKitchenFitNotOwned
	}
	if err := s.recipes.SetKitchenFitVisibility(ctx, userID, versionID, isPublic); err != nil {
		if errors.Is(err, repository.ErrKitchenFitNotOwned) {
			return nil, ErrKitchenFitNotOwned
		}
		return nil, err
	}
	detail.Version.IsPublic = isPublic
	return &VersionDetailResult{Detail: detail}, nil
}

func (s *AdaptationService) ListMine(ctx context.Context, userID uuid.UUID) ([]repository.KitchenFitListItem, error) {
	return s.recipes.ListKitchenFitsByAuthor(ctx, userID)
}

func parsedFromVersion(version *model.RecipeVersion) *model.ParsedRecipe {
	parsed := &model.ParsedRecipe{
		Title:            version.Title,
		Servings:         version.Servings,
		EstimatedMinutes: version.EstimatedMinutes,
		Instructions:     version.Instructions,
	}
	for _, ingredient := range version.Ingredients {
		parsed.Ingredients = append(parsed.Ingredients, model.ParsedIngredient{
			Name:          ingredient.Name,
			Amount:        ingredient.Amount,
			Category:      ingredient.Category,
			IsExplicit:    ingredient.IsExplicit,
			SubstituteFor: ingredient.SubstituteFor,
		})
	}
	for _, step := range version.Steps {
		parsed.Steps = append(parsed.Steps, model.ParsedStep{
			StepOrder:       step.StepOrder,
			Description:     step.Description,
			VideoTimestamp:  step.VideoTimestamp,
			DurationSeconds: step.DurationSeconds,
			Heat:            step.Heat,
		})
	}
	return parsed
}

func hasAllergyRestriction(restrictions []model.DietaryRestriction) bool {
	for _, restriction := range restrictions {
		if restriction.RestrictionType == model.RestrictionTypeAllergy &&
			strings.TrimSpace(restriction.IngredientName) != "" {
			return true
		}
	}
	return false
}

func findAllergenInResult(result *model.AdaptationResult, restrictions []model.DietaryRestriction) string {
	var texts []string
	texts = append(texts, result.Title)
	if result.Instructions != nil {
		texts = append(texts, *result.Instructions)
	}
	for _, ingredient := range result.Ingredients {
		texts = append(texts, ingredient.Name)
	}
	for _, step := range result.Steps {
		texts = append(texts, step.Description)
	}
	for _, restriction := range restrictions {
		if restriction.RestrictionType != model.RestrictionTypeAllergy {
			continue
		}
		for _, text := range texts {
			if containsFold(text, restriction.IngredientName) {
				return restriction.IngredientName
			}
		}
	}
	return ""
}

func containsFold(text, needle string) bool {
	needle = strings.TrimSpace(needle)
	return needle != "" && strings.Contains(strings.ToLower(text), strings.ToLower(needle))
}

func blockReason(result *model.AdaptationResult) string {
	if result.BlockReason == nil {
		return "无法安全适配"
	}
	return *result.BlockReason
}
