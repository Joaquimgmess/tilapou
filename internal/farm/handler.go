package farm

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const (
	seriesPoints   = 21
	microsPerMilli = 1_000
	gramsPerKilo   = 1_000
)

// BatchView is one batch inside a tank: money in cents, weight in grams, ratios in PPM.
// A tank holds up to sim.MaxBatchesPerTank of them, and every one is a decision of its own.
type BatchView struct {
	ID             uint32       `json:"batch_id"`
	Fish           int32        `json:"fish"`
	MeanGrams      int64        `json:"mean_grams"`
	Ready          bool         `json:"ready_to_harvest"`
	Sick           bool         `json:"sick"`
	PriceKgCents   int64        `json:"price_kg_cents"`
	ValueCents     int64        `json:"value_cents"`
	CostCents      int64        `json:"cost_cents"`
	MarginCents    int64        `json:"margin_cents"`
	CostPerKg      int64        `json:"cost_per_kg_cents"`
	ClassPPM       int64        `json:"class_ppm"`
	NextClassGrams int64        `json:"next_class_grams"`
	NextClassGain  int64        `json:"next_class_gain_ppm"`
	Decision       DecisionView `json:"decision"`
}

// TankView is the tank in the API: money in cents, weight in grams, oxygen in
// micrograms per litre, density in thousandths of kg/m3 and ratios in PPM. What belongs to
// a batch lives in Batches, so a tank with four of them does not hide three.
type TankView struct {
	ID           uint32 `json:"id"`
	Kind         string `json:"kind"`
	Fish         int32  `json:"fish"`
	FeedKg       int64  `json:"feed_kg"`
	OxygenUgL    int32  `json:"oxygen_ugl"`
	Aerating     bool   `json:"aerating"`
	DensityMilli int64  `json:"density_milli_kg_m3"`

	Batches     []BatchView   `json:"batches"`
	Capacity    int64         `json:"capacity_fish"`
	StockAdvice int64         `json:"stock_advice_fish"`
	BatchCount  int32         `json:"batch_count"`
	MaxBatches  int32         `json:"max_batches"`
	BreakEven   int64         `json:"break_even_fish"`
	CostPerFish int64         `json:"stock_cost_per_fish_cents"`
	LoanAdvice  int64         `json:"loan_advice_cents"`
	LoanBlock   string        `json:"loan_block"`
	ServedFor   int64         `json:"served_for_ticks"`
	Upgrades    []UpgradeView `json:"upgrades"`
}

// EventView is the event in the API, with mass in grams and cash in cents.
type EventView struct {
	Seq       uint64 `json:"seq"`
	Kind      string `json:"kind"`
	From      int64  `json:"from_tick"`
	To        int64  `json:"to_tick"`
	Tank      uint32 `json:"tank_id"`
	Fish      int32  `json:"fish"`
	MassGrams int64  `json:"mass_grams"`
	CashCents int64  `json:"cash_cents"`
	Reason    string `json:"reason"`
}

// UpgradeView carries the automation cost in cents.
type UpgradeView struct {
	Kind      string `json:"kind"`
	Owned     bool   `json:"owned"`
	CostCents int64  `json:"cost_cents"`
}

// OutcomeView carries the reason for the refusal and the cash that was missing, in cents.
type OutcomeView struct {
	Applied    bool   `json:"applied"`
	Reason     string `json:"reason"`
	NeededCash int64  `json:"needed_cents"`
}

// PriceView carries the current tick's prices in cents and the feed-to-fish exchange in PPM.
type PriceView struct {
	FeedKgCents     int64 `json:"feed_kg_cents"`
	FingerlingCents int64 `json:"fingerling_cents"`
	FishKgCents     int64 `json:"fish_kg_cents"`
	RatioPPM        int64 `json:"equivalence_ppm"`
	ViablePPM       int64 `json:"equivalence_viable_ppm"`
}

// CycleView closes the last cycle: mass in grams, values in cents and feed
// conversion in PPM.
type CycleView struct {
	Fish         int32 `json:"fish"`
	MassGrams    int64 `json:"mass_grams"`
	RevenueCents int64 `json:"revenue_cents"`
	CostCents    int64 `json:"cost_cents"`
	MarginCents  int64 `json:"margin_cents"`
	CostPerKg    int64 `json:"cost_per_kg_cents"`
	PricePerKg   int64 `json:"price_per_kg_cents"`
	FCRPPM       int64 `json:"fcr_ppm"`
}

