package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// AppDownloadPath 是 APK 下载接口的绝对路径，下发给客户端后拼到 BASE_URL 上。
const AppDownloadPath = "/api/v1/app/download"

// AppUpdateHandler 提供 App 自更新所需的版本信息与安装包下载。
type AppUpdateHandler struct {
	service *service.AppUpdateService
}

// NewAppUpdateHandler 创建 Handler。
func NewAppUpdateHandler(svc *service.AppUpdateService) *AppUpdateHandler {
	return &AppUpdateHandler{service: svc}
}

// Check 返回最新版本信息。GET /api/v1/app/update
func (h *AppUpdateHandler) Check(c *gin.Context) {
	release, err := h.service.Latest(c.Request.Context())
	if err != nil {
		response.Fail(c, apperror.Wrap("APP_UPDATE_UNAVAILABLE", "暂时无法获取最新版本", http.StatusServiceUnavailable, err))
		return
	}
	response.Ok(c, gin.H{
		"version_code":  release.VersionCode,
		"version_name":  release.VersionName,
		"size":          release.Size,
		"sha256":        release.SHA256,
		"commit":        release.Commit,
		"published_at":  release.PublishedAt,
		"download_path": AppDownloadPath,
	})
}

// Download 代理下载最新 APK。GET /api/v1/app/download
//
// 走服务端代理而不是让客户端直连 GitHub：国内网络下 GitHub Releases 经常不可达，
// 而 api.susie.one 本身已经能访问。
func (h *AppUpdateHandler) Download(c *gin.Context) {
	body, size, err := h.service.Open(c.Request.Context())
	if err != nil {
		response.Fail(c, apperror.Wrap("APP_DOWNLOAD_UNAVAILABLE", "安装包下载失败，请稍后再试", http.StatusBadGateway, err))
		return
	}
	defer body.Close()

	c.Header("Content-Disposition", `attachment; filename="wanli.apk"`)
	c.DataFromReader(http.StatusOK, size, "application/vnd.android.package-archive", body, nil)
}
