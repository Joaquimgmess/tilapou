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
		t.Aerating = t.Oxygen < b.Water.IdealMin && s.Cash > 0
	}

	if t.Owns(AutoFeeder) {
		restockFeed(s, b, t, tick, sink)
		serve(b, t, tick)
	}

	if t.Owns(AutoHarvester) {
		harvestReady(s, b, t, tick, sink)
	}
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

	s.Cash = Coins(subSat(int64(s.Cash), int64(b.Economy.AeratorCostTick)))
}

func restockFeed(s *State, b *Balance, t *Tank, tick Tick, sink *eventSink) {
	if t.FeedStock > 0 || t.Fish() == 0 {
		return
	}

	kilos := int64(autoRestockKg)
	price := Coins(mulDivCeil(int64(b.Economy.FeedPricePerKg), kilos, 1))
	if s.Cash < price {
		return
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	mass := Micrograms(kilos) * MicrogramsPerKilogram
	t.FeedStock = Micrograms(addSat(int64(t.FeedStock), int64(mass)))

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
	revenue := Coins(mulDivFloor(int64(b.Economy.FishPricePerKg), int64(mass), int64(MicrogramsPerKilogram)))
	if t.Owns(AutoContract) {
		revenue = Coins(mulDivFloor(int64(revenue), int64(UnitPPM)+int64(b.Progression.ContractBonusPPM), int64(UnitPPM)))
	}

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

const autoRestockKg = 200
