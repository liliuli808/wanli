package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/middleware"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// KitchenHandler 厨房配置接口。
type KitchenHandler struct {
	kitchenService *service.KitchenService
}

// NewKitchenHandler 创建 KitchenHandler。
func NewKitchenHandler(kitchenService *service.KitchenService) *KitchenHandler {
	return &KitchenHandler{kitchenService: kitchenService}
}

type updateKitchenRequest struct {
	OwnedIngredients []string `json:"owned_ingredients"`
	OwnedTools       []string `json:"owned_tools"`
}

type restrictionItem struct {
	RestrictionType string `json:"restriction_type" binding:"required,oneof=ALLERGY DISLIKE MEDICAL"`
	IngredientName  string `json:"ingredient_name" binding:"required"`
}

type updateRestrictionsRequest struct {
	Restrictions []restrictionItem `json:"restrictions"`
}

func currentUserID(c *gin.Context) (uuid.UUID, *apperror.AppError) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return uuid.Nil, apperror.ErrUnauthorized
	}
	id, err := uuid.Parse(user.UserID)
	if err != nil {
		return uuid.Nil, apperror.ErrUnauthorized
	}
	return id, nil
}

// GetKitchen 获取当前用户厨房配置。
func (h *KitchenHandler) GetKitchen(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	profile, svcErr := h.kitchenService.GetProfile(c.Request.Context(), userID)
	if svcErr != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "获取厨房配置失败", http.StatusInternalServerError, svcErr))
		return
	}

	response.Ok(c, gin.H{
		"owned_ingredients": profile.OwnedIngredients,
		"owned_tools":       profile.OwnedTools,
	})
}

// UpdateKitchen 更新当前用户厨房配置。
func (h *KitchenHandler) UpdateKitchen(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req updateKitchenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	profile, svcErr := h.kitchenService.UpdateProfile(c.Request.Context(), userID, req.OwnedIngredients, req.OwnedTools)
	if svcErr != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "更新厨房配置失败", http.StatusInternalServerError, svcErr))
		return
	}

	response.Ok(c, gin.H{
		"owned_ingredients": profile.OwnedIngredients,
		"owned_tools":       profile.OwnedTools,
	})
}

// GetRestrictions 获取当前用户忌口。
func (h *KitchenHandler) GetRestrictions(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	restrictions, svcErr := h.kitchenService.GetRestrictions(c.Request.Context(), userID)
	if svcErr != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "获取忌口失败", http.StatusInternalServerError, svcErr))
		return
	}

	response.Ok(c, restrictionsToDTO(restrictions))
}

// UpdateRestrictions 更新当前用户忌口。
func (h *KitchenHandler) UpdateRestrictions(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	var req updateRestrictionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	items := make([]model.DietaryRestriction, 0, len(req.Restrictions))
	for _, item := range req.Restrictions {
		items = append(items, model.DietaryRestriction{
			UserID:          userID,
			RestrictionType: item.RestrictionType,
			IngredientName:  item.IngredientName,
		})
	}

	restrictions, svcErr := h.kitchenService.UpdateRestrictions(c.Request.Context(), userID, items)
	if svcErr != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "更新忌口失败", http.StatusInternalServerError, svcErr))
		return
	}

	response.Ok(c, restrictionsToDTO(restrictions))
}

func restrictionsToDTO(restrictions []model.DietaryRestriction) []gin.H {
	result := make([]gin.H, 0, len(restrictions))
	for _, r := range restrictions {
		result = append(result, gin.H{
			"id":               r.ID,
			"restriction_type": r.RestrictionType,
			"ingredient_name":  r.IngredientName,
			"created_at":       r.CreatedAt,
		})
	}
	return result
}
