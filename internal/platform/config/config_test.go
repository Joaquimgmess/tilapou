package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/Joaquimgmess/tilapou/internal/platform/config"
)

const validURL = "postgres://user:pass@localhost:5432/db"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want an error")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", validURL)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("DBMaxConns = %d, want 10", cfg.DBMaxConns)
	}
	if cfg.DBTimeout != 3*time.Second {
		t.Errorf("DBTimeout = %v, want %v", cfg.DBTimeout, 3*time.Second)
	}
}

func TestLoadParsesOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", validURL)
	t.Setenv("ADDR", ":9000")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DB_MAX_CONNS", "42")
	t.Setenv("REQUEST_TIMEOUT", "90s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != ":9000" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":9000")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.DBMaxConns != 42 {
		t.Errorf("DBMaxConns = %d, want 42", cfg.DBMaxConns)
	}
	if cfg.RequestTimeout != 90*time.Second {
		t.Errorf("RequestTimeout = %v, want %v", cfg.RequestTimeout, 90*time.Second)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := map[string]struct{ key, value string }{
		"log level": {"LOG_LEVEL", "loud"},
		"max conns": {"DB_MAX_CONNS", "many"},
		"overflow":  {"DB_MAX_CONNS", "99999999999"},
		"duration":  {"REQUEST_TIMEOUT", "30 seconds"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", validURL)
			t.Setenv(tt.key, tt.value)

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() with %s=%q error = nil, want an error", tt.key, tt.value)
			}
		})
	}
}

func TestLoadWiresEveryDurationToItsOwnVariable(t *testing.T) {
	t.Setenv("DATABASE_URL", validURL)
	t.Setenv("DB_TIMEOUT", "1s")
	t.Setenv("REQUEST_TIMEOUT", "2s")
	t.Setenv("READ_TIMEOUT", "3s")
	t.Setenv("WRITE_TIMEOUT", "4s")
	t.Setenv("SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("TRUSTED_PROXIES", "7")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := map[string]time.Duration{
		"DB_TIMEOUT":       cfg.DBTimeout,
		"REQUEST_TIMEOUT":  cfg.RequestTimeout,
		"READ_TIMEOUT":     cfg.ReadTimeout,
		"WRITE_TIMEOUT":    cfg.WriteTimeout,
		"SHUTDOWN_TIMEOUT": cfg.ShutdownTimeout,
	}
	want := map[string]time.Duration{
		"DB_TIMEOUT":       1 * time.Second,
		"REQUEST_TIMEOUT":  2 * time.Second,
		"READ_TIMEOUT":     3 * time.Second,
		"WRITE_TIMEOUT":    4 * time.Second,
		"SHUTDOWN_TIMEOUT": 5 * time.Second,
	}

	for key, w := range want {
		if got[key] != w {
			t.Errorf("%s virou %v, queria %v: a env caiu no campo errado", key, got[key], w)
		}
	}

	if cfg.TrustedProxies != 7 {
		t.Errorf("TrustedProxies = %d, queria 7", cfg.TrustedProxies)
	}
}