// DecisionView compares selling now with holding to the next weight class:
// cents, daily gain in milligrams, daily feed in grams.
type DecisionView struct {
	SellNowCents  int64 `json:"sell_now_cents"`
	SellNowMargin int64 `json:"sell_now_margin_cents"`
	HoldToGrams   int64 `json:"hold_to_grams"`
	HoldDays      int64 `json:"hold_days"`
	HoldCents     int64 `json:"hold_cents"`
	HoldMargin    int64 `json:"hold_margin_cents"`
	HoldCostCents int64 `json:"hold_cost_cents"`
	// HoldReached diz que a projecao alcanca o alvo dentro do teto de dias, e nao que o
	// peixe ja chegou la: com 30 g e alvo de 400 g ela e verdadeira e correta.
	HoldReached    bool  `json:"hold_reached"`
	BreakEvenPerKg int64 `json:"break_even_per_kg_cents"`
	GainPerDayMg   int64 `json:"gain_per_day_mg"`
	FeedPerDayG    int64 `json:"feed_per_day_grams"`
	CostPerDay     int64 `json:"cost_per_day_cents"`
	DaysOfFeed     int64 `json:"days_of_feed"`
	CycleDays      int64 `json:"cycle_days"`
}

// SeriesView carries prices in cents per kilo, one point every StepTicks.
type SeriesView struct {
	FishKgCents []int64 `json:"fish_kg_cents"`
	FeedKgCents []int64 `json:"feed_kg_cents"`
	StepTicks   int64   `json:"step_ticks"`
}

// SnapshotView is the farm in the API: cents, grams and thousandths of a degree, with
// RunwayDays at -1 when there is no daily cost.
type SnapshotView struct {
	FarmID        string       `json:"farm_id"`
	Name          string       `json:"name"`
	Tick          int64        `json:"tick"`
	Hour          int32        `json:"hour"`
	TempMilliC    int32        `json:"temp_milli_c"`
	CashCents     int64        `json:"cash_cents"`
	LifetimeCents int64        `json:"lifetime_cents"`
	BiomassGrams  int64        `json:"biomass_grams"`
	Fish          int32        `json:"fish"`
	Prestige      uint32       `json:"prestige"`
	Tanks         []TankView   `json:"tanks"`
	PrestigeNow   uint32       `json:"prestige_available"`
	NextTankCents int64        `json:"next_tank_cents"`
	Prices        PriceView    `json:"prices"`
	Debt          int64        `json:"debt_cents"`
	LastCycle     CycleView    `json:"last_cycle"`
	Series        SeriesView   `json:"series"`
	InterestDay   int64        `json:"interest_per_day_cents"`
	RunwayDays    int64        `json:"runway_days"`
	Broke         bool         `json:"broke"`
	Events        []EventView  `json:"events"`
	LastOutcome   *OutcomeView `json:"last_outcome,omitempty"`
}

type snapshotOutput struct {
	Body SnapshotView
}

type tankKindName string

func (tankKindName) Schema(huma.Registry) *huma.Schema {
	names := sim.TankKindNames()
	allowed := make([]any, len(names))

	for i, name := range names {
		allowed[i] = name
	}

	return &huma.Schema{Type: "string", Description: "Tipo de tanque a comprar", Enum: allowed}
}

type actionBody struct {
	Key      uint64       `doc:"Chave de idempotencia da acao"   json:"key"`
	Kind     string       `doc:"Acao a executar"                 enum:"feed,buy_feed,aerate,harvest,stock,buy_tank,buy_upgrade,treat,prestige,restart,borrow,repay" json:"kind"`
	Tank     uint32       `doc:"Tanque alvo"                     json:"tank_id,omitempty"`
	Batch    uint32       `doc:"Lote alvo"                       json:"batch_id,omitempty"`
	TankKind tankKindName `json:"tank_kind,omitempty"`
	Auto     string       `doc:"Automacao a comprar"             enum:"comedouro,aerador,peao,tecnico,contrato"                                                     json:"auto,omitempty"`
	Amount   int64        `doc:"Quantidade, quando a acao pedir" json:"amount,omitempty"`
}

type actionInput struct {
	Body actionBody
}

