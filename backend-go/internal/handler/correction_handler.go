package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// CorrectionHandler 纠错建议 Handler（阶段 4）。
type CorrectionHandler struct {
	correctionService *service.CorrectionService
}

// NewCorrectionHandler 创建纠错建议 Handler。
func NewCorrectionHandler(correctionService *service.CorrectionService) *CorrectionHandler {
	return &CorrectionHandler{correctionService: correctionService}
}

type submitCorrectionRequest struct {
	SuggestedChanges correctionContentInput `json:"suggested_changes" binding:"required"`
	Reason           string                 `json:"reason" binding:"required"`
}

type correctionContentInput struct {
	Title            string                 `json:"title" binding:"required"`
	Servings         *string                `json:"servings"`
	EstimatedMinutes *int                   `json:"estimated_minutes"`
	Instructions     *string                `json:"instructions"`
	Ingredients      []draftIngredientInput `json:"ingredients" binding:"required"`
	Steps            []draftStepInput       `json:"steps" binding:"required"`
}

func (in *correctionContentInput) toContent() *service.CorrectionContent {
	content := &service.CorrectionContent{
		Title:            in.Title,
		Servings:         in.Servings,
		EstimatedMinutes: in.EstimatedMinutes,
		Instructions:     in.Instructions,
	}
	for _, ing := range in.Ingredients {
		content.Ingredients = append(content.Ingredients, model.ParsedIngredient{
			Name:       ing.Name,
			Amount:     ing.Amount,
			Category:   ing.Category,
			IsExplicit: ing.IsExplicit,
		})
	}
	for _, step := range in.Steps {
		content.Steps = append(content.Steps, model.ParsedStep{
			StepOrder:       step.StepOrder,
			Description:     step.Description,
			VideoTimestamp:  step.VideoTimestamp,
			DurationSeconds: step.DurationSeconds,
			Heat:            step.Heat,
		})
	}
	return content
}

// SubmitCorrection 提交纠错建议。POST /api/v1/recipe-versions/:id/corrections
func (h *CorrectionHandler) SubmitCorrection(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	versionID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	var req submitCorrectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	proposal, err := h.correctionService.SubmitCorrection(
		c.Request.Context(), userID, versionID, req.SuggestedChanges.toContent(), req.Reason)
	if err != nil {
		failCorrectionError(c, err)
		return
	}
	response.Created(c, gin.H{
		"proposal_id": proposal.ID.String(),
		"status":      proposal.Status,
	})
}

type reviewCorrectionRequest struct {
	Action          string `json:"action" binding:"required"` // approve | reject
	RejectionReason string `json:"rejection_reason"`
}

// ReviewCorrection 审核纠错建议。POST /api/v1/correction-proposals/:id/review
func (h *CorrectionHandler) ReviewCorrection(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	proposalID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	var req reviewCorrectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	newVersionID, err := h.correctionService.ReviewCorrection(
		c.Request.Context(), userID, proposalID, req.Action, req.RejectionReason)
	if err != nil {
		failCorrectionError(c, err)
		return
	}

	data := gin.H{"action": req.Action}
	if newVersionID != nil {
		data["recipe_version_id"] = newVersionID.String()
	}
	response.Ok(c, data)
}

// ListPending 查询当前用户待审核的纠错建议。GET /api/v1/me/correction-proposals
func (h *CorrectionHandler) ListPending(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	items, err := h.correctionService.ListPending(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}

	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":                item.Proposal.ID.String(),
			"recipe_version_id": item.Proposal.RecipeVersionID.String(),
			"recipe_title":      item.RecipeTitle,
			"proposer_nickname": item.ProposerNickname,
			"reason":            item.Proposal.Reason,
			"suggested_changes": item.Proposal.SuggestedChanges,
			"created_at":        item.Proposal.CreatedAt,
		})
	}
	response.Ok(c, gin.H{"items": list})
}

// failCorrectionError 将纠错服务错误映射为统一错误响应。
func failCorrectionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrProposalNotFound):
		response.Fail(c, apperror.ErrResourceNotFound)
	case errors.Is(err, service.ErrVersionNotFound):
		response.Fail(c, apperror.ErrResourceNotFound)
	case errors.Is(err, service.ErrNotSourcePublisher):
		response.FailWithDetails(c, apperror.ErrForbidden, "仅来源发布者可以审核纠错建议")
	case errors.Is(err, service.ErrProposalReviewed):
		response.FailWithDetails(c, apperror.ErrValidation, "该建议已被处理")
	case errors.Is(err, service.ErrInvalidReviewAction):
		response.FailWithDetails(c, apperror.ErrValidation, "无效的审核动作，仅支持 approve / reject")
	case errors.Is(err, service.ErrCorrectionInvalid):
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
	default:
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "操作失败，请稍后再试", http.StatusInternalServerError, err))
	}
}
