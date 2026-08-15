package sim

type Cycle struct {
	Fish       FishCount
	Mass       Micrograms
	Revenue    Coins
	Cost       Coins
	CostPerKg  Coins
	PricePerKg Coins
	FCRPPM     PPM
}

func (c Cycle) Margin() Coins {
	return Coins(subSat(int64(c.Revenue), int64(c.Cost)))
}

func charge(s *State, amount Coins) {
	if amount <= 0 {
		return
	}

	if s.Cash >= amount {
		s.Cash = Coins(subSat(int64(s.Cash), int64(amount)))

		return
	}

	short := Coins(subSat(int64(amount), int64(s.Cash)))
	s.Cash = 0
	s.Debt = Coins(addSat(int64(s.Debt), int64(short)))
}

func chargeUpkeep(s *State, b *Balance, t *Tank) {
	daily := int64(b.Tanks[t.Kind].UpkeepPerDay)
	if daily <= 0 {
		return
	}

	due := carryTake(daily, int64(TicksPerDay), &t.UpkeepCarry)
	if due <= 0 {
		return
	}

	charge(s, Coins(due))
	spread(t, Coins(due))
}

func spread(t *Tank, amount Coins) {
	if amount <= 0 || t.BatchCount == 0 {
		return
	}

	total := int64(t.Biomass())
	if total <= 0 {
		share := int64(amount) / int64(t.BatchCount)
		for i := range t.BatchCount {
			t.Batches[i].Cost = Coins(addSat(int64(t.Batches[i].Cost), share))
		}

		return
	}

	for i := range t.BatchCount {
		batch := &t.Batches[i]
		share := mulDivFloor(int64(amount), int64(batch.Biomass()), total)
		batch.Cost = Coins(addSat(int64(batch.Cost), share))
	}
}

func accrueInterest(s *State, b *Balance) {
	if s.Debt <= 0 || b.Credit.DailyRatePPM <= 0 {
		return
	}

	daily := mulDivFloor(int64(s.Debt), int64(b.Credit.DailyRatePPM), int64(UnitPPM))
	due := carryTake(daily, int64(TicksPerDay), &s.DebtCarry)
	if due <= 0 {
		return
	}

	charge(s, Coins(due))
}

func borrow(s *State, b *Balance, amount Coins, at Tick, sink *eventSink) (RejectReason, Coins) {
	if amount <= 0 {
		return RejectBadAmount, 0
	}

	room := Coins(subSat(int64(b.Credit.MaxPrincipal), int64(s.Debt)))
	if room <= 0 || amount > room {
		return RejectCreditLimit, room
	}

	s.Debt = Coins(addSat(int64(s.Debt), int64(amount)))
	s.Cash = Coins(addSat(int64(s.Cash), int64(amount)))

	sink.emit(Event{Kind: EventBorrowed, From: at, To: at, Cash: amount})

	return RejectNone, 0
}

func repay(s *State, amount Coins, at Tick, sink *eventSink) (RejectReason, Coins) {
	if s.Debt <= 0 {
		return RejectNoDebt, 0
	}

	due := min(amount, s.Debt)
	if due <= 0 {
		due = s.Debt
	}
	if s.Cash < due {
		return RejectNotEnoughCash, due
	}

	s.Cash = Coins(subSat(int64(s.Cash), int64(due)))
	s.Debt = Coins(subSat(int64(s.Debt), int64(due)))

	sink.emit(Event{Kind: EventRepaid, From: at, To: at, Cash: due})

	return RejectNone, 0
}

func closeCycle(s *State, batch *Batch, count FishCount, mass Micrograms, revenue Coins) {
	if count <= 0 {
		return
	}

	share := Coins(mulDivFloor(int64(batch.Cost), int64(count), int64(count)+int64(batch.Fish)))
	batch.Cost = Coins(subSat(int64(batch.Cost), int64(share)))

	kilos := int64(mass) / int64(MicrogramsPerKilogram)
	cycle := Cycle{
		Fish:    count,
		Mass:    mass,
		Revenue: revenue,
		Cost:    share,
	}

	if kilos > 0 {
		cycle.CostPerKg = Coins(mulDivFloor(int64(share), 1, kilos))
		cycle.PricePerKg = Coins(mulDivFloor(int64(revenue), 1, kilos))
	}
	if batch.MassGained > 0 {
		cycle.FCRPPM = PPM(mulDivFloor(int64(batch.FeedEaten), int64(UnitPPM), int64(batch.MassGained)))
	}

	s.LastCycle = cycle
}