// RegisterRoutes publishes GET /farm and POST /farm/actions, which return the
// already advanced snapshot.
func RegisterRoutes(api huma.API, sessions *Sessions, player uuid.UUID, b *sim.Balance) {
	// O mesmo cache que a sessao usa para adiantar a simulacao: dois caches pagariam duas
	// vezes a primeira simulacao de cada dia.
	p := sessions.plans

	huma.Register(api, huma.Operation{
		OperationID: "get-farm",
		Method:      http.MethodGet,
		Path:        "/farm",
		Summary:     "Estado atual da fazenda",
		Tags:        []string{"farm"},
	}, func(ctx context.Context, _ *struct{}) (*snapshotOutput, error) {
		snap, err := sessions.Sync(ctx, player)
		if err != nil {
			return nil, reportError(ctx, "get-farm", err)
		}

		return &snapshotOutput{Body: viewOf(snap, b, p)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "act-on-farm",
		Method:        http.MethodPost,
		Path:          "/farm/actions",
		Summary:       "Executa uma acao na fazenda",
		Tags:          []string{"farm"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *actionInput) (*snapshotOutput, error) {
		action, err := actionOf(in.Body)
		if err != nil {
			return nil, reportError(ctx, in.Body.Kind, err)
		}

		snap, err := sessions.Act(ctx, player, action)
		if err != nil {
			return nil, reportError(ctx, in.Body.Kind, err)
		}

		return &snapshotOutput{Body: viewOf(snap, b, p)}, nil
	})
}

func actionOf(body actionBody) (sim.Action, error) {
	kind, ok := sim.ActionKindNamed(body.Kind)
	if !ok {
		return sim.Action{}, ErrUnknownAction
	}

	action := sim.Action{
		ID:     sim.ActionID(body.Key),
		Kind:   kind,
		Tank:   sim.TankID(body.Tank),
		Batch:  sim.BatchID(body.Batch),
		Amount: body.Amount,
	}

	if kind == sim.ActionBuyUpgrade {
		auto, ok := sim.AutoKindNamed(body.Auto)
		if !ok {
			return sim.Action{}, ErrMissingAuto
		}
		action.Auto = auto
	}

	if kind == sim.ActionBuyTank {
		tankKind, ok := sim.TankKindNamed(string(body.TankKind))
		if !ok {
			return sim.Action{}, ErrMissingTankKind
		}
		action.TankKind = tankKind
	}

	if body.Tank == 0 && needsTank(kind) {
		return sim.Action{}, ErrMissingTank
	}

	return action, nil
}

func needsTank(kind sim.ActionKind) bool {
	switch kind {
	case sim.ActionFeed, sim.ActionBuyFeed, sim.ActionAerate, sim.ActionHarvest,
		sim.ActionStock, sim.ActionBuyUpgrade, sim.ActionTreat:
		return true
	case sim.ActionBuyTank, sim.ActionPrestige, sim.ActionRestart, sim.ActionBorrow, sim.ActionRepay, sim.ActionUnknown:
		return false
	}

	return false
}

func viewOf(snap Snapshot, b *sim.Balance, p *plans) SnapshotView {
	state := &snap.Farm.State
	market := sim.MarketAt(b, state.Tick)

	view := SnapshotView{
		FarmID:        snap.Farm.ID.String(),
		Name:          snap.Farm.Name,
		Tick:          int64(snap.Projection.Tick),
		Hour:          snap.Projection.Tick.At(state.Zone).Hour,
		TempMilliC:    int32(snap.Temp),
		CashCents:     int64(snap.Projection.Cash),
		LifetimeCents: int64(snap.Projection.Lifetime),
		BiomassGrams:  snap.Projection.Biomass.Grams(),
		Fish:          int32(snap.Projection.Fish),
		Prestige:      snap.Projection.Prestige,
		Tanks:         make([]TankView, 0, state.TankCount),
		PrestigeNow:   sim.PrestigePointsFor(state.LifetimeEarned, b.Progression.PrestigeDivisor),
		NextTankCents: int64(state.NextTankCost(b, sim.TankEarthPond)),
		Series:        seriesOf(state, b),
		InterestDay:   int64(state.Debt) * int64(b.Credit.DailyRatePPM) / int64(sim.UnitPPM),
		RunwayDays:    runwayDays(state, b),
		Broke:         state.Broke(b),
		Prices: PriceView{
			FeedKgCents:     int64(market.FeedKg),
			FingerlingCents: int64(b.Economy.FingerlingPrice),
			FishKgCents:     int64(market.FishKg),
			RatioPPM:        int64(market.RatioPPM),
			ViablePPM:       int64(b.Market.ViableRatioPPM),
		},
		Debt: int64(state.Debt),
		LastCycle: CycleView{
			Fish:         int32(state.LastCycle.Fish),
			MassGrams:    state.LastCycle.Mass.Grams(),
			RevenueCents: int64(state.LastCycle.Revenue),
			CostCents:    int64(state.LastCycle.Cost),
			MarginCents:  int64(state.LastCycle.Margin()),
			CostPerKg:    int64(state.LastCycle.CostPerKg),
			PricePerKg:   int64(state.LastCycle.PricePerKg),
			FCRPPM:       int64(state.LastCycle.FCRPPM),
		},
		Events: make([]EventView, 0, len(snap.Events)),
	}

	if snap.Outcome != nil {
		view.LastOutcome = &OutcomeView{
			Applied:    snap.Outcome.Applied,
			Reason:     snap.Outcome.Reason.String(),
			NeededCash: int64(snap.Outcome.Needed),
		}
	}

	for i := range state.TankCount {
		tank := &state.Tanks[i]
		plan := p.at(b, tank.Kind, state.Tick, state.Zone)
		fish, cost := state.StockAdvice(b, tank.ID, plan)
		advice, perFish := int64(fish), int64(cost)
		loan, block := state.LoanAdvice(b, tank.ID, plan)
		tv := TankView{
			ID:          uint32(tank.ID),
			Kind:        tank.Kind.String(),
			Fish:        int32(tank.Fish()),
			FeedKg:      int64(tank.FeedStock / sim.MicrogramsPerKilogram),
			OxygenUgL:   int32(tank.Oxygen),
			Aerating:    tank.Aerating,
			Capacity:    tank.Capacity(b),
			StockAdvice: advice,
			BatchCount:  tank.BatchCount,
			MaxBatches:  sim.MaxBatchesPerTank,
			CostPerFish: perFish,
			BreakEven:   int64(plan.BreakEven),
			LoanAdvice:  int64(loan),
			LoanBlock:   block.String(),
			ServedFor:   int64(tank.ServedUntil - state.Tick),
			Upgrades:    upgradesOf(tank, b),
		}
		if tank.Litres > 0 {
			tv.DensityMilli = int64(tank.Biomass()) / (sim.LitresPerCubicMetre * int64(tank.Litres))
		}
		for j := range tank.BatchCount {
			tv.Batches = append(tv.Batches, batchViewOf(state, b, tank, &tank.Batches[j], p))
		}

		view.Tanks = append(view.Tanks, tv)
	}

	for _, e := range snap.Events {
		view.Events = append(view.Events, EventView{
			Seq:       e.Seq,
			Kind:      e.Kind,
			From:      int64(e.From),
			To:        int64(e.To),
			Tank:      uint32(e.Tank),
			Fish:      int32(e.Fish),
			MassGrams: e.Mass.Grams(),
			CashCents: int64(e.Cash),
			Reason:    e.Reason,
		})
	}

	return view
}

func batchViewOf(state *sim.State, b *sim.Balance, tank *sim.Tank, batch *sim.Batch, p *plans) BatchView {
	bv := BatchView{
		ID:           uint32(batch.ID),
		Fish:         int32(batch.Fish),
		MeanGrams:    batch.MeanMass.Grams(),
		Ready:        batch.MeanMass >= b.Growth.HarvestMass,
		Sick:         batch.Sick > 0,
		PriceKgCents: int64(b.PriceFor(batch.MeanMass, state.Tick)),
		ClassPPM:     int64(b.ClassPPM(batch.MeanMass)),
		CostCents:    int64(batch.Cost),
	}

	if entry, gain, ok := b.NextClass(batch.MeanMass); ok {
		bv.NextClassGrams, bv.NextClassGain = entry.Grams(), int64(gain)
	}

	kilos := int64(batch.Biomass()) / int64(sim.MicrogramsPerKilogram)
	bv.ValueCents = bv.PriceKgCents * kilos
	bv.MarginCents = bv.ValueCents - bv.CostCents

	if kilos > 0 {
		bv.CostPerKg = bv.CostCents / kilos
	}

	in := newDecisionInput(state, tank, batch)

	bv.Decision = p.decision(in, func() DecisionView { return decisionFor(b, in) })
	bv.Decision.SellNowCents = bv.ValueCents
	bv.Decision.SellNowMargin = bv.MarginCents
	bv.Decision.BreakEvenPerKg = bv.CostPerKg
	bv.Decision.HoldMargin = bv.Decision.HoldCents - bv.CostCents - bv.Decision.HoldCostCents
	bv.Decision.CycleDays = p.at(b, tank.Kind, state.Tick, state.Zone).Days

	return bv
}

// decisionFor projects the batch; the sell-now cents are filled by the caller on every
// request, because they follow the price of the tick and must not be cached by day.
func decisionFor(b *sim.Balance, in decisionInput) DecisionView {
	var view DecisionView

	tank := in.tank
	// newDecisionInput quantiza o lote da chave para a posicao 0: e o unico lote que o
	// input carrega, e o cache depende disso para nao misturar lotes.
	batch := tank.Batches[0]
	feedKg := int64(tank.FeedStock) / int64(sim.MicrogramsPerKilogram)

	ahead := in.forecast(b, batch.MeanMass+sim.MicrogramsPerGram)
	if ahead.Days > 0 {
		view.GainPerDayMg = int64(ahead.MeanMass-batch.MeanMass) / ahead.Days / microsPerMilli
		view.CostPerDay = int64(ahead.Cost) / ahead.Days
		view.FeedPerDayG = int64(ahead.FeedEaten) / ahead.Days / microsPerMilli / microsPerMilli
	}
	if view.FeedPerDayG > 0 {
		view.DaysOfFeed = feedKg * gramsPerKilo / view.FeedPerDayG
	}

	target, _, ok := b.NextClass(batch.MeanMass)
	if !ok {
		return view
	}

	hold := in.forecast(b, target)
	view.HoldToGrams = target.Grams()
	view.HoldDays = hold.Days
	view.HoldCents = int64(hold.Value)
	view.HoldCostCents = int64(hold.Cost)
	view.HoldReached = hold.Reached

	return view
}

func seriesOf(state *sim.State, b *sim.Balance) SeriesView {
	fish, feed := state.Series(b, seriesPoints, sim.TicksPerDay)

	view := SeriesView{
		FishKgCents: make([]int64, 0, len(fish)),
		FeedKgCents: make([]int64, 0, len(feed)),
		StepTicks:   int64(sim.TicksPerDay),
	}
	for i := range fish {
		view.FishKgCents = append(view.FishKgCents, int64(fish[i]))
		view.FeedKgCents = append(view.FeedKgCents, int64(feed[i]))
	}

	return view
}

func runwayDays(state *sim.State, b *sim.Balance) int64 {
	daily := int64(state.Debt) * int64(b.Credit.DailyRatePPM) / int64(sim.UnitPPM)

	for i := range state.TankCount {
		tank := &state.Tanks[i]
		daily += int64(b.Tanks[tank.Kind].UpkeepPerDay)
	}
	if daily <= 0 {
		return -1
	}

	return int64(state.Cash) / daily
}

var upgradeOrder = [5]sim.AutoKind{
	sim.AutoFeeder,
	sim.AutoAerator,
	sim.AutoHarvester,
	sim.AutoTechnician,
	sim.AutoContract,
}

var _ [len(upgradeOrder) - sim.AutoKindCount]struct{}

func upgradesOf(tank *sim.Tank, b *sim.Balance) []UpgradeView {
	views := make([]UpgradeView, 0, len(upgradeOrder))
	for _, kind := range upgradeOrder {
		views = append(views, UpgradeView{
			Kind:      kind.String(),
			Owned:     tank.Owns(kind),
			CostCents: int64(b.Automation[kind].Cost),
		})
	}

	return views
}

func reportError(ctx context.Context, operation string, err error) error {
	if converted := toHTTPError(err); converted != nil {
		return converted
	}

	logging.FromContext(ctx).ErrorContext(ctx, "farm request failed",
		slog.String("operation", operation), slog.Any("error", err))

	return huma.Error500InternalServerError("erro interno")
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return huma.Error404NotFound("fazenda nao encontrada")
	case errors.Is(err, ErrUnknownAction), errors.Is(err, ErrMissingAuto),
		errors.Is(err, ErrMissingTankKind), errors.Is(err, ErrMissingTank):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, ErrStaleRevision):
		return huma.Error409Conflict("a fazenda mudou durante a escrita, tente de novo")
	default:
		return nil
	}
}
