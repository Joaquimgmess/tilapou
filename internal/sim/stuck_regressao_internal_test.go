package sim

import "testing"

// Tanque com lote dentro e sem racao nao esta preso enquanto o galpao vender um saco: a menor
// compra que resolve e o saco, e nao um ciclo novo. Comparar o alcance com o piso de
// povoamento marcava a fazenda como quebrada com 2000 peixes vivos no tanque, e a falencia
// por dias sem saida trocava o lote crescido pelo lote inicial.
func TestFazendaComLoteVivoESemRacaoNaoEstaPresaEnquantoOSacoCabe(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := NewState(1, 0, 0)
	s.Cash = 0
	s.Debt = 30

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	s.StockTank(id, 2_000, 200*MicrogramsPerGram, 1_000)
	s.tank(id).FeedStock = 0

	var plans Plans
	plans[TankEarthPond] = b.CycleAt(TankEarthPond, s.Tick, s.Zone)

	offer := s.LoanAdvice(b, id, plans[TankEarthPond])
	if offer.Block != LoanOpen || offer.Cents <= 0 {
		t.Fatalf("o galpao recusou (%v): este teste precisa do credito que compra a racao", offer.Block)
	}

	if s.stuck(b, plans) {
		t.Errorf("fazenda com %d peixes vivos e emprestimo de %d disponivel contou como presa: a falencia por dias sem saida troca o lote crescido pelo inicial",
			s.tank(id).Fish(), offer.Cents)
	}
	if s.Broke(b, plans) {
		t.Error("a mesma fazenda contou como quebrada e a tela oferece [b] com o lote vivo dentro")
	}
}
