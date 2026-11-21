package helpers

import (
	"api-server/v2/internal/observability"
	"net/http"
	"time"
)

func RequestTrace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := observability.JobContext(r.Context())
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", observability.RequestID(ctx))
		response := &loggingResponseWriter{ResponseWriter: w, statusCode: 200}
		start := time.Now()
		defer func() {
			observability.Event(ctx, "http_request", map[string]any{"method": r.Method, "path": r.URL.EscapedPath(), "status": response.statusCode, "duration_ms": time.Since(start).Milliseconds()})
		}()
		next.ServeHTTP(response, r)
	})
}
