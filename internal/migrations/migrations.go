package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
)

//go:embed sql/*.sql
var files embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	provider, err := newProvider(ctx, pool)
	if err != nil {
		return err
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}

	logger := logging.FromContext(ctx)
	for _, result := range results {
		logger.InfoContext(ctx, "migration applied",
			slog.Int64("version", result.Source.Version),
			slog.String("source", result.Source.Path),
			slog.Duration("duration", result.Duration),
		)
	}

	return nil
}

func Pending(ctx context.Context, pool *pgxpool.Pool) ([]int64, error) {
	provider, err := newProvider(ctx, pool)
	if err != nil {
		return nil, err
	}

	statuses, err := provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading migration status: %w", err)
	}

	var pending []int64
	for _, status := range statuses {
		if status.State == goose.StatePending {
			pending = append(pending, status.Source.Version)
		}
	}

	return pending, nil
}

func newProvider(ctx context.Context, pool *pgxpool.Pool) (*goose.Provider, error) {
	sqlFS, err := fs.Sub(files, "sql")
	if err != nil {
		return nil, fmt.Errorf("opening migrations directory: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		stdlib.OpenDBFromPool(pool),
		sqlFS,
		goose.WithSlog(logging.FromContext(ctx)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating migration provider: %w", err)
	}

	return provider, nil
}
