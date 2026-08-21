package sim

import "testing"

// O conselho de povoamento nao pode sugerir um lote que o jogo recusa: applyStock exige
// MinStockFish, entao sugerir 51 peixes manda o jogador apertar [s] para ouvir "quantidade
// invalida" por um numero que quem escolheu foi a propria tela.
func TestConselhoDePovoamentoNaoSugereLoteQueOJogoRecusa(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	for _, cash := range []Coins{159_994, 175_000, 189_999, 204_999, 220_000, 400_000} {
		s := NewState(1, 0, 0)
		s.Cash = cash

		id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
		if !ok {
			t.Fatal("sem tanque")
		}

		plan := b.CycleAt(TankEarthPond, s.Tick, s.Zone)

		fish, _ := s.StockAdvice(b, id, plan)
		if fish <= 0 {
			continue
		}

		reason, _ := applyStock(&s, b, Action{Kind: ActionStock, Tank: id, Amount: int64(fish)},
			s.Tick, &eventSink{})
		if reason != RejectNone {
			t.Errorf("com caixa %d o conselho sugeriu povoar %d peixes e o jogo recusou com %v",
				cash, fish, reason)
		}
	}
}
