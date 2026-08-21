package farm

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const (
	seriesPoints   = 21
	microsPerMilli = 1_000
	gramsPerKilo   = 1_000
)

type snapshotOutput struct {
	Body api.Snapshot
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
	Key      uint64       `doc:"Chave de idempotencia da acao"    json:"key"`
	Kind     string       `doc:"Acao a executar"                  enum:"feed,buy_feed,aerate,harvest,stock,buy_tank,buy_upgrade,treat,prestige,restart,borrow,repay" json:"kind"`
	Tank     uint32       `doc:"Tanque alvo"                      json:"tank_id,omitempty"`
	Batch    uint32       `doc:"Lote alvo"                        json:"batch_id,omitempty"`
	TankKind tankKindName `json:"tank_kind,omitempty"`
	Auto     string       `doc:"Automacao a comprar"              enum:"comedouro,aerador,peao,tecnico,contrato"                                                     json:"auto,omitempty"`
	Amount   int64        `doc:"Quantidade, quando a acao pedir"  json:"amount,omitempty"`
	SeenTick int64        `doc:"Tick que o jogador tinha na tela" json:"seen_tick,omitempty"`
}

type actionInput struct {
	Body actionBody
}

// RegisterRoutes publishes GET /farm and POST /farm/actions, which return the
// already advanced snapshot.
func RegisterRoutes(router huma.API, sessions *Sessions, player uuid.UUID, b *sim.Balance) {
	// O mesmo cache que a sessao usa para adiantar a simulacao: dois caches pagariam duas
	// vezes a primeira simulacao de cada dia.
	p := sessions.plans

	huma.Register(router, huma.Operation{
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

	huma.Register(router, huma.Operation{
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
		SeenAt: sim.Tick(body.SeenTick),
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

func viewOf(snap Snapshot, b *sim.Balance, p *plans) api.Snapshot {
	state := &snap.Farm.State
	market := sim.MarketAt(b, state.Tick)

	view := api.Snapshot{
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
		Tanks:         make([]api.Tank, 0, state.TankCount),
		PrestigeNow:   sim.PrestigePointsFor(state.LifetimeEarned, b.Progression.PrestigeDivisor),
		NextTankCents: int64(state.NextTankCost(b, sim.TankEarthPond)),
		Series:        seriesOf(state, b),
		InterestDay:   int64(state.Debt) * int64(b.Credit.DailyRatePPM) / int64(sim.UnitPPM),
		RunwayDays:    runwayDays(state, b),
		Broke:         state.Broke(b, p.forFarm(b, state)),
		Prices: api.Prices{
			FeedKgCents:     int64(market.FeedKg),
			FingerlingCents: int64(b.Economy.FingerlingPrice),
			FishKgCents:     int64(market.FishKg),
			RatioPPM:        int64(market.RatioPPM),
			ViablePPM:       int64(b.Market.ViableRatioPPM),
		},
		Debt: int64(state.Debt),
		LastCycle: api.Cycle{
			Fish:         int32(state.LastCycle.Fish),
			MassGrams:    state.LastCycle.Mass.Grams(),
			RevenueCents: int64(state.LastCycle.Revenue),
			CostCents:    int64(state.LastCycle.Cost),
			MarginCents:  int64(state.LastCycle.Margin()),
			CostPerKg:    int64(state.LastCycle.CostPerKg),
			PricePerKg:   int64(state.LastCycle.PricePerKg),
			FCRPPM:       int64(state.LastCycle.FCRPPM),
		},
		Events: make([]api.Event, 0, len(snap.Events)),
	}

	if snap.Outcome != nil {
		view.LastOutcome = &api.Outcome{
			Applied:    snap.Outcome.Applied,
			Reason:     snap.Outcome.Reason.String(),
			NeededCash: int64(snap.Outcome.Needed),
		}
	}

	for i := range state.TankCount {
		tank := &state.Tanks[i]
		plan := p.at(b, tank.Kind, state.Tick, state.Zone)
		stock := state.StockAdvice(b, tank.ID, plan)
		advice := int64(stock.Fish)
		loan := state.LoanAdvice(b, tank.ID, plan)
		tv := api.Tank{
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
			BreakEven:   int64(plan.BreakEven),
			LoanAdvice:  int64(loan.Cents),
			LoanFish:    int64(loan.Fish),
			LoanOwed:    int64(sim.OwedOn(b, plan, loan.Cents)),
			CycleDays:   plan.Days,
			CycleMargin: int64(plan.Margin),
			LoanBlock:   loanBlockAPI[loan.Block],
			StockBlock:  stockBlockAPI[stock.Block],
			StockShort:  int64(stock.Short),
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
		view.Events = append(view.Events, api.Event{
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

func batchViewOf(state *sim.State, b *sim.Balance, tank *sim.Tank, batch *sim.Batch, p *plans) api.Batch {
	bv := api.Batch{
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

	bv.Decision = p.decision(in, func() api.Decision { return decisionFor(b, in) })
	bv.Decision.SellNowCents = bv.ValueCents
	bv.Decision.SellNowMargin = bv.MarginCents
	bv.Decision.BreakEvenPerKg = bv.CostPerKg
	bv.Decision.HoldMargin = bv.Decision.HoldCents - bv.CostCents - bv.Decision.HoldCostCents
	bv.Decision.CycleDays = p.at(b, tank.Kind, state.Tick, state.Zone).Days

	return bv
}

// decisionFor projects the batch; the sell-now cents are filled by the caller on every
// request, because they follow the price of the tick and must not be cached by day.
func decisionFor(b *sim.Balance, in decisionInput) api.Decision {
	var view api.Decision

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

func seriesOf(state *sim.State, b *sim.Balance) api.Series {
	fish, feed := state.Series(b, seriesPoints, sim.TicksPerDay)

	view := api.Series{
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

// loanBlockAPI e stockBlockAPI convertem os enums do sim para o contrato. A tabela indexada
// pelo enum, e nao o String(), e o que faz o compilador cobrar o motivo novo: valor sem
// entrada aqui derruba a sentinela abaixo, e nao vira string desconhecida na tela.
var loanBlockAPI = [...]api.LoanBlock{
	sim.LoanOpen:     api.LoanOpen,
	sim.LoanNoCredit: api.LoanNoCredit,
	sim.LoanNoRoom:   api.LoanNoRoom,
	sim.LoanNoNeed:   api.LoanNoNeed,
	sim.LoanNoCycle:  api.LoanNoCycle,
}

var _ [len(loanBlockAPI) - int(sim.LoanBlockCount)]struct{}

var stockBlockAPI = [...]api.StockBlock{
	sim.StockOpen:    api.StockOpen,
	sim.StockNoTank:  api.StockNoTank,
	sim.StockNoRoom:  api.StockNoRoom,
	sim.StockNoBatch: api.StockNoBatch,
	sim.StockNoCash:  api.StockNoCash,
	sim.StockNoCycle: api.StockNoCycle,
}

var _ [len(stockBlockAPI) - int(sim.StockBlockCount)]struct{}

func upgradesOf(tank *sim.Tank, b *sim.Balance) []api.Upgrade {
	views := make([]api.Upgrade, 0, len(upgradeOrder))
	for _, kind := range upgradeOrder {
		views = append(views, api.Upgrade{
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
