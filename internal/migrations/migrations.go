package migrations

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Joaquimgmess/catalog/internal/platform/logging"
)

//go:embed sql/*.sql
var files embed.FS

const createTable = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version     TEXT PRIMARY KEY,
		applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
	)`

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	pending, err := fs.Glob(files, "sql/*.sql")
	if err != nil {
		return fmt.Errorf("listing migrations: %w", err)
	}
	slices.Sort(pending)

	logger := logging.FromContext(ctx)
	for _, name := range pending {
		if slices.Contains(applied, name) {
			continue
		}
		if err := applyOne(ctx, pool, name); err != nil {
			return err
		}
		logger.InfoContext(ctx, "migration applied", slog.String("version", name))
	}

	return nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("reading schema_migrations: %w", err)
	}

	versions, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("collecting schema_migrations: %w", err)
	}

	return versions, nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, name string) error {
	statements, err := files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("reading %s: %w", name, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", name, err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logging.FromContext(ctx).ErrorContext(ctx, "rolling back migration",
				slog.String("version", name), slog.Any("error", err))
		}
	}()

	if _, err := tx.Exec(ctx, string(statements)); err != nil {
		return fmt.Errorf("applying %s: %w", name, err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("recording %s: %w", name, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s: %w", name, err)
	}

	return nil
}
