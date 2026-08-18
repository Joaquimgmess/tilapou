package farm_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/farm"
	"github.com/Joaquimgmess/tilapou/internal/platform/metrics"
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

	sessions := farm.NewSessions(store, &b, func() time.Time { return epoch }, metrics.NewRegistry())

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

func TestAcaoRecusadaContaNaMetricaDeNegocio(t *testing.T) {
	t.Parallel()

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
	store.farm.State.Cash = 0

	registry := metrics.NewRegistry()
	sessions := farm.NewSessions(store, &b, func() time.Time { return epoch }, registry)

	before := registry.Render()

	if _, actErr := sessions.Act(t.Context(), player,
		sim.Action{ID: 1, Kind: sim.ActionBuyTank, TankKind: sim.TankRecirculation}); actErr != nil {
		t.Fatalf("agindo: %v", actErr)
	}

	const want = `farm_actions_rejected_total{reason="not_enough_cash"}`

	after := registry.Render()
	if countOf(t, before, want)+1 != countOf(t, after, want) {
		t.Errorf("a recusa nao contou em %s\nantes:\n%s\ndepois:\n%s", want, before, after)
	}
}

func countOf(t *testing.T, rendered, series string) int64 {
	t.Helper()

	for line := range strings.SplitSeq(rendered, "\n") {
		name, value, found := strings.Cut(line, " ")
		if !found || name != series {
			continue
		}

		count, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			t.Fatalf("valor de %s nao e numero: %q", series, value)
		}

		return count
	}

	return 0
}
