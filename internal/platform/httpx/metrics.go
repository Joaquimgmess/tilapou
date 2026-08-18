package httpx

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
	"github.com/Joaquimgmess/tilapou/internal/platform/metrics"
)

// httpDuration is the latency series of every request, by method, route and status.
const httpDuration = "http_request_duration_seconds"

func renderMetrics(registry *metrics.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		if _, err := io.WriteString(w, registry.Render()); err != nil {
			logging.FromContext(r.Context()).ErrorContext(r.Context(), "writing metrics", slog.Any("error", err))
		}
	}
}

func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "unmatched"
	}

	pattern := rctx.RoutePattern()
	if pattern == "" {
		return "unmatched"
	}

	return pattern
}
