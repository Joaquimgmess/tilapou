package sim

const (
	forecastCapDays   = 400
	forecastFeedFloor = 50 * MicrogramsPerKilogram
	forecastFeedTopUp = 500 * MicrogramsPerKilogram
)

type Forecast struct {
	Reached    bool
	Days       int64
	MeanMass   Micrograms
	Fish       FishCount
	FeedCost   Coins
	Value      Coins
	Margin     Coins
	PricePerKg Coins
}

func (s *State) Forecast(b *Balance, tank TankID, batch BatchID, target Micrograms) Forecast {
	start := *s
	found := start.batch(tank, batch)
	if found == nil || found.Empty() {
		return Forecast{}
	}

	spent := found.Cost
	from := s.Tick

	for day := range int64(forecastCapDays) {
		keepManaged(&start, b, tank)

		out, err := Advance(Input{State: start, Until: from + Tick(day+1)*TicksPerDay, Balance: b})
		if err != nil {
			break
		}
		start = out.State

		current := start.batch(tank, batch)
		if current == nil || current.Empty() {
			break
		}
		if current.MeanMass >= target {
			return closeForecast(&start, b, current, spent, day+1, true)
		}
	}

	current := start.batch(tank, batch)
	if current == nil {
		return Forecast{}
	}

	return closeForecast(&start, b, current, spent, forecastCapDays, false)
}

func keepManaged(s *State, b *Balance, tank TankID) {
	t := s.tank(tank)
	if t == nil {
		return
	}

	t.ServedUntil = s.Tick + TicksPerDay
	t.Aerating = wantsAeration(t, b)

	if t.FeedStock >= forecastFeedFloor {
		return
	}

	missing := forecastFeedTopUp - t.FeedStock
	kilos := int64(missing) / int64(MicrogramsPerKilogram)
	price := Coins(mulDivCeil(int64(MarketAt(b, s.Tick).FeedKg), kilos, 1))

	t.FeedStock = forecastFeedTopUp
	spread(t, price)
}

func closeForecast(s *State, b *Balance, batch *Batch, spent Coins, days int64, reached bool) Forecast {
	price := b.PriceFor(batch.MeanMass, s.Tick)
	value := Coins(mulDivFloor(int64(price), int64(batch.Biomass()), int64(MicrogramsPerKilogram)))

	return Forecast{
		Reached:    reached,
		Days:       days,
		MeanMass:   batch.MeanMass,
		Fish:       batch.Fish,
		FeedCost:   Coins(subSat(int64(batch.Cost), int64(spent))),
		Value:      value,
		Margin:     Coins(subSat(int64(value), int64(batch.Cost))),
		PricePerKg: price,
	}
}

func (s *State) batch(tank TankID, batch BatchID) *Batch {
	t := s.tank(tank)
	if t == nil {
		return nil
	}

	for i := range t.BatchCount {
		if t.Batches[i].ID == batch {
			return &t.Batches[i]
		}
	}

	return nil
}

func (b *Balance) NextClass(mass Micrograms) (upTo Micrograms, gain PPM, ok bool) {
	current := b.ClassPPM(mass)

	for i := range b.Market.ClassCount {
		class := b.Market.Classes[i]
		if class.UpToMass <= mass || class.PPM <= current {
			continue
		}

		return class.UpToMass, PPM(mulDivFloor(int64(class.PPM), int64(UnitPPM), int64(current))) - UnitPPM, true
	}

	return 0, 0, false
}

func (s *State) Series(b *Balance, points int, step Tick) (fish, feed []Coins) {
	if points <= 0 || step <= 0 {
		return nil, nil
	}

	points = min(points, int(s.Tick/step)+1)
	fish = make([]Coins, 0, points)
	feed = make([]Coins, 0, points)

	for i := points - 1; i >= 0; i-- {
		at := s.Tick - Tick(i)*step
		market := MarketAt(b, at)
		fish = append(fish, market.FishKg)
		feed = append(feed, market.FeedKg)
	}

	return fish, feed
}
