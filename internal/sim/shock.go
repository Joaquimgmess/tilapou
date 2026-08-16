package sim

const maxDiseases = 4

type DiseaseSpec struct {
	MinTemp     MilliCelsius
	MaxTemp     MilliCelsius
	OutbreakPPM PPM
	DeathPPM    PPM
	Ticks       int32
}

type ShockBalance struct {
	Diseases       [maxDiseases]DiseaseSpec
	DiseaseCount   int32
	CheckEvery     Tick
	TreatmentCost  Coins
	CarrierTicks   Tick
	CarrierRiskPPM PPM
}

func diseaseFor(b *Balance, temp MilliCelsius) (DiseaseSpec, int32, bool) {
	for i := range b.Shock.DiseaseCount {
		spec := b.Shock.Diseases[i]
		if temp >= spec.MinTemp && temp <= spec.MaxTemp && spec.OutbreakPPM > 0 {
			return spec, i, true
		}
	}

	return DiseaseSpec{}, 0, false
}

func rollDisease(s *State, b *Balance, t *Tank, temp MilliCelsius, tick Tick, sink *eventSink) {
	if b.Shock.CheckEvery <= 0 || tick%b.Shock.CheckEvery != 0 {
		return
	}

	spec, _, ok := diseaseFor(b, temp)
	if !ok {
		return
	}

	for i := range t.BatchCount {
		batch := &t.Batches[i]
		if batch.Empty() || batch.Sick > 0 {
			continue
		}

		key := RollKey{Tick: tick, Tank: t.ID, Batch: batch.ID, Purpose: PurposeDisease}
		if !s.Seed.Chance(key, riskFor(b, t, spec.OutbreakPPM, tick)) {
			continue
		}

		batch.Sick = spec.Ticks
		sink.emit(Event{
			Kind:  EventDisease,
			From:  tick,
			To:    tick + Tick(spec.Ticks),
			Tank:  t.ID,
			Batch: batch.ID,
		})
	}
}

func riskFor(b *Balance, t *Tank, base PPM, tick Tick) PPM {
	risk := crowdedRisk(b, t, base)
	if t.CarrierUntil > tick && b.Shock.CarrierRiskPPM > 0 {
		risk = PPM(min(mulDivFloor(int64(risk), int64(b.Shock.CarrierRiskPPM), int64(UnitPPM)), int64(UnitPPM)))
	}

	return risk
}

func treat(s *State, b *Balance, a Action, at Tick, sink *eventSink) (RejectReason, Coins) {
	t := s.tank(a.Tank)
	if t == nil {
		return RejectNoSuchTank, 0
	}

	sick := false
	for i := range t.BatchCount {
		if t.Batches[i].Sick > 0 {
			sick = true
		}
	}
	if !sick {
		return RejectNothingSick, 0
	}
	if s.Cash < b.Shock.TreatmentCost {
		return RejectNotEnoughCash, b.Shock.TreatmentCost
	}

	charge(s, b.Shock.TreatmentCost)
	spread(t, b.Shock.TreatmentCost)

	for i := range t.BatchCount {
		t.Batches[i].Sick = 0
	}
	t.CarrierUntil = at + b.Shock.CarrierTicks

	sink.emit(Event{Kind: EventTreated, From: at, To: at, Tank: t.ID, Cash: b.Shock.TreatmentCost})

	return RejectNone, 0
}

func crowdedRisk(b *Balance, t *Tank, base PPM) PPM {
	capacity := t.Capacity(b)
	if capacity <= 0 {
		return base
	}

	crowding := mulDivFloor(int64(t.Fish()), int64(UnitPPM), capacity)

	return PPM(min(int64(base)+mulDivFloor(int64(base), crowding, int64(UnitPPM)), int64(UnitPPM)))
}

func killByDisease(s *State, b *Balance, t *Tank, batch *Batch, tick Tick) {
	if batch.Sick <= 0 {
		return
	}

	batch.Sick--

	spec, _, ok := diseaseFor(b, seasonalTemp(b, tick, s.Zone))
	if !ok {
		return
	}

	deaths := killFish(batch, int64(spec.DeathPPM), s.Seed,
		RollKey{Tick: tick, Tank: t.ID, Batch: batch.ID, Purpose: PurposeDiseaseDeath})
	t.Accrual.DiseaseDeaths = FishCount(addSat(int64(t.Accrual.DiseaseDeaths), int64(deaths)))
}
