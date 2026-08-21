package sim

import "testing"

// O custo por dia da projecao nao pode depender de quanta racao ha no silo: a mesma fazenda
// comendo o mesmo tanto tem o mesmo custo, e o que muda com o nivel do silo e so a media
// contabil do estoque ja pago. Com ela dentro da projecao, a decisao segurar-vs-vender
// mudava de resposta por causa da ultima compra de racao.
func TestCustoPorDiaDaProjecaoNaoDependeDoNivelDoSilo(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)

	custo := func(kilos int64) (Coins, int64) {
		s := stockedFarm(t, 71)
		s.Cash = 10_000_000
		s.Tanks[0].Batches[0].Fish = 500
		s.Tanks[0].ServedUntil = Tick(maxInt32)
		s.Tanks[0].FeedStock = Micrograms(kilos) * MicrogramsPerKilogram
		s.Tanks[0].FeedUnitCost = MarketAt(b, s.Tick).FeedKg

		got := s.Forecast(b, 1, 1, 600*MicrogramsPerGram)
		if got.Days <= 0 {
			t.Fatalf("com %d kg no silo a projecao nao fechou nenhum dia", kilos)
		}

		return Coins(int64(got.Cost) / got.Days), got.Days
	}

	base, days := custo(49)

	for _, kilos := range []int64{5, 30, 51, 60, 200, 490} {
		got, gotDays := custo(kilos)
		if got != base {
			t.Errorf("com %d kg no silo o custo por dia e %d (%d dias) e com 49 kg e %d (%d dias): o numero muda com a ultima compra de racao",
				kilos, got, gotDays, base, days)
		}
	}
}
