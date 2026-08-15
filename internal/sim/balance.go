package sim

const maxRationSteps = 8

type TankSpec struct {
	MaxDensityPerM3   int64
	RenewalPPMPerHour PPM
	BaseCost          Coins
	Litres            Litres
}

type GrowthBalance struct {
	TGCPPM         PPM
	ReferenceTemp  MilliCelsius
	MaxMass        Micrograms
	FingerlingMass Micrograms
	HarvestMass    Micrograms
	TempMultiplier Curve
}

type RationStep struct {
	UpToMass    Micrograms
	RatePPMDay  PPM
	MealsPerDay int32
}

type RationBalance struct {
	Steps [maxRationSteps]RationStep
	Len   int32
}

func (r RationBalance) For(mass Micrograms) RationStep {
	for i := range r.Len {
		if mass <= r.Steps[i].UpToMass {
			return r.Steps[i]
		}
	}
	if r.Len == 0 {
		return RationStep{}
	}

	return r.Steps[r.Len-1]
}

type WaterBalance struct {
	BaseTemp        MilliCelsius
	DailyTempSwing  MilliCelsius
	TempPeakHour    int32
	IdealMin        MicrogramsPerLiter
	FeedingMin      MicrogramsPerLiter
	Critical        MicrogramsPerLiter
	Lethal          MicrogramsPerLiter
	PeakHour        int32
	TroughHour      int32
	DailySwing      MicrogramsPerLiter
	BaselineOxygen  MicrogramsPerLiter
	BiomassDrawPPM  PPM
	AeratorRecovery MicrogramsPerLiter
}

type MortalityBalance struct {
	HypoxiaTicksToLethal int32
	HypoxiaRatePPM       PPM
	StarvationTicksGrace int32
	StarvationRatePPM    PPM
}

type EconomyBalance struct {
	FishPricePerKg  Coins
	FeedPricePerKg  Coins
	FingerlingPrice Coins
	AeratorCostTick Coins
}

type Balance struct {
	Version uint16
	Growth  GrowthBalance
	Ration  RationBalance
	Water   WaterBalance
	Death   MortalityBalance
	Economy EconomyBalance
	Tanks   [tankKindCount]TankSpec
}

func (b Balance) Validate() error {
	if b.Version == 0 {
		return ErrBalanceUnversioned
	}
	if b.Growth.TempMultiplier.Len == 0 {
		return ErrBalanceNoTempCurve
	}
	if b.Growth.MaxMass <= 0 || b.Growth.HarvestMass <= 0 {
		return ErrBalanceNoMass
	}
	if b.Ration.Len == 0 {
		return ErrBalanceNoRation
	}
	for kind := range tankKindCount {
		if b.Tanks[kind].Litres <= 0 || b.Tanks[kind].MaxDensityPerM3 <= 0 {
			return ErrBalanceTankSlotEmpty
		}
	}

	return nil
}

func (b Balance) TempMultiplier(temp MilliCelsius) PPM {
	return PPM(b.Growth.TempMultiplier.At(int64(temp)))
}
