package sim

const (
	breakEvenCapDays  = 600
	probeSeed         = Seed(1)
	probeUnlimited    = Coins(1) << 40
	probeThin         = 100
	probeDensityShare = 8
	probeFloor        = 1_000
)

// CyclePlan carries duration in days, mass in micrograms, price per kilo in cents and the stocking that pays the fixed costs.
type CyclePlan struct {
	Days       int64
	Mass       Micrograms
	PricePerKg Coins
	BreakEven  FishCount
}

// CycleAt returns the best-margin plan for the tank kind, or the zero value if no cycle completes.
func (b *Balance) CycleAt(kind TankKind, at Tick, zone ZoneOffset) CyclePlan {
	return b.cycleAtSeed(kind, at, zone, probeSeed)
}

// cycleAtSeed recebe a semente para o teste cobrar que o plano nao depende dela.
func (b *Balance) cycleAtSeed(kind TankKind, at Tick, zone ZoneOffset, seed Seed) CyclePlan {
	// As sondas rodam sem doenca: com ela, o break-even exibido passa a depender de probeSeed
	// ter calhado numa temporada limpa. A mortalidade volta no fim, determinista.
	healthy := *b
	healthy.Shock.DiseaseCount = 0

	// A sonda do plano acompanha a densidade do tanque: com uma lotacao fixa, o tanque caro
	// sai sub-lotado e escolhe um ciclo que ninguem joga.
	dense := max(probeStocking(b, kind)/probeDensityShare, probeFloor)

	plan, denseCash, ok := probeBestCycle(&healthy, kind, at, zone, dense, seed)
	if !ok {
		return CyclePlan{}
	}

	// A segunda sonda fica na faixa magra, perto do break-even, e fecha no dia do plano:
	// sondas de ciclos diferentes nao formam reta. A reta entre as duas so descreve a curva
	// nessa vizinhanca, entao mover probeThin move o numero — ele nao e uma constante livre.
	thinCash, ok := probeCycleUntil(&healthy, kind, at, zone, probeThin, plan.Days, seed)
	if !ok {
		return plan
	}

	// O break-even promete "N peixes pagam a manutencao": e a lotacao onde a fazenda fecha o
	// ciclo com o caixa que comecou. A margem do lote nao ve a manutencao nem a energia, que
	// saem do caixa por dia de tanque; por isso a conta e sobre o caixa, e nao sobre o lote.
	perFish := int64(denseCash-thinCash) / (dense - probeThin)
	if perFish <= 0 {
		return plan
	}
	// O cruzamento vale para um lote que chega inteiro na despesca. Como a doenca so mata,
	// o lote real chega menor, e povoar o cruzamento fecha o ciclo no vermelho: o break-even
	// honesto e o cruzamento dividido pela sobrevivencia esperada do ciclo.
	crossing := max(probeThin-int64(thinCash)/perFish, 0)
	if survival := expectedSurvival(b, at, zone, plan.Days); survival > 0 {
		crossing = mulDivCeil(crossing, int64(UnitPPM), int64(survival))
	}
	plan.BreakEven = FishCount(min(crossing, maxInt32))

	return plan
}

// expectedSurvival devolve, em PPM, a fracao do lote que atravessa days dias de doenca. Sai
// da tabela de doencas e do calendario de checagem, e nao de uma constante: mexer no
// balanceamento move o break-even junto.
//
// Ignora a lotacao, que em crowdedRisk aumenta o risco: o break-even mora na faixa magra,
// onde esse acrescimo e pequeno.
func expectedSurvival(b *Balance, at Tick, zone ZoneOffset, days int64) PPM {
	every := b.Shock.CheckEvery
	if every <= 0 {
		return UnitPPM
	}

	survival := int64(UnitPPM)
	last := at + Tick(days)*TicksPerDay

	for tick := at + every - at%every; tick <= last; tick += every {
		spec, ok := diseaseFor(b, seasonalTemp(b, tick, zone))
		if !ok {
			continue
		}

		risk := mulDivFloor(int64(spec.OutbreakPPM), outbreakLoss(spec), int64(UnitPPM))
		survival -= mulDivFloor(survival, risk, int64(UnitPPM))
	}

	return PPM(survival)
}

// outbreakLoss devolve, em PPM, quanto do lote um surto leva do primeiro ao ultimo dia.
func outbreakLoss(spec DiseaseSpec) int64 {
	alive := int64(UnitPPM)
	for range spec.Ticks {
		alive -= mulDivFloor(alive, int64(spec.DeathPPM), int64(UnitPPM))
	}

	return int64(UnitPPM) - alive
}

