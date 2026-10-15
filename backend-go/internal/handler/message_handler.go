package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// MessageHandler 消息聚合接口。
type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

// List 当前用户的消息列表。GET /api/v1/me/messages
func (h *MessageHandler) List(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	items, err := h.messageService.ListMessages(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询消息失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		var versionID any
		if item.RecipeVersionID != nil {
			versionID = item.RecipeVersionID.String()
		}
		list = append(list, gin.H{
			"type":              string(item.Type),
			"id":                item.ID.String(),
			"actor_nickname":    item.ActorNickname,
			"text":              item.Text,
			"rating":            item.Rating,
			"recipe_version_id": versionID,
			"recipe_title":      item.RecipeTitle,
			"created_at":        item.CreatedAt,
		})
	}
	response.Ok(c, gin.H{"items": list})
}
