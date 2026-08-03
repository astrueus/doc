package logging

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

// WithLogger stores logger in context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// LoggerFromCtx returns logger from context, or zap.L() if missing.
func LoggerFromCtx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.L()
}
