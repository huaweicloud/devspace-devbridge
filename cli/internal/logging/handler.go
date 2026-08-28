package logging

import (
	"io"
	"log/slog"
)

// levelVar 是标准库提供的原子可变级别变量，实现 slog.Leveler 接口。
// 传给 slog.HandlerOptions{Level: &levelVar} 后，TextHandler 每次写日志
// 都会调用 levelVar.Level() 判断是否输出，SetLevel 修改立即生效。
var levelVar slog.LevelVar

func init() {
	levelVar.Set(slog.LevelError)
}

func LogLevel() slog.Level {
	return levelVar.Level()
}

func SetLevel(level slog.Level) {
	levelVar.Set(level)
}

// NewHandler 创建日志 handler，级别由 logging 包动态控制。
func NewHandler(w io.Writer) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{Level: &levelVar})
}
