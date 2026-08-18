package farm_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/farm"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func routedAPI(t *testing.T) humatest.TestAPI {
	t.Helper()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	player := uuid.New()
	epoch := time.Unix(0, 0).UTC()
	store := &memoryStore{
		farm:    farm.New(uuid.New(), player, "t", epoch, 0, 1, &b),
		actions: map[sim.ActionID]sim.Outcome{},
	}
	store.farm.State.Cash = 100_000_000

	sessions := farm.NewSessions(store, &b, func() time.Time { return epoch })

	_, api := humatest.New(t, huma.DefaultConfig("tilapou", "test"))
	farm.RegisterRoutes(api, sessions, player, &b)

	return api
}

func TestTodoTipoDeTanqueDoEnumPassaNaValidacaoDaRota(t *testing.T) {
	t.Parallel()

	api := routedAPI(t)

	for i, kind := range sim.TankKindNames() {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			resp := api.Post("/farm/actions", map[string]any{
				"key":       i + 1,
				"kind":      "buy_tank",
				"tank_kind": kind,
			})

			if resp.Code == http.StatusUnprocessableEntity {
				t.Errorf("a rota recusou o tipo %q do enum com 422: %s", kind, resp.Body.String())
			}
		})
	}
}

func TestTipoDeTanqueForaDoEnumEhRecusadoPelaRota(t *testing.T) {
	t.Parallel()

	api := routedAPI(t)

	resp := api.Post("/farm/actions", map[string]any{
		"key":       1,
		"kind":      "buy_tank",
		"tank_kind": "aquario",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("a rota aceitou um tipo fora do enum com %d, queria 422", resp.Code)
	}
}
