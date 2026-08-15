package save

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Joaquimgmess/catalog/internal/sim"
)

var (
	ErrUnknownVersion = errors.New("save: unknown state version")
	ErrTooManyTanks   = errors.New("save: more tanks than the state can hold")
	ErrTooManyBatches = errors.New("save: more batches than a tank can hold")
)

type document struct {
	Version        uint16         `json:"version"`
	BalanceVersion uint16         `json:"balance_version"`
	RngVersion     uint16         `json:"rng_version"`
	Seed           uint64         `json:"seed"`
	Zone           int32          `json:"zone"`
	Tick           int64          `json:"tick"`
	Cash           int64          `json:"cash"`
	LifetimeEarned int64          `json:"lifetime_earned"`
	Prestige       uint32         `json:"prestige"`
	Upgrades       uint32         `json:"upgrades"`
	NextTankID     uint32         `json:"next_tank_id"`
	NextBatchID    uint32         `json:"next_batch_id"`
	EventSeq       uint64         `json:"event_seq"`
	Saturated      bool           `json:"saturated"`
	Tanks          []tankDocument `json:"tanks"`
}

type tankDocument struct {
	ID        uint32          `json:"id"`
	Kind      uint8           `json:"kind"`
	Litres    int64           `json:"litres"`
	FeedStock int64           `json:"feed_stock"`
	Oxygen    int32           `json:"oxygen"`
	Aerating  bool            `json:"aerating"`
	FeedCarry int64           `json:"feed_carry"`
	Accrual   accrualDocument `json:"accrual"`
	Batches   []batchDocument `json:"batches"`
}

type accrualDocument struct {
	Window           int64 `json:"window"`
	HypoxiaDeaths    int32 `json:"hypoxia_deaths"`
	StarvationDeaths int32 `json:"starvation_deaths"`
	FeedEaten        int64 `json:"feed_eaten"`
	MassGained       int64 `json:"mass_gained"`
}

type batchDocument struct {
	ID              uint32 `json:"id"`
	Fish            int32  `json:"fish"`
	MeanMass        int64  `json:"mean_mass"`
	GrowthCarry     int64  `json:"growth_carry"`
	StockedAt       int64  `json:"stocked_at"`
	FeedEaten       int64  `json:"feed_eaten"`
	MassGained      int64  `json:"mass_gained"`
	HypoxiaTicks    int32  `json:"hypoxia_ticks"`
	StarvationTicks int32  `json:"starvation_ticks"`
}

func Encode(s sim.State) ([]byte, error) {
	doc := document{
		Version:        s.Version,
		BalanceVersion: s.BalanceVersion,
		RngVersion:     s.RngVersion,
		Seed:           uint64(s.Seed),
		Zone:           int32(s.Zone),
		Tick:           int64(s.Tick),
		Cash:           int64(s.Cash),
		LifetimeEarned: int64(s.LifetimeEarned),
		Prestige:       s.Prestige,
		Upgrades:       s.Upgrades,
		NextTankID:     uint32(s.NextTankID),
		NextBatchID:    uint32(s.NextBatchID),
		EventSeq:       s.EventSeq,
		Saturated:      s.Saturated,
		Tanks:          make([]tankDocument, 0, s.TankCount),
	}

	for i := range s.TankCount {
		t := s.Tanks[i]
		tank := tankDocument{
			ID:        uint32(t.ID),
			Kind:      uint8(t.Kind),
			Litres:    int64(t.Litres),
			FeedStock: int64(t.FeedStock),
			Oxygen:    int32(t.Oxygen),
			Aerating:  t.Aerating,
			FeedCarry: t.FeedCarry,
			Accrual: accrualDocument{
				Window:           int64(t.Accrual.Window),
				HypoxiaDeaths:    int32(t.Accrual.HypoxiaDeaths),
				StarvationDeaths: int32(t.Accrual.StarvationDeaths),
				FeedEaten:        int64(t.Accrual.FeedEaten),
				MassGained:       int64(t.Accrual.MassGained),
			},
			Batches: make([]batchDocument, 0, t.BatchCount),
		}

		for j := range t.BatchCount {
			batch := t.Batches[j]
			tank.Batches = append(tank.Batches, batchDocument{
				ID:              uint32(batch.ID),
				Fish:            int32(batch.Fish),
				MeanMass:        int64(batch.MeanMass),
				GrowthCarry:     batch.GrowthCarry,
				StockedAt:       int64(batch.StockedAt),
				FeedEaten:       int64(batch.FeedEaten),
				MassGained:      int64(batch.MassGained),
				HypoxiaTicks:    batch.HypoxiaTicks,
				StarvationTicks: batch.StarvationTicks,
			})
		}

		doc.Tanks = append(doc.Tanks, tank)
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("save: encoding state: %w", err)
	}

	return raw, nil
}

