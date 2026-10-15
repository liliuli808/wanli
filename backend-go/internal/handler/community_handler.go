package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/middleware"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

type CommunityHandler struct {
	service *service.CommunityService
}

func NewCommunityHandler(communityService *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{service: communityService}
}

type createCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *CommunityHandler) CreateComment(c *gin.Context) {
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
	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	comment, err := h.service.CreateComment(c.Request.Context(), userID, versionID, req.Content)
	if err != nil {
		failCommunityError(c, err)
		return
	}
	response.Created(c, gin.H{
		"id":                comment.ID.String(),
		"recipe_version_id": comment.RecipeVersionID.String(),
		"content":           comment.Content,
		"is_mine":           true,
		"created_at":        comment.CreatedAt,
	})
}

func (h *CommunityHandler) ListComments(c *gin.Context) {
	versionID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	items, err := h.service.ListComments(c.Request.Context(), versionID)
	if err != nil {
		failCommunityError(c, err)
		return
	}
	viewerID := optionalCurrentUserID(c)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		isMine := viewerID != nil && item.UserID != nil && *viewerID == *item.UserID
		list = append(list, gin.H{
			"id":              item.ID.String(),
			"content":         item.Content,
			"author_nickname": item.AuthorNickname,
			"is_mine":         isMine,
			"created_at":      item.CreatedAt,
		})
	}
	response.Ok(c, gin.H{"items": list})
}

func (h *CommunityHandler) DeleteComment(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	commentID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	if err := h.service.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		failCommunityError(c, err)
		return
	}
	response.NoContent(c)
}

type createCookingRecordRequest struct {
	Rating   *int    `json:"rating"`
	PhotoURL *string `json:"photo_url"`
	Feedback *string `json:"feedback"`
}

func (h *CommunityHandler) CreateCookingRecord(c *gin.Context) {
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
	var req createCookingRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	record, err := h.service.CreateCookingRecord(
		c.Request.Context(), userID, versionID, req.Rating, req.PhotoURL, req.Feedback,
	)
	if err != nil {
		failCommunityError(c, err)
		return
	}
	response.Created(c, gin.H{
		"id":                record.ID.String(),
		"recipe_version_id": record.RecipeVersionID.String(),
		"rating":            record.Rating,
		"photo_url":         record.PhotoURL,
		"feedback":          record.Feedback,
		"is_mine":           true,
		"created_at":        record.CreatedAt,
	})
}

func (h *CommunityHandler) ListCookingRecords(c *gin.Context) {
	versionID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	items, err := h.service.ListCookingRecords(c.Request.Context(), versionID)
	if err != nil {
		failCommunityError(c, err)
		return
	}
	viewerID := optionalCurrentUserID(c)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":              item.ID.String(),
			"rating":          item.Rating,
			"photo_url":       item.PhotoURL,
			"feedback":        item.Feedback,
			"author_nickname": item.AuthorNickname,
			"is_mine":         viewerID != nil && *viewerID == item.UserID,
			"created_at":      item.CreatedAt,
		})
	}
	response.Ok(c, gin.H{"items": list})
}

func (h *CommunityHandler) DeleteCookingRecord(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	recordID, appErr := parseUUIDParam(c, "id")
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	if err := h.service.DeleteCookingRecord(c.Request.Context(), userID, recordID); err != nil {
		failCommunityError(c, err)
		return
	}
	response.NoContent(c)
}

// ListCookingFeed 全站「做过」动态流。GET /api/v1/feed/cooking-records?limit=20
func (h *CommunityHandler) ListCookingFeed(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.service.ListCookingFeed(c.Request.Context(), limit)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询动态失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":                item.ID.String(),
			"rating":            item.Rating,
			"photo_url":         item.PhotoURL,
			"feedback":          item.Feedback,
			"author_nickname":   item.AuthorNickname,
			"created_at":        item.CreatedAt,
			"recipe_version_id": item.RecipeVersionID.String(),
			"recipe_id":         item.RecipeID.String(),
			"recipe_title":      item.RecipeTitle,
			"cover_url":         item.CoverURL,
		})
	}
	response.Ok(c, gin.H{"items": list})
}

// ListMyCookingRecords 当前用户的「做过」列表。GET /api/v1/me/cooking-records
func (h *CommunityHandler) ListMyCookingRecords(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "60"))
	items, err := h.service.ListMyCookingRecords(c.Request.Context(), userID, limit)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":                item.ID.String(),
			"rating":            item.Rating,
			"photo_url":         item.PhotoURL,
			"feedback":          item.Feedback,
			"created_at":        item.CreatedAt,
			"recipe_version_id": item.RecipeVersionID.String(),
			"recipe_title":      item.RecipeTitle,
			"cover_url":         item.CoverURL,
		})
	}
	response.Ok(c, gin.H{"items": list})
}

func optionalCurrentUserID(c *gin.Context) *uuid.UUID {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return nil
	}
	id, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil
	}
	return &id
}

func failCommunityError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCommunityVersionNotFound):
		response.Fail(c, apperror.ErrResourceNotFound)
	case errors.Is(err, service.ErrCommunityContentInvalid):
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
	case errors.Is(err, service.ErrCommunityItemNotOwned):
		response.Fail(c, apperror.ErrForbidden)
	default:
		response.Fail(c, apperror.Wrap(
			"INTERNAL_ERROR", "社区内容操作失败", http.StatusInternalServerError, err,
		))
	}
}
