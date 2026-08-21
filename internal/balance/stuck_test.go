package balance_test

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

// Ou o galpao empresta, ou a fazenda esta quebrada e o [b] libera: contar como saida um
// credito que o galpao se recusa a soltar deixava o jogador com toda tecla negando, o [b]
// recusado e o cronometro do resgate zerado — o estado que o QA reproduziu jogando.
func TestOuOGalpaoEmprestaOuAFazendaEstaQuebrada(t *testing.T) {
	t.Parallel()

	loaded, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	b := &loaded

	for _, debt := range []sim.Coins{0, 100_000, 772_318, 1_400_000} {
		s := sim.NewState(1, 0, 0)
		s.Cash = 0
		s.Debt = debt

		id, ok := s.AddTank(b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}

		var plans sim.Plans
		plans[sim.TankEarthPond] = b.CycleAt(sim.TankEarthPond, s.Tick, s.Zone)

		offer := s.LoanAdvice(b, id, plans[sim.TankEarthPond])
		emprestou := offer.Block == sim.LoanOpen && offer.Cents > 0

		if !emprestou && !s.Broke(b, plans) {
			t.Errorf("com divida %d o galpao recusou (%v) e a fazenda nao contou como quebrada: nao sobra tecla nenhuma e o resgate nem comeca a contar",
				debt, offer.Block)
		}
	}
}
