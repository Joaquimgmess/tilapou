package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/farm"
	"github.com/Joaquimgmess/tilapou/internal/migrations"
	"github.com/Joaquimgmess/tilapou/internal/platform/config"
	"github.com/Joaquimgmess/tilapou/internal/platform/httpx"
	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
	"github.com/Joaquimgmess/tilapou/internal/platform/metrics"
	"github.com/Joaquimgmess/tilapou/internal/platform/postgres"
)

var errNotReady = errors.New("service is not ready")

const (
	serviceName        = "tilapou"
	serviceVersion     = "1.0.0"
	readHeaderTimeout  = 5 * time.Second
	errorDocsPrefix    = "https://github.com/Joaquimgmess/tilapou/blob/main/docs/errors.md#"
	localPlayer        = "00000000-0000-0000-0000-000000000001"
	healthProbeTimeout = 3 * time.Second
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

	logger := logging.New(cfg.LogLevel, logging.Service{
		Name:    serviceName,
		Env:     cfg.Env,
		Version: serviceVersion,
	})

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

	registry := metrics.NewRegistry()
	sessions := farm.NewSessions(farm.NewDB(pool, cfg.DBTimeout), &rules, time.Now, registry)

	router, api := httpx.NewAPI(logger, registry, "Tilapou", serviceVersion, "/v1", cfg.RequestTimeout,
		httpx.WithTrustedProxies(int(cfg.TrustedProxies)),
		httpx.WithErrorDocs(errorDocsPrefix),
	)
	httpx.RegisterHealth(router, pool.Ping,
		httpx.WithBuild(buildRevision()),
		httpx.WithDatabase(databaseName(cfg.DatabaseURL)))
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

// buildRevision le o commit do proprio binario. E o que deixa o portao de teste falhar com
// "daemon rodando build X, teste esperando Y" em vez de deixar alguem medir binario velho e
// reportar regressao que nao existe.
func buildRevision() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	return revision, modified
}

// databaseName tira o nome do banco da URL de conexao. Publicar isso e o que permite a guarda
// morar do lado que sabe: quem escreve e o daemon, e so ele conhece o destino de verdade.
func databaseName(url string) string {
	parsed, err := neturl.Parse(url)
	if err != nil {
		return ""
	}

	return strings.TrimPrefix(parsed.Path, "/")
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
