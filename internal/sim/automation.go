package sim

type AutoKind uint8

const (
	AutoFeeder AutoKind = iota
	AutoAerator
	AutoHarvester
	AutoTechnician
	AutoContract
	autoKindCount
)

var autoKindNames = [autoKindCount]string{
	AutoFeeder:     "comedouro",
	AutoAerator:    "aerador",
	AutoHarvester:  "peao",
	AutoTechnician: "tecnico",
	AutoContract:   "contrato",
}

func (k AutoKind) String() string {
	if k >= autoKindCount {
		return invalidName
	}

	return autoKindNames[k]
}

func (t *Tank) Owns(k AutoKind) bool {
	if k >= autoKindCount {
		return false
	}

	return t.Upgrades&(1<<k) != 0
}

func (t *Tank) grant(k AutoKind) {
	t.Upgrades |= 1 << k
}

func automate(s *State, b *Balance, t *Tank, tick Tick, sink *eventSink) {
	if t.Owns(AutoAerator) {
		t.Aerating = wantsAeration(t, b) && s.Cash > 0
	}

	if t.Owns(AutoFeeder) {
		restockFeed(s, b, t, tick, sink)
		serve(b, t, tick)
	}

	if t.Owns(AutoHarvester) {
		harvestReady(s, b, t, tick, sink)
	}
}

func wantsAeration(t *Tank, b *Balance) bool {
	if t.Aerating {
		return t.Oxygen < b.Water.AeratorOff
	}

	return t.Oxygen < b.Water.AeratorOn
}

func serve(b *Balance, t *Tank, tick Tick) {
	if t.BatchCount == 0 {
		return
	}

	meals := max(b.Ration.For(t.Batches[0].MeanMass).MealsPerDay, 1)
	t.ServedUntil = tick + TicksPerDay/Tick(meals)
}

func payEnergy(s *State, b *Balance, t *Tank) {
	if !t.Aerating {
		return
	}

	if s.Cash < b.Economy.AeratorCostTick {
		t.Aerating = false

		return
	}

	charge(s, b.Economy.AeratorCostTick)
	spread(t, b.Economy.AeratorCostTick)
}

func restockFeed(s *State, b *Balance, t *Tank, tick Tick, sink *eventSink) {
	if t.FeedStock > 0 || t.Fish() == 0 || MarketAt(b, tick).FeedKg <= 0 {
		return
	}

	unit := MarketAt(b, tick).FeedKg
	kilos := min(int64(autoRestockKg), int64(s.Cash)/int64(unit))
	if kilos < autoMinRestockKg {
		return
	}

	price := Coins(mulDivCeil(int64(unit), kilos, 1))
	if s.Cash < price {
		return
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	mass := Micrograms(kilos) * MicrogramsPerKilogram
	loadFeed(t, mass, price)

	sink.emit(Event{Kind: EventFeedBought, From: tick, To: tick, Tank: t.ID, Mass: mass, Cash: price})
}

func harvestReady(s *State, b *Balance, t *Tank, tick Tick, sink *eventSink) {
	for i := range t.BatchCount {
		batch := &t.Batches[i]
		if batch.Empty() || batch.MeanMass < b.Growth.HarvestMass {
			continue
		}

		sell(s, b, t, batch, batch.Fish, tick, sink)
	}
}

func sell(s *State, b *Balance, t *Tank, batch *Batch, count FishCount, tick Tick, sink *eventSink) {
	if count <= 0 || count > batch.Fish {
		count = batch.Fish
	}
	if count <= 0 {
		return
	}

	mass := Micrograms(mulDivFloor(int64(batch.MeanMass), int64(count), 1))
	price := b.PriceFor(batch.MeanMass, tick)
	revenue := Coins(mulDivFloor(int64(price), int64(mass), int64(MicrogramsPerKilogram)))
	if t.Owns(AutoContract) {
		revenue = Coins(mulDivFloor(int64(revenue), int64(UnitPPM)+int64(b.Progression.ContractBonusPPM), int64(UnitPPM)))
	}

	closeCycle(s, batch, count, mass, revenue)

	batch.Fish -= count
	s.Cash = Coins(addSat(int64(s.Cash), int64(revenue)))
	s.LifetimeEarned = Coins(addSat(int64(s.LifetimeEarned), int64(revenue)))

	sink.emit(Event{
		Kind:  EventHarvest,
		From:  tick,
		To:    tick,
		Tank:  t.ID,
		Batch: batch.ID,
		Fish:  count,
		Mass:  mass,
		Cash:  revenue,
	})
}

const (
	autoRestockKg    = 200
	autoMinRestockKg = 10
)