func Decode(raw []byte) (sim.State, error) {
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return sim.State{}, fmt.Errorf("save: decoding state: %w", err)
	}
	if doc.Version != sim.StateVersion {
		return sim.State{}, fmt.Errorf("%w: %d", ErrUnknownVersion, doc.Version)
	}

	state := sim.NewState(sim.Seed(doc.Seed), sim.ZoneOffset(doc.Zone), sim.Tick(doc.Tick))
	state.BalanceVersion = doc.BalanceVersion
	state.RngVersion = doc.RngVersion
	state.Cash = sim.Coins(doc.Cash)
	state.LifetimeEarned = sim.Coins(doc.LifetimeEarned)
	state.Prestige = doc.Prestige
	state.Upgrades = doc.Upgrades
	state.NextTankID = sim.TankID(doc.NextTankID)
	state.NextBatchID = sim.BatchID(doc.NextBatchID)
	state.EventSeq = doc.EventSeq
	state.Saturated = doc.Saturated

	if len(doc.Tanks) > len(state.Tanks) {
		return sim.State{}, ErrTooManyTanks
	}

	for i, tank := range doc.Tanks {
		if len(tank.Batches) > len(state.Tanks[i].Batches) {
			return sim.State{}, ErrTooManyBatches
		}

		state.Tanks[i] = sim.Tank{
			ID:        sim.TankID(tank.ID),
			Kind:      sim.TankKind(tank.Kind),
			Litres:    sim.Litres(tank.Litres),
			FeedStock: sim.Micrograms(tank.FeedStock),
			Oxygen:    sim.MicrogramsPerLiter(tank.Oxygen),
			Aerating:  tank.Aerating,
			FeedCarry: tank.FeedCarry,
			Accrual: sim.Accrual{
				Window:           sim.Tick(tank.Accrual.Window),
				HypoxiaDeaths:    sim.FishCount(tank.Accrual.HypoxiaDeaths),
				StarvationDeaths: sim.FishCount(tank.Accrual.StarvationDeaths),
				FeedEaten:        sim.Micrograms(tank.Accrual.FeedEaten),
				MassGained:       sim.Micrograms(tank.Accrual.MassGained),
			},
			BatchCount: int32(len(tank.Batches)),
		}

		for j, batch := range tank.Batches {
			state.Tanks[i].Batches[j] = sim.Batch{
				ID:              sim.BatchID(batch.ID),
				Fish:            sim.FishCount(batch.Fish),
				MeanMass:        sim.Micrograms(batch.MeanMass),
				MassRoot:        sim.MassRootOf(sim.Micrograms(batch.MeanMass)),
				GrowthCarry:     batch.GrowthCarry,
				StockedAt:       sim.Tick(batch.StockedAt),
				FeedEaten:       sim.Micrograms(batch.FeedEaten),
				MassGained:      sim.Micrograms(batch.MassGained),
				HypoxiaTicks:    batch.HypoxiaTicks,
				StarvationTicks: batch.StarvationTicks,
			}
		}
	}
	state.TankCount = int32(len(doc.Tanks))

	return state, nil
}