func probeStocking(b *Balance, kind TankKind) int64 {
	spec := b.Tanks[kind]

	return spec.MaxDensityPerM3 * int64(spec.Litres) / LitresPerCubicMetre
}

// probeFarm monta a fazenda da sonda: um tanque do tipo, um lote e as duas automacoes, com
// caixa que nao acaba para a sonda medir a producao e nao a falta de dinheiro.
func probeFarm(b *Balance, kind TankKind, at Tick, zone ZoneOffset, fish int64, seed Seed) (State, Coins, bool) {
	s := NewState(seed, zone, at)
	s.Cash = probeUnlimited

	id, ok := s.AddTank(b, kind, b.Tanks[kind].Litres)
	if !ok {
		return State{}, 0, false
	}
	// Depois do tanque: a compra dele e custo de entrada, e nao do ciclo que roda dentro.
	start := s.Cash

	fingerlings := Coins(int64(b.Economy.FingerlingPrice) * fish)
	s.StockTank(id, FishCount(fish), b.Growth.FingerlingMass, fingerlings)
	s.Cash -= fingerlings
	s.SeedOxygen(b)

	tank := s.tank(id)
	tank.grant(AutoFeeder)
	tank.grant(AutoAerator)

	return s, start, true
}

// probeDay is one day of the probe: the plan the farm could close on, and the cash it holds.
type probeDay struct {
	plan CyclePlan
	cash Coins
}

// probeStep roda um dia e devolve onde a fazenda ficou, ou false quando o lote se perdeu.
func probeStep(s *State, b *Balance, at Tick, day int64, start Coins) (probeDay, bool) {
	out, err := Advance(Input{State: *s, Until: at + Tick(day+1)*TicksPerDay, Balance: b})
	if err != nil {
		return probeDay{}, false
	}
	*s = out.State

	if s.TankCount == 0 || s.Tanks[0].BatchCount == 0 {
		return probeDay{}, false
	}

	batch := s.Tanks[0].Batches[0]
	price := b.PriceFor(batch.MeanMass, s.Tick)
	value := mulDivFloor(int64(price), int64(batch.Biomass()), int64(MicrogramsPerKilogram))
	// A racao no silo foi paga mas nao comida: e estoque, nao prejuizo, e o comedouro compra
	// em lote, entao a sobra muda com a lotacao.
	left := mulDivFloor(int64(MarketAt(b, s.Tick).FeedKg),
		int64(s.Tanks[0].FeedStock), int64(MicrogramsPerKilogram))

	return probeDay{
		plan: CyclePlan{Days: day + 1, Mass: batch.MeanMass, PricePerKg: price},
		cash: s.Cash - start + Coins(value+left),
	}, true
}

func probeBestCycle(b *Balance, kind TankKind, at Tick, zone ZoneOffset, fish int64, seed Seed) (CyclePlan, Coins, bool) {
	s, start, ok := probeFarm(b, kind, at, zone, fish, seed)
	if !ok {
		return CyclePlan{}, 0, false
	}

	var (
		best   probeDay
		margin int64
	)

	for day := range int64(breakEvenCapDays) {
		here, stepOK := probeStep(&s, b, at, day, start)
		if !stepOK {
			return CyclePlan{}, 0, false
		}

		batch := s.Tanks[0].Batches[0]

		// Margem por dia, e nao margem total: peixe parado cobra aluguel, entao esperar
		// mais para faturar um pouco mais e uma jogada pior que recomecar o ciclo.
		if now := int64(here.cash); best.plan.Days == 0 || now*best.plan.Days > margin*here.plan.Days {
			best, margin = here, now
		}

		if batch.MeanMass >= topClass(b) {
			break
		}
	}

	return best.plan, best.cash, best.plan.Days > 0
}

// probeCycleUntil fecha o ciclo no dia pedido: sondas de lotacoes diferentes so se comparam
// quando param no mesmo dia.
func probeCycleUntil(b *Balance, kind TankKind, at Tick, zone ZoneOffset, fish, day int64, seed Seed) (Coins, bool) {
	s, start, ok := probeFarm(b, kind, at, zone, fish, seed)
	if !ok {
		return 0, false
	}

	var last probeDay
	for i := range day {
		last, ok = probeStep(&s, b, at, i, start)
		if !ok {
			return 0, false
		}
	}

	return last.cash, day > 0
}

func topClass(b *Balance) Micrograms {
	var top Micrograms
	var best PPM

	for i := range b.Market.ClassCount {
		class := b.Market.Classes[i]
		if class.PPM > best {
			best, top = class.PPM, class.UpToMass
		}
	}
	if top == 0 {
		return b.Growth.HarvestMass
	}

	return top
}
