package observabilitylogger

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

// WithLogger attaches the logger to the context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext gets the logger from the context
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return l
	}
	return nil
}
