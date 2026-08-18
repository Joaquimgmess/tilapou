package farm

import (
	"errors"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func TestOInsertGuardaApenasOsUltimosEventos(t *testing.T) {
	t.Parallel()

	events := make([]sim.Event, eventsKept+1)
	for i := range events {
		events[i] = sim.Event{Seq: uint64(i)}
	}

	kept := lastEvents(events)

	if len(kept) != eventsKept {
		t.Fatalf("guardou %d eventos, queria %d", len(kept), eventsKept)
	}
	if kept[0].Seq != 1 {
		t.Fatalf("o primeiro evento guardado tem seq %d, queria 1: a poda cortou do lado errado", kept[0].Seq)
	}
}

func TestOInsertNaoPodaQuandoCabeTudo(t *testing.T) {
	t.Parallel()

	events := make([]sim.Event, eventsKept)

	if got := len(lastEvents(events)); got != eventsKept {
		t.Fatalf("guardou %d eventos, queria %d", got, eventsKept)
	}
}

func TestRazaoForaDoEnumNaoViraAcaoAplicada(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		reason string
		want   sim.RejectReason
		fails  bool
	}{
		"sem recusa":        {reason: "none", want: sim.RejectNone},
		"recusa conhecida":  {reason: "not_enough_cash", want: sim.RejectNotEnoughCash},
		"nome fora do enum": {reason: "sem_gasolina", fails: true},
		"coluna vazia":      {reason: "", fails: true},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := storedOutcome(sim.Outcome{Applied: false}, tc.reason)
			if tc.fails {
				if !errors.Is(err, ErrUnknownReason) {
					t.Fatalf("storedOutcome(%q) devolveu %v, queria ErrUnknownReason", tc.reason, err)
				}

				return
			}
			if err != nil {
				t.Fatalf("storedOutcome(%q) falhou: %v", tc.reason, err)
			}
			if got.Reason != tc.want {
				t.Errorf("storedOutcome(%q) deu %v, queria %v", tc.reason, got.Reason, tc.want)
			}
		})
	}
}
