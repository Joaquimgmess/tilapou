package sim

const (
	breakEvenCapDays = 600
	probeSeed        = Seed(1)
	probeUnlimited   = Coins(1) << 40
	probeFloor       = 1_000
	probeDivisor     = 8
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
	// As sondas acompanham a densidade do tanque: com uma lotacao fixa, o tanque caro sai
	// sub-lotado, a margem fica negativa em todo dia e o plano volta vazio. Ficam perto uma
	// da outra e perto do piso, senao a reta que estima o break-even extrapola de longe.
	probeLow := max(probeStocking(b, kind)/probeDivisor, probeFloor)
	probeHigh := probeLow * 2

	low, lowMargin, ok := probeCycle(b, kind, at, zone, probeLow)
	if !ok {
		return CyclePlan{}
	}

	_, highMargin, ok := probeCycle(b, kind, at, zone, probeHigh)
	if !ok {
		return low
	}

	perFish := (highMargin - lowMargin) / (probeHigh - probeLow)
	if perFish <= 0 {
		return low
	}

	fixed := probeLow*perFish - lowMargin
	low.BreakEven = FishCount(min(max((fixed+perFish-1)/perFish, 0), maxInt32))

	return low
}

func probeStocking(b *Balance, kind TankKind) int64 {
	spec := b.Tanks[kind]

	return spec.MaxDensityPerM3 * int64(spec.Litres) / LitresPerCubicMetre
}

func probeCycle(b *Balance, kind TankKind, at Tick, zone ZoneOffset, fish int64) (CyclePlan, int64, bool) {
	s := NewState(probeSeed, zone, at)
	s.Cash = probeUnlimited

	id, ok := s.AddTank(b, kind, b.Tanks[kind].Litres)
	if !ok {
		return CyclePlan{}, 0, false
	}
	s.StockTank(id, FishCount(fish), b.Growth.FingerlingMass, Coins(int64(b.Economy.FingerlingPrice)*fish))
	s.SeedOxygen(b)

	tank := s.tank(id)
	tank.grant(AutoFeeder)
	tank.grant(AutoAerator)

	var (
		best   CyclePlan
		margin int64
	)

	for day := range int64(breakEvenCapDays) {
		out, err := Advance(Input{State: s, Until: at + Tick(day+1)*TicksPerDay, Balance: b})
		if err != nil {
			return CyclePlan{}, 0, false
		}
		s = out.State

		if s.TankCount == 0 || s.Tanks[0].BatchCount == 0 {
			return CyclePlan{}, 0, false
		}
		batch := s.Tanks[0].Batches[0]
		price := b.PriceFor(batch.MeanMass, s.Tick)
		value := mulDivFloor(int64(price), int64(batch.Biomass()), int64(MicrogramsPerKilogram))

		// Margem por dia, e nao margem total: peixe parado cobra aluguel, entao esperar
		// mais para faturar um pouco mais e uma jogada pior que recomecar o ciclo.
		if now := value - int64(batch.Cost); best.Days == 0 || now*best.Days > margin*(day+1) {
			best, margin = CyclePlan{Days: day + 1, Mass: batch.MeanMass, PricePerKg: price}, now
		}

		if batch.MeanMass >= topClass(b) {
			break
		}
	}

	return best, margin, best.Days > 0
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
