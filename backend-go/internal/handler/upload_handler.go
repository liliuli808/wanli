package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

type UploadHandler struct {
	service *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{service: uploadService}
}

type presignUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	SizeBytes   int64  `json:"size_bytes" binding:"required"`
}

func (h *UploadHandler) Presign(c *gin.Context) {
	if _, appErr := currentUserID(c); appErr != nil {
		response.Fail(c, appErr)
		return
	}
	var req presignUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	result, err := h.service.PresignCookingPhoto(
		c.Request.Context(), req.ContentType, req.SizeBytes,
	)
	if err != nil {
		if errors.Is(err, service.ErrUploadInvalid) {
			response.FailWithDetails(c, apperror.ErrValidation, err.Error())
			return
		}
		response.Fail(c, apperror.Wrap(
			"INTERNAL_ERROR", "生成上传地址失败", http.StatusInternalServerError, err,
		))
		return
	}
	response.Ok(c, gin.H{
		"upload_url":  result.UploadURL,
		"public_url":  result.PublicURL,
		"object_name": result.ObjectName,
		"expires_in":  result.ExpiresIn,
	})
}
