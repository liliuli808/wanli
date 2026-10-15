package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/pkg/response"
)

// HealthHandler 处理健康检查。
type HealthHandler struct{}

// NewHealthHandler 创建 HealthHandler。
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health 返回服务健康状态。
func (h *HealthHandler) Health(c *gin.Context) {
	response.Ok(c, gin.H{"status": "ok"})
}

// PingDB 返回数据库连通状态（阶段 0 占位，后续实现）。
func (h *HealthHandler) PingDB(c *gin.Context) {
	response.Ok(c, gin.H{"database": "ok"})
}
