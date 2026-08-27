package logging

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
)

var currentLevel atomic.Int32 //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	currentLevel.Store(int32(slog.LevelError + 1))
}

func LogLevel() slog.Level {
	return slog.Level(currentLevel.Load())
}

func SetLevel(level slog.Level) {
	currentLevel.Store(int32(level)) //nolint:gosec
}

// NewHandler 创建日志 handler，级别由 logging 包动态控制。
func NewHandler(w io.Writer) slog.Handler {
	return &levelHandler{
		inner: slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}
}

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
