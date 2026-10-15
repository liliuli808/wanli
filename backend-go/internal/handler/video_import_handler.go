package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/repository"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// VideoImportHandler 视频导入与解析任务 Handler。
type VideoImportHandler struct {
	importService *service.VideoImportService
}

// NewVideoImportHandler 创建视频导入 Handler。
func NewVideoImportHandler(importService *service.VideoImportService) *VideoImportHandler {
	return &VideoImportHandler{importService: importService}
}

type createVideoImportRequest struct {
	URL string `json:"url" binding:"required"`
}

// Create 提交视频解析任务。POST /api/v1/video-imports
func (h *VideoImportHandler) Create(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}

	var req createVideoImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	task, err := h.importService.Submit(c.Request.Context(), userID, req.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVideoURL):
			response.FailWithDetails(c, apperror.ErrValidation, "请输入有效的B站视频链接")
		case isDuplicate(err):
			dup := err.(*service.DuplicateVideoError)
			// AC1：重复 BVID 直接跳转已有菜谱
			details := gin.H{"recipe_id": dup.RecipeID.String()}
			if dup.VersionID != nil {
				details["recipe_version_id"] = dup.VersionID.String()
			}
			response.FailWithDetails(c, apperror.ErrBvidAlreadyParsed, details)
		default:
			response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "任务创建失败，请稍后再试", http.StatusInternalServerError, err))
		}
		return
	}

	response.Created(c, gin.H{
		"task_id": task.TaskID,
		"status":  task.Status,
		"bvid":    task.BVID,
	})
}

// Get 查询解析任务状态。GET /api/v1/video-imports/:task_id
func (h *VideoImportHandler) Get(c *gin.Context) {
	task, err := h.importService.Get(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		if errors.Is(err, repository.ErrImportTaskNotFound) {
			response.Fail(c, apperror.ErrResourceNotFound)
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询任务失败", http.StatusInternalServerError, err))
		return
	}

	data := gin.H{
		"task_id": task.TaskID,
		"status":  task.Status,
		"bvid":    task.BVID,
		"message": task.Message,
	}
	if task.ErrorCode != "" {
		data["error_code"] = task.ErrorCode
	}
	if task.RecipeVersionID != "" {
		data["recipe_version_id"] = task.RecipeVersionID
	}
	response.Ok(c, data)
}

func isDuplicate(err error) bool {
	var dup *service.DuplicateVideoError
	return errors.As(err, &dup)
}
