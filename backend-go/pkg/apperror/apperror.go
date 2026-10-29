package apperror

import "net/http"

// AppError 是业务错误类型。
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建业务错误。
func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

// Wrap 包装底层错误。
func Wrap(code, message string, status int, err error) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status, Err: err}
}

// 预定义错误码，与 docs/api-contract.md 保持一致。
var (
	ErrUnauthorized        = New("UNAUTHORIZED", "未登录或登录已过期", http.StatusUnauthorized)
	ErrForbidden           = New("FORBIDDEN", "无权操作", http.StatusForbidden)
	ErrResourceNotFound    = New("RESOURCE_NOT_FOUND", "资源不存在", http.StatusNotFound)
	ErrValidation          = New("VALIDATION_ERROR", "请求参数校验失败", http.StatusBadRequest)
	ErrBvidAlreadyParsed   = New("BVID_ALREADY_PARSED", "该视频已有公开来源解析版", http.StatusConflict)
	ErrVideoFetchFailed    = New("VIDEO_FETCH_FAILED", "获取视频信息失败，请检查链接或稍后再试", http.StatusUnprocessableEntity)
	ErrSubtitleUnavailable = New("SUBTITLE_NOT_AVAILABLE", "该视频没有可用字幕，无法解析", http.StatusUnprocessableEntity)
	// ErrBilibiliLoginRequired 表示字幕仅 B 站登录可见，需要用户先授权自己的账号。
	ErrBilibiliLoginRequired = New("BILIBILI_LOGIN_REQUIRED", "该视频的字幕需要登录 B 站才能读取，请先在「我的 - B 站账号」中授权", http.StatusUnprocessableEntity)
	ErrAIParseFailed         = New("AI_PARSE_FAILED", "AI 解析失败", http.StatusUnprocessableEntity)
	ErrKitchenFitCannotFork  = New("KITCHEN_FIT_CANNOT_FORK", "个人适配版不支持派生改编", http.StatusUnprocessableEntity)
	ErrAllergyBlocked        = New("ALLERGY_BLOCKED", "由于安全限制，该菜谱无法适配", http.StatusUnprocessableEntity)
	ErrDietaryBlocked        = New("DIETARY_BLOCKED", "当前饮食限制无法安全适配", http.StatusUnprocessableEntity)
	ErrAIAdaptationFailed    = New("AI_ADAPTATION_FAILED", "厨房适配生成失败，请稍后重试", http.StatusBadGateway)
	ErrInternal              = New("INTERNAL_ERROR", "服务器内部错误", http.StatusInternalServerError)
)
