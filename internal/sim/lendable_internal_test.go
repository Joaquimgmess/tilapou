package sim

import "testing"

// Quem pergunta "da para comecar?" e quem pergunta "quanto?" tem de responder do mesmo lugar:
// com o maximo sendo mais generoso que qualquer alvo, oferta aberta implica fazenda de pe. O
// contrario deixava a tela mandar recomecar — que e irreversivel e o @qa confirmou que o jogo
// aceita — com credito disponivel na mesma tela.
func TestOfertaAbertaImplicaFazendaDePe(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	for _, cash := range []Coins{0, 10_000, 50_000, 99_994, 127_979, 200_000} {
		s := NewState(1, 0, 0)
		s.Cash = cash

		id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}

		var plans Plans
		plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

		offer := s.LoanAdvice(b, id, plans[TankEarthPond])
		if offer.Block != LoanOpen || offer.Cents <= 0 {
			continue
		}

		if s.Broke(b, plans) {
			t.Errorf("com caixa %d o galpao empresta %d e a tela manda recomecar do zero: [b] e irreversivel e queima o caixa",
				cash, offer.Cents)
		}
	}
}

// O maximo nunca pode ser menor que o que a oferta concede: sao a mesma pergunta, e o
// parametro want de significado duplo (0 querendo dizer "sem alvo") escondia a divergencia.
func TestOMaximoNuncaFicaAbaixoDoQueAOfertaConcede(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	for _, cash := range []Coins{0, 50_000, 200_000, 900_000} {
		for _, debt := range []Coins{0, 300_000, 1_000_000} {
			s := NewState(1, 0, 0)
			s.Cash, s.Debt = cash, debt

			id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
			if !ok {
				t.Fatal("sem tanque")
			}

			plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)
			tank := s.tank(id)

			maximo := lendable(b, tank, plan, s.Debt, s.Cash)
			offer := s.LoanAdvice(b, id, plan)

			if offer.Block == LoanOpen && offer.Cents > maximo {
				t.Errorf("caixa %d divida %d: a oferta concede %d e o maximo diz %d",
					cash, debt, offer.Cents, maximo)
			}
			if offer.Block == LoanOpen && maximo <= 0 {
				t.Errorf("caixa %d divida %d: a oferta esta aberta com %d e o maximo diz que nao ha o que emprestar",
					cash, debt, offer.Cents)
			}
		}
	}
}
