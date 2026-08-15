package sim

const (
	breakEvenCapDays = 600
	probeLow         = 1_000
	probeHigh        = 2_000
	probeSeed        = Seed(1)
	probeUnlimited   = Coins(1) << 40
)

type CyclePlan struct {
	Days       int64
	Mass       Micrograms
	PricePerKg Coins
	BreakEven  FishCount
}

func (b *Balance) CycleAt(kind TankKind, at Tick, zone ZoneOffset) CyclePlan {
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

func probeCycle(b *Balance, kind TankKind, at Tick, zone ZoneOffset, fish int64) (CyclePlan, int64, bool) {
	s := NewState(probeSeed, zone, at)
	s.Cash = probeUnlimited

	id, ok := s.AddTank(kind, b.Tanks[kind].Litres)
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

		if now := value - int64(batch.Cost); best.Days == 0 || now > margin {
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
