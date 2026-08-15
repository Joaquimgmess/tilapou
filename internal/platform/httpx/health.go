package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
)

const readyTimeout = 2 * time.Second

func RegisterHealth(router chi.Router, ready func(ctx context.Context) error) {
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := ready(ctx); err != nil {
			logging.FromContext(ctx).ErrorContext(ctx, "readiness check failed", slog.Any("error", err))
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
