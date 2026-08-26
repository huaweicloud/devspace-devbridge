package logging

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
)

var currentLevel atomic.Int32 //nolint:gochecknoglobals // cobra CLI 惯用全局变量

func init() { //nolint:gochecknoinits // cobra CLI 惯用 init 函数
	currentLevel.Store(int32(slog.LevelError + 1))
}

func LogLevel() slog.Level {
	return slog.Level(currentLevel.Load())
}

func SetLevel(level slog.Level) {
	currentLevel.Store(int32(level)) //nolint:gosec // slog.Level 实际取值 -8~8，远在 int32 范围内
}

// NewHandler 创建一个基于 slog.TextHandler 的日志 handler，
// 日志级别由 logging 包动态控制.
func NewHandler(w io.Writer) slog.Handler {
	return &levelHandler{
		inner: slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}
}

// levelHandler 包装 slog.TextHandler，动态检查日志级别.
type levelHandler struct {
	inner slog.Handler
}

func (h *levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= LogLevel()
}

func (h *levelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.inner.Handle(ctx, r)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{inner: h.inner.WithGroup(name)}
}
