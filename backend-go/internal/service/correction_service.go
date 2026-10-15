package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// 阶段 4：纠错与派生改编的业务错误。
var (
	ErrProposalNotFound     = errors.New("纠错建议不存在")
	ErrProposalReviewed     = errors.New("该建议已被处理")
	ErrNotSourcePublisher   = errors.New("仅来源发布者可以审核纠错建议")
	ErrInvalidReviewAction  = errors.New("无效的审核动作")
	ErrKitchenFitCannotFork = errors.New("个人适配版属于定制节点，不支持派生改编")
	ErrCorrectionInvalid    = errors.New("纠错内容校验失败")
)

// CorrectionContent 是纠错建议提交的校对副本全量内容。
type CorrectionContent struct {
	Title            string                   `json:"title"`
	Servings         *string                  `json:"servings"`
	EstimatedMinutes *int                     `json:"estimated_minutes"`
	Instructions     *string                  `json:"instructions"`
	Ingredients      []model.ParsedIngredient `json:"ingredients"`
	Steps            []model.ParsedStep       `json:"steps"`
}

func (c *CorrectionContent) toParsed() *model.ParsedRecipe {
	return &model.ParsedRecipe{
		Title:            c.Title,
		Servings:         c.Servings,
		EstimatedMinutes: c.EstimatedMinutes,
		Instructions:     c.Instructions,
		Ingredients:      c.Ingredients,
		Steps:            c.Steps,
	}
}

func (c *CorrectionContent) validate() error {
	if c == nil {
		return errors.New("纠错内容不能为空")
	}
	return c.toParsed().Validate()
}

// CorrectionService 纠错建议服务。
type CorrectionService struct {
	recipes     repository.RecipeRepository
	corrections repository.CorrectionRepository
}

// NewCorrectionService 创建纠错建议服务。
func NewCorrectionService(recipes repository.RecipeRepository, corrections repository.CorrectionRepository) *CorrectionService {
	return &CorrectionService{recipes: recipes, corrections: corrections}
}

// SubmitCorrection 对公开的来源解析版提交纠错建议（修改副本 + 理由）。
func (s *CorrectionService) SubmitCorrection(ctx context.Context, proposerID, versionID uuid.UUID, content *CorrectionContent, reason string) (*model.CorrectionProposal, error) {
	if reason == "" {
		return nil, fmt.Errorf("%w: 请填写纠错理由", ErrCorrectionInvalid)
	}
	if err := content.validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorrectionInvalid, err)
	}

	detail, err := s.recipes.FindVersionDetail(ctx, versionID)
	if err != nil {
		return nil, err
	}
	// 纠错只针对已公开的来源解析版
	if detail == nil || !isReadable(detail) || !detail.Version.IsPublic ||
		detail.Version.VersionType != model.VersionTypeSource {
		return nil, ErrVersionNotFound
	}

	changes, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal suggested changes: %w", err)
	}

	proposal := &model.CorrectionProposal{
		RecipeVersionID:  versionID,
		ProposerID:       proposerID,
		SuggestedChanges: changes,
		Reason:           reason,
		Status:           model.CorrectionStatusPending,
	}
	if err := s.corrections.CreateProposal(ctx, proposal); err != nil {
		return nil, err
	}
	return proposal, nil
}

// ReviewCorrection 来源发布者审核纠错建议；approve 时合入并生成新的 SOURCE 公开版本。
// 返回新生成的版本 ID（reject 时为 nil）。
func (s *CorrectionService) ReviewCorrection(ctx context.Context, reviewerID, proposalID uuid.UUID, action, rejectionReason string) (*uuid.UUID, error) {
	proposal, err := s.corrections.FindProposalByID(ctx, proposalID)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, ErrProposalNotFound
	}
	if proposal.Status != model.CorrectionStatusPending {
		return nil, ErrProposalReviewed
	}

	detail, err := s.recipes.FindVersionDetail(ctx, proposal.RecipeVersionID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrProposalNotFound
	}
	// 仅来源发布者（recipes.creator_id）可审核
	if detail.Recipe.CreatorID == nil || *detail.Recipe.CreatorID != reviewerID {
		return nil, ErrNotSourcePublisher
	}

	switch action {
	case "reject":
		if rejectionReason == "" {
			return nil, fmt.Errorf("%w: 请填写拒绝理由", ErrCorrectionInvalid)
		}
		err := s.corrections.RejectProposal(ctx, proposalID, rejectionReason)
		if errors.Is(err, repository.ErrCorrectionAlreadyReviewed) {
			return nil, ErrProposalReviewed
		}
		return nil, err
	case "approve":
		var content CorrectionContent
		if err := json.Unmarshal(proposal.SuggestedChanges, &content); err != nil {
			return nil, fmt.Errorf("unmarshal suggested changes: %w", err)
		}
		if err := content.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCorrectionInvalid, err)
		}

		version, err := s.corrections.ApproveWithNewVersion(ctx, proposalID, repository.DerivedVersionParams{
			RecipeID:    detail.Recipe.ID,
			ParentID:    &detail.Version.ID,
			AuthorID:    reviewerID,
			VersionType: model.VersionTypeSource,
			ChangeLog:   "合入纠错：" + proposal.Reason,
			Parsed:      content.toParsed(),
		})
		if err != nil {
			if errors.Is(err, repository.ErrCorrectionAlreadyReviewed) {
				return nil, ErrProposalReviewed
			}
			return nil, err
		}
		return &version.ID, nil
	default:
		return nil, ErrInvalidReviewAction
	}
}

// ListPending 查询当前用户作为来源发布者待审核的纠错建议。
func (s *CorrectionService) ListPending(ctx context.Context, creatorID uuid.UUID) ([]repository.PendingProposalItem, error) {
	return s.corrections.ListPendingByCreator(ctx, creatorID)
}

// ForkVersion 在公开版本基础上派生改编版（ADAPTED），KITCHEN_FIT 禁止派生。
func (s *RecipeService) ForkVersion(ctx context.Context, userID, versionID uuid.UUID, changeLog string, parsed *model.ParsedRecipe) (*VersionDetailResult, error) {
	detail, err := s.repo.FindVersionDetail(ctx, versionID)
	if err != nil {
		return nil, err
	}
	// 草稿不可被派生（对外表现为不存在）
	if detail == nil || !isReadable(detail) || !detail.Version.IsPublic {
		return nil, ErrVersionNotFound
	}
	if detail.Version.VersionType == model.VersionTypeKitchenFit {
		return nil, ErrKitchenFitCannotFork
	}
	if strings.TrimSpace(changeLog) == "" {
		return nil, fmt.Errorf("%w: 请填写改编说明", ErrDraftInvalid)
	}

	if err := parsed.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDraftInvalid, err)
	}

	version, err := s.repo.CreateDerivedVersion(ctx, repository.DerivedVersionParams{
		RecipeID:    detail.Recipe.ID,
		ParentID:    &versionID,
		AuthorID:    userID,
		VersionType: model.VersionTypeAdapted,
		ChangeLog:   strings.TrimSpace(changeLog),
		Parsed:      parsed,
	})
	if err != nil {
		return nil, err
	}
	return s.GetVersionDetail(ctx, &userID, version.ID)
}
