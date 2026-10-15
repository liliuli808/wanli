package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// memoryCorrectionRepo 是 CorrectionRepository 的内存实现。
type memoryCorrectionRepo struct {
	proposals map[uuid.UUID]*model.CorrectionProposal
	recipes   *memoryRecipeRepo // approve 时复用其 CreateDerivedVersion
}

func newMemoryCorrectionRepo(recipes *memoryRecipeRepo) *memoryCorrectionRepo {
	return &memoryCorrectionRepo{
		proposals: map[uuid.UUID]*model.CorrectionProposal{},
		recipes:   recipes,
	}
}

func (r *memoryCorrectionRepo) CreateProposal(_ context.Context, proposal *model.CorrectionProposal) error {
	proposal.ID = uuid.New()
	r.proposals[proposal.ID] = proposal
	return nil
}

func (r *memoryCorrectionRepo) FindProposalByID(_ context.Context, id uuid.UUID) (*model.CorrectionProposal, error) {
	return r.proposals[id], nil
}

func (r *memoryCorrectionRepo) RejectProposal(_ context.Context, id uuid.UUID, rejectionReason string) error {
	proposal := r.proposals[id]
	if proposal == nil || proposal.Status != model.CorrectionStatusPending {
		return assert.AnError
	}
	proposal.Status = model.CorrectionStatusRejected
	proposal.RejectionReason = &rejectionReason
	return nil
}

func (r *memoryCorrectionRepo) ApproveWithNewVersion(ctx context.Context, proposalID uuid.UUID, params repository.DerivedVersionParams) (*model.RecipeVersion, error) {
	proposal := r.proposals[proposalID]
	if proposal == nil || proposal.Status != model.CorrectionStatusPending {
		return nil, assert.AnError
	}
	version, err := r.recipes.CreateDerivedVersion(ctx, params)
	if err != nil {
		return nil, err
	}
	proposal.Status = model.CorrectionStatusApproved
	proposal.ApprovedVersionID = &version.ID
	return version, nil
}

func (r *memoryCorrectionRepo) ListReviewedByProposer(_ context.Context, proposerID uuid.UUID, limit int) ([]repository.ReviewedProposalItem, error) {
	return nil, nil
}

func (r *memoryCorrectionRepo) ListPendingByCreator(_ context.Context, creatorID uuid.UUID) ([]repository.PendingProposalItem, error) {
	var items []repository.PendingProposalItem
	for _, p := range r.proposals {
		if p.Status != model.CorrectionStatusPending {
			continue
		}
		version := r.recipes.versions[p.RecipeVersionID]
		if version == nil {
			continue
		}
		recipe := r.recipes.recipes[version.RecipeID]
		if recipe == nil || recipe.CreatorID == nil || *recipe.CreatorID != creatorID {
			continue
		}
		items = append(items, repository.PendingProposalItem{
			Proposal:         p,
			RecipeTitle:      recipe.Title,
			ProposerNickname: r.recipes.users[p.ProposerID],
		})
	}
	return items, nil
}

func validCorrectionContent() *CorrectionContent {
	amount := "600克"
	return &CorrectionContent{
		Title: "家常红烧肉",
		Ingredients: []model.ParsedIngredient{
			{Name: "五花肉", Amount: &amount, Category: model.IngredientCategoryMain, IsExplicit: true},
		},
		Steps: []model.ParsedStep{
			{StepOrder: 1, Description: "焯水去腥"},
			{StepOrder: 2, Description: "炒糖色"},
		},
	}
}

func TestSubmitCorrection_Success(t *testing.T) {
	recipes := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, versionID := recipes.seed(authorID, model.VersionTypeSource, true)
	svc := NewCorrectionService(recipes, newMemoryCorrectionRepo(recipes))

	proposal, err := svc.SubmitCorrection(context.Background(), uuid.New(), versionID, validCorrectionContent(), "视频第2步说要焯水")
	require.NoError(t, err)
	assert.Equal(t, model.CorrectionStatusPending, proposal.Status)

	// suggested_changes 可还原为提交内容
	var content CorrectionContent
	require.NoError(t, json.Unmarshal(proposal.SuggestedChanges, &content))
	require.Len(t, content.Ingredients, 1)
	assert.Equal(t, "600克", *content.Ingredients[0].Amount)
}

