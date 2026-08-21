package sim

import "testing"

// O peixe que a oferta promete e, por definicao, o peixe que povoar entrega depois do
// emprestimo: fishFor dimensiona pelo caixa e nao conhece o espaco livre do tanque, entao a
// tela promete lote que o jogo recusa assim que o tanque, e nao o caixa, e o limite.
func TestOEmprestimoNaoPrometeMaisPeixeDoQueCabeNoTanque(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 900_000

	// Tanque pequeno de caso: o break-even do plano nao cabe nele, entao quem limita o
	// povoar e o espaco, e nao o dinheiro.
	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres*3/20)
	if !ok {
		t.Fatal("sem tanque")
	}

	plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	// O tanque quase cheio: o que falta para o break-even nao cabe mais nele, e quem limita
	// o povoar passa a ser o espaco, nao o dinheiro.
	tank := s.tank(id)
	room := int64(20)
	tank.addBatch(s.NextBatchID, FishCount(tank.Capacity(b)-room), b.Growth.FingerlingMass, s.Tick)
	s.NextBatchID++

	offer := s.LoanAdvice(b, id, plan)
	if offer.Block != LoanOpen {
		t.Skipf("o galpao recusou (%v): este teste precisa da oferta aberta", offer.Block)
	}

	fish, _ := s.StockAdvice(b, id, plan)
	livre := room - int64(fish)

	if int64(offer.Fish) > livre {
		t.Errorf("a oferta promete %d peixes a mais e no tanque so sobram %d", offer.Fish, livre)
	}

	s.Cash += offer.Cents
	s.Debt += offer.Cents

	depois, _ := s.StockAdvice(b, id, plan)
	if int64(offer.Fish) > int64(depois)-int64(fish) {
		t.Errorf("a oferta promete %d peixes a mais e o povoar so passa de %d para %d",
			offer.Fish, fish, depois)
	}
}
