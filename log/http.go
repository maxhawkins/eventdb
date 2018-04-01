package log

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// WrapHandler wraps an http.Handler, adding request logging and decorating
// its request context with the logger.
//
// When you call FromContext with a wrapped http handler's request object it
// will return the logger passed here.
func WrapHandler(h http.Handler, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// cut down on noise, don't log health checks
		if r.URL.Path == "/healthz" {
			h.ServeHTTP(w, r)
			return
		}

		fields := []zapcore.Field{
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		}
		if ua := r.Header.Get("User-Agent"); ua != "" {
			fields = append(fields, zap.String("user_agent", ua))
		}

		reqLogger := logger.With(fields...)

		// Send logger through the request context
		ctx := r.Context()
		ctx = ToContext(ctx, reqLogger)
		r = r.WithContext(ctx)

		metrics := httpsnoop.CaptureMetrics(h, w, r)

		reqLogger.Info("handled",
			zap.Int("code", metrics.Code),
			zap.Int64("size", metrics.Written),
			zap.Duration("duration", metrics.Duration))
	})
}
