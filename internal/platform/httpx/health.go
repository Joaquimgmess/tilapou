package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
)

const readyTimeout = 2 * time.Second

// health is what /healthz publishes: which build is running and which database it is on.
//
// Nao entra no api.Snapshot por decisao: aquilo e contrato de jogo. Isto e o daemon dizendo
// quem ele e, e existe para quem opera — o portao de teste recusa antes de escrever quando o
// banco nao e o de teste, e falha dizendo "daemon rodando build X" quando o binario e outro.
type health struct {
	Revision string `json:"revision"`
	Modified bool   `json:"modified"`
	Database string `json:"database"`
}

// HealthOption preenche o que o /healthz publica.
type HealthOption func(*health)

// WithBuild informs which commit the daemon was built from, and whether the tree was dirty.
func WithBuild(revision string, modified bool) HealthOption {
	return func(h *health) {
		h.Revision, h.Modified = revision, modified
	}
}

// WithDatabase informs which database the daemon is writing to.
func WithDatabase(name string) HealthOption {
	return func(h *health) {
		h.Database = name
	}
}

// RegisterHealth registers GET /healthz and GET /readyz; ready runs with a 2s timeout.
func RegisterHealth(router chi.Router, ready func(ctx context.Context) error, opts ...HealthOption) {
	var body health
	for _, opt := range opts {
		opt(&body)
	}

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(body); err != nil {
			logging.FromContext(r.Context()).ErrorContext(r.Context(), "healthz encode", slog.Any("error", err))
		}
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
