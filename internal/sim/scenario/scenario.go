// Package scenario runs deterministic simulator games to check balancing and lock
// down regressions.
package scenario

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Joaquimgmess/tilapou/internal/sim"
)

// Scenario describes a game with Cash in cents, Setup building the state at tick 0
// and Actions scheduled in ticks.
type Scenario struct {
	Name    string
	Zone    sim.ZoneOffset
	Seed    sim.Seed
	Days    int64
	Cash    sim.Coins
	Setup   func(s *sim.State, b *sim.Balance)
	Actions []sim.Action
}

// Sample photographs the first tank at the end of the day, in the units of the suffix
// and with Cash in cents and Density in thousandths of kg per cubic metre.
type Sample struct {
	Day     int64
	Fish    sim.FishCount
	MeanG   int64
	FeedKg  int64
	Cash    sim.Coins
	Oxygen  sim.MicrogramsPerLiter
	Density int64
}

// Result carries one sample per day, the event count by kind and the final state.
type Result struct {
	Scenario Scenario
	Samples  []Sample
	Totals   map[string]int64
	Final    sim.State
}

func (s Scenario) initial(b *sim.Balance) sim.State {
	state := sim.NewState(s.Seed, s.Zone, 0)
	state.Cash = s.Cash
	if s.Setup != nil {
		s.Setup(&state, b)
	}

	return state
}

// Run advances the scenario day by day, sampling the end of each one.
func Run(s Scenario, b *sim.Balance) (Result, error) {
	state := s.initial(b)

	result := Result{Scenario: s, Totals: map[string]int64{}}

	for day := int64(1); day <= s.Days; day++ {
		out, err := sim.Advance(sim.Input{
			State:   state,
			Until:   sim.Tick(day) * sim.TicksPerDay,
			Balance: b,
			Actions: actionsWithin(s.Actions, state.Tick, sim.Tick(day)*sim.TicksPerDay),
		})
		if err != nil {
			return Result{}, fmt.Errorf("day %d: %w", day, err)
		}

		state = out.State
		for _, e := range out.Events {
			result.Totals[e.Kind.String()]++
		}
		result.Samples = append(result.Samples, sampleOf(day, &state))
	}

	result.Final = state

	return result, nil
}

// RunWhole advances the scenario in a single Advance, to check that the result
// matches Run's.
func RunWhole(s Scenario, b *sim.Balance) (sim.State, error) {
	state := s.initial(b)

	out, err := sim.Advance(sim.Input{
		State:   state,
		Until:   sim.Tick(s.Days) * sim.TicksPerDay,
		Balance: b,
		Actions: s.Actions,
	})
	if err != nil {
		return sim.State{}, err
	}

	return out.State, nil
}

func actionsWithin(actions []sim.Action, from, to sim.Tick) []sim.Action {
	var due []sim.Action
	for _, a := range actions {
		if a.At > from && a.At <= to {
			due = append(due, a)
		}
	}

	return due
}

func sampleOf(day int64, state *sim.State) Sample {
	sample := Sample{Day: day, Cash: state.Cash}
	if state.TankCount == 0 {
		return sample
	}

	tank := &state.Tanks[0]
	sample.Fish = tank.Fish()
	sample.FeedKg = int64(tank.FeedStock / sim.MicrogramsPerKilogram)
	sample.Oxygen = tank.Oxygen
	if tank.Litres > 0 {
		sample.Density = int64(tank.Biomass()) / (sim.LitresPerCubicMetre * int64(tank.Litres))
	}
	if tank.BatchCount > 0 {
		sample.MeanG = tank.Batches[0].MeanMass.Grams()
	}

	return sample
}

// Render builds the text table, with the cash already out of cents.
func (r Result) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "cenario: %s\n", r.Scenario.Name)
	fmt.Fprintf(&b, "semente: %d  fuso: %d  dias: %d\n\n", r.Scenario.Seed, r.Scenario.Zone, r.Scenario.Days)
	fmt.Fprintf(&b, "%4s %8s %8s %8s %12s %8s %9s\n", "dia", "peixes", "peso_g", "racao_kg", "caixa_tc", "o2_ugl", "kg_por_m3")

	for _, s := range r.Samples {
		fmt.Fprintf(&b, "%4d %8d %8d %8d %12d %8d %9d\n",
			s.Day, s.Fish, s.MeanG, s.FeedKg, int64(s.Cash)/100, s.Oxygen, s.Density)
	}

	fmt.Fprint(&b, "\neventos\n")
	for _, kind := range slices.Sorted(maps.Keys(r.Totals)) {
		fmt.Fprintf(&b, "%-20s %6d\n", kind, r.Totals[kind])
	}

	return b.String()
}
