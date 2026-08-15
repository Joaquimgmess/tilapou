package farm

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const litresPerCubicMetre = 1_000

type TankView struct {
	ID        uint32 `json:"id"`
	Kind      string `json:"kind"`
	Fish      int32  `json:"fish"`
	MeanGrams int64  `json:"mean_grams"`
	FeedKg    int64  `json:"feed_kg"`
	OxygenUgL int32  `json:"oxygen_ugl"`
	Aerating  bool   `json:"aerating"`
	DensityKg int64  `json:"density_kg_m3"`
	Ready     bool   `json:"ready_to_harvest"`
	BatchID   uint32 `json:"batch_id"`

	Capacity  int64         `json:"capacity_fish"`
	ServedFor int64         `json:"served_for_ticks"`
	Upgrades  []UpgradeView `json:"upgrades"`
}

type EventView struct {
	Seq    uint64 `json:"seq"`
	Kind   string `json:"kind"`
	From   int64  `json:"from_tick"`
	To     int64  `json:"to_tick"`
	Tank   uint32 `json:"tank_id"`
	Fish   int32  `json:"fish"`
	MassG  int64  `json:"mass_grams"`
	CashTC int64  `json:"cash_cents"`
	Reason string `json:"reason"`
}

type UpgradeView struct {
	Kind      string `json:"kind"`
	Owned     bool   `json:"owned"`
	CostCents int64  `json:"cost_cents"`
}

type OutcomeView struct {
	Applied    bool   `json:"applied"`
	Reason     string `json:"reason"`
	NeededCash int64  `json:"needed_cents"`
}

type PriceView struct {
	FeedKgCents     int64 `json:"feed_kg_cents"`
	FingerlingCents int64 `json:"fingerling_cents"`
	FishKgCents     int64 `json:"fish_kg_cents"`
}

type SnapshotView struct {
	FarmID      string       `json:"farm_id"`
	Name        string       `json:"name"`
	Tick        int64        `json:"tick"`
	Hour        int32        `json:"hour"`
	TempMilliC  int32        `json:"temp_milli_c"`
	CashCents   int64        `json:"cash_cents"`
	LifetimeTC  int64        `json:"lifetime_cents"`
	BiomassG    int64        `json:"biomass_grams"`
	Fish        int32        `json:"fish"`
	Prestige    uint32       `json:"prestige"`
	Tanks       []TankView   `json:"tanks"`
	PrestigeNow uint32       `json:"prestige_available"`
	Prices      PriceView    `json:"prices"`
	Events      []EventView  `json:"events"`
	LastApplied bool         `json:"last_action_applied"`
	LastReason  string       `json:"last_action_reason"`
	LastOutcome *OutcomeView `json:"last_outcome,omitempty"`
}

type snapshotOutput struct {
	Body SnapshotView
}

type actionBody struct {
	Key      uint64 `doc:"Chave de idempotencia da acao"   json:"key"`
	Kind     string `doc:"Acao a executar"                 enum:"feed,buy_feed,aerate,harvest,stock,buy_tank,buy_upgrade,prestige" json:"kind"`
	Tank     uint32 `doc:"Tanque alvo"                     json:"tank_id,omitempty"`
	Batch    uint32 `doc:"Lote alvo"                       json:"batch_id,omitempty"`
	TankKind string `doc:"Tipo de tanque a comprar"        enum:"viveiro_escavado,tanque_rede,bioflocos,recirculacao"              json:"tank_kind,omitempty"`
	Auto     string `doc:"Automacao a comprar"             enum:"comedouro,aerador,peao,tecnico,contrato"                          json:"auto,omitempty"`
	Amount   int64  `doc:"Quantidade, quando a acao pedir" json:"amount,omitempty"`
}

type actionInput struct {
	Body actionBody
}

var actionKindByName = map[string]sim.ActionKind{
	"feed":        sim.ActionFeed,
	"buy_tank":    sim.ActionBuyTank,
	"stock":       sim.ActionStock,
	"buy_feed":    sim.ActionBuyFeed,
	"aerate":      sim.ActionAerate,
	"harvest":     sim.ActionHarvest,
	"buy_upgrade": sim.ActionBuyUpgrade,
	"prestige":    sim.ActionPrestige,
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

var tankKindNames = map[sim.TankKind]string{
	sim.TankEarthPond:     "viveiro_escavado",
	sim.TankNetCage:       "tanque_rede",
	sim.TankBiofloc:       "bioflocos",
	sim.TankRecirculation: "recirculacao",
}

func RegisterRoutes(api huma.API, sessions *Sessions, player uuid.UUID, b *sim.Balance) {
	huma.Register(api, huma.Operation{
		OperationID: "get-farm",
		Method:      http.MethodGet,
		Path:        "/farm",
		Summary:     "Estado atual da fazenda",
		Tags:        []string{"farm"},
	}, func(ctx context.Context, _ *struct{}) (*snapshotOutput, error) {
		snap, err := sessions.Sync(ctx, player)
		if err != nil {
			return nil, toHTTPError(err)
		}

		return &snapshotOutput{Body: viewOf(snap, b)}, nil
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
			return nil, toHTTPError(err)
		}

		snap, err := sessions.Act(ctx, player, action)
		if err != nil {
			return nil, toHTTPError(err)
		}

		return &snapshotOutput{Body: viewOf(snap, b)}, nil
	})
}

