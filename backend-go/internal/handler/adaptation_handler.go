package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// AdaptationHandler 提供个人厨房适配、可见性和个人适配版列表接口。
type AdaptationHandler struct {
	adaptationService *service.AdaptationService
}

func NewAdaptationHandler(adaptationService *service.AdaptationService) *AdaptationHandler {
	return &AdaptationHandler{adaptationService: adaptationService}
}

// AdaptVersion 生成默认私有的 KITCHEN_FIT 版本。
// POST /api/v1/recipe-versions/:id/adaptations
func (h *AdaptationHandler) AdaptVersion(c *gin.Context) {
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

	result, err := h.adaptationService.AdaptVersion(c.Request.Context(), userID, versionID)
	if err != nil {
		failAdaptationError(c, err)
		return
	}
	response.Ok(c, versionDetailResponse(result))
}

type updateVisibilityRequest struct {
	IsPublic *bool `json:"is_public" binding:"required"`
}

// SetVisibility 仅允许作者修改个人适配版的公开状态。
// PATCH /api/v1/recipe-versions/:id/visibility
func (h *AdaptationHandler) SetVisibility(c *gin.Context) {
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

	var req updateVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.IsPublic == nil {
		response.FailWithDetails(c, apperror.ErrValidation, "is_public 必须为布尔值")
		return
	}
	result, err := h.adaptationService.SetVisibility(
		c.Request.Context(), userID, versionID, *req.IsPublic,
	)
	if err != nil {
		failAdaptationError(c, err)
		return
	}
	response.Ok(c, versionDetailResponse(result))
}

// ListMine 查询当前用户的全部个人适配版，包括私有版本。
// GET /api/v1/me/adaptations
func (h *AdaptationHandler) ListMine(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	items, err := h.adaptationService.ListMine(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, apperror.Wrap(
			"INTERNAL_ERROR", "查询个人适配版失败", http.StatusInternalServerError, err,
		))
		return
	}

	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":         item.ID.String(),
			"recipe_id":  item.RecipeID.String(),
			"parent_id":  item.ParentID,
			"title":      item.Title,
			"is_public":  item.IsPublic,
			"created_at": item.CreatedAt,
		})
	}
	response.Ok(c, gin.H{"items": list})
}

func failAdaptationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVersionNotFound):
		response.Fail(c, apperror.ErrResourceNotFound)
	case errors.Is(err, service.ErrNotVersionAuthor):
		response.FailWithDetails(c, apperror.ErrForbidden, "仅版本作者可以修改公开状态")
	case errors.Is(err, service.ErrKitchenFitNotOwned):
		response.FailWithDetails(c, apperror.ErrForbidden, "仅可修改自己的个人适配版")
	case errors.Is(err, service.ErrKitchenFitCannotFork):
		response.Fail(c, apperror.ErrKitchenFitCannotFork)
	case errors.Is(err, service.ErrAllergyBlocked):
		response.FailWithDetails(c, apperror.ErrAllergyBlocked, gin.H{"reason": err.Error()})
	case errors.Is(err, service.ErrDietaryBlocked):
		response.FailWithDetails(c, apperror.ErrDietaryBlocked, gin.H{"reason": err.Error()})
	case errors.Is(err, service.ErrAIAdaptationFailed):
		response.Fail(c, apperror.ErrAIAdaptationFailed)
	default:
		response.Fail(c, apperror.Wrap(
			"INTERNAL_ERROR", "厨房适配操作失败", http.StatusInternalServerError, err,
		))
	}
}
