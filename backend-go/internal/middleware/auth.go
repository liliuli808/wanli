package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/auth"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

const (
	CurrentUserKey = "currentUser"
)

// CurrentUser 是上下文中保存的当前用户信息。
type CurrentUser struct {
	UserID string
	Email  string
}

// Auth 从 Authorization 头解析 JWT，并将当前用户注入 Context。
// 若未携带 Token 或 Token 无效，currentUser 为 nil，但不直接返回错误（由后续 RequireAuth 处理）。
func Auth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := extractUser(c, jwtManager)
		c.Set(CurrentUserKey, user)
		c.Next()
	}
}

// RequireAuth 要求请求必须携带有效 Token。
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.MustGet(CurrentUserKey) == nil {
			response.Fail(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Next()
	}
}

func extractUser(c *gin.Context, jwtManager *auth.JWTManager) *CurrentUser {
	header := c.GetHeader("Authorization")
	if header == "" {
		return nil
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil
	}

	claims, err := jwtManager.Parse(parts[1])
	if err != nil {
		return nil
	}

	return &CurrentUser{
		UserID: claims.UserID,
		Email:  claims.Email,
	}
}

// GetCurrentUser 从 Context 获取当前用户，可能为 nil。
func GetCurrentUser(c *gin.Context) *CurrentUser {
	v, exists := c.Get(CurrentUserKey)
	if !exists || v == nil {
		return nil
	}
	return v.(*CurrentUser)
}
