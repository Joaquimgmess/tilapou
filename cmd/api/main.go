package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Joaquimgmess/catalog/internal/platform/config"
	"github.com/Joaquimgmess/catalog/internal/platform/httpx"
	"github.com/Joaquimgmess/catalog/internal/platform/logging"
	"github.com/Joaquimgmess/catalog/internal/platform/postgres"
	"github.com/Joaquimgmess/catalog/internal/product"
)

const (
	readHeaderTimeout = 5 * time.Second
	errorDocsPrefix   = "https://github.com/Joaquimgmess/catalog/blob/main/docs/errors.md#"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
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

	httpx.SetErrorDocsPrefix(errorDocsPrefix)

	router, api := httpx.NewAPI(logger, httpx.Options{
		Title:          "Catalog API",
		Version:        "1.0.0",
		APIPrefix:      "/v1",
		RequestTimeout: cfg.RequestTimeout,
		TrustedProxies: int(cfg.TrustedProxies),
	})
	httpx.RegisterHealth(router, pool.Ping)
	product.RegisterRoutes(api, product.NewDB(pool, cfg.DBTimeout))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}
	return nil
}
