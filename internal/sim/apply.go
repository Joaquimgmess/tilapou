package sim

func apply(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	switch a.Kind {
	case ActionBuyTank:
		return applyBuyTank(s, b, a, at, sink)
	case ActionStock:
		return applyStock(s, b, a, at, sink)
	case ActionBuyFeed:
		return applyBuyFeed(s, b, a, at, sink)
	case ActionFeed:
		return applyFeed(s, a)
	case ActionAerate:
		return applyAerate(s, a)
	case ActionHarvest:
		return applyHarvest(s, b, a, at, sink)
	case ActionBuyUpgrade:
		return applyBuyUpgrade(s, b, a, at, sink)
	case ActionPrestige:
		return prestige(s, b, at, sink)
	case ActionUnknown, actionKindCount:
	}

	return RejectUnknownKind
}

func applyBuyTank(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	if a.TankKind >= tankKindCount {
		return RejectUnknownKind
	}

	spec := b.Tanks[a.TankKind]
	price := ladderCost(spec.BaseCost, int64(s.TankCount), b.Progression.CostFactorPPM)
	if s.Cash < price {
		return RejectNotEnoughCash
	}

	id, ok := s.addTank(a.TankKind, spec.Litres)
	if !ok {
		return RejectFarmFull
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	sink.emit(Event{Kind: EventTankBought, From: at, To: at, Tank: id, Cash: price})

	return RejectNone
}

func applyStock(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	if a.Amount <= 0 || a.Amount > maxInt32 {
		return RejectBadAmount
	}

	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank
	}

	spec := b.Tanks[t.Kind]
	capacity := spec.MaxDensityPerM3 * int64(t.Litres) / 1000
	if int64(t.Fish())+a.Amount > capacity {
		return RejectTooDense
	}

	price := Coins(mulDivCeil(int64(b.Economy.FingerlingPrice), a.Amount, 1))
	if s.Cash < price {
		return RejectNotEnoughCash
	}

	id := s.NextBatchID
	if !t.addBatch(id, FishCount(a.Amount), b.Growth.FingerlingMass, at) {
		return RejectTankFull
	}
	s.NextBatchID++
	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))

	sink.emit(Event{
		Kind:  EventStocked,
		From:  at,
		To:    at,
		Tank:  t.ID,
		Batch: id,
		Fish:  FishCount(a.Amount),
		Cash:  price,
	})

	return RejectNone
}

func applyBuyFeed(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	if a.Amount <= 0 {
		return RejectBadAmount
	}

	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank
	}

	mass := Micrograms(mulDivFloor(a.Amount, int64(MicrogramsPerKilogram), 1))
	price := Coins(mulDivCeil(int64(b.Economy.FeedPricePerKg), a.Amount, 1))
	if s.Cash < price {
		return RejectNotEnoughCash
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	t.FeedStock = Micrograms(addSat(int64(t.FeedStock), int64(mass)))

	sink.emit(Event{Kind: EventFeedBought, From: at, To: at, Tank: t.ID, Mass: mass, Cash: price})

	return RejectNone
}

func applyFeed(s *State, a Action) RejectReason {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank
	}
	if t.FeedStock <= 0 {
		return RejectNotEnoughFeed
	}

	return RejectNone
}

func applyAerate(s *State, a Action) RejectReason {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank
	}

	t.Aerating = a.Amount != 0

	return RejectNone
}

func applyHarvest(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank
	}

	for i := range t.BatchCount {
		batch := &t.Batches[i]
		if batch.ID != a.Batch {
			continue
		}
		if batch.Empty() {
			return RejectNoSuchBatch
		}

		count := batch.Fish
		if a.Amount > 0 && a.Amount < int64(count) {
			count = FishCount(a.Amount)
		}
		sell(s, b, t, batch, count, at, sink)

		return RejectNone
	}

	return RejectNoSuchBatch
}

func applyBuyUpgrade(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	if a.Auto >= autoKindCount {
		return RejectUnknownKind
	}
	if s.Owns(a.Auto) {
		return RejectAlreadyOwned
	}

	price := b.Automation[a.Auto].Cost
	if s.Cash < price {
		return RejectNotEnoughCash
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	s.grant(a.Auto)

	sink.emit(Event{Kind: EventUpgradeBought, From: at, To: at, Cash: price, Fish: FishCount(a.Auto)})

	return RejectNone
}

func ladderCost(base Coins, owned int64, factor PPM) Coins {
	if factor <= 0 {
		factor = UnitPPM
	}

	cost := int64(base)
	for range owned {
		cost = mulDivCeil(cost, int64(factor), int64(UnitPPM))
	}

	return Coins(cost)
}
