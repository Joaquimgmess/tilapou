package farm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
	"github.com/Joaquimgmess/tilapou/internal/sim"
	"github.com/Joaquimgmess/tilapou/internal/sim/save"
)

type Store interface {
	ByPlayer(ctx context.Context, playerID uuid.UUID) (Farm, error)
	Insert(ctx context.Context, f Farm) error
	Save(ctx context.Context, f Farm, events []sim.Event, outcome *sim.Outcome) error
	Events(ctx context.Context, id ID, limit int32) ([]StoredEvent, error)
	AppliedOutcome(ctx context.Context, id ID, key sim.ActionID) (sim.Outcome, bool, error)
}

type StoredEvent struct {
	Seq    uint64
	Kind   string
	From   sim.Tick
	To     sim.Tick
	Tank   sim.TankID
	Fish   sim.FishCount
	Mass   sim.Micrograms
	Cash   sim.Coins
	Reason string
}

type DB struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewDB(pool *pgxpool.Pool, timeout time.Duration) *DB {
	return &DB{pool: pool, timeout: timeout}
}

func (d *DB) ByPlayer(ctx context.Context, playerID uuid.UUID) (Farm, error) {
	const query = `
		SELECT id, player_id, name, epoch, revision, state, created_at
		FROM farms
		WHERE player_id = $1`

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	var (
		f   Farm
		raw []byte
	)

	err := d.pool.QueryRow(ctx, query, playerID).
		Scan(&f.ID, &f.PlayerID, &f.Name, &f.Epoch, &f.Revision, &raw, &f.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Farm{}, ErrNotFound
	case err != nil:
		return Farm{}, fmt.Errorf("select farm of player %s: %w", playerID, err)
	}

	f.State, err = save.Decode(raw)
	if err != nil {
		return Farm{}, fmt.Errorf("farm %s: %w", f.ID, err)
	}

	return f, nil
}

func (d *DB) Insert(ctx context.Context, f Farm) error {
	const query = `
		INSERT INTO farms (id, player_id, name, epoch, revision, state, cash_cents, biomass_ug, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	raw, err := save.Encode(f.State)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	projection := sim.Project(&f.State)
	_, err = d.pool.Exec(ctx, query,
		f.ID, f.PlayerID, f.Name, f.Epoch, f.Revision, raw,
		int64(projection.Cash), int64(projection.Biomass), f.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert farm %s: %w", f.ID, err)
	}

	return nil
}

func (d *DB) Save(ctx context.Context, f Farm, events []sim.Event, outcome *sim.Outcome) error {
	raw, err := save.Encode(f.State)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save of farm %s: %w", f.ID, err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			logging.FromContext(ctx).ErrorContext(ctx, "rolling back farm save", slog.Any("error", rbErr))
		}
	}()

	projection := sim.Project(&f.State)

	const update = `
		UPDATE farms
		SET state = $1, revision = revision + 1, cash_cents = $2, biomass_ug = $3, updated_at = now()
		WHERE id = $4 AND revision = $5`

	tag, err := tx.Exec(ctx, update, raw, int64(projection.Cash), int64(projection.Biomass), f.ID, f.Revision)
	if err != nil {
		return fmt.Errorf("update farm %s: %w", f.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStaleRevision
	}

	if err := insertEvents(ctx, tx, f.ID, events); err != nil {
		return err
	}

	if err := insertOutcome(ctx, tx, f.ID, outcome); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save of farm %s: %w", f.ID, err)
	}

	return nil
}

func insertOutcome(ctx context.Context, tx pgx.Tx, id ID, outcome *sim.Outcome) error {
	if outcome == nil {
		return nil
	}

	const query = `
		INSERT INTO farm_actions (farm_id, idempotency_key, applied, reason, at_tick, needed_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (farm_id, idempotency_key) DO NOTHING`

	_, err := tx.Exec(ctx, query, id, int64(outcome.ID), outcome.Applied,
		outcome.Reason.String(), int64(outcome.At), int64(outcome.Needed))
	if err != nil {
		return fmt.Errorf("record action of farm %s: %w", id, err)
	}

	return nil
}

func insertEvents(ctx context.Context, tx pgx.Tx, id ID, events []sim.Event) error {
	const insert = `
		INSERT INTO farm_events (farm_id, seq, kind, from_tick, to_tick, tank_id, fish, mass_ug, cash_cents, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (farm_id, seq) DO NOTHING`

	for _, e := range events {
		_, err := tx.Exec(ctx, insert,
			id, int64(e.Seq), e.Kind.String(), int64(e.From), int64(e.To),
			int64(e.Tank), int64(e.Fish), int64(e.Mass), int64(e.Cash), e.Reason.String())
		if err != nil {
			return fmt.Errorf("insert event %d of farm %s: %w", e.Seq, id, err)
		}
	}

	return nil
}

func (d *DB) Events(ctx context.Context, id ID, limit int32) ([]StoredEvent, error) {
	const query = `
		SELECT seq, kind, from_tick, to_tick, tank_id, fish, mass_ug, cash_cents, reason
		FROM farm_events
		WHERE farm_id = $1
		ORDER BY seq DESC
		LIMIT $2`

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	rows, err := d.pool.Query(ctx, query, id, limit)
	if err != nil {
		return nil, fmt.Errorf("select events of farm %s: %w", id, err)
	}

	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredEvent, error) {
		var (
			e                   StoredEvent
			seq, from, to, tank int64
			fish, mass, cash    int64
		)
		if scanErr := row.Scan(&seq, &e.Kind, &from, &to, &tank, &fish, &mass, &cash, &e.Reason); scanErr != nil {
			return StoredEvent{}, fmt.Errorf("scan event row: %w", scanErr)
		}

		e.Seq = uint64(seq)
		e.From, e.To = sim.Tick(from), sim.Tick(to)
		e.Tank = sim.TankID(tank)
		e.Fish, e.Mass, e.Cash = sim.FishCount(fish), sim.Micrograms(mass), sim.Coins(cash)

		return e, nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect events of farm %s: %w", id, err)
	}

	return events, nil
}

func (d *DB) AppliedOutcome(ctx context.Context, id ID, key sim.ActionID) (sim.Outcome, bool, error) {
	const query = `
		SELECT applied, reason, at_tick, needed_cents
		FROM farm_actions WHERE farm_id = $1 AND idempotency_key = $2`

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	var (
		reason string
		tick   int64
		needed int64
		out    = sim.Outcome{ID: key}
	)

	err := d.pool.QueryRow(ctx, query, id, int64(key)).Scan(&out.Applied, &reason, &tick, &needed)
	if errors.Is(err, pgx.ErrNoRows) {
		return sim.Outcome{}, false, nil
	}
	if err != nil {
		return sim.Outcome{}, false, fmt.Errorf("read idempotency of farm %s: %w", id, err)
	}

	out.At, out.Needed = sim.Tick(tick), sim.Coins(needed)
	out.Reason, _ = sim.RejectReasonNamed(reason)

	return out, true, nil
}