func actionOf(body actionBody) (sim.Action, error) {
	kind, ok := actionKindByName[body.Kind]
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
		auto, ok := autoKindByName[body.Auto]
		if !ok {
			return sim.Action{}, ErrMissingAuto
		}
		action.Auto = auto
	}

	if kind == sim.ActionBuyTank {
		tankKind, ok := tankKindByName[body.TankKind]
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
		sim.ActionStock, sim.ActionBuyUpgrade:
		return true
	case sim.ActionBuyTank, sim.ActionPrestige, sim.ActionUnknown:
		return false
	}

	return false
}

func viewOf(snap Snapshot, b *sim.Balance) SnapshotView {
	state := &snap.Farm.State

	view := SnapshotView{
		FarmID:      snap.Farm.ID.String(),
		Name:        snap.Farm.Name,
		Tick:        int64(snap.Projection.Tick),
		Hour:        snap.Projection.Tick.At(state.Zone).Hour,
		TempMilliC:  int32(snap.Temp),
		CashCents:   int64(snap.Projection.Cash),
		LifetimeTC:  int64(snap.Projection.Lifetime),
		BiomassG:    snap.Projection.Biomass.Grams(),
		Fish:        int32(snap.Projection.Fish),
		Prestige:    snap.Projection.Prestige,
		Tanks:       make([]TankView, 0, state.TankCount),
		PrestigeNow: sim.PrestigePointsFor(state.LifetimeEarned, b.Progression.PrestigeDivisor),
		Prices: PriceView{
			FeedKgCents:     int64(b.Economy.FeedPricePerKg),
			FingerlingCents: int64(b.Economy.FingerlingPrice),
			FishKgCents:     int64(b.Economy.FishPricePerKg),
		},
		Events: make([]EventView, 0, len(snap.Events)),
	}

	if snap.Outcome != nil {
		view.LastApplied = snap.Outcome.Applied
		view.LastReason = snap.Outcome.Reason.String()
		view.LastOutcome = &OutcomeView{
			Applied:    snap.Outcome.Applied,
			Reason:     snap.Outcome.Reason.String(),
			NeededCash: int64(snap.Outcome.Needed),
		}
	}

	for i := range state.TankCount {
		tank := &state.Tanks[i]
		tv := TankView{
			ID:        uint32(tank.ID),
			Kind:      tankKindNames[tank.Kind],
			Fish:      int32(tank.Fish()),
			FeedKg:    int64(tank.FeedStock / sim.MicrogramsPerKilogram),
			OxygenUgL: int32(tank.Oxygen),
			Aerating:  tank.Aerating,
			Capacity:  tank.Capacity(b),
			ServedFor: int64(tank.ServedUntil - state.Tick),
			Upgrades:  upgradesOf(tank, b),
		}
		if tank.Litres > 0 {
			tv.DensityKg = int64(tank.Biomass()) / (litresPerCubicMetre * int64(tank.Litres))
		}
		if tank.BatchCount > 0 {
			batch := &tank.Batches[0]
			tv.MeanGrams = batch.MeanMass.Grams()
			tv.BatchID = uint32(batch.ID)
			tv.Ready = batch.MeanMass >= b.Growth.HarvestMass
		}
		view.Tanks = append(view.Tanks, tv)
	}

	for _, e := range snap.Events {
		view.Events = append(view.Events, EventView{
			Seq:    e.Seq,
			Kind:   e.Kind,
			From:   int64(e.From),
			To:     int64(e.To),
			Tank:   uint32(e.Tank),
			Fish:   int32(e.Fish),
			MassG:  e.Mass.Grams(),
			CashTC: int64(e.Cash),
			Reason: e.Reason,
		})
	}

	return view
}

var upgradeOrder = []sim.AutoKind{
	sim.AutoFeeder,
	sim.AutoAerator,
	sim.AutoHarvester,
	sim.AutoTechnician,
	sim.AutoContract,
}

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

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return huma.Error404NotFound("fazenda nao encontrada")
	case errors.Is(err, ErrUnknownAction), errors.Is(err, ErrBadAmount),
		errors.Is(err, ErrMissingAuto), errors.Is(err, ErrMissingTankKind), errors.Is(err, ErrMissingTank):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, ErrStaleRevision):
		return huma.Error409Conflict("a fazenda mudou durante a escrita, tente de novo")
	default:
		return err
	}
}
