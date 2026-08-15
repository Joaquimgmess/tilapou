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
	case ActionUnknown, actionKindCount:
	}

	return RejectUnknownKind
}

func applyBuyTank(s *State, b *Balance, a Action, at Tick, sink *eventSink) RejectReason {
	if a.TankKind >= tankKindCount {
		return RejectUnknownKind
	}

	spec := b.Tanks[a.TankKind]
	price := ladderCost(spec.BaseCost, int64(s.TankCount))
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

		mass := Micrograms(mulDivFloor(int64(batch.MeanMass), int64(count), 1))
		revenue := Coins(mulDivFloor(int64(b.Economy.FishPricePerKg), int64(mass), int64(MicrogramsPerKilogram)))

		batch.Fish -= count
		s.Cash = Coins(addSat(int64(s.Cash), int64(revenue)))
		s.LifetimeEarned = Coins(addSat(int64(s.LifetimeEarned), int64(revenue)))

		sink.emit(Event{
			Kind:  EventHarvest,
			From:  at,
			To:    at,
			Tank:  t.ID,
			Batch: batch.ID,
			Fish:  count,
			Mass:  mass,
			Cash:  revenue,
		})

		return RejectNone
	}

	return RejectNoSuchBatch
}

func ladderCost(base Coins, owned int64) Coins {
	cost := int64(base)
	for range owned {
		cost = mulDivCeil(cost, 115, 100)
	}

	return Coins(cost)
}
