package farm

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const (
	defaultZone    = sim.ZoneOffset(-180)
	eventFeedLimit = int32(40)
)

// Clock gives the real instant that defines up to which tick the farm advances.
type Clock func() time.Time

// Sessions advances the farm and applies actions, serializing the writes of a
// single farm within the process.
type Sessions struct {
	store   Store
	balance *sim.Balance
	clock   Clock

	mu    sync.Mutex
	locks map[ID]*sync.Mutex
}

// NewSessions builds the sessions on top of the store, the balance and the clock.
func NewSessions(store Store, balance *sim.Balance, clock Clock) *Sessions {
	return &Sessions{store: store, balance: balance, clock: clock, locks: make(map[ID]*sync.Mutex)}
}

// Snapshot is the already advanced farm, with a nil Outcome when there was no action.
type Snapshot struct {
	Farm       Farm
	Projection sim.Projection
	Events     []StoredEvent
	Temp       sim.MilliCelsius
	Outcome    *sim.Outcome
}

// Sync advances the farm up to now, creating it if the player has none.
func (s *Sessions) Sync(ctx context.Context, playerID uuid.UUID) (Snapshot, error) {
	return s.withFarm(ctx, playerID, nil)
}

// Act applies the action at the current tick. Repeating the same action.ID only returns
// the already recorded result.
func (s *Sessions) Act(ctx context.Context, playerID uuid.UUID, action sim.Action) (Snapshot, error) {
	return s.withFarm(ctx, playerID, &action)
}

func (s *Sessions) withFarm(ctx context.Context, playerID uuid.UUID, action *sim.Action) (Snapshot, error) {
	f, err := s.ensure(ctx, playerID)
	if err != nil {
		return Snapshot{}, err
	}

	lock := s.lockFor(f.ID)
	lock.Lock()
	defer lock.Unlock()

	for range 2 {
		snap, attemptErr := s.attempt(ctx, playerID, action)
		if errors.Is(attemptErr, ErrAlreadyApplied) {
			continue
		}

		return snap, attemptErr
	}

	return Snapshot{}, ErrAlreadyApplied
}

func (s *Sessions) attempt(ctx context.Context, playerID uuid.UUID, action *sim.Action) (Snapshot, error) {
	f, err := s.store.ByPlayer(ctx, playerID)
	if err != nil {
		return Snapshot{}, err
	}

	pending := action

	var replayed *sim.Outcome
	if action != nil {
		stored, applied, appliedErr := s.store.AppliedOutcome(ctx, f.ID, action.ID)
		if appliedErr != nil {
			return Snapshot{}, appliedErr
		}
		if applied {
			pending, replayed = nil, &stored
		}
	}

	now := max(TickAt(f.Epoch, s.clock()), f.State.Tick)

	var actions []sim.Action
	if pending != nil {
		scheduled := *pending
		scheduled.At = now
		actions = append(actions, scheduled)
	}

	out, err := sim.Advance(sim.Input{State: f.State, Until: now, Balance: s.balance, Actions: actions})
	if err != nil {
		return Snapshot{}, err
	}

	var recorded *sim.Outcome
	if len(out.Outcomes) > 0 {
		recorded = &out.Outcomes[0]
	}

	outcome := replayed
	if recorded != nil {
		outcome = recorded
	}

	changed := out.State.Tick != f.State.Tick || len(out.Outcomes) > 0
	f.State = out.State

	if changed {
		if err := s.store.Save(ctx, f, out.Events, recorded); err != nil {
			return Snapshot{}, err
		}
		f.Revision++
	}

	logging.FromContext(ctx).DebugContext(ctx, "farm advanced",
		slog.String("farm_id", f.ID.String()),
		slog.Int64("tick", int64(f.State.Tick)),
		slog.Int("events", len(out.Events)),
	)

	return s.snapshot(ctx, f, outcome)
}

func (s *Sessions) ensure(ctx context.Context, playerID uuid.UUID) (Farm, error) {
	f, err := s.store.ByPlayer(ctx, playerID)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Farm{}, err
	}

	created := New(uuid.New(), playerID, "Tilapou", s.clock(), defaultZone, sim.Seed(playerID.ID()), s.balance)
	if err := s.store.Insert(ctx, created); err != nil {
		return Farm{}, err
	}

	return created, nil
}

func (s *Sessions) snapshot(ctx context.Context, f Farm, outcome *sim.Outcome) (Snapshot, error) {
	events, err := s.store.Events(ctx, f.ID, eventFeedLimit)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Farm:       f,
		Projection: sim.Project(&f.State),
		Events:     events,
		Temp:       sim.TemperatureAt(s.balance, f.State.Tick, f.State.Zone),
		Outcome:    outcome,
	}, nil
}

func (s *Sessions) lockFor(id ID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.locks[id]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[id] = lock
	}

	return lock
}
