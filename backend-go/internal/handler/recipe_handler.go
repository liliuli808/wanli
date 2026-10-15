package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/middleware"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// RecipeHandler 菜谱阅读、校对与发布 Handler（阶段 3）。
type RecipeHandler struct {
	recipeService *service.RecipeService
}

// NewRecipeHandler 创建菜谱 Handler。
func NewRecipeHandler(recipeService *service.RecipeService) *RecipeHandler {
	return &RecipeHandler{recipeService: recipeService}
}

// GetVersion 获取菜谱版本详情。GET /api/v1/recipe-versions/:id（游客可浏览公开版本）
func (h *RecipeHandler) GetVersion(c *gin.Context) {
	versionID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	var userID *uuid.UUID
	if user := middleware.GetCurrentUser(c); user != nil {
		if id, err := uuid.Parse(user.UserID); err == nil {
			userID = &id
		}
	}

	result, err := h.recipeService.GetVersionDetail(c.Request.Context(), userID, versionID)
	if err != nil {
		failRecipeError(c, err)
		return
	}
	response.Ok(c, versionDetailResponse(result))
}

// PublishVersion 发布来源解析草稿。POST /api/v1/recipe-versions/:id/publish
func (h *RecipeHandler) PublishVersion(c *gin.Context) {
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

	var req updateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	if _, err := h.recipeService.PublishSource(c.Request.Context(), userID, versionID, req.Title, req.toParsed()); err != nil {
		failRecipeError(c, err)
		return
	}
	response.Ok(c, gin.H{
		"recipe_version_id": versionID.String(),
		"is_public":         true,
	})
}

type updateDraftRequest struct {
	Title            string                 `json:"title" binding:"required"`
	Servings         *string                `json:"servings"`
	EstimatedMinutes *int                   `json:"estimated_minutes"`
	Instructions     *string                `json:"instructions"`
	Ingredients      []draftIngredientInput `json:"ingredients" binding:"required"`
	Steps            []draftStepInput       `json:"steps" binding:"required"`
}

type draftIngredientInput struct {
	Name       string  `json:"name" binding:"required"`
	Amount     *string `json:"amount"`
	Category   string  `json:"category" binding:"required"`
	IsExplicit bool    `json:"is_explicit"`
}

type draftStepInput struct {
	StepOrder       int     `json:"step_order" binding:"required"`
	Description     string  `json:"description" binding:"required"`
	VideoTimestamp  *int    `json:"video_timestamp"`
	DurationSeconds *int    `json:"duration_seconds"`
	Heat            *string `json:"heat"`
}

