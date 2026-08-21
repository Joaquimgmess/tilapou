package sim

// apply runs the action and, when it is refused, says how many cents would settle it:
// the price for what needs cash, and the room left in the limit for RejectCreditLimit.
// staleViewTicks e quanto o mundo pode andar entre o jogador ver a tela e a acao chegar. Um
// pedido represado volta minutos depois e seria aplicado contra uma fazenda que ja mudou:
// melhor recusar visivelmente do que aplicar em silencio uma decisao de outro mundo.
const staleViewTicks = Tick(5)

func apply(s *State, b *Balance, a Action, at Tick, sink *eventSink, plans Plans) (reason RejectReason, needed Coins) {
	if a.stale(at) {
		return RejectStaleView, 0
	}

	return dispatch(s, b, a, at, sink, plans)
}

// stale reporta se a tela que o jogador viu ja tinha morrido quando a acao chegou.
func (a Action) stale(at Tick) bool {
	return a.SeenAt > 0 && at-a.SeenAt > staleViewTicks
}

func dispatch(s *State, b *Balance, a Action, at Tick, sink *eventSink, plans Plans) (reason RejectReason, needed Coins) {
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
		return applyHarvest(s, b, a, at, sink)
	case ActionBuyUpgrade:
		return applyBuyUpgrade(s, b, a, at, sink)
	case ActionRestart:
		return restart(s, b, at, sink, plans), 0
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

func applyBuyTank(s *State, b *Balance, a Action, at Tick, sink *eventSink) (reason RejectReason, needed Coins) {
	if a.TankKind >= tankKindCount {
		return RejectUnknownKind, 0
	}

	spec := b.Tanks[a.TankKind]
	price := ladderCost(spec.BaseCost, int64(s.TankCount), b.Progression.CostFactorPPM)
	if s.Cash < price {
		return RejectNotEnoughCash, price
	}

	id, ok := s.AddTank(b, a.TankKind, spec.Litres)
	if !ok {
		return RejectFarmFull, 0
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(price)))
	sink.emit(Event{Kind: EventTankBought, From: at, To: at, Tank: id, Cash: price})

	return RejectNone, 0
}

func applyStock(s *State, b *Balance, a Action, at Tick, sink *eventSink) (reason RejectReason, needed Coins) {
	// Abaixo do piso nao ha ciclo: povoar um punhado de alevinos gasta o caixa, cobra o custo
	// fixo do ciclo inteiro e nao paga nem a manutencao. O resgate ja conta com este piso.
	if a.Amount < MinStockFish || a.Amount > maxInt32 {
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

func applyBuyFeed(s *State, b *Balance, a Action, at Tick, sink *eventSink) (reason RejectReason, needed Coins) {
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
	loadFeed(t, mass, price)

	sink.emit(Event{Kind: EventFeedBought, From: at, To: at, Tank: t.ID, Mass: mass, Cash: price})

	return RejectNone, 0
}

func applyFeed(s *State, b *Balance, a Action, at Tick) (reason RejectReason, needed Coins) {
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

func applyAerate(s *State, a Action) (reason RejectReason, needed Coins) {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}

	t.Aerating = a.Amount != 0

	return RejectNone, 0
}

func applyHarvest(s *State, b *Balance, a Action, at Tick, sink *eventSink) (reason RejectReason, needed Coins) {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}

	for i := range t.BatchCount {
		batch := &t.Batches[i]
		if batch.ID != a.Batch {
			continue
		}
		if batch.Empty() {
			return RejectNoSuchBatch, 0
		}

		count := batch.Fish
		if a.Amount > 0 && a.Amount < int64(count) {
			count = FishCount(a.Amount)
		}
		sell(s, b, t, batch, count, at, sink)

		return RejectNone, 0
	}

	return RejectNoSuchBatch, 0
}

func applyBuyUpgrade(s *State, b *Balance, a Action, at Tick, sink *eventSink) (reason RejectReason, needed Coins) {
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

// NextTankCost is what the next tank of that kind costs now, in cents: the ladder makes
// every tank dearer than the last.
func (s *State) NextTankCost(b *Balance, kind TankKind) Coins {
	if !kind.Known() {
		return 0
	}

	return ladderCost(b.Tanks[kind].BaseCost, int64(s.TankCount), b.Progression.CostFactorPPM)
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
