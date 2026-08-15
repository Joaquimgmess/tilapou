package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Joaquimgmess/catalog/internal/migrations"
	"github.com/Joaquimgmess/catalog/internal/platform/config"
	"github.com/Joaquimgmess/catalog/internal/platform/logging"
	"github.com/Joaquimgmess/catalog/internal/platform/postgres"
)

func main() {
	status := flag.Bool("status", false, "list pending migrations instead of applying them")
	flag.Parse()

	if err := run(*status); err != nil {
		slog.Error("migrate failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(statusOnly bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	ctx = logging.WithLogger(ctx, logger)

	if statusOnly {
		pending, err := migrations.Pending(ctx, pool)
		if err != nil {
			return err
		}
		logger.InfoContext(ctx, "pending migrations", slog.Any("versions", pending))

		return nil
	}

	return migrations.Apply(ctx, pool)
}
