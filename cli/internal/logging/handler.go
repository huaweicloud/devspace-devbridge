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

type PlainHandler struct {
	w io.Writer
}

func NewPlainHandler(w io.Writer) *PlainHandler {
	return &PlainHandler{w: w}
}

func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= LogLevel()
}

func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	var buf []byte
	buf = append(buf, r.Message...)
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
		return true
	})
	buf = append(buf, '\n')
	_, err := h.w.Write(buf)
	return err
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PlainHandler) WithGroup(name string) slog.Handler {
	return h
}
