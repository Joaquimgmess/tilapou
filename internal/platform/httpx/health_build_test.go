package httpx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Joaquimgmess/tilapou/internal/platform/httpx"
)

// O /healthz publica em qual build e em qual banco o daemon esta. Sem isso, o portao de teste
// nao tem como saber que esta medindo um binario velho — o @qa quase reportou regressao
// inexistente porque um build antigo reapareceu na porta — nem como recusar antes de escrever
// no banco errado, que e o unico jeito de essa guarda morar do lado que sabe.
func TestHealthzPublicaBuildEBanco(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	httpx.RegisterHealth(router, func(context.Context) error { return nil },
		httpx.WithBuild("abc1234", true), httpx.WithDatabase("tilapou_qa"))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("healthz devolveu %d", rec.Code)
	}

	var got struct {
		Revision string `json:"revision"`
		Modified bool   `json:"modified"`
		Database string `json:"database"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("healthz nao devolveu JSON: %v", err)
	}

	if got.Revision != "abc1234" || !got.Modified || got.Database != "tilapou_qa" {
		t.Errorf("healthz devolveu %+v", got)
	}
}
