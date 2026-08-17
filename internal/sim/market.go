package sim

const maxPriceClasses = 6

// PriceClass is a weight band and the multiplier in PPM over the base price.
type PriceClass struct {
	UpToMass Micrograms
	PPM      PPM
}

// MarketBalance carries base prices per kilo in cents and the period in ticks.
type MarketBalance struct {
	FishBasePerKg  Coins
	FeedBasePerKg  Coins
	SwingPPM       PPM
	FeedSwingPPM   PPM
	PeriodTicks    Tick
	Seed           Seed
	Classes        [maxPriceClasses]PriceClass
	ClassCount     int32
	ViableRatioPPM PPM
}

// Market carries prices per kilo in cents and the equivalence ratio in PPM.
type Market struct {
	FishKg   Coins
	FeedKg   Coins
	RatioPPM PPM
}

// MarketAt deterministically computes the fish and feed prices at the given tick.
func MarketAt(b *Balance, tick Tick) Market {
	fish := drift(b.Market.Seed, tick, b.Market.PeriodTicks, b.Market.FishBasePerKg, b.Market.SwingPPM, PurposeMarket)
	feed := drift(b.Market.Seed, tick, b.Market.PeriodTicks, b.Market.FeedBasePerKg, b.Market.FeedSwingPPM, PurposeEvent)

	return Market{
		FishKg:   fish,
		FeedKg:   feed,
		RatioPPM: equivalence(fish, feed, b.Ration.TargetFCRPPM),
	}
}

func equivalence(fish, feed Coins, fcrPPM PPM) PPM {
	cost := mulDivFloor(int64(feed), int64(fcrPPM), int64(UnitPPM))
	if cost <= 0 {
		return 0
	}

	return PPM(mulDivFloor(int64(fish), int64(UnitPPM), cost))
}

func drift(seed Seed, tick, period Tick, base Coins, swing PPM, purpose Purpose) Coins {
	if period <= 0 || base <= 0 {
		return base
	}

	index := floorDiv(int64(tick), int64(period))
	within := floorMod(int64(tick), int64(period))

	from := anchor(seed, index, purpose)
	to := anchor(seed, index+1, purpose)

	blended := from + mulDivFloor(to-from, within, int64(period))
	factor := int64(UnitPPM) + mulDivFloor(blended, int64(swing), int64(UnitPPM))

	return Coins(max(mulDivFloor(int64(base), factor, int64(UnitPPM)), 1))
}

func anchor(seed Seed, index int64, purpose Purpose) int64 {
	roll := seed.RollBelow(RollKey{Tick: Tick(index), Purpose: purpose}, 2*int64(UnitPPM))

	return roll - int64(UnitPPM)
}

// ClassPPM falls on the last class above the band and on UnitPPM if there are no classes.
func (b *Balance) ClassPPM(mass Micrograms) PPM {
	for i := range b.Market.ClassCount {
		if mass <= b.Market.Classes[i].UpToMass {
			return b.Market.Classes[i].PPM
		}
	}
	if b.Market.ClassCount == 0 {
		return UnitPPM
	}

	return b.Market.Classes[b.Market.ClassCount-1].PPM
}

// PriceFor returns the price per kilo in cents, already with the mass's class applied.
func (b *Balance) PriceFor(mass Micrograms, tick Tick) Coins {
	market := MarketAt(b, tick)

	return Coins(mulDivFloor(int64(market.FishKg), int64(b.ClassPPM(mass)), int64(UnitPPM)))
}
