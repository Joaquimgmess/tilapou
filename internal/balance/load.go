package balance

import (
	"embed"
	"fmt"
	"math"

	"github.com/BurntSushi/toml"

	"github.com/Joaquimgmess/catalog/internal/sim"
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
	Tempo        tempoSection       `toml:"tempo"`
	Crescimento  crescimentoSection `toml:"crescimento"`
	Arracoamento []arracoamentoRow  `toml:"arracoamento"`
	Agua         aguaSection        `toml:"agua"`
	Mortalidade  mortalidadeSection `toml:"mortalidade"`
	Tanques      []tanqueRow        `toml:"tanques"`
	Economia     economiaSection    `toml:"economia"`
	Progressao   progressaoSection  `toml:"progressao"`
	Automacao    []automacaoRow     `toml:"automacao"`
}

type tempoSection struct {
	TickSegundosDeJogo int64 `toml:"tick_segundos_de_jogo"`
	Compressao         int64 `toml:"compressao"`
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
	ODVariacaoDiariaUgl        int64   `toml:"od_variacao_diaria_ugl"`
	ODBaseUgl                  int64   `toml:"od_base_ugl"`
	ODConsumoPorKgM3Ugl        int64   `toml:"od_consumo_por_kg_m3_ugl"`
	ODRecuperacaoAeradorUgl    int64   `toml:"od_recuperacao_aerador_ugl"`
	ODIdealMinUgl              int64   `toml:"od_ideal_min_ugl"`
	ODParaAlimentarMinUgl      int64   `toml:"od_para_alimentar_min_ugl"`
	ODCriticoUgl               int64   `toml:"od_critico_ugl"`
	ODLetalUgl                 int64   `toml:"od_letal_ugl"`
	ODPicoHora                 int32   `toml:"od_pico_hora"`
	ODMinimoHora               int32   `toml:"od_minimo_hora"`
}

type mortalidadeSection struct {
	HorasParaLetalEmHipoxia float64 `toml:"horas_para_letal_em_hipoxia"`
	HipoxiaPerdaPorTickPct  float64 `toml:"hipoxia_perda_por_tick_pct"`
	FomeDiasAtePerda        float64 `toml:"fome_dias_ate_perda"`
	FomePerdaDiariaPct      float64 `toml:"fome_perda_diaria_pct"`
}

type tanqueRow struct {
	Tipo                 string  `toml:"tipo"`
	VolumeLitros         int64   `toml:"volume_litros"`
	DensidadeMaxPeixesM3 int64   `toml:"densidade_max_peixes_m3"`
	RenovacaoPorHoraPct  float64 `toml:"renovacao_por_hora_pct"`
	CustoBaseTC          int64   `toml:"custo_base_tc"`
}

type economiaSection struct {
	PrecoPeixeTCKg            int64 `toml:"preco_peixe_tc_kg"`
	PrecoRacaoTCKg            int64 `toml:"preco_racao_tc_kg"`
	CustoAlevinoTC            int64 `toml:"custo_alevino_tc"`
	CustoEnergiaAeradorTCHora int64 `toml:"custo_energia_aerador_tc_hora"`
}

type progressaoSection struct {
	FatorCusto                  float64 `toml:"fator_custo"`
	MarcosX2                    []int64 `toml:"marcos_x2"`
	PrestigioDivisor            int64   `toml:"prestigio_divisor"`
	PrestigioBonusPorUnidadePct float64 `toml:"prestigio_bonus_por_unidade_pct"`
	ContratoBonusPct            float64 `toml:"contrato_bonus_pct"`
	ReinicioCaixaTC             int64   `toml:"reinicio_caixa_tc"`
	ReinicioPeixes              int32   `toml:"reinicio_peixes"`
	ReinicioRacaoKg             int64   `toml:"reinicio_racao_kg"`
}

type automacaoRow struct {
	Nome    string `toml:"nome"`
	CustoTC int64  `toml:"custo_tc"`
	Remove  string `toml:"remove"`
}

var autoKindByName = map[string]sim.AutoKind{
	"comedouro": sim.AutoFeeder,
	"aerador":   sim.AutoAerator,
	"peao":      sim.AutoHarvester,
	"tecnico":   sim.AutoTechnician,
	"contrato":  sim.AutoContract,
}

