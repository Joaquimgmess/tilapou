// Package save serializa o sim.State em JSON e o le de volta, recusando save
// que o estado atual nao comporta.
package save

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

// Erros de leitura do save, todos devolvidos por Decode.
var (
	ErrUnknownVersion = errors.New("save: unknown state version")
	ErrTooManyTanks   = errors.New("save: more tanks than the state can hold")
	ErrTooManyBatches = errors.New("save: more batches than a tank can hold")
	ErrUnknownKind    = errors.New("save: tank kind outside the enum")
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
	Upgrades       uint32         `json:"upgrades,omitempty"`
	NextTankID     uint32         `json:"next_tank_id"`
	NextBatchID    uint32         `json:"next_batch_id"`
	EventSeq       uint64         `json:"event_seq"`
	Debt           int64          `json:"debt"`
	DebtCarry      int64          `json:"debt_carry"`
	LastCycle      cycleDocument  `json:"last_cycle,omitzero"`
	Tanks          []tankDocument `json:"tanks"`
}

type cycleDocument struct {
	Fish       int32 `json:"fish,omitempty"`
	Mass       int64 `json:"mass,omitempty"`
	Revenue    int64 `json:"revenue,omitempty"`
	Cost       int64 `json:"cost,omitempty"`
	CostPerKg  int64 `json:"cost_per_kg,omitempty"`
	PricePerKg int64 `json:"price_per_kg,omitempty"`
	FCRPPM     int64 `json:"fcr_ppm,omitempty"`
}

type tankDocument struct {
	ID           uint32          `json:"id"`
	Kind         uint8           `json:"kind"`
	Litres       int64           `json:"litres"`
	FeedStock    int64           `json:"feed_stock"`
	ServedUntil  int64           `json:"served_until"`
	Upgrades     uint32          `json:"upgrades"`
	Oxygen       int32           `json:"oxygen"`
	Aerating     bool            `json:"aerating"`
	FeedCarry    int64           `json:"feed_carry"`
	FeedUnitCost int64           `json:"feed_unit_cost,omitempty"`
	UpkeepCarry  int64           `json:"upkeep_carry"`
	CarrierUntil int64           `json:"carrier_until"`
	Accrual      accrualDocument `json:"accrual"`
	Batches      []batchDocument `json:"batches"`
}

type accrualDocument struct {
	Window           int64 `json:"window"`
	HypoxiaDeaths    int32 `json:"hypoxia_deaths"`
	DiseaseDeaths    int32 `json:"disease_deaths"`
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
	Cost            int64  `json:"cost"`
	CostCarry       int64  `json:"cost_carry,omitempty"`
	Sick            int32  `json:"sick"`
	HypoxiaTicks    int32  `json:"hypoxia_ticks"`
	StarvationTicks int32  `json:"starvation_ticks"`
}

