package sim

const technicianBonusPPM = 100_000

const (
	rootScale    = 1000
	rootCubeUnit = rootScale * rootScale * rootScale
)

func massRootOf(mass Micrograms) int64 {
	return icbrt(mulDivFloor(int64(mass), rootCubeUnit, 1))
}

func massFromRoot(root int64) Micrograms {
	return Micrograms(mulDivFloor(root*root, root, rootCubeUnit))
}

func step(s *State, b *Balance, tick Tick, sink *eventSink) {
	temp := temperatureAt(b, tick, s.Zone)
	tempMult := b.TempMultiplier(temp)

	for i := range s.TankCount {
		t := &s.Tanks[i]
		t.Oxygen = oxygenAt(b, t, tick, s.Zone)
		automate(s, b, t, tick, sink)
		payEnergy(s, b, t)

		oxygen := oxygenAt(b, t, tick, s.Zone)
		t.Oxygen = oxygen

		feeding := oxygen >= b.Water.FeedingMin && tempMult > 0 && tick <= t.ServedUntil

		for j := range t.BatchCount {
			batch := &t.Batches[j]
			if batch.Empty() {
				continue
			}

			eaten, wanted := feedBatch(t, batch, b, tempMult, feeding)
			growBatch(batch, b, tempMult, eaten, wanted, s.prestigeBonus(b), t.Owns(AutoTechnician))
			killByHypoxia(t, batch, b, oxygen, tick, s.Seed)
			killByStarvation(t, batch, b, eaten, tick, s.Seed)
		}

		if t.FeedStock <= 0 && feeding && t.Fish() > 0 && tick%WindowTicks == 0 {
			sink.emit(Event{Kind: EventFeedExhausted, From: tick, To: tick, Tank: t.ID})
		}

		t.compact()
	}
}

func feedBatch(t *Tank, batch *Batch, b *Balance, tempMult PPM, feeding bool) (eatenMass, wantedMass Micrograms) {
	if !feeding {
		return 0, 0
	}

	step := b.Ration.For(batch.MeanMass)
	if step.RatePPMDay <= 0 {
		return 0, 0
	}

	daily := mulDivFloor(int64(batch.Biomass()), int64(step.RatePPMDay), int64(UnitPPM))
	daily = mulDivFloor(daily, int64(tempMult), int64(UnitPPM))

	wanted := carryTake(daily, int64(TicksPerDay), &t.FeedCarry)
	if wanted <= 0 {
		return 0, 0
	}

	eaten := min(wanted, int64(t.FeedStock))
	if eaten <= 0 {
		return 0, Micrograms(wanted)
	}

	t.FeedStock = Micrograms(subSat(int64(t.FeedStock), eaten))
	batch.FeedEaten = Micrograms(addSat(int64(batch.FeedEaten), eaten))
	t.Accrual.FeedEaten = Micrograms(addSat(int64(t.Accrual.FeedEaten), eaten))

	return Micrograms(eaten), Micrograms(wanted)
}

func growBatch(batch *Batch, b *Balance, tempMult PPM, eaten, wanted Micrograms, bonus PPM, technician bool) {
	if eaten <= 0 || wanted <= 0 || tempMult <= 0 {
		return
	}

	if batch.MassRoot <= 0 {
		batch.MassRoot = massRootOf(batch.MeanMass)
	}

	dailyRoot := mulDivFloor(int64(b.Growth.ReferenceTemp)/1000*int64(b.Growth.TGCPPM), 1, 10)
	dailyRoot = mulDivFloor(dailyRoot, int64(tempMult), int64(UnitPPM))

	headroom := int64(UnitPPM) - mulDivFloor(int64(batch.MeanMass), int64(UnitPPM), int64(b.Growth.MaxMass))
	if headroom <= 0 {
		return
	}
	dailyRoot = mulDivFloor(dailyRoot, headroom, int64(UnitPPM))

	dailyRoot = mulDivFloor(dailyRoot, int64(bonus), int64(UnitPPM))
	if technician {
		dailyRoot = mulDivFloor(dailyRoot, int64(UnitPPM)+technicianBonusPPM, int64(UnitPPM))
	}

	adequacy := mulDivFloor(int64(eaten), int64(UnitPPM), int64(wanted))
	dailyRoot = mulDivFloor(dailyRoot, min(adequacy, int64(UnitPPM)), int64(UnitPPM))

	delta := carryTake(dailyRoot, int64(TicksPerDay), &batch.GrowthCarry)
	if delta <= 0 {
		return
	}

	before := batch.MeanMass
	batch.MassRoot += delta
	after := massFromRoot(batch.MassRoot)
	if after <= before {
		return
	}

	batch.MeanMass = after
	gained := Micrograms(mulDivFloor(int64(after-before), int64(batch.Fish), 1))
	batch.MassGained = Micrograms(addSat(int64(batch.MassGained), int64(gained)))
}

func killByHypoxia(t *Tank, batch *Batch, b *Balance, oxygen MicrogramsPerLiter, tick Tick, seed Seed) {
	if oxygen >= b.Water.Critical {
		if batch.HypoxiaTicks > 0 {
			batch.HypoxiaTicks--
		}
		return
	}

	batch.HypoxiaTicks++
	if batch.HypoxiaTicks < b.Death.HypoxiaTicksToLethal {
		return
	}

	severity := int64(b.Water.Critical - oxygen)
	rate := mulDivFloor(int64(b.Death.HypoxiaRatePPM), severity, int64(b.Water.Critical))
	if oxygen <= b.Water.Lethal {
		rate = int64(b.Death.HypoxiaRatePPM)
	}

	deaths := killFish(batch, rate, seed, RollKey{Tick: tick, Tank: t.ID, Batch: batch.ID, Purpose: PurposeMortality})
	t.Accrual.HypoxiaDeaths = FishCount(addSat(int64(t.Accrual.HypoxiaDeaths), int64(deaths)))
}

func killByStarvation(t *Tank, batch *Batch, b *Balance, eaten Micrograms, tick Tick, seed Seed) {
	if eaten > 0 {
		batch.StarvationTicks = 0
		return
	}

	batch.StarvationTicks++
	if batch.StarvationTicks < b.Death.StarvationTicksGrace {
		return
	}

	deaths := killFish(batch, int64(b.Death.StarvationRatePPM), seed,
		RollKey{Tick: tick, Tank: t.ID, Batch: batch.ID, Purpose: PurposeStarvation})
	t.Accrual.StarvationDeaths = FishCount(addSat(int64(t.Accrual.StarvationDeaths), int64(deaths)))
}

func killFish(batch *Batch, ratePPM int64, seed Seed, key RollKey) FishCount {
	if ratePPM <= 0 || batch.Fish <= 0 {
		return 0
	}

	expected := mulDivFloor(int64(batch.Fish), ratePPM, int64(UnitPPM))
	remainder := mulDivFloor(int64(batch.Fish), ratePPM, 1) - expected*int64(UnitPPM)

	if remainder > 0 && key.Purpose != PurposeUnknown && seed.RollBelow(key, int64(UnitPPM)) < remainder {
		expected++
	}
	if expected <= 0 {
		return 0
	}

	deaths := FishCount(min(expected, int64(batch.Fish)))
	batch.Fish -= deaths

	return deaths
}

func MassRootOf(mass Micrograms) int64 {
	return massRootOf(mass)
}
