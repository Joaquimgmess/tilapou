package sim

// Input parte de State e roda ate Until aplicando Actions ordenadas por tick.
// Budget limita o trabalho por chamada; se nao for positivo, nao trunca.
type Input struct {
	State   State
	Until   Tick
	Balance *Balance
	Actions []Action
	Budget  int64
}

// Output traz o estado novo; Stopped so fica aquem de Input.Until quando Truncated,
// e Outcomes tem uma entrada por acao.
type Output struct {
	State     State
	Events    []Event
	Outcomes  []Outcome
	Stopped   Tick
	Truncated bool
}

const defaultBudget = int64(1) << 62

// Advance roda a simulacao ate in.Until sem mutar in.State e devolve sempre a mesma
// saida para a mesma entrada. Erra com ErrNoBalance, o erro de Balance.Validate,
// ErrBackwardsTime ou ErrActionsUnsorted.
func Advance(in Input) (Output, error) {
	if in.Balance == nil {
		return Output{}, ErrNoBalance
	}
	if err := in.Balance.Validate(); err != nil {
		return Output{}, err
	}
	if in.Until < in.State.Tick {
		return Output{}, ErrBackwardsTime
	}
	if !actionsAreSorted(in.Actions) {
		return Output{}, ErrActionsUnsorted
	}

	budget := in.Budget
	if budget <= 0 {
		budget = defaultBudget
	}

	state := in.State
	sink := &eventSink{nextSeq: state.EventSeq}
	outcomes := make([]Outcome, 0, len(in.Actions))

	pending := in.Actions
	truncated := false
	tick := state.Tick

	var overdue []Action
	overdue, pending = actionsDue(pending, tick)
	outcomes = applyDue(&state, in.Balance, overdue, tick, sink, outcomes)

	for tick < in.Until {
		cost := int64(state.TankCount)
		if cost == 0 {
			cost = 1
		}
		if budget < cost {
			truncated = true
			break
		}
		budget -= cost

		tick++

		var due []Action
		due, pending = actionsDue(pending, tick)
		outcomes = applyDue(&state, in.Balance, due, tick, sink, outcomes)

		step(&state, in.Balance, tick, sink)
		closeWindows(&state, tick, sink)
	}

	state.Tick = tick
	state.EventSeq = sink.nextSeq

	return Output{
		State:     state,
		Events:    sink.events,
		Outcomes:  outcomes,
		Stopped:   tick,
		Truncated: truncated,
	}, nil
}

func closeWindows(s *State, tick Tick, sink *eventSink) {
	window := tick.WindowStart(WindowTicks)

	for i := range s.TankCount {
		t := &s.Tanks[i]
		if t.Accrual.Window == window {
			continue
		}
		if t.Accrual.Window == 0 && t.Accrual.empty() {
			t.Accrual.Window = window
			continue
		}
		t.flushAccrual(sink, window)
	}
}

func actionsAreSorted(actions []Action) bool {
	for i := 1; i < len(actions); i++ {
		if actions[i].At < actions[i-1].At {
			return false
		}
	}

	return true
}

func applyDue(state *State, b *Balance, due []Action, tick Tick, sink *eventSink, outcomes []Outcome) []Outcome {
	for _, a := range due {
		reason, needed := apply(state, b, a, tick, sink)
		if reason != RejectNone {
			sink.emit(Event{
				Kind:   EventActionRejected,
				From:   tick,
				To:     tick,
				Tank:   a.Tank,
				Batch:  a.Batch,
				Reason: reason,
			})
		}
		outcomes = append(outcomes, Outcome{ID: a.ID, At: tick, Applied: reason == RejectNone, Reason: reason, Needed: needed})
	}

	return outcomes
}

func actionsDue(actions []Action, tick Tick) (due, rest []Action) {
	cut := 0
	for cut < len(actions) && actions[cut].At <= tick {
		cut++
	}

	return actions[:cut], actions[cut:]
}