// Encode serializa o estado em JSON, gravando so os tanques e lotes em uso.
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
		NextTankID:     uint32(s.NextTankID),
		NextBatchID:    uint32(s.NextBatchID),
		EventSeq:       s.EventSeq,
		Debt:           int64(s.Debt),
		DebtCarry:      s.DebtCarry,
		LastCycle: cycleDocument{
			Fish:       int32(s.LastCycle.Fish),
			Mass:       int64(s.LastCycle.Mass),
			Revenue:    int64(s.LastCycle.Revenue),
			Cost:       int64(s.LastCycle.Cost),
			CostPerKg:  int64(s.LastCycle.CostPerKg),
			PricePerKg: int64(s.LastCycle.PricePerKg),
			FCRPPM:     int64(s.LastCycle.FCRPPM),
		},
		Tanks: make([]tankDocument, 0, s.TankCount),
	}

	for i := range s.TankCount {
		t := s.Tanks[i]
		tank := tankDocument{
			ID:           uint32(t.ID),
			Kind:         uint8(t.Kind),
			Litres:       int64(t.Litres),
			FeedStock:    int64(t.FeedStock),
			ServedUntil:  int64(t.ServedUntil),
			Upgrades:     t.Upgrades,
			Oxygen:       int32(t.Oxygen),
			Aerating:     t.Aerating,
			FeedCarry:    t.FeedCarry,
			FeedUnitCost: int64(t.FeedUnitCost),
			UpkeepCarry:  t.UpkeepCarry,
			CarrierUntil: int64(t.CarrierUntil),
			Accrual: accrualDocument{
				Window:           int64(t.Accrual.Window),
				HypoxiaDeaths:    int32(t.Accrual.HypoxiaDeaths),
				DiseaseDeaths:    int32(t.Accrual.DiseaseDeaths),
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
				Cost:            int64(batch.Cost),
				CostCarry:       batch.CostCarry,
				Sick:            batch.Sick,
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

// Decode reconstroi o estado do JSON, recalculando o que e derivado.
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
	state.NextTankID = sim.TankID(doc.NextTankID)
	state.NextBatchID = sim.BatchID(doc.NextBatchID)
	state.EventSeq = doc.EventSeq
	state.Debt = sim.Coins(doc.Debt)
	state.DebtCarry = doc.DebtCarry
	state.LastCycle = sim.Cycle{
		Fish:       sim.FishCount(doc.LastCycle.Fish),
		Mass:       sim.Micrograms(doc.LastCycle.Mass),
		Revenue:    sim.Coins(doc.LastCycle.Revenue),
		Cost:       sim.Coins(doc.LastCycle.Cost),
		CostPerKg:  sim.Coins(doc.LastCycle.CostPerKg),
		PricePerKg: sim.Coins(doc.LastCycle.PricePerKg),
		FCRPPM:     sim.PPM(doc.LastCycle.FCRPPM),
	}

	if len(doc.Tanks) > len(state.Tanks) {
		return sim.State{}, ErrTooManyTanks
	}

	for i := range doc.Tanks {
		tank := &doc.Tanks[i]
		if len(tank.Batches) > len(state.Tanks[i].Batches) {
			return sim.State{}, ErrTooManyBatches
		}
		if !sim.TankKind(tank.Kind).Known() {
			return sim.State{}, fmt.Errorf("%w: %d", ErrUnknownKind, tank.Kind)
		}

		state.Tanks[i] = sim.Tank{
			ID:           sim.TankID(tank.ID),
			Kind:         sim.TankKind(tank.Kind),
			Litres:       sim.Litres(tank.Litres),
			FeedStock:    sim.Micrograms(tank.FeedStock),
			ServedUntil:  sim.Tick(tank.ServedUntil),
			Upgrades:     tank.Upgrades | doc.Upgrades,
			Oxygen:       sim.MicrogramsPerLiter(tank.Oxygen),
			Aerating:     tank.Aerating,
			FeedCarry:    tank.FeedCarry,
			FeedUnitCost: sim.Coins(tank.FeedUnitCost),
			UpkeepCarry:  tank.UpkeepCarry,
			CarrierUntil: sim.Tick(tank.CarrierUntil),
			Accrual: sim.Accrual{
				Window:           sim.Tick(tank.Accrual.Window),
				HypoxiaDeaths:    sim.FishCount(tank.Accrual.HypoxiaDeaths),
				DiseaseDeaths:    sim.FishCount(tank.Accrual.DiseaseDeaths),
				StarvationDeaths: sim.FishCount(tank.Accrual.StarvationDeaths),
				FeedEaten:        sim.Micrograms(tank.Accrual.FeedEaten),
				MassGained:       sim.Micrograms(tank.Accrual.MassGained),
			},
			BatchCount: int32(len(tank.Batches)),
		}

		for j := range tank.Batches {
			batch := &tank.Batches[j]
			state.Tanks[i].Batches[j] = sim.Batch{
				ID:              sim.BatchID(batch.ID),
				Fish:            sim.FishCount(batch.Fish),
				MeanMass:        sim.Micrograms(batch.MeanMass),
				MassRoot:        sim.MassRootOf(sim.Micrograms(batch.MeanMass)),
				GrowthCarry:     batch.GrowthCarry,
				StockedAt:       sim.Tick(batch.StockedAt),
				FeedEaten:       sim.Micrograms(batch.FeedEaten),
				MassGained:      sim.Micrograms(batch.MassGained),
				Cost:            sim.Coins(batch.Cost),
				CostCarry:       batch.CostCarry,
				Sick:            batch.Sick,
				HypoxiaTicks:    batch.HypoxiaTicks,
				StarvationTicks: batch.StarvationTicks,
			}
		}
	}
	state.TankCount = int32(len(doc.Tanks))

	return state, nil
}
