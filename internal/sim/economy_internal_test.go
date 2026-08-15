package sim

import "testing"

func TestMarketMovesAndStaysDeterministic(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	seen := map[Coins]bool{}
	for day := range int64(120) {
		price := MarketAt(b, Tick(day)*TicksPerDay).FishKg
		seen[price] = true

		if price != MarketAt(b, Tick(day)*TicksPerDay).FishKg {
			t.Fatalf("o preco do dia %d mudou entre duas leituras", day)
		}
	}

	if len(seen) < 20 {
		t.Errorf("o mercado ficou parado: so %d precos diferentes em 120 dias", len(seen))
	}
}

func TestMarketStaysWithinTheSwing(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	base := int64(b.Market.FishBasePerKg)
	limit := base + mulDivFloor(base, int64(b.Market.SwingPPM), int64(UnitPPM))

	for tick := range Tick(200_000) {
		if tick%997 != 0 {
			continue
		}

		price := int64(MarketAt(b, tick).FishKg)
		if price <= 0 || price > limit {
			t.Fatalf("preco %d no tick %d saiu da banda (base %d, teto %d)", price, tick, base, limit)
		}
	}
}

func TestBiggerFishFetchesMorePerKilo(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	small := b.PriceFor(350*MicrogramsPerGram, 0)
	medium := b.PriceFor(550*MicrogramsPerGram, 0)
	large := b.PriceFor(850*MicrogramsPerGram, 0)
	oversized := b.PriceFor(1_500*MicrogramsPerGram, 0)

	if small >= medium || medium >= large {
		t.Errorf("o preco por quilo nao sobe com a classe: %d %d %d", small, medium, large)
	}
	if oversized >= large {
		t.Errorf("peixe grande demais deveria valer menos por quilo: %d contra %d", oversized, large)
	}
}

func TestHoldingLongerRaisesRevenuePerFish(t *testing.T) {
	t.Parallel()

	b := isothermalBalance(t, 28)

	revenue := func(untilGrams int64) Coins {
		s := stockedFarm(t, 41)
		s.Tanks[0].Batches[0].Fish = 200
		s.Tanks[0].FeedStock = 100_000 * MicrogramsPerKilogram
		s.Tanks[0].ServedUntil = Tick(maxInt32)
		s.Cash = 5_000_000

		for day := range int64(400) {
			out, err := Advance(Input{State: s, Until: Tick(day+1) * TicksPerDay, Balance: b})
			if err != nil {
				t.Fatalf("Advance() error = %v", err)
			}
			s = out.State

			if s.Tanks[0].Batches[0].MeanMass.Grams() >= untilGrams {
				break
			}
		}

		before := s.Cash
		out, err := Advance(Input{
			State:   s,
			Until:   s.Tick + 5,
			Balance: b,
			Actions: []Action{{ID: 1, Kind: ActionHarvest, At: s.Tick + 1, Tank: 1, Batch: 1}},
		})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}

		return Coins(subSat(int64(out.State.Cash), int64(before)))
	}

	early, late := revenue(450), revenue(820)
	if late <= early {
		t.Errorf("segurar ate a classe maior nao pagou: 450g=%d 820g=%d", early, late)
	}
}

func TestUpkeepDrainsAnIdleTank(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 42)
	s.Tanks[0].BatchCount = 0
	s.Cash = 100_000

	out, err := Advance(Input{State: s, Until: 30 * TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if out.State.Cash >= 100_000 {
		t.Error("tanque parado nao custou nada: a manutencao nao esta cobrando")
	}
}

func TestDebtGrowsWithInterestAndCanBeRepaid(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 43)
	s.Cash = 0

	borrowed, err := Advance(Input{
		State:   s,
		Until:   10,
		Balance: b,
		Actions: []Action{{ID: 1, Kind: ActionBorrow, At: 1, Amount: 500_000}},
	})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if borrowed.State.Debt != 500_000 || borrowed.State.Cash < 400_000 {
		t.Fatalf("emprestimo nao entrou: divida=%d caixa=%d", borrowed.State.Debt, borrowed.State.Cash)
	}

	waited, err := Advance(Input{State: borrowed.State, Until: 60 * TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if waited.State.Cash >= borrowed.State.Cash {
		t.Error("os juros nao cobraram nada em 60 dias")
	}

	paid, err := Advance(Input{
		State:   waited.State,
		Until:   waited.State.Tick + 5,
		Balance: b,
		Actions: []Action{{ID: 2, Kind: ActionRepay, At: waited.State.Tick + 1, Amount: 100_000}},
	})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if paid.State.Debt >= waited.State.Debt {
		t.Errorf("pagamento nao abateu: antes=%d depois=%d", waited.State.Debt, paid.State.Debt)
	}
}

func TestBorrowingIsCappedByTheLimit(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 44)

	out, err := Advance(Input{
		State:   s,
		Until:   5,
		Balance: b,
		Actions: []Action{{ID: 1, Kind: ActionBorrow, At: 1, Amount: int64(b.Credit.MaxPrincipal) + 1}},
	})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	if len(out.Outcomes) != 1 || out.Outcomes[0].Reason != RejectCreditLimit {
		t.Errorf("outcomes = %+v, queria recusa por limite", out.Outcomes)
	}
}

func TestCrowdingRaisesDiseaseRisk(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	b.Shock.CheckEvery = TicksPerDay

	outbreaks := func(fish FishCount) int {
		s := stockedFarm(t, 45)
		s.Tanks[0].Batches[0].Fish = fish
		s.Tanks[0].FeedStock = 500_000 * MicrogramsPerKilogram
		s.Tanks[0].ServedUntil = Tick(maxInt32)
		s.Tanks[0].Aerating = true
		s.Cash = 100_000_000

		out, err := Advance(Input{State: s, Until: 200 * TicksPerDay, Balance: b})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}

		count := 0
		for _, e := range out.Events {
			if e.Kind == EventDisease {
				count++
			}
		}

		return count
	}

	sparse, packed := outbreaks(200), outbreaks(4_500)
	if packed <= sparse {
		t.Errorf("lotacao nao aumentou o risco de surto: folgado=%d lotado=%d", sparse, packed)
	}
}
