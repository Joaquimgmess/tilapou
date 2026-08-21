package sim

import "testing"

// Fazenda com caixa que povoa nao esta quebrada: a tela mandava recomecar — que e
// irreversivel — com 4532,83 TC na barra e o [s] funcionando. stuck perguntava se cabia um
// ciclo inteiro, enquanto a acao perguntava se cabiam 100 alevinos.
func TestFazendaComCaixaQuePovoaNaoEstaQuebrada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 453_283
	s.Debt = 1_200_000

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	antes := s
	reason, _ := applyStock(&s, b, Action{Kind: ActionStock, Tank: id, Amount: MinStockFish},
		s.Tick, &eventSink{})
	if reason != RejectNone {
		t.Fatalf("o jogo recusou povoar o minimo com %d de caixa: %v", antes.Cash, reason)
	}

	if antes.Broke(b, plans) {
		t.Errorf("o jogo aceita povoar com %d de caixa e a tela manda recomecar do zero: [b] e irreversivel e queima o caixa",
			antes.Cash)
	}
}

// O contrario tem de valer tambem: sem caixa para a jogada mais barata e sem credito, a
// fazenda esta quebrada de verdade e o resgate precisa existir.
func TestFazendaSemCaixaParaAJogadaMaisBarataEstaQuebrada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 0
	s.Debt = b.Credit.MaxPrincipal

	if _, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
		t.Fatal("sem tanque")
	}

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	if !s.Broke(b, plans) {
		t.Error("sem caixa, sem credito e sem peixe a fazenda nao contou como quebrada: nao sobra jogada nenhuma")
	}
}

// O registro tem de responder por TODA acao: entrada vazia devolveria "nao e jogada" em
// silencio, que e como a divergencia entre a tela e o jogo nasceu quatro vezes.
func TestRegistroDeJogadasCobreTodaAcao(t *testing.T) {
	t.Parallel()

	if len(playable) != int(actionKindCount) {
		t.Fatalf("o registro tem %d entradas e existem %d acoes", len(playable), actionKindCount)
	}

	for kind, can := range playable {
		if can == nil {
			t.Errorf("a acao %v nao tem entrada no registro de jogadas", ActionKind(kind))
		}
	}
}
