package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/catalog/internal/balance"
	"github.com/Joaquimgmess/catalog/internal/farm"
	"github.com/Joaquimgmess/catalog/internal/migrations"
	"github.com/Joaquimgmess/catalog/internal/platform/config"
	"github.com/Joaquimgmess/catalog/internal/platform/httpx"
	"github.com/Joaquimgmess/catalog/internal/platform/logging"
	"github.com/Joaquimgmess/catalog/internal/platform/postgres"
)

const (
	readHeaderTimeout = 5 * time.Second
	errorDocsPrefix   = "https://github.com/Joaquimgmess/catalog/blob/main/docs/errors.md#"
	localPlayer       = "00000000-0000-0000-0000-000000000001"
)

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("lendo flags de serve: %w", err)
	}

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

	if migErr := migrations.Apply(logging.WithLogger(ctx, logger), pool); migErr != nil {
		return migErr
	}

	rules, err := balance.Load()
	if err != nil {
		return err
	}

	player, err := uuid.Parse(localPlayer)
	if err != nil {
		return fmt.Errorf("player local invalido: %w", err)
	}

	sessions := farm.NewSessions(farm.NewDB(pool, cfg.DBTimeout), &rules, time.Now)

	router, api := httpx.NewAPI(logger, httpx.Options{
		Title:          "Tilapou",
		Version:        "1.0.0",
		APIPrefix:      "/v1",
		RequestTimeout: cfg.RequestTimeout,
		TrustedProxies: int(cfg.TrustedProxies),
		ErrorDocsURL:   errorDocsPrefix,
	})
	httpx.RegisterHealth(router, pool.Ping)
	farm.RegisterRoutes(api, sessions, player, &rules)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("daemon escutando", slog.String("addr", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("encerrando daemon")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}

	return nil
}

func runHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("lendo flags de health: %w", err)
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+addr+"/readyz", http.NoBody)
	if err != nil {
		return fmt.Errorf("building probe request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("probing readiness: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", errNotReady, resp.StatusCode)
	}

	return nil
}

const healthProbeTimeout = 3 * time.Second

var errNotReady = errors.New("service is not ready")