var tankKindByName = map[string]sim.TankKind{
	"viveiro_escavado": sim.TankEarthPond,
	"tanque_rede":      sim.TankNetCage,
	"bioflocos":        sim.TankBiofloc,
	"recirculacao":     sim.TankRecirculation,
}

func Load() (sim.Balance, error) {
	raw, err := files.ReadFile("balance.toml")
	if err != nil {
		return sim.Balance{}, fmt.Errorf("reading embedded balance: %w", err)
	}

	var f file
	if err = toml.Unmarshal(raw, &f); err != nil {
		return sim.Balance{}, fmt.Errorf("parsing balance: %w", err)
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
			IdealMin:        sim.MicrogramsPerLiter(f.Agua.ODIdealMinUgl),
			FeedingMin:      sim.MicrogramsPerLiter(f.Agua.ODParaAlimentarMinUgl),
			Critical:        sim.MicrogramsPerLiter(f.Agua.ODCriticoUgl),
			Lethal:          sim.MicrogramsPerLiter(f.Agua.ODLetalUgl),
			PeakHour:        f.Agua.ODPicoHora,
			TroughHour:      f.Agua.ODMinimoHora,
			DailySwing:      sim.MicrogramsPerLiter(f.Agua.ODVariacaoDiariaUgl),
			BaselineOxygen:  sim.MicrogramsPerLiter(f.Agua.ODBaseUgl),
			BiomassDrawPPM:  sim.PPM(f.Agua.ODConsumoPorKgM3Ugl),
			AeratorRecovery: sim.MicrogramsPerLiter(f.Agua.ODRecuperacaoAeradorUgl),
		},
		Death: sim.MortalityBalance{
			HypoxiaTicksToLethal: int32(round(f.Mortalidade.HorasParaLetalEmHipoxia * minutesPerHour)),
			HypoxiaRatePPM:       ppmOf(f.Mortalidade.HipoxiaPerdaPorTickPct / percentScale),
			StarvationTicksGrace: int32(round(f.Mortalidade.FomeDiasAtePerda * float64(sim.TicksPerDay))),
			StarvationRatePPM:    ppmOf(f.Mortalidade.FomePerdaDiariaPct / percentScale / float64(sim.TicksPerDay)),
		},
		Economy: sim.EconomyBalance{
			FishPricePerKg:  sim.Coins(f.Economia.PrecoPeixeTCKg),
			FeedPricePerKg:  sim.Coins(f.Economia.PrecoRacaoTCKg),
			FingerlingPrice: sim.Coins(f.Economia.CustoAlevinoTC),
			AeratorCostTick: sim.Coins(round(float64(f.Economia.CustoEnergiaAeradorTCHora) / minutesPerHour)),
		},
		Progression: sim.ProgressionBalance{
			CostFactorPPM:    ppmOf(f.Progressao.FatorCusto),
			PrestigeDivisor:  f.Progressao.PrestigioDivisor,
			PrestigeBonusPPM: ppmOf(f.Progressao.PrestigioBonusPorUnidadePct / percentScale),
			ContractBonusPPM: ppmOf(f.Progressao.ContratoBonusPct / percentScale),
			RestartCash:      sim.Coins(f.Progressao.ReinicioCaixaTC),
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

	for _, row := range f.Automacao {
		kind, ok := autoKindByName[row.Nome]
		if !ok {
			return sim.Balance{}, fmt.Errorf("%w: %q", ErrUnknownAutomation, row.Nome)
		}
		b.Automation[kind] = sim.AutomationSpec{Cost: sim.Coins(row.CustoTC)}
	}

	for _, row := range f.Tanques {
		kind, ok := tankKindByName[row.Tipo]
		if !ok {
			return sim.Balance{}, fmt.Errorf("%w: %q", ErrUnknownTankKind, row.Tipo)
		}
		b.Tanks[kind] = sim.TankSpec{
			MaxDensityPerM3:   row.DensidadeMaxPeixesM3,
			RenewalPPMPerHour: ppmOf(row.RenovacaoPorHoraPct / percentScale),
			BaseCost:          sim.Coins(row.CustoBaseTC),
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
