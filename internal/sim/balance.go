package sim

const maxRationSteps = 8

// TankSpec sao as constantes de um tipo de tanque: densidade em peixes por metro
// cubico, renovacao em PPM por hora, custos em centavos e volume em litros.
type TankSpec struct {
	MaxDensityPerM3   int64
	RenewalPPMPerHour PPM
	BaseCost          Coins
	UpkeepPerDay      Coins
	Litres            Litres
}

// GrowthBalance rege o crescimento: massas em microgramas, temperatura em milesimos
// de grau e TempMultiplier mapeando calor para um fator em PPM.
type GrowthBalance struct {
	TGCPPM         PPM
	ReferenceTemp  MilliCelsius
	MaxMass        Micrograms
	FingerlingMass Micrograms
	HarvestMass    Micrograms
	TempMultiplier Curve
}

// RationStep vale ate peixes de UpToMass microgramas, com taxa diaria em PPM da biomassa.
type RationStep struct {
	UpToMass    Micrograms
	RatePPMDay  PPM
	MealsPerDay int32
}

// RationBalance e a tabela de arracoamento; so os primeiros Len passos valem, em
// ordem crescente de UpToMass.
type RationBalance struct {
	Steps          [maxRationSteps]RationStep
	Len            int32
	TargetFCRPPM   PPM
	MaintenancePPM PPM
}

// For cai no ultimo passo se a massa passar de todas as faixas, e no passo zero se a
// tabela estiver vazia.
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

// WaterBalance rege a agua: calor em milesimos de grau, oxigenio em microgramas por
// litro e os ciclos diario e sazonal por hora e dia de pico.
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

// MortalityBalance rege mortes por hipoxia e fome: carencia em ticks e taxa em PPM.
type MortalityBalance struct {
	HypoxiaTicksToLethal int32
	HypoxiaRatePPM       PPM
	StarvationTicksGrace int32
	StarvationRatePPM    PPM
}

// EconomyBalance sao os precos fixos, em centavos.
type EconomyBalance struct {
	FingerlingPrice Coins
	AeratorCostTick Coins
}

// CreditBalance rege o emprestimo: teto em centavos e juros diarios em PPM.
type CreditBalance struct {
	MaxPrincipal Coins
	DailyRatePPM PPM
}

// AutomationSpec e o custo de uma automacao, em centavos.
type AutomationSpec struct {
	Cost Coins
}

// ProgressionBalance rege progressao e recomeco: fatores em PPM e a partida em
// centavos, peixes e microgramas.
type ProgressionBalance struct {
	CostFactorPPM    PPM
	PrestigeDivisor  int64
	PrestigeBonusPPM PPM
	ContractBonusPPM PPM
	RestartCash      Coins
	RestartFish      FishCount
	RestartFeed      Micrograms
}

// Balance sao as constantes que Advance consulta; so leitura, e precisa passar por
// Validate antes do uso.
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

// Validate devolve o ErrBalance* da primeira falha encontrada.
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

// TempMultiplier devolve o fator de crescimento em PPM, interpolado na curva de Growth.
func (b *Balance) TempMultiplier(temp MilliCelsius) PPM {
	return PPM(b.Growth.TempMultiplier.At(int64(temp)))
}
