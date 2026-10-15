package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/infrastructure/ai"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

type fakeAdaptationRepo struct {
	details      map[uuid.UUID]*repository.VersionDetail
	created      *repository.KitchenFitCreateParams
	visibilityID uuid.UUID
	visibility   bool
	list         []repository.KitchenFitListItem
}

func (r *fakeAdaptationRepo) FindVersionDetail(_ context.Context, versionID uuid.UUID) (*repository.VersionDetail, error) {
	return r.details[versionID], nil
}

func (r *fakeAdaptationRepo) CreateKitchenFitVersion(
	_ context.Context,
	params repository.KitchenFitCreateParams,
) (*model.RecipeVersion, error) {
	r.created = &params
	id := uuid.New()
	version := &model.RecipeVersion{
		ID:          id,
		RecipeID:    params.RecipeID,
		ParentID:    &params.ParentID,
		AuthorID:    &params.AuthorID,
		VersionType: model.VersionTypeKitchenFit,
		Title:       params.Result.Title,
		IsPublic:    false,
		Status:      model.ContentStatusActive,
	}
	r.details[id] = &repository.VersionDetail{
		Version: version,
		Recipe:  r.details[params.ParentID].Recipe,
	}
	return version, nil
}

func (r *fakeAdaptationRepo) SetKitchenFitVisibility(
	_ context.Context,
	_ uuid.UUID,
	versionID uuid.UUID,
	isPublic bool,
) error {
	r.visibilityID = versionID
	r.visibility = isPublic
	return nil
}

func (r *fakeAdaptationRepo) ListKitchenFitsByAuthor(
	_ context.Context,
	_ uuid.UUID,
) ([]repository.KitchenFitListItem, error) {
	return r.list, nil
}

type fakeAdaptationKitchenRepo struct {
	profile      *model.KitchenProfile
	restrictions []model.DietaryRestriction
}

func (r *fakeAdaptationKitchenRepo) GetProfile(context.Context, uuid.UUID) (*model.KitchenProfile, error) {
	return r.profile, nil
}

func (r *fakeAdaptationKitchenRepo) SaveProfile(context.Context, *model.KitchenProfile) error {
	return nil
}

func (r *fakeAdaptationKitchenRepo) GetRestrictions(context.Context, uuid.UUID) ([]model.DietaryRestriction, error) {
	return r.restrictions, nil
}

func (r *fakeAdaptationKitchenRepo) SaveRestrictions(context.Context, uuid.UUID, []model.DietaryRestriction) error {
	return nil
}

type fakeAdaptationAI struct {
	result *model.AdaptationResult
	err    error
	input  *ai.AdaptationInput
}

func (f *fakeAdaptationAI) AdaptRecipe(_ context.Context, input ai.AdaptationInput) (*model.AdaptationResult, error) {
	f.input = &input
	return f.result, f.err
}

func adaptationFixture() (
	uuid.UUID,
	uuid.UUID,
	uuid.UUID,
	*fakeAdaptationRepo,
	*fakeAdaptationKitchenRepo,
) {
	userID := uuid.New()
	recipeID := uuid.New()
	versionID := uuid.New()
	amount := "2勺"
	repo := &fakeAdaptationRepo{details: map[uuid.UUID]*repository.VersionDetail{}}
	repo.details[versionID] = &repository.VersionDetail{
		Recipe: &model.Recipe{
			ID:     recipeID,
			Status: model.ContentStatusActive,
		},
		Version: &model.RecipeVersion{
			ID:          versionID,
			RecipeID:    recipeID,
			AuthorID:    &userID,
			VersionType: model.VersionTypeSource,
			Title:       "麻酱拌面",
			IsPublic:    true,
			Status:      model.ContentStatusActive,
			Ingredients: []model.RecipeIngredient{
				{Name: "花生酱", Amount: &amount, Category: model.IngredientCategorySeasoning, IsExplicit: true},
			},
			Steps: []model.RecipeStep{{StepOrder: 1, Description: "拌匀"}},
		},
	}
	kitchen := &fakeAdaptationKitchenRepo{
		profile: &model.KitchenProfile{
			UserID:           userID,
			OwnedIngredients: []string{"芝麻酱"},
			OwnedTools:       []string{"汤锅"},
		},
		restrictions: []model.DietaryRestriction{{
			UserID:          userID,
			RestrictionType: model.RestrictionTypeAllergy,
			IngredientName:  "花生",
		}},
	}
	return userID, recipeID, versionID, repo, kitchen
}

