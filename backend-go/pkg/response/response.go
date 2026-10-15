package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wanli/backend-go/pkg/apperror"
)

// Response 是统一 API 响应结构。
type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Ok 返回成功响应。
func Ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

// Created 返回 201 成功响应。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

// NoContent 返回 204。
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Fail 返回业务错误响应。
func Fail(c *gin.Context, err *apperror.AppError) {
	c.JSON(err.HTTPStatus, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
		},
	})
}

// FailWithDetails 返回带 details 的业务错误响应。
func FailWithDetails(c *gin.Context, err *apperror.AppError, details any) {
	c.JSON(err.HTTPStatus, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
			Details: details,
		},
	})
}