func (req *updateDraftRequest) toParsed() *model.ParsedRecipe {
	parsed := &model.ParsedRecipe{
		Title:            req.Title,
		Servings:         req.Servings,
		EstimatedMinutes: req.EstimatedMinutes,
		Instructions:     req.Instructions,
	}
	for _, ing := range req.Ingredients {
		parsed.Ingredients = append(parsed.Ingredients, model.ParsedIngredient{
			Name:       ing.Name,
			Amount:     ing.Amount,
			Category:   ing.Category,
			IsExplicit: ing.IsExplicit,
		})
	}
	for _, step := range req.Steps {
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

// UpdateDraft 校对阶段更新来源解析草稿。PUT /api/v1/recipe-versions/:id/draft
func (h *RecipeHandler) UpdateDraft(c *gin.Context) {
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

	var req updateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	result, err := h.recipeService.UpdateSourceDraft(c.Request.Context(), userID, versionID, req.Title, req.toParsed())
	if err != nil {
		failRecipeError(c, err)
		return
	}
	response.Ok(c, versionDetailResponse(result))
}

type forkVersionRequest struct {
	Title            string                 `json:"title" binding:"required"`
	ChangeLog        string                 `json:"change_log" binding:"required"`
	Servings         *string                `json:"servings"`
	EstimatedMinutes *int                   `json:"estimated_minutes"`
	Instructions     *string                `json:"instructions"`
	Ingredients      []draftIngredientInput `json:"ingredients" binding:"required"`
	Steps            []draftStepInput       `json:"steps" binding:"required"`
}

// ForkVersion 派生改编。POST /api/v1/recipe-versions/:id/forks
func (h *RecipeHandler) ForkVersion(c *gin.Context) {
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

	var req forkVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	parsed := &model.ParsedRecipe{
		Title:            req.Title,
		Servings:         req.Servings,
		EstimatedMinutes: req.EstimatedMinutes,
		Instructions:     req.Instructions,
	}
	for _, ing := range req.Ingredients {
		parsed.Ingredients = append(parsed.Ingredients, model.ParsedIngredient{
			Name:       ing.Name,
			Amount:     ing.Amount,
			Category:   ing.Category,
			IsExplicit: ing.IsExplicit,
		})
	}
	for _, step := range req.Steps {
		parsed.Steps = append(parsed.Steps, model.ParsedStep{
			StepOrder:       step.StepOrder,
			Description:     step.Description,
			VideoTimestamp:  step.VideoTimestamp,
			DurationSeconds: step.DurationSeconds,
			Heat:            step.Heat,
		})
	}

	result, err := h.recipeService.ForkVersion(c.Request.Context(), userID, versionID, req.ChangeLog, parsed)
	if err != nil {
		failRecipeError(c, err)
		return
	}
	response.Created(c, versionDetailResponse(result))
}

// ListRecipes 获取公开菜谱列表。GET /api/v1/recipes?sort=latest&page=1&page_size=20
func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	sort := c.DefaultQuery("sort", repository.RecipeSortLatest)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.recipeService.ListPublicRecipes(c.Request.Context(), sort, page, pageSize)
	if err != nil {
		failRecipeError(c, err)
		return
	}

	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"recipe_id":            item.RecipeID.String(),
			"title":                item.Title,
			"cover_url":            item.CoverURL,
			"up_nickname":          item.UpNickname,
			"video_status":         item.VideoStatus,
			"recipe_version_id":    item.SourceVersionID.String(),
			"servings":             item.Servings,
			"estimated_minutes":    item.EstimatedMinutes,
			"author_nickname":      item.AuthorNickname,
			"public_version_count": item.PublicVersionCount,
			"cooking_record_count": item.CookingRecordCount,
			"created_at":           item.CreatedAt,
		})
	}
	response.Ok(c, gin.H{
		"items":     list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetVersionTree 获取菜谱版本派生树。GET /api/v1/recipes/:id/version-tree
func (h *RecipeHandler) GetVersionTree(c *gin.Context) {
	recipeID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	nodes, err := h.recipeService.GetVersionTree(c.Request.Context(), recipeID)
	if err != nil {
		failRecipeError(c, err)
		return
	}

	list := make([]gin.H, 0, len(nodes))
	for _, node := range nodes {
		var parentID any
		if node.ParentID != nil {
			parentID = node.ParentID.String()
		}
		list = append(list, gin.H{
			"id":              node.ID.String(),
			"title":           node.Title,
			"version_type":    node.VersionType,
			"parent_id":       parentID,
			"author_nickname": node.AuthorNickname,
			"change_log":      node.ChangeLog,
			"created_at":      node.CreatedAt,
		})
	}
	response.Ok(c, gin.H{
		"recipe_id": recipeID.String(),
		"nodes":     list,
	})
}

// versionDetailResponse 组装版本详情响应（对齐 api-contract.md §2.3.1，另附校对所需字段）。
func versionDetailResponse(result *service.VersionDetailResult) gin.H {
	detail := result.Detail
	version := detail.Version

	videoStatus := model.VideoStatusExpired
	videoURL := any(nil)
	upNickname := ""
	coverURL := any(nil)
	if detail.Source != nil {
		videoStatus = detail.Source.Status
		videoURL = detail.Source.VideoURL
		upNickname = detail.Source.UpNickname
		if detail.Source.CoverURL != nil {
			coverURL = *detail.Source.CoverURL
		}
	}

	ingredients := make([]gin.H, 0, len(version.Ingredients))
	for _, ing := range version.Ingredients {
		ingredients = append(ingredients, gin.H{
			"name":           ing.Name,
			"amount":         ing.Amount,
			"category":       ing.Category,
			"is_explicit":    ing.IsExplicit,
			"substitute_for": ing.SubstituteFor,
		})
	}
	steps := make([]gin.H, 0, len(version.Steps))
	for _, step := range version.Steps {
		steps = append(steps, gin.H{
			"step_order":       step.StepOrder,
			"description":      step.Description,
			"video_timestamp":  step.VideoTimestamp,
			"duration_seconds": step.DurationSeconds,
			"heat":             step.Heat,
		})
	}

	videoWarning := any(nil)
	if result.VideoWarning != "" {
		videoWarning = result.VideoWarning
	}

	return gin.H{
		"id":                version.ID.String(),
		"recipe_id":         version.RecipeID.String(),
		"parent_id":         version.ParentID,
		"title":             version.Title,
		"up_nickname":       upNickname,
		"video_url":         videoURL,
		"video_status":      videoStatus,
		"video_warning":     videoWarning,
		"cover_url":         coverURL,
		"version_type":      version.VersionType,
		"is_public":         version.IsPublic,
		"author_nickname":   detail.AuthorNickname,
		"change_log":        version.ChangeLog,
		"servings":          version.Servings,
		"estimated_minutes": version.EstimatedMinutes,
		"instructions":      version.Instructions,
		"adaptation_notes":  version.AdaptationNotes,
		"ingredients":       ingredients,
		"steps":             steps,
	}
}

// parseUUIDParam 解析 URL 中的 UUID 参数。
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, *apperror.AppError) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, apperror.Wrap("VALIDATION_ERROR", "无效的 ID", http.StatusBadRequest, err)
	}
	return id, nil
}

// failRecipeError 将菜谱服务错误映射为统一错误响应。
func failRecipeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVersionNotFound):
		response.Fail(c, apperror.ErrResourceNotFound)
	case errors.Is(err, service.ErrNotVersionAuthor):
		response.FailWithDetails(c, apperror.ErrForbidden, "仅版本作者可以执行此操作")
	case errors.Is(err, service.ErrNotSourceDraft):
		response.FailWithDetails(c, apperror.ErrValidation, "仅待发布的来源解析草稿可以执行此操作")
	case errors.Is(err, service.ErrInvalidSort):
		response.FailWithDetails(c, apperror.ErrValidation, "不支持的排序方式")
	case errors.Is(err, service.ErrKitchenFitCannotFork):
		response.Fail(c, apperror.ErrKitchenFitCannotFork)
	case errors.Is(err, service.ErrDraftInvalid):
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
	default:
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "操作失败，请稍后再试", http.StatusInternalServerError, err))
	}
}