func safeAdaptationResult() *model.AdaptationResult {
	replacement := "花生酱"
	return &model.AdaptationResult{
		Safe:  true,
		Title: "芝麻酱拌面",
		Ingredients: []model.ParsedIngredient{{
			Name:          "芝麻酱",
			Category:      model.IngredientCategorySeasoning,
			IsExplicit:    true,
			SubstituteFor: &replacement,
		}},
		Steps: []model.ParsedStep{{StepOrder: 1, Description: "加入芝麻酱拌匀"}},
		AdaptationNotes: model.AdaptationNotes{
			Substitutions: []model.AdaptationSubstitution{{
				Original: "花生酱", Replacement: "芝麻酱", Reason: "规避过敏源",
			}},
			FlavorChanges: []string{"坚果香气会有变化"},
		},
	}
}

func TestAdaptVersion_SavesPrivateSafeKitchenFit(t *testing.T) {
	userID, recipeID, versionID, repo, kitchen := adaptationFixture()
	aiClient := &fakeAdaptationAI{result: safeAdaptationResult()}
	svc := NewAdaptationService(repo, kitchen, aiClient)

	result, err := svc.AdaptVersion(context.Background(), userID, versionID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, repo.created)
	assert.Equal(t, recipeID, repo.created.RecipeID)
	assert.Equal(t, versionID, repo.created.ParentID)
	assert.Equal(t, userID, repo.created.AuthorID)
	assert.Equal(t, "芝麻酱拌面", repo.created.Result.Title)
	assert.Equal(t, "芝麻酱", aiClient.input.KitchenProfile.OwnedIngredients[0])
	assert.False(t, result.Detail.Version.IsPublic, "个人适配版必须默认私有")
	assert.Equal(t, model.VersionTypeKitchenFit, result.Detail.Version.VersionType)
}

func TestAdaptVersion_AIUnsafeWithAllergyIsBlocked(t *testing.T) {
	userID, _, versionID, repo, kitchen := adaptationFixture()
	reason := "无法确认安全替代品"
	aiClient := &fakeAdaptationAI{result: &model.AdaptationResult{Safe: false, BlockReason: &reason}}
	svc := NewAdaptationService(repo, kitchen, aiClient)

	_, err := svc.AdaptVersion(context.Background(), userID, versionID)
	assert.ErrorIs(t, err, ErrAllergyBlocked)
	assert.Nil(t, repo.created, "被拦截的结果不得保存")
}

func TestAdaptVersion_PostValidationRejectsAllergenLeak(t *testing.T) {
	userID, _, versionID, repo, kitchen := adaptationFixture()
	unsafeResult := safeAdaptationResult()
	unsafeResult.Steps[0].Description = "最后撒入花生碎"
	svc := NewAdaptationService(repo, kitchen, &fakeAdaptationAI{result: unsafeResult})

	_, err := svc.AdaptVersion(context.Background(), userID, versionID)
	assert.ErrorIs(t, err, ErrAllergyBlocked)
	assert.ErrorContains(t, err, "花生")
	assert.Nil(t, repo.created, "后置安全校验失败时不得保存")
}

func TestAdaptVersion_AIFailureDoesNotSave(t *testing.T) {
	userID, _, versionID, repo, kitchen := adaptationFixture()
	svc := NewAdaptationService(repo, kitchen, &fakeAdaptationAI{err: errors.New("timeout")})

	_, err := svc.AdaptVersion(context.Background(), userID, versionID)
	assert.ErrorIs(t, err, ErrAIAdaptationFailed)
	assert.Nil(t, repo.created)
}

func TestAdaptVersion_KitchenFitIsTerminalNode(t *testing.T) {
	userID, _, versionID, repo, kitchen := adaptationFixture()
	repo.details[versionID].Version.VersionType = model.VersionTypeKitchenFit
	svc := NewAdaptationService(repo, kitchen, &fakeAdaptationAI{result: safeAdaptationResult()})

	_, err := svc.AdaptVersion(context.Background(), userID, versionID)
	assert.ErrorIs(t, err, ErrKitchenFitCannotFork)
	assert.Nil(t, repo.created)
}

func TestAdaptationVisibilityAndList(t *testing.T) {
	userID, recipeID, versionID, repo, kitchen := adaptationFixture()
	repo.details[versionID].Version.VersionType = model.VersionTypeKitchenFit
	repo.details[versionID].Version.IsPublic = false
	repo.list = []repository.KitchenFitListItem{{
		ID: versionID, RecipeID: recipeID, Title: "我的拌面", CreatedAt: time.Now(),
	}}
	svc := NewAdaptationService(repo, kitchen, &fakeAdaptationAI{})

	result, err := svc.SetVisibility(context.Background(), userID, versionID, true)
	require.NoError(t, err)
	assert.True(t, result.Detail.Version.IsPublic)
	assert.Equal(t, versionID, repo.visibilityID)
	assert.True(t, repo.visibility)

	items, err := svc.ListMine(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "我的拌面", items[0].Title)
}
