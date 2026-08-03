package logging

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

// WithLogger 将 logger 存入 context。
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// LoggerFromCtx 从 context 取 logger；不存在则返回 zap.L()。
func LoggerFromCtx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.L()
}
