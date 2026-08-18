package farm

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const brasilia = sim.ZoneOffset(-3 * 60)

func planBalance(t *testing.T) *sim.Balance {
	t.Helper()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	return &b
}

func TestPlanoDeCicloEsqueceOQueFoiCalculadoOntem(t *testing.T) {
	t.Parallel()

	b := planBalance(t)
	p := newPlans()

	p.at(b, sim.TankEarthPond, 0, brasilia)
	p.at(b, sim.TankNetCage, sim.TicksPerDay, brasilia)

	if len(p.cache) != 1 {
		t.Fatalf("o dia virou e o cache guardou %d planos, queria 1", len(p.cache))
	}
	if p.day != 1 {
		t.Fatalf("o cache diz dia %d, queria 1", p.day)
	}
}

func TestPlanoDeCicloEsqueceQuandoOFusoMuda(t *testing.T) {
	t.Parallel()

	b := planBalance(t)
	p := newPlans()

	p.at(b, sim.TankEarthPond, 0, brasilia)
	p.at(b, sim.TankNetCage, 0, 0)

	if len(p.cache) != 1 {
		t.Fatalf("o fuso mudou e o cache guardou %d planos, queria 1", len(p.cache))
	}
	if p.zone != 0 {
		t.Fatalf("o cache diz fuso %d, queria 0", p.zone)
	}
}

func TestPlanoDeCicloDoMesmoDiaSoCalculaUmaVez(t *testing.T) {
	t.Parallel()

	b := planBalance(t)
	p := newPlans()

	first := p.at(b, sim.TankEarthPond, 0, brasilia)
	again := p.at(b, sim.TankEarthPond, 1, brasilia)

	if first != again {
		t.Fatalf("o mesmo dia devolveu planos diferentes: %+v e %+v", first, again)
	}
	if len(p.cache) != 1 {
		t.Fatalf("o mesmo dia guardou %d planos, queria 1", len(p.cache))
	}
}

func TestDecisaoRecalculaDepoisQueODiaVira(t *testing.T) {
	t.Parallel()

	p := newPlans()
	calls := 0
	compute := func() DecisionView {
		calls++

		return DecisionView{SellNowCents: 1}
	}

	today := decisionInput{batch: 1, day: 0}
	tomorrow := today
	tomorrow.day = 1

	p.decision(today, compute)
	p.decision(tomorrow, compute)
	p.decision(today, compute)

	if calls != 3 {
		t.Fatalf("compute rodou %d vezes, queria 3: a decisao de ontem sobreviveu a virada do dia", calls)
	}
}

func TestDecisaoDoMesmoDiaSoCalculaUmaVez(t *testing.T) {
	t.Parallel()

	p := newPlans()
	calls := 0
	compute := func() DecisionView {
		calls++

		return DecisionView{SellNowCents: 1}
	}

	key := decisionInput{batch: 1, day: 0}

	p.decision(key, compute)
	p.decision(key, compute)

	if calls != 1 {
		t.Fatalf("compute rodou %d vezes, queria 1", calls)
	}
}
