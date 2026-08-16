package farm_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/farm"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

type memoryStore struct {
	farm    farm.Farm
	actions map[sim.ActionID]bool
	saves   int
}

func (m *memoryStore) ByPlayer(context.Context, uuid.UUID) (farm.Farm, error) {
	return m.farm, nil
}

func (m *memoryStore) Insert(_ context.Context, f farm.Farm) error {
	m.farm = f

	return nil
}

func (m *memoryStore) Save(_ context.Context, f farm.Farm, _ []sim.Event) error {
	m.farm = f
	m.farm.Revision++
	m.saves++

	return nil
}

func (*memoryStore) Events(context.Context, farm.ID, int32) ([]farm.StoredEvent, error) {
	return nil, nil
}

func (m *memoryStore) AlreadyApplied(_ context.Context, _ farm.ID, key sim.ActionID) (bool, error) {
	return m.actions[key], nil
}

func (m *memoryStore) RecordAction(_ context.Context, _ farm.ID, key sim.ActionID, _ sim.Outcome) error {
	m.actions[key] = true

	return nil
}

func TestAnActionLandsAtTheCurrentTickNotTheStaleOne(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	epoch := time.Unix(0, 0).UTC()
	store := &memoryStore{
		farm:    farm.New(uuid.New(), uuid.New(), "t", epoch, 0, 1, &b),
		actions: map[sim.ActionID]bool{},
	}

	away := 3 * 24 * time.Hour
	sessions := farm.NewSessions(store, &b, func() time.Time { return epoch.Add(away) })

	snap, err := sessions.Act(context.Background(), store.farm.PlayerID,
		sim.Action{ID: 1, Kind: sim.ActionFeed, Tank: 1})
	if err != nil {
		t.Fatal(err)
	}

	if !snap.Outcome.Applied {
		t.Fatalf("o trato foi recusado: %v", snap.Outcome.Reason)
	}

	served := store.farm.State.Tanks[0].ServedUntil - store.farm.State.Tick
	if served <= 0 {
		t.Errorf("o trato foi servido %d ticks no passado: a acao caiu no tick velho", -served)
	}
}

func TestAReadWithNoTimePassedDoesNotWriteTheState(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	epoch := time.Unix(0, 0).UTC()
	store := &memoryStore{
		farm:    farm.New(uuid.New(), uuid.New(), "t", epoch, 0, 1, &b),
		actions: map[sim.ActionID]bool{},
	}

	frozen := epoch.Add(time.Hour)
	sessions := farm.NewSessions(store, &b, func() time.Time { return frozen })
	player := store.farm.PlayerID

	if _, err := sessions.Sync(context.Background(), player); err != nil {
		t.Fatal(err)
	}
	after := store.saves

	for range 5 {
		if _, err := sessions.Sync(context.Background(), player); err != nil {
			t.Fatal(err)
		}
	}

	if store.saves != after {
		t.Errorf("%d leituras sem tempo passar gravaram %d vezes", 5, store.saves-after)
	}
}

func TestActingDoesNotPushTheFarmClockAheadOfRealTime(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	epoch := time.Unix(0, 0).UTC()
	store := &memoryStore{
		farm:    farm.New(uuid.New(), uuid.New(), "t", epoch, 0, 1, &b),
		actions: map[sim.ActionID]bool{},
	}

	frozen := epoch.Add(time.Hour)
	sessions := farm.NewSessions(store, &b, func() time.Time { return frozen })
	player := store.farm.PlayerID

	for i := range 50 {
		if _, err := sessions.Act(context.Background(), player,
			sim.Action{ID: sim.ActionID(i + 1), Kind: sim.ActionFeed, Tank: 1}); err != nil {
			t.Fatal(err)
		}
	}

	if drift := store.farm.State.Tick - farm.TickAt(epoch, frozen); drift != 0 {
		t.Errorf("50 acoes instantaneas adiantaram o relogio da fazenda em %d ticks", drift)
	}
}

func TestAnActionWithNoTimePassingIsStillPersisted(t *testing.T) {
	t.Parallel()

	b, err := balance.Load()
	if err != nil {
		t.Fatal(err)
	}

	epoch := time.Unix(0, 0).UTC()
	store := &memoryStore{
		farm:    farm.New(uuid.New(), uuid.New(), "t", epoch, 0, 1, &b),
		actions: map[sim.ActionID]bool{},
	}

	frozen := epoch.Add(time.Hour)
	sessions := farm.NewSessions(store, &b, func() time.Time { return frozen })
	player := store.farm.PlayerID

	if _, err := sessions.Sync(context.Background(), player); err != nil {
		t.Fatal(err)
	}

	before := store.farm.State.Tanks[0].FeedStock
	saves := store.saves

	snap, actErr := sessions.Act(context.Background(), player,
		sim.Action{ID: 1, Kind: sim.ActionBuyFeed, Tank: 1, Amount: 50})
	if actErr != nil {
		t.Fatal(actErr)
	}
	if !snap.Outcome.Applied {
		t.Fatalf("a compra foi recusada: %v", snap.Outcome.Reason)
	}

	if store.saves == saves {
		t.Error("a acao foi aplicada e nao foi gravada: o relogio nao andou e a guarda descartou tudo")
	}
	if store.farm.State.Tanks[0].FeedStock <= before {
		t.Errorf("a racao comprada nao sobreviveu: %d, antes %d", store.farm.State.Tanks[0].FeedStock, before)
	}
}
