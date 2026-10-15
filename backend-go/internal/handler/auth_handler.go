package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// userResponse 返回用户信息的统一响应结构。
func userResponse(u *model.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"email":      u.Email,
		"nickname":   u.Nickname,
		"avatar_url": u.AvatarURL,
	}
}

// AuthHandler 认证接口。
type AuthHandler struct {
	authService *service.AuthService
	siteKey     string
}

// NewAuthHandler 创建 AuthHandler。
func NewAuthHandler(authService *service.AuthService, siteKey string) *AuthHandler {
	return &AuthHandler{authService: authService, siteKey: siteKey}
}

type registerRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8,max=72"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

type loginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

// Register 邮箱+密码注册。
func (h *AuthHandler) Register(c *gin.Context) {
	if err := prepareBodyWithCleanEmail(c); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.TurnstileToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			response.Fail(c, apperror.New("AUTH_EMAIL_TAKEN", err.Error(), http.StatusConflict))
		case errors.Is(err, service.ErrCaptchaFailed):
			response.Fail(c, apperror.New("CAPTCHA_FAILED", err.Error(), http.StatusBadRequest))
		default:
			response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "注册失败", http.StatusInternalServerError, err))
		}
		return
	}

	response.Ok(c, gin.H{"token": token, "user": userResponse(user)})
}

// Login 邮箱+密码登录。
func (h *AuthHandler) Login(c *gin.Context) {
	if err := prepareBodyWithCleanEmail(c); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, req.TurnstileToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCaptchaFailed):
			response.Fail(c, apperror.New("CAPTCHA_FAILED", err.Error(), http.StatusBadRequest))
		case errors.Is(err, service.ErrTooManyAttempts):
			response.Fail(c, apperror.New("AUTH_TOO_MANY_ATTEMPTS", err.Error(), http.StatusTooManyRequests))
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Fail(c, apperror.New("AUTH_INVALID_CREDENTIALS", err.Error(), http.StatusUnauthorized))
		default:
			response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "登录失败", http.StatusInternalServerError, err))
		}
		return
	}

	response.Ok(c, gin.H{"token": token, "user": userResponse(user)})
}

// Me 获取当前登录用户信息。GET /api/v1/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	user, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(c, apperror.ErrResourceNotFound)
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询用户信息失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, userResponse(user))
}

type updateProfileRequest struct {
	Nickname  string  `json:"nickname" binding:"required"`
	AvatarURL *string `json:"avatar_url"`
}

// UpdateProfile 更新昵称与头像。PUT /api/v1/me/profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	user, err := h.authService.UpdateProfile(c.Request.Context(), userID, req.Nickname, req.AvatarURL)
	if err != nil {
		if errors.Is(err, service.ErrProfileInvalid) {
			response.FailWithDetails(c, apperror.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(c, apperror.ErrResourceNotFound)
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "更新资料失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, userResponse(user))
}

// CaptchaWidget 返回内嵌 Turnstile 组件的页面（供 Android WebView 加载）。
func (h *AuthHandler) CaptchaWidget(c *gin.Context) {
	page := fmt.Sprintf(captchaPageTemplate, h.siteKey)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

const captchaPageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>人机验证</title>
<style>body{margin:0;display:flex;justify-content:center;padding-top:16px;background:transparent;}</style>
</head>
<body>
<div id="turnstile-widget"></div>
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onTurnstileLoad" async defer></script>
<script>
function onTurnstileLoad() {
  turnstile.render('#turnstile-widget', {
    sitekey: '%s',
    callback: function(token) {
      if (window.AndroidBridge) { window.AndroidBridge.onTokenReceived(token); }
    },
    'error-callback': function(code) {
      if (window.AndroidBridge) { window.AndroidBridge.onError(code); }
    }
  });
}
</script>
</body>
</html>`
