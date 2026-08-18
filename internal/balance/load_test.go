package balance_test

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

func TestLoadQuantizesTheShippedFile(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := map[string]struct {
		got, want int64
	}{
		"versao":              {int64(b.Version), 1},
		"tgc em ppm":          {int64(b.Growth.TGCPPM), 1_400},
		"temperatura de ref":  {int64(b.Growth.ReferenceTemp), 26_000},
		"peso maximo em ug":   {int64(b.Growth.MaxMass), 1_100_000_000},
		"peso de abate em ug": {int64(b.Growth.HarvestMass), 600_000_000},
		"od critico":          {int64(b.Water.Critical), 2_000},
		"fator de custo":      {int64(b.Progression.CostFactorPPM), 1_120_000},
		"passos de racao":     {int64(b.Ration.Len), 5},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", name, tt.got, tt.want)
			}
		})
	}
}

func TestLoadFillsEveryTankSlot(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, kind := range []sim.TankKind{sim.TankEarthPond, sim.TankNetCage, sim.TankBiofloc, sim.TankRecirculation} {
		spec := b.Tanks[kind]
		if spec.Litres <= 0 || spec.MaxDensityPerM3 <= 0 || spec.BaseCost <= 0 {
			t.Errorf("tanque %d ficou sem spec: %+v", kind, spec)
		}
	}
}

func TestLoadIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	second, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if first != second {
		t.Error("duas cargas do mesmo arquivo produziram balances diferentes")
	}
}

func TestShippedBalanceGrowsFishToHarvest(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	state := sim.NewState(1, -180, 0)
	state.Cash = 5_000_000

	actions := []sim.Action{
		{ID: 1, Kind: sim.ActionBuyTank, At: 1, TankKind: sim.TankEarthPond},
		{ID: 2, Kind: sim.ActionStock, At: 2, Tank: 1, Amount: 2_000},
		{ID: 3, Kind: sim.ActionBuyFeed, At: 3, Tank: 1, Amount: 2_000},
		{ID: 4, Kind: sim.ActionBuyUpgrade, At: 4, Tank: 1, Auto: sim.AutoFeeder},
	}

	out, err := sim.Advance(sim.Input{State: state, Until: 120 * sim.TicksPerDay, Balance: &b, Actions: actions})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	for _, o := range out.Outcomes {
		if !o.Applied {
			t.Fatalf("acao %d rejeitada: %v", o.ID, o.Reason)
		}
	}

	grams := out.State.Tanks[0].Batches[0].MeanMass.Grams()
	if grams < 100 || grams > 600 {
		t.Errorf("peso apos 120 dias partindo de alevino = %d g, esperava entre 100 e 600", grams)
	}
	if fish := out.State.Tanks[0].Batches[0].Fish; fish < 1_200 {
		t.Errorf("sobraram %d de 2000 peixes em 120 dias: mortalidade alta demais", fish)
	}
}

func TestTodoTanqueDoBalanceTemPlanoDeCicloUsavel(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	for _, kind := range []sim.TankKind{
		sim.TankEarthPond, sim.TankNetCage, sim.TankBiofloc, sim.TankRecirculation,
	} {
		plan := b.CycleAt(kind, 0, 0)

		switch {
		case plan.Days <= 0:
			t.Errorf("%s nao tem plano de ciclo: o conselho de lotacao fica cego nesse tanque", kind)
		case plan.Days > 300:
			t.Errorf("%s planeja %d dias ate %d g: esperar tanto derruba a margem por dia",
				kind, plan.Days, plan.Mass.Grams())
		case plan.Mass.Grams() < b.Growth.HarvestMass.Grams():
			t.Errorf("%s planeja vender a %d g, antes do ponto de abate de %d g",
				kind, plan.Mass.Grams(), b.Growth.HarvestMass.Grams())
		case plan.BreakEven <= 0:
			t.Errorf("%s volta break-even zero, e o conselho pula tanque com break-even zero", kind)
		}
	}
}

func TestOPeaoVendeNoPontoDeMargemDeCadaTanque(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	for _, kind := range []sim.TankKind{
		sim.TankEarthPond, sim.TankNetCage, sim.TankBiofloc, sim.TankRecirculation,
	} {
		plan := b.CycleAt(kind, 0, 0)
		point := sim.HarvestPoint(&b, kind, 0, 0)

		if point < b.Growth.HarvestMass {
			t.Errorf("%s: o peao venderia a %d g, antes do peso de abate", kind, point.Grams())
		}
		if plan.Mass > b.Growth.HarvestMass && point != plan.Mass {
			t.Errorf("%s: o peao vende a %d g e o ponto de margem do tanque e %d g: joga fora a diferenca",
				kind, point.Grams(), plan.Mass.Grams())
		}
	}
}

func TestEmprestimoOferecidoCobreUmSacoDeRacao(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	// O cenario que o QA viveu: fazenda logo depois da falencia, caixa zero, divida
	// pequena e limite de credito quase todo livre.
	s := sim.NewState(1, 0, 0)
	s.Cash = 0
	s.Debt = 5_985

	id, ok := s.AddTank(&b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}
	plan := b.CycleAt(sim.TankEarthPond, 0, 0)

	// Doze peixes abaixo do break-even: e ai que o conselho oferece um emprestimo
	// dimensionado por esses doze peixes, que nao paga nem racao.
	s.StockTank(id, plan.BreakEven-12, 450*sim.MicrogramsPerGram, 1_000)

	loan, block := s.LoanAdvice(&b, id, plan)
	if block != sim.LoanOpen {
		t.Fatalf("o credito esta bloqueado por %v com o limite quase todo livre", block)
	}

	// 100 kg e o saco que a loja vende: um emprestimo que nao paga um saco nao serve.
	sack := int64(sim.MarketAt(&b, s.Tick).FeedKg) * 100

	if int64(loan)+int64(s.Cash) < sack {
		t.Errorf("o emprestimo oferecido e %d e com o caixa da %d, e um saco de racao custa %d: o jogador aceita a divida e segue sem poder alimentar",
			loan, int64(loan)+int64(s.Cash), sack)
	}
}

