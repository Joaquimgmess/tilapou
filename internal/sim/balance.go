package sim

const maxRationSteps = 8

type TankSpec struct {
	MaxDensityPerM3   int64
	RenewalPPMPerHour PPM
	BaseCost          Coins
	UpkeepPerDay      Coins
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
	Steps          [maxRationSteps]RationStep
	Len            int32
	TargetFCRPPM   PPM
	MaintenancePPM PPM
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
	SeasonSwing     MilliCelsius
	SeasonDays      int64
	SeasonPeakDay   int64
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
	AeratorOn       MicrogramsPerLiter
	AeratorOff      MicrogramsPerLiter
}

type MortalityBalance struct {
	HypoxiaTicksToLethal int32
	HypoxiaRatePPM       PPM
	StarvationTicksGrace int32
	StarvationRatePPM    PPM
}

type EconomyBalance struct {
	FingerlingPrice Coins
	AeratorCostTick Coins
}

type CreditBalance struct {
	MaxPrincipal Coins
	DailyRatePPM PPM
}

type AutomationSpec struct {
	Cost Coins
}

type ProgressionBalance struct {
	CostFactorPPM    PPM
	PrestigeDivisor  int64
	PrestigeBonusPPM PPM
	ContractBonusPPM PPM
	RestartCash      Coins
	RestartFish      FishCount
	RestartFeed      Micrograms
}

type Balance struct {
	Version     uint16
	Growth      GrowthBalance
	Ration      RationBalance
	Water       WaterBalance
	Death       MortalityBalance
	Economy     EconomyBalance
	Market      MarketBalance
	Credit      CreditBalance
	Shock       ShockBalance
	Progression ProgressionBalance
	Tanks       [tankKindCount]TankSpec
	Automation  [autoKindCount]AutomationSpec
}

func (b *Balance) Validate() error {
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
	if b.Market.FishBasePerKg <= 0 || b.Market.FeedBasePerKg <= 0 || b.Market.PeriodTicks <= 0 {
		return ErrBalanceNoMarket
	}
	for kind := range tankKindCount {
		if b.Tanks[kind].Litres <= 0 || b.Tanks[kind].MaxDensityPerM3 <= 0 {
			return ErrBalanceTankSlotEmpty
		}
	}
	for kind := range autoKindCount {
		if b.Automation[kind].Cost <= 0 {
			return ErrBalanceAutomationSlotEmpty
		}
	}

	return nil
}

func (b *Balance) TempMultiplier(temp MilliCelsius) PPM {
	return PPM(b.Growth.TempMultiplier.At(int64(temp)))
}
