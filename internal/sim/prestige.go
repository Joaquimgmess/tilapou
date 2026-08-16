package sim

const prestigeScale = 10

func PrestigePointsFor(lifetime Coins, divisor int64) uint32 {
	if divisor <= 0 || lifetime <= 0 {
		return 0
	}

	return uint32(prestigeScale * isqrt(int64(lifetime)/divisor))
}

func (s *State) prestigeBonus(b *Balance) PPM {
	return PPM(addSat(int64(UnitPPM), int64(s.Prestige)*int64(b.Progression.PrestigeBonusPPM)))
}

func (s *State) Broke(b *Balance) bool {
	if s.Fish() > 0 || s.Cash >= b.Economy.FingerlingPrice {
		return false
	}
	if PrestigePointsFor(s.LifetimeEarned, b.Progression.PrestigeDivisor) > s.Prestige {
		return false
	}

	return Coins(subSat(int64(b.Credit.MaxPrincipal), int64(s.Debt))) < b.Economy.FingerlingPrice
}

func restart(s *State, b *Balance, at Tick, sink *eventSink) RejectReason {
	if !s.Broke(b) {
		return RejectNotBroke
	}

	s.Debt, s.DebtCarry = 0, 0
	rebuild(s, b, at, s.Prestige)

	sink.emit(Event{Kind: EventRestarted, From: at, To: at})

	return RejectNone
}

func prestige(s *State, b *Balance, at Tick, sink *eventSink) RejectReason {
	earned := PrestigePointsFor(s.LifetimeEarned, b.Progression.PrestigeDivisor)
	if earned <= s.Prestige {
		return RejectNotEnoughLifetime
	}

	rebuild(s, b, at, earned)

	sink.emit(Event{Kind: EventPrestiged, From: at, To: at})

	return RejectNone
}

func rebuild(s *State, b *Balance, at Tick, prestige uint32) {
	kept := State{
		Version:        s.Version,
		BalanceVersion: s.BalanceVersion,
		RngVersion:     s.RngVersion,
		Seed:           s.Seed,
		Zone:           s.Zone,
		Tick:           s.Tick,
		Cash:           b.Progression.RestartCash,
		LifetimeEarned: s.LifetimeEarned,
		Prestige:       prestige,
		NextTankID:     1,
		NextBatchID:    1,
		EventSeq:       s.EventSeq,
		Debt:           s.Debt,
		DebtCarry:      s.DebtCarry,
	}

	tank, ok := kept.addTank(TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if ok {
		target := kept.tank(tank)
		target.addBatch(kept.NextBatchID, b.Progression.RestartFish, b.Growth.FingerlingMass, at)
		kept.NextBatchID++
		target.FeedStock = b.Progression.RestartFeed
	}

	*s = kept
}
