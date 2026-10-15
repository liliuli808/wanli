package model

import "time"

// ImportTaskStatus 是视频解析任务的状态机状态。
type ImportTaskStatus string

const (
	ImportTaskPending   ImportTaskStatus = "PENDING"
	ImportTaskFetching  ImportTaskStatus = "FETCHING"
	ImportTaskParsing   ImportTaskStatus = "PARSING"
	ImportTaskCompleted ImportTaskStatus = "COMPLETED"
	ImportTaskFailed    ImportTaskStatus = "FAILED"
)

// IsTerminal 判断状态是否为终态。
func (s ImportTaskStatus) IsTerminal() bool {
	return s == ImportTaskCompleted || s == ImportTaskFailed
}

// VideoImportTask 是一次视频解析任务（存储在 Redis 中，有 TTL）。
type VideoImportTask struct {
	TaskID          string
	BVID            string
	URL             string
	Status          ImportTaskStatus
	Message         string // 面向用户的进度或失败提示
	ErrorCode       string // 失败时的业务错误码（对应 docs/api-contract.md）
	RecipeVersionID string // 完成时生成的草稿版本 ID
	CreatedAt       time.Time
	UpdatedAt       time.Time // 心跳时间，卡死任务依据它判定
}
