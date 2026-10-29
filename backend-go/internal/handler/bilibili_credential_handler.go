package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// BilibiliCredentialHandler 用户级 B 站登录态授权接口。
type BilibiliCredentialHandler struct {
	service *service.BilibiliCredentialService
}

// NewBilibiliCredentialHandler 创建 Handler。
func NewBilibiliCredentialHandler(svc *service.BilibiliCredentialService) *BilibiliCredentialHandler {
	return &BilibiliCredentialHandler{service: svc}
}

type bindBilibiliCredentialRequest struct {
	// Sessdata 是 B 站登录 Cookie 中的 SESSDATA 值，不接受完整 Cookie 串。
	Sessdata string `json:"sessdata" binding:"required"`
}

// credentialResponse 只返回展示信息，绝不回传凭证内容。
func credentialResponse(status *service.BilibiliCredentialStatus) gin.H {
	out := gin.H{
		"bound": status.Bound,
		"mid":   status.Mid,
		"uname": status.Uname,
	}
	if status.UpdatedAt != nil {
		out["updated_at"] = status.UpdatedAt.Format(time.RFC3339)
	}
	return out
}

// Get 查询当前用户的 B 站授权状态。GET /api/v1/me/bilibili-credential
func (h *BilibiliCredentialHandler) Get(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	status, err := h.service.Status(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询授权状态失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, credentialResponse(status))
}

// Bind 保存用户授权的 SESSDATA。PUT /api/v1/me/bilibili-credential
func (h *BilibiliCredentialHandler) Bind(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	var req bindBilibiliCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	status, err := h.service.Bind(c.Request.Context(), userID, req.Sessdata)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBilibiliCredentialInvalid):
			response.Fail(c, apperror.New("BILIBILI_CREDENTIAL_INVALID",
				"SESSDATA 无效或已过期，请重新登录 B 站后再试", http.StatusBadRequest))
		case errors.Is(err, service.ErrBilibiliCredentialDisabled):
			response.Fail(c, apperror.New("BILIBILI_CREDENTIAL_DISABLED",
				"服务端未配置凭证加密密钥，暂不支持授权", http.StatusServiceUnavailable))
		default:
			response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "保存授权失败", http.StatusInternalServerError, err))
		}
		return
	}
	response.Ok(c, credentialResponse(status))
}

// Unbind 解除授权。DELETE /api/v1/me/bilibili-credential
func (h *BilibiliCredentialHandler) Unbind(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	if err := h.service.Unbind(c.Request.Context(), userID); err != nil {
		if errors.Is(err, service.ErrBilibiliCredentialDisabled) {
			response.Fail(c, apperror.New("BILIBILI_CREDENTIAL_DISABLED",
				"服务端未配置凭证加密密钥，暂不支持授权", http.StatusServiceUnavailable))
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "解除授权失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, gin.H{"bound": false})
}
