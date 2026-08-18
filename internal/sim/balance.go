package sim

const maxRationSteps = 8

// TankSpec holds the constants of a tank kind: density in fish per cubic metre,
// renewal in PPM per hour, costs in cents and volume in litres.
type TankSpec struct {
	MaxDensityPerM3   int64
	RenewalPPMPerHour PPM
	BaseCost          Coins
	UpkeepPerDay      Coins
	Litres            Litres
	// TempFactorPPM replaces the season inside the tank, for the kinds that control the
	// water. Zero means the tank follows the weather like any pond.
	TempFactorPPM PPM
}

// GrowthBalance governs growth: masses in micrograms, temperature in thousandths of a
// degree and TempMultiplier mapping heat to a factor in PPM.
type GrowthBalance struct {
	TGCPPM         PPM
	ReferenceTemp  MilliCelsius
	MaxMass        Micrograms
	FingerlingMass Micrograms
	HarvestMass    Micrograms
	TempMultiplier Curve
}

// RationStep applies up to fish of UpToMass micrograms, with a daily rate in PPM of the biomass.
type RationStep struct {
	UpToMass    Micrograms
	RatePPMDay  PPM
	MealsPerDay int32
}

// RationBalance is the feeding table; only the first Len steps are valid, in
// increasing order of UpToMass.
type RationBalance struct {
	Steps          [maxRationSteps]RationStep
	Len            int32
	TargetFCRPPM   PPM
	MaintenancePPM PPM
}

// For falls on the last step if the mass exceeds every band, and on the zero step if the
// table is empty.
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

// WaterBalance governs the water: heat in thousandths of a degree, oxygen in micrograms
// per litre and the daily and seasonal cycles by peak hour and day.
type WaterBalance struct {
	BaseTemp        MilliCelsius
	DailyTempSwing  MilliCelsius
	TempPeakHour    int32
	SeasonSwing     MilliCelsius
	SeasonDays      int64
	SeasonPeakDay   int64
	FeedingMin      MicrogramsPerLiter
	Critical        MicrogramsPerLiter
	Lethal          MicrogramsPerLiter
	PeakHour        int32
	DailySwing      MicrogramsPerLiter
	BaselineOxygen  MicrogramsPerLiter
	BiomassDrawPPM  PPM
	AeratorRecovery MicrogramsPerLiter
	AeratorOn       MicrogramsPerLiter
	AeratorOff      MicrogramsPerLiter
}

// MortalityBalance governs deaths by hypoxia and starvation: grace in ticks and rate in PPM.
type MortalityBalance struct {
	HypoxiaTicksToLethal int32
	HypoxiaRatePPM       PPM
	StarvationTicksGrace int32
	StarvationRatePPM    PPM
}

// EconomyBalance holds the fixed prices, in cents.
type EconomyBalance struct {
	FingerlingPrice Coins
	AeratorCostTick Coins
}

// CreditBalance governs the loan: cap in cents and daily interest in PPM.
type CreditBalance struct {
	MaxPrincipal Coins
	DailyRatePPM PPM
	// BankruptcyPrincipal is the debt that ends the farm: past it the interest alone
	// outruns any cycle, so the farm is wound up instead of left with nothing to do.
	BankruptcyPrincipal Coins
}

// AutomationSpec is the cost of an automation, in cents.
type AutomationSpec struct {
	Cost Coins
}

// ProgressionBalance governs progression and restart: factors in PPM and the start in
// cents, fish and micrograms.
type ProgressionBalance struct {
	// StuckDaysToBankruptcy is how long the farm may sit with no possible action before it
	// is wound up. The trigger is the absence of a move, not the size of the debt: debt
	// grows too slowly to rescue anyone.
	StuckDaysToBankruptcy int64

	CostFactorPPM    PPM
	PrestigeDivisor  int64
	PrestigeBonusPPM PPM
	ContractBonusPPM PPM
	RestartCash      Coins
	RestartFish      FishCount
	RestartFeed      Micrograms
}

// Balance holds the constants Advance reads; read-only, and it must pass through
// Validate before use.
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

// Validate returns the ErrBalance* of the first failure found.
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

// TempMultiplier returns the growth factor in PPM, interpolated on the Growth curve.
func (b *Balance) TempMultiplier(temp MilliCelsius) PPM {
	return PPM(b.Growth.TempMultiplier.At(int64(temp)))
}
