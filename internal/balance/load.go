// Package balance le o balance.toml embutido, em unidades humanas, e o converte
// no sim.Balance em inteiros.
package balance

import (
	"embed"
	"fmt"
	"math"

	"github.com/BurntSushi/toml"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

//go:embed balance.toml
var files embed.FS

const (
	milliScale     = 1_000
	ppmScale       = 1_000_000
	percentScale   = 100
	minutesPerHour = 60
	microsPerMilli = 1_000
)

type file struct {
	Version      uint16             `toml:"version"`
	Crescimento  crescimentoSection `toml:"crescimento"`
	Arracoamento []arracoamentoRow  `toml:"arracoamento"`
	Agua         aguaSection        `toml:"agua"`
	Mortalidade  mortalidadeSection `toml:"mortalidade"`
	Tanques      []tanqueRow        `toml:"tanques"`
	Economia     economiaSection    `toml:"economia"`
	Progressao   progressaoSection  `toml:"progressao"`
	Mercado      mercadoSection     `toml:"mercado"`
	Credito      creditoSection     `toml:"credito"`
	Choques      choquesSection     `toml:"choques"`
	Doencas      []doencaRow        `toml:"doencas"`
	Automacao    []automacaoRow     `toml:"automacao"`
}

type temperaturaRow struct {
	Graus         float64 `toml:"graus"`
	Multiplicador float64 `toml:"multiplicador"`
}

type crescimentoSection struct {
	TGC                    float64          `toml:"tgc"`
	TemperaturaReferenciaC float64          `toml:"temperatura_referencia_c"`
	PesoMaximoMg           int64            `toml:"peso_maximo_mg"`
	PesoAlevinoMg          int64            `toml:"peso_alevino_mg"`
	PesoAbateMg            int64            `toml:"peso_abate_mg"`
	Temperatura            []temperaturaRow `toml:"temperatura"`
}

type arracoamentoRow struct {
	AtePesoMg       int64   `toml:"ate_peso_mg"`
	TaxaBiomassaPct float64 `toml:"taxa_biomassa_pct"`
	TratosDia       int32   `toml:"tratos_dia"`
	ProteinaPct     int32   `toml:"proteina_pct"`
}

type aguaSection struct {
	TemperaturaBaseC           float64 `toml:"temperatura_base_c"`
	TemperaturaVariacaoDiariaC float64 `toml:"temperatura_variacao_diaria_c"`
	TemperaturaPicoHora        int32   `toml:"temperatura_pico_hora"`
	TemperaturaVariacaoAnoC    float64 `toml:"temperatura_variacao_anual_c"`
	EstacaoDias                int64   `toml:"estacao_dias"`
	EstacaoPicoDia             int64   `toml:"estacao_pico_dia"`
	ODVariacaoDiariaUgl        int64   `toml:"od_variacao_diaria_ugl"`
	ODBaseUgl                  int64   `toml:"od_base_ugl"`
	ODConsumoPorKgM3Ugl        int64   `toml:"od_consumo_por_kg_m3_ugl"`
	ODRecuperacaoAeradorUgl    int64   `toml:"od_recuperacao_aerador_ugl"`
	ODLigaAeradorUgl           int64   `toml:"od_liga_aerador_ugl"`
	ODDesligaAeradorUgl        int64   `toml:"od_desliga_aerador_ugl"`
	ODParaAlimentarMinUgl      int64   `toml:"od_para_alimentar_min_ugl"`
	ODCriticoUgl               int64   `toml:"od_critico_ugl"`
	ODLetalUgl                 int64   `toml:"od_letal_ugl"`
	ODPicoHora                 int32   `toml:"od_pico_hora"`
}

type mortalidadeSection struct {
	HorasParaLetalEmHipoxia float64 `toml:"horas_para_letal_em_hipoxia"`
	HipoxiaPerdaPorTickPct  float64 `toml:"hipoxia_perda_por_tick_pct"`
	FomeDiasAtePerda        float64 `toml:"fome_dias_ate_perda"`
	FomePerdaDiariaPct      float64 `toml:"fome_perda_diaria_pct"`
}

type tanqueRow struct {
	Tipo                  string  `toml:"tipo"`
	VolumeLitros          int64   `toml:"volume_litros"`
	DensidadeMaxPeixesM3  int64   `toml:"densidade_max_peixes_m3"`
	RenovacaoPorHoraPct   float64 `toml:"renovacao_por_hora_pct"`
	CustoBaseCentavos     int64   `toml:"custo_base_centavos"`
	ManutencaoCentavosDia int64   `toml:"manutencao_centavos_dia"`
}

type classeRow struct {
	AtePesoMg int64   `toml:"ate_peso_mg"`
	PrecoPct  float64 `toml:"preco_pct"`
}

type mercadoSection struct {
	PeixeBaseCentavosKg int64       `toml:"peixe_base_centavos_kg"`
	RacaoBaseCentavosKg int64       `toml:"racao_base_centavos_kg"`
	OscilacaoPeixePct   float64     `toml:"oscilacao_peixe_pct"`
	OscilacaoRacaoPct   float64     `toml:"oscilacao_racao_pct"`
	PeriodoDias         int64       `toml:"periodo_dias"`
	Semente             uint64      `toml:"semente"`
	EquivalenciaViavel  float64     `toml:"equivalencia_viavel"`
	Classes             []classeRow `toml:"classes"`
}

type creditoSection struct {
	LimiteCentavos int64   `toml:"limite_centavos"`
	JurosDiaPct    float64 `toml:"juros_dia_pct"`
}

type choquesSection struct {
	ChecagemDias       int64   `toml:"checagem_dias"`
	TratamentoCentavos int64   `toml:"tratamento_centavos"`
	PortadorDias       int64   `toml:"portador_dias"`
	PortadorRiscoPct   float64 `toml:"portador_risco_pct"`
}

type doencaRow struct {
	Nome              string  `toml:"nome"`
	TemperaturaMinC   float64 `toml:"temperatura_min_c"`
	TemperaturaMaxC   float64 `toml:"temperatura_max_c"`
	SurtoPct          float64 `toml:"surto_pct"`
	MortalidadeDiaPct float64 `toml:"mortalidade_dia_pct"`
	DuracaoDias       int64   `toml:"duracao_dias"`
}

type economiaSection struct {
	CustoAlevinoCentavos    int64   `toml:"custo_alevino_centavos"`
	CustoEnergiaAeradorHora int64   `toml:"custo_energia_aerador_centavos_hora"`
	CAAReferencia           float64 `toml:"caa_referencia"`
	ManutencaoPctDia        float64 `toml:"manutencao_pct_biomassa_dia"`
}

type progressaoSection struct {
	FatorCusto                  float64 `toml:"fator_custo"`
	PrestigioDivisor            int64   `toml:"prestigio_divisor"`
	PrestigioBonusPorUnidadePct float64 `toml:"prestigio_bonus_por_unidade_pct"`
	ContratoBonusPct            float64 `toml:"contrato_bonus_pct"`
	ReinicioCaixaCentavos       int64   `toml:"reinicio_caixa_centavos"`
	ReinicioPeixes              int32   `toml:"reinicio_peixes"`
	ReinicioRacaoKg             int64   `toml:"reinicio_racao_kg"`
}

type automacaoRow struct {
	Nome          string `toml:"nome"`
	CustoCentavos int64  `toml:"custo_centavos"`
}

// Load devolve o sim.Balance ja validado.
func Load() (sim.Balance, error) {
	raw, err := files.ReadFile("balance.toml")
	if err != nil {
		return sim.Balance{}, fmt.Errorf("reading embedded balance: %w", err)
	}

	var f file

	meta, err := toml.Decode(string(raw), &f)
	if err != nil {
		return sim.Balance{}, fmt.Errorf("parsing balance: %w", err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return sim.Balance{}, fmt.Errorf("%w: %v", ErrUnusedKeys, undecoded)
	}

	b, err := convert(f)
	if err != nil {
		return sim.Balance{}, err
	}
	if err := b.Validate(); err != nil {
		return sim.Balance{}, fmt.Errorf("balance is not usable: %w", err)
	}

	return b, nil
}

func convert(f file) (sim.Balance, error) {
	points := make([]sim.CurvePoint, 0, len(f.Crescimento.Temperatura))
	for _, row := range f.Crescimento.Temperatura {
		points = append(points, sim.CurvePoint{X: milli(row.Graus), Y: ppm(row.Multiplicador)})
	}

	curve, err := sim.NewCurve(points)
	if err != nil {
		return sim.Balance{}, fmt.Errorf("temperature curve: %w", err)
	}

	b := sim.Balance{
		Version: f.Version,
		Growth: sim.GrowthBalance{
			TGCPPM:         ppmOf(f.Crescimento.TGC),
			ReferenceTemp:  milliCelsius(f.Crescimento.TemperaturaReferenciaC),
			MaxMass:        micrograms(f.Crescimento.PesoMaximoMg),
			FingerlingMass: micrograms(f.Crescimento.PesoAlevinoMg),
			HarvestMass:    micrograms(f.Crescimento.PesoAbateMg),
			TempMultiplier: curve,
		},
		Water: sim.WaterBalance{
			BaseTemp:        milliCelsius(f.Agua.TemperaturaBaseC),
			DailyTempSwing:  milliCelsius(f.Agua.TemperaturaVariacaoDiariaC),
			TempPeakHour:    f.Agua.TemperaturaPicoHora,
			SeasonSwing:     milliCelsius(f.Agua.TemperaturaVariacaoAnoC),
			SeasonDays:      f.Agua.EstacaoDias,
			SeasonPeakDay:   f.Agua.EstacaoPicoDia,
			FeedingMin:      sim.MicrogramsPerLiter(f.Agua.ODParaAlimentarMinUgl),
			Critical:        sim.MicrogramsPerLiter(f.Agua.ODCriticoUgl),
			Lethal:          sim.MicrogramsPerLiter(f.Agua.ODLetalUgl),
			PeakHour:        f.Agua.ODPicoHora,
			DailySwing:      sim.MicrogramsPerLiter(f.Agua.ODVariacaoDiariaUgl),
			BaselineOxygen:  sim.MicrogramsPerLiter(f.Agua.ODBaseUgl),
			BiomassDrawPPM:  sim.PPM(f.Agua.ODConsumoPorKgM3Ugl),
			AeratorRecovery: sim.MicrogramsPerLiter(f.Agua.ODRecuperacaoAeradorUgl),
			AeratorOn:       sim.MicrogramsPerLiter(f.Agua.ODLigaAeradorUgl),
			AeratorOff:      sim.MicrogramsPerLiter(f.Agua.ODDesligaAeradorUgl),
		},
		Death: sim.MortalityBalance{
			HypoxiaTicksToLethal: int32(round(f.Mortalidade.HorasParaLetalEmHipoxia * minutesPerHour)),
			HypoxiaRatePPM:       ppmOf(f.Mortalidade.HipoxiaPerdaPorTickPct / percentScale),
			StarvationTicksGrace: int32(round(f.Mortalidade.FomeDiasAtePerda * float64(sim.TicksPerDay))),
			StarvationRatePPM:    ppmOf(f.Mortalidade.FomePerdaDiariaPct / percentScale / float64(sim.TicksPerDay)),
		},
		Economy: sim.EconomyBalance{
			FingerlingPrice: sim.Coins(f.Economia.CustoAlevinoCentavos),
			AeratorCostTick: sim.Coins(round(float64(f.Economia.CustoEnergiaAeradorHora) / minutesPerHour)),
		},
		Market: sim.MarketBalance{
			FishBasePerKg:  sim.Coins(f.Mercado.PeixeBaseCentavosKg),
			FeedBasePerKg:  sim.Coins(f.Mercado.RacaoBaseCentavosKg),
			SwingPPM:       ppmOf(f.Mercado.OscilacaoPeixePct / percentScale),
			FeedSwingPPM:   ppmOf(f.Mercado.OscilacaoRacaoPct / percentScale),
			PeriodTicks:    sim.Tick(f.Mercado.PeriodoDias) * sim.TicksPerDay,
			Seed:           sim.Seed(f.Mercado.Semente),
			ViableRatioPPM: ppmOf(f.Mercado.EquivalenciaViavel),
		},
		Credit: sim.CreditBalance{
			MaxPrincipal: sim.Coins(f.Credito.LimiteCentavos),
			DailyRatePPM: ppmOf(f.Credito.JurosDiaPct / percentScale),
		},
		Shock: sim.ShockBalance{
			CheckEvery:     sim.Tick(f.Choques.ChecagemDias) * sim.TicksPerDay,
			TreatmentCost:  sim.Coins(f.Choques.TratamentoCentavos),
			CarrierTicks:   sim.Tick(f.Choques.PortadorDias) * sim.TicksPerDay,
			CarrierRiskPPM: ppmOf(f.Choques.PortadorRiscoPct / percentScale),
		},
		Progression: sim.ProgressionBalance{
			CostFactorPPM:    ppmOf(f.Progressao.FatorCusto),
			PrestigeDivisor:  f.Progressao.PrestigioDivisor,
			PrestigeBonusPPM: ppmOf(f.Progressao.PrestigioBonusPorUnidadePct / percentScale),
			ContractBonusPPM: ppmOf(f.Progressao.ContratoBonusPct / percentScale),
			RestartCash:      sim.Coins(f.Progressao.ReinicioCaixaCentavos),
			RestartFish:      sim.FishCount(f.Progressao.ReinicioPeixes),
			RestartFeed:      sim.Micrograms(f.Progressao.ReinicioRacaoKg) * sim.MicrogramsPerKilogram,
		},
	}

	for i, row := range f.Arracoamento {
		if i >= len(b.Ration.Steps) {
			break
		}
		b.Ration.Steps[i] = sim.RationStep{
			UpToMass:    micrograms(row.AtePesoMg),
			RatePPMDay:  ppmOf(row.TaxaBiomassaPct / percentScale),
			MealsPerDay: row.TratosDia,
		}
		b.Ration.Len = int32(i + 1)
	}

	b.Ration.TargetFCRPPM = ppmOf(f.Economia.CAAReferencia)
	b.Ration.MaintenancePPM = ppmOf(f.Economia.ManutencaoPctDia / percentScale)

	for i, row := range f.Mercado.Classes {
		if i >= len(b.Market.Classes) {
			break
		}
		b.Market.Classes[i] = sim.PriceClass{
			UpToMass: micrograms(row.AtePesoMg),
			PPM:      ppmOf(row.PrecoPct / percentScale),
		}
		b.Market.ClassCount = int32(i + 1)
	}

	for i, row := range f.Doencas {
		if i >= len(b.Shock.Diseases) {
			break
		}
		b.Shock.Diseases[i] = sim.DiseaseSpec{
			MinTemp:     milliCelsius(row.TemperaturaMinC),
			MaxTemp:     milliCelsius(row.TemperaturaMaxC),
			OutbreakPPM: ppmOf(row.SurtoPct / percentScale),
			DeathPPM:    ppmOf(row.MortalidadeDiaPct / percentScale / float64(sim.TicksPerDay)),
			Ticks:       int32(row.DuracaoDias * int64(sim.TicksPerDay)),
		}
		b.Shock.DiseaseCount = int32(i + 1)
	}

	for _, row := range f.Automacao {
		kind, ok := sim.AutoKindNamed(row.Nome)
		if !ok {
			return sim.Balance{}, fmt.Errorf("%w: %q", ErrUnknownAutomation, row.Nome)
		}
		b.Automation[kind] = sim.AutomationSpec{Cost: sim.Coins(row.CustoCentavos)}
	}

	for _, row := range f.Tanques {
		kind, ok := sim.TankKindNamed(row.Tipo)
		if !ok {
			return sim.Balance{}, fmt.Errorf("%w: %q", ErrUnknownTankKind, row.Tipo)
		}
		b.Tanks[kind] = sim.TankSpec{
			MaxDensityPerM3:   row.DensidadeMaxPeixesM3,
			RenewalPPMPerHour: ppmOf(row.RenovacaoPorHoraPct / percentScale),
			BaseCost:          sim.Coins(row.CustoBaseCentavos),
			UpkeepPerDay:      sim.Coins(row.ManutencaoCentavosDia),
			Litres:            sim.Litres(row.VolumeLitros),
		}
	}

	return b, nil
}

func micrograms(milligrams int64) sim.Micrograms {
	return sim.Micrograms(milligrams * microsPerMilli)
}

func milliCelsius(v float64) sim.MilliCelsius {
	return sim.MilliCelsius(milli(v))
}

func ppmOf(v float64) sim.PPM {
	return sim.PPM(ppm(v))
}

func milli(v float64) int64 {
	return round(v * milliScale)
}

func ppm(v float64) int64 {
	return round(v * ppmScale)
}

func round(v float64) int64 {
	return int64(math.Round(v))
}