func TestSubmitCorrection_RejectsInvalid(t *testing.T) {
	recipes := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, publicSourceID := recipes.seed(authorID, model.VersionTypeSource, true)
	_, draftID := recipes.seed(authorID, model.VersionTypeSource, false)
	_, adaptedID := recipes.seed(authorID, model.VersionTypeAdapted, true)
	svc := NewCorrectionService(recipes, newMemoryCorrectionRepo(recipes))

	// 空理由
	_, err := svc.SubmitCorrection(context.Background(), uuid.New(), publicSourceID, validCorrectionContent(), "")
	assert.ErrorIs(t, err, ErrCorrectionInvalid)

	// 空食材
	_, err = svc.SubmitCorrection(context.Background(), uuid.New(), publicSourceID,
		&CorrectionContent{Title: "红烧肉", Steps: []model.ParsedStep{{StepOrder: 1, Description: "x"}}}, "理由")
	assert.ErrorIs(t, err, ErrCorrectionInvalid)

	// 草稿不可被纠错
	_, err = svc.SubmitCorrection(context.Background(), uuid.New(), draftID, validCorrectionContent(), "理由")
	assert.ErrorIs(t, err, ErrVersionNotFound)

	// ADAPTED 版本不走来源纠错（用派生改编）
	_, err = svc.SubmitCorrection(context.Background(), uuid.New(), adaptedID, validCorrectionContent(), "理由")
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestReviewCorrection_ApproveGeneratesNewSourceVersion(t *testing.T) {
	recipes := newMemoryRecipeRepo()
	publisherID := uuid.New()
	proposerID := uuid.New()
	recipeID, versionID := recipes.seed(publisherID, model.VersionTypeSource, true)
	corrections := newMemoryCorrectionRepo(recipes)
	svc := NewCorrectionService(recipes, corrections)

	proposal, err := svc.SubmitCorrection(context.Background(), proposerID, versionID, validCorrectionContent(), "AI 漏了焯水")
	require.NoError(t, err)

	newVersionID, err := svc.ReviewCorrection(context.Background(), publisherID, proposal.ID, "approve", "")
	require.NoError(t, err)
	require.NotNil(t, newVersionID)

	// 生成的新版本：SOURCE、公开、挂在原版本下、作者为来源发布者、change_log 记录理由
	newVersion := recipes.versions[*newVersionID]
	require.NotNil(t, newVersion)
	assert.Equal(t, recipeID, newVersion.RecipeID)
	assert.Equal(t, &versionID, newVersion.ParentID)
	assert.Equal(t, model.VersionTypeSource, newVersion.VersionType)
	assert.True(t, newVersion.IsPublic)
	assert.Equal(t, &publisherID, newVersion.AuthorID)
	assert.Contains(t, *newVersion.ChangeLog, "AI 漏了焯水")
	require.Len(t, newVersion.Ingredients, 1)
	require.Len(t, newVersion.Steps, 2)

	// 建议状态与 approved_version_id 回写
	updated := corrections.proposals[proposal.ID]
	assert.Equal(t, model.CorrectionStatusApproved, updated.Status)
	assert.Equal(t, newVersionID, updated.ApprovedVersionID)
}

func TestReviewCorrection_PermissionsAndState(t *testing.T) {
	recipes := newMemoryRecipeRepo()
	publisherID := uuid.New()
	_, versionID := recipes.seed(publisherID, model.VersionTypeSource, true)
	svc := NewCorrectionService(recipes, newMemoryCorrectionRepo(recipes))

	proposal, err := svc.SubmitCorrection(context.Background(), uuid.New(), versionID, validCorrectionContent(), "理由")
	require.NoError(t, err)

	// 非来源发布者（含提议人自己）不能审核
	_, err = svc.ReviewCorrection(context.Background(), uuid.New(), proposal.ID, "approve", "")
	assert.ErrorIs(t, err, ErrNotSourcePublisher)

	// 无效动作
	_, err = svc.ReviewCorrection(context.Background(), publisherID, proposal.ID, "maybe", "")
	assert.ErrorIs(t, err, ErrInvalidReviewAction)

	// 拒绝
	_, err = svc.ReviewCorrection(context.Background(), publisherID, proposal.ID, "reject", "与原视频不符")
	require.NoError(t, err)
	assert.Equal(t, "与原视频不符", *proposal.RejectionReason)

	// 已处理的建议不能重复审核
	_, err = svc.ReviewCorrection(context.Background(), publisherID, proposal.ID, "approve", "")
	assert.ErrorIs(t, err, ErrProposalReviewed)

	// 不存在的建议
	_, err = svc.ReviewCorrection(context.Background(), publisherID, uuid.New(), "approve", "")
	assert.ErrorIs(t, err, ErrProposalNotFound)
}

func TestListPending_OnlyOwnRecipes(t *testing.T) {
	recipes := newMemoryRecipeRepo()
	publisherID := uuid.New()
	otherPublisherID := uuid.New()
	_, myVersionID := recipes.seed(publisherID, model.VersionTypeSource, true)
	_, otherVersionID := recipes.seed(otherPublisherID, model.VersionTypeSource, true)
	svc := NewCorrectionService(recipes, newMemoryCorrectionRepo(recipes))

	_, err := svc.SubmitCorrection(context.Background(), uuid.New(), myVersionID, validCorrectionContent(), "我的菜被纠错")
	require.NoError(t, err)
	_, err = svc.SubmitCorrection(context.Background(), uuid.New(), otherVersionID, validCorrectionContent(), "别人的菜被纠错")
	require.NoError(t, err)

	items, err := svc.ListPending(context.Background(), publisherID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "我的菜被纠错", items[0].Proposal.Reason)
	assert.Equal(t, "家常红烧肉", items[0].RecipeTitle)
}

func TestForkVersion_Success(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	recipeID, sourceID := repo.seed(authorID, model.VersionTypeSource, true)
	svc := NewRecipeService(repo)

	forkerID := uuid.New()
	parsed := &model.ParsedRecipe{
		Title:       "空气炸锅红烧肉",
		Ingredients: []model.ParsedIngredient{{Name: "鸡胸肉", Category: model.IngredientCategoryMain, IsExplicit: true}},
		Steps:       []model.ParsedStep{{StepOrder: 1, Description: "空气炸锅 180 度 20 分钟"}},
	}
	result, err := svc.ForkVersion(context.Background(), forkerID, sourceID, "减脂空气炸锅版", parsed)
	require.NoError(t, err)

	version := result.Detail.Version
	assert.Equal(t, recipeID, version.RecipeID)
	assert.Equal(t, &sourceID, version.ParentID)
	assert.Equal(t, model.VersionTypeAdapted, version.VersionType)
	assert.True(t, version.IsPublic)
	assert.Equal(t, &forkerID, version.AuthorID)
	assert.Equal(t, "减脂空气炸锅版", *version.ChangeLog)
	assert.Equal(t, "鸡胸肉", version.Ingredients[0].Name)
}

func TestForkVersion_Restrictions(t *testing.T) {
	repo := newMemoryRecipeRepo()
	authorID := uuid.New()
	_, sourceID := repo.seed(authorID, model.VersionTypeSource, true)
	_, adaptedID := repo.seed(authorID, model.VersionTypeAdapted, true)
	_, kitchenFitID := repo.seed(authorID, model.VersionTypeKitchenFit, true)
	_, draftID := repo.seed(authorID, model.VersionTypeSource, false)
	svc := NewRecipeService(repo)

	parsed := &model.ParsedRecipe{
		Title:       "空气炸锅红烧肉",
		Ingredients: []model.ParsedIngredient{{Name: "鸡胸肉", Category: model.IngredientCategoryMain}},
		Steps:       []model.ParsedStep{{StepOrder: 1, Description: "空气炸锅"}},
	}

	// KITCHEN_FIT 禁止派生（验收：个人适配版无法被改编）
	_, err := svc.ForkVersion(context.Background(), uuid.New(), kitchenFitID, "", parsed)
	assert.ErrorIs(t, err, ErrKitchenFitCannotFork)

	// 草稿不可被派生
	_, err = svc.ForkVersion(context.Background(), uuid.New(), draftID, "", parsed)
	assert.ErrorIs(t, err, ErrVersionNotFound)

	// ADAPTED 可被再次改编（验收：社区改编版可以被再次改编）
	result, err := svc.ForkVersion(context.Background(), uuid.New(), adaptedID, "二次改编", parsed)
	require.NoError(t, err)
	assert.Equal(t, &adaptedID, result.Detail.Version.ParentID)
	assert.Equal(t, model.VersionTypeAdapted, result.Detail.Version.VersionType)

	// 改编说明必填
	_, err = svc.ForkVersion(context.Background(), uuid.New(), sourceID, " ", parsed)
	assert.ErrorIs(t, err, ErrDraftInvalid)

	// 内容校验
	_, err = svc.ForkVersion(context.Background(), uuid.New(), sourceID, "", &model.ParsedRecipe{})
	assert.ErrorIs(t, err, ErrDraftInvalid)
}
