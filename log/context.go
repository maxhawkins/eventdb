package log

import (
	"context"

	"go.uber.org/zap"
)

type ctxMarker struct{}

var (
	ctxMarkerKey = &ctxMarker{}
	nullLogger   = zap.NewNop()
)

// FromContext retrieves a *zap.Logger embedded in a context.Context using ToContext.
func FromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(ctxMarkerKey).(*zap.Logger)
	if !ok {
		return nullLogger
	}
	return logger.With() // copy
}

// ToContext embeds a *zap.Logger in a context.Context
func ToContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxMarkerKey, logger)
}
