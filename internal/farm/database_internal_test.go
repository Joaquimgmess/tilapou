package farm

import (
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
