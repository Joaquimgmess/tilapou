package sim

const technicianBonusPPM = 100_000

const (
	rootScale    = 1000
	rootCubeUnit = rootScale * rootScale * rootScale
)

// MassRootOf returns the cube root of the mass in thousandths.
func MassRootOf(mass Micrograms) int64 {
	return icbrt(mulDivFloor(int64(mass), rootCubeUnit, 1))
}

func massFromRoot(root int64) Micrograms {
	return Micrograms(mulDivFloor(root*root, root, rootCubeUnit))
}

func step(s *State, b *Balance, tick Tick, sink *eventSink) {
	accrueInterest(s, b)

	if bankrupt(s, b, tick, sink) {
		return
	}

	temp := TemperatureAt(b, tick, s.Zone)
	tempMult := b.TempMultiplier(temp)

	for i := range s.TankCount {
		t := &s.Tanks[i]
		t.Oxygen = oxygenAt(b, t, tick, s.Zone)
		automate(s, b, t, tick, sink)
		payEnergy(s, b, t)
		chargeUpkeep(s, b, t)

		oxygen := oxygenAt(b, t, tick, s.Zone)
		t.Oxygen = oxygen

		rollDisease(s, b, t, seasonalTemp(b, tick, s.Zone), tick, sink)

		feeding := oxygen >= b.Water.FeedingMin && tempMult > 0 && tick <= t.ServedUntil

		for j := range t.BatchCount {
			batch := &t.Batches[j]
			if batch.Empty() {
				continue
			}

			var eaten Micrograms
			if feeding {
				eaten = feedAndGrow(t, batch, b, tempMult, s.prestigeBonus(b))
			}

			killByHypoxia(t, batch, b, oxygen, tick, s.Seed)
			killByDisease(s, b, t, batch, tick)
			killByStarvation(t, batch, b, eaten, tick, s.Seed)
		}

		if t.FeedStock <= 0 && feeding && t.Fish() > 0 && tick%WindowTicks == 0 {
			sink.emit(Event{Kind: EventFeedExhausted, From: tick, To: tick, Tank: t.ID})
		}

		t.compact()
	}
}

func feedAndGrow(t *Tank, batch *Batch, b *Balance, tempMult, bonus PPM) Micrograms {
	maintenance := carryTake(
		mulDivFloor(int64(batch.Biomass()), int64(b.Ration.MaintenancePPM), int64(UnitPPM)),
		int64(TicksPerDay), &t.FeedCarry)

	delta := growthDelta(batch, b, tempMult, bonus, t.Owns(AutoTechnician))
	gain := gainFor(batch, delta)

	wanted := addSat(maintenance, mulDivCeil(int64(gain), int64(b.Ration.TargetFCRPPM), int64(UnitPPM)))
	wanted = min(wanted, rationCap(batch, b, tempMult))
	if wanted <= 0 {
		return 0
	}

	eaten := min(wanted, int64(t.FeedStock))
	if eaten <= 0 {
		return 0
	}

	t.FeedStock = Micrograms(subSat(int64(t.FeedStock), eaten))
	batch.FeedEaten = Micrograms(addSat(int64(batch.FeedEaten), eaten))
	batch.Cost = Coins(addSat(int64(batch.Cost),
		carryTake(eaten*int64(t.FeedUnitCost), int64(MicrogramsPerKilogram), &batch.CostCarry)))
	t.Accrual.FeedEaten = Micrograms(addSat(int64(t.Accrual.FeedEaten), eaten))

	forGrowth := subSat(eaten, maintenance)
	available := subSat(wanted, maintenance)
	if forGrowth <= 0 || available <= 0 {
		batch.GrowthCarry = addSat(batch.GrowthCarry, delta)

		return Micrograms(eaten)
	}
	if forGrowth < available {
		delta = mulDivFloor(delta, forGrowth, available)
	}

	applyGrowth(batch, delta)

	return Micrograms(eaten)
}

func rationCap(batch *Batch, b *Balance, tempMult PPM) int64 {
	ration := b.Ration.For(batch.MeanMass)
	if ration.RatePPMDay <= 0 {
		return 0
	}

	daily := mulDivFloor(int64(batch.Biomass()), int64(ration.RatePPMDay), int64(UnitPPM))
	daily = mulDivFloor(daily, int64(tempMult), int64(UnitPPM))

	return daily / int64(TicksPerDay)
}

func growthDelta(batch *Batch, b *Balance, tempMult, bonus PPM, technician bool) int64 {
	if tempMult <= 0 {
		return 0
	}

	if batch.MassRoot <= 0 {
		batch.MassRoot = MassRootOf(batch.MeanMass)
	}

	dailyRoot := mulDivFloor(int64(b.Growth.ReferenceTemp)/1000*int64(b.Growth.TGCPPM), 1, 10)
	dailyRoot = mulDivFloor(dailyRoot, int64(tempMult), int64(UnitPPM))

	headroom := int64(UnitPPM) - mulDivFloor(int64(batch.MeanMass), int64(UnitPPM), int64(b.Growth.MaxMass))
	if headroom <= 0 {
		return 0
	}
	dailyRoot = mulDivFloor(dailyRoot, headroom, int64(UnitPPM))
	dailyRoot = mulDivFloor(dailyRoot, int64(bonus), int64(UnitPPM))

	if technician {
		dailyRoot = mulDivFloor(dailyRoot, int64(UnitPPM)+technicianBonusPPM, int64(UnitPPM))
	}

	return carryTake(dailyRoot, int64(TicksPerDay), &batch.GrowthCarry)
}

func gainFor(batch *Batch, delta int64) Micrograms {
	if delta <= 0 {
		return 0
	}

	after := massFromRoot(batch.MassRoot + delta)
	if after <= batch.MeanMass {
		return 0
	}

	return Micrograms(mulDivFloor(int64(after-batch.MeanMass), int64(batch.Fish), 1))
}

func applyGrowth(batch *Batch, delta int64) {
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