// Apertar a tecla de povoar repetidas vezes nao pode encher o tanque sozinho: cada aperto
// tem a bencao do jogo, e o custo fixo do ciclo que a primeira sugestao reservou evapora.
// Quem enche o tanque e o jogador que tem caixa para isso, e nao a repeticao.
func TestApertarPovoarDeNovoNaoLotaOTanqueSozinho(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	for _, cash := range []int64{500_000, 1_000_000, 2_183_221, 4_000_000} {
		for _, debt := range []sim.Coins{0, b.Credit.MaxPrincipal / 2, b.Credit.MaxPrincipal} {
			s := sim.NewState(1, 0, 0)
			s.Cash, s.Debt = sim.Coins(cash), debt

			id, ok := s.AddTank(&b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
			if !ok {
				t.Fatal("sem tanque")
			}

			capacity := s.Tanks[0].Capacity(&b)
			plan := b.CycleAt(sim.TankEarthPond, s.Tick, s.Zone)

			first, perFish := s.StockAdvice(&b, id, plan)
			if first <= 0 {
				continue
			}

			s.StockTank(id, first, b.Growth.FingerlingMass, sim.Coins(int64(first))*perFish)
			s.Cash -= sim.Coins(int64(first)) * perFish

			if again, _ := s.StockAdvice(&b, id, plan); again > 0 {
				t.Errorf("caixa %d divida %d: povoou %d de %d e o conselho ainda manda povoar mais %d",
					cash, debt, first, capacity, again)
			}
		}
	}
}

// O break-even promete "N peixes pagam a manutencao". A promessa cobrada aqui e a ordem de
// grandeza: com metade dele a fazenda perde, com o dobro ela ganha. Perto do numero exato a
// doenca decide o sinal, entao uma assercao ali seria sorteio e nao juiz.
//
// O numero e otimista em ~17%: a sonda repoe a biomassa que a mortalidade tirou, mas nao a
// racao ja paga no peixe que morreu antes da despesca. O cruzamento medido em 24 sementes
// fica em ~890 contra os 760 de hoje. Ver TASK-134.
func TestOBreakEvenPagaMesmoAManutencao(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}
	b.Market.SwingPPM = 0

	plan := b.CycleAt(sim.TankEarthPond, 0, 0)
	if plan.BreakEven <= 0 {
		t.Fatal("o viveiro nao tem break-even")
	}

	if lucro := meanCycleProfit(t, &b, plan.BreakEven/2); lucro >= 0 {
		t.Errorf("com metade do break-even (%d peixes) a fazenda ainda lucrou %d",
			plan.BreakEven/2, lucro)
	}
	if lucro := meanCycleProfit(t, &b, plan.BreakEven*2); lucro <= 0 {
		t.Errorf("com o dobro do break-even (%d peixes) a fazenda ainda perdeu %d",
			plan.BreakEven*2, -lucro)
	}
}

// meanCycleProfit roda o ciclo em varias sementes e devolve o caixa medio. Oito e o que faz o
// sinal parar de virar entre janelas de sementes.
func meanCycleProfit(t *testing.T, b *sim.Balance, fish sim.FishCount) int64 {
	t.Helper()

	const seeds = 8

	var sum int64
	for seed := sim.Seed(1); seed <= seeds; seed++ {
		sum += cycleProfit(t, b, fish, seed)
	}

	return sum / seeds
}

// cycleProfit devolve quanto a fazenda ganha ou perde de caixa num ciclo completo.
func cycleProfit(t *testing.T, b *sim.Balance, fish sim.FishCount, seed sim.Seed) int64 {
	t.Helper()

	s := sim.NewState(seed, 0, 0)
	s.Cash = 100_000_000

	id, ok := s.AddTank(b, sim.TankEarthPond, b.Tanks[sim.TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	start := s.Cash
	s.StockTank(id, fish, b.Growth.FingerlingMass, sim.Coins(int64(fish))*b.Economy.FingerlingPrice)
	s.Cash -= sim.Coins(int64(fish)) * b.Economy.FingerlingPrice
	s.SeedOxygen(b)
	s.Tanks[0].Upgrades = 0b11

	for range 400 {
		out, err := sim.Advance(sim.Input{State: s, Until: s.Tick + sim.TicksPerDay, Balance: b})
		if err != nil {
			t.Fatal(err)
		}
		s = out.State

		batch := &s.Tanks[0].Batches[0]
		if batch.Empty() {
			break
		}

		if batch.MeanMass >= b.Growth.HarvestMass {
			price := b.PriceFor(batch.MeanMass, s.Tick)
			kilos := int64(batch.Biomass()) / int64(sim.MicrogramsPerKilogram)
			left := mulKg(int64(s.Tanks[0].FeedStock), int64(sim.MarketAt(b, s.Tick).FeedKg))

			return int64(s.Cash) + int64(price)*kilos + left - int64(start)
		}
	}

	return int64(s.Cash) - int64(start)
}

func mulKg(micrograms, perKg int64) int64 {
	return micrograms / int64(sim.MicrogramsPerKilogram) * perKg
}
