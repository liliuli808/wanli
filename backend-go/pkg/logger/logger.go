package logger

import (
	"log/slog"
	"os"
)

// New 根据 level 创建结构化 slog 日志实例。
func New(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lv}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}
