package sim

func apply(s *State, b *Balance, a Action, at Tick, sink *eventSink) (RejectReason, Coins) {
	switch a.Kind {
	case ActionBuyTank:
		return applyBuyTank(s, b, a, at, sink)
	case ActionStock:
		return applyStock(s, b, a, at, sink)
	case ActionBuyFeed:
		return applyBuyFeed(s, b, a, at, sink)
	case ActionFeed:
		return applyFeed(s, b, a, at)
	case ActionAerate:
		return applyAerate(s, a)
	case ActionHarvest:
		return applyHarvest(s, b, a, at, sink), 0
	case ActionBuyUpgrade:
		return applyBuyUpgrade(s, b, a, at, sink)
	case ActionRestart:
		return restart(s, b, at, sink), 0
	case ActionPrestige:
		return prestige(s, b, at, sink), 0
	case ActionBorrow:
		return borrow(s, b, Coins(a.Amount), at, sink)
	case ActionRepay:
		return repay(s, Coins(a.Amount), at, sink)
	case ActionTreat:
		return treat(s, b, a, at, sink)
	case ActionUnknown, actionKindCount:
	}

	return RejectUnknownKind, 0
}

func applyBuyTank(s *State, b *Balance, a Action, at Tick, sink *eventSink) (RejectReason, Coins) {
	if a.TankKind >= tankKindCount {
		return RejectUnknownKind, 0
	}

	spec := b.Tanks[a.TankKind]
	price := ladderCost(spec.BaseCost, int64(s.TankCount), b.Progression.CostFactorPPM)
	if s.Cash < price {
		return RejectNotEnoughCash, price
	}

	id, ok := s.addTank(a.TankKind, spec.Litres)
	if !ok {
		return RejectFarmFull, 0
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	sink.emit(Event{Kind: EventTankBought, From: at, To: at, Tank: id, Cash: price})

	return RejectNone, 0
}

func applyStock(s *State, b *Balance, a Action, at Tick, sink *eventSink) (RejectReason, Coins) {
	if a.Amount <= 0 || a.Amount > maxInt32 {
		return RejectBadAmount, 0
	}

	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}

	if int64(t.Fish())+a.Amount > t.Capacity(b) {
		return RejectTooDense, 0
	}

	price := Coins(mulDivCeil(int64(b.Economy.FingerlingPrice), a.Amount, 1))
	if s.Cash < price {
		return RejectNotEnoughCash, price
	}

	id := s.NextBatchID
	if !t.addBatch(id, FishCount(a.Amount), b.Growth.FingerlingMass, at) {
		return RejectTankFull, 0
	}
	t.Batches[t.BatchCount-1].Cost = price
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

	return RejectNone, 0
}

func applyBuyFeed(s *State, b *Balance, a Action, at Tick, sink *eventSink) (RejectReason, Coins) {
	if a.Amount <= 0 {
		return RejectBadAmount, 0
	}

	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}

	mass := Micrograms(mulDivFloor(a.Amount, int64(MicrogramsPerKilogram), 1))
	price := Coins(mulDivCeil(int64(MarketAt(b, at).FeedKg), a.Amount, 1))
	if s.Cash < price {
		return RejectNotEnoughCash, price
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	t.FeedStock = Micrograms(addSat(int64(t.FeedStock), int64(mass)))
	spread(t, price)

	sink.emit(Event{Kind: EventFeedBought, From: at, To: at, Tank: t.ID, Mass: mass, Cash: price})

	return RejectNone, 0
}

func applyFeed(s *State, b *Balance, a Action, at Tick) (RejectReason, Coins) {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}
	if t.FeedStock <= 0 {
		return RejectNotEnoughFeed, 0
	}
	if t.BatchCount == 0 {
		return RejectNoSuchBatch, 0
	}

	serve(b, t, at)

	return RejectNone, 0
}

func applyAerate(s *State, a Action) (RejectReason, Coins) {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}

	t.Aerating = a.Amount != 0

	return RejectNone, 0
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

func applyBuyUpgrade(s *State, b *Balance, a Action, at Tick, sink *eventSink) (RejectReason, Coins) {
	if a.Auto >= autoKindCount {
		return RejectUnknownKind, 0
	}

	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}
	if t.Owns(a.Auto) {
		return RejectAlreadyOwned, 0
	}

	price := b.Automation[a.Auto].Cost
	if s.Cash < price {
		return RejectNotEnoughCash, price
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	t.grant(a.Auto)

	sink.emit(Event{Kind: EventUpgradeBought, From: at, To: at, Tank: t.ID, Cash: price})

	return RejectNone, 0
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
