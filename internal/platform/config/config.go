package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	LogLevel        slog.Level
	DBMaxConns      int32
	DBTimeout       time.Duration
	RequestTimeout  time.Duration
	TrustedProxies  int32
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

var errMissingDatabaseURL = errors.New("config: DATABASE_URL is required")

const (
	defaultMaxConns      = 10
	defaultDBTimeout     = 3 * time.Second
	defaultReqTimeout    = 30 * time.Second
	defaultReadTimeout   = 10 * time.Second
	defaultWriteTimeout  = 15 * time.Second
	defaultShutdownGrace = 15 * time.Second
)

func Load() (Config, error) {
	var err error

	cfg := Config{
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errMissingDatabaseURL
	}

	cfg.LogLevel, err = levelEnv("LOG_LEVEL", slog.LevelInfo)
	if err != nil {
		return Config{}, err
	}

	cfg.DBMaxConns, err = int32Env("DB_MAX_CONNS", defaultMaxConns)
	if err != nil {
		return Config{}, err
	}

	cfg.TrustedProxies, err = int32Env("TRUSTED_PROXIES", 0)
	if err != nil {
		return Config{}, err
	}

	durations := []struct {
		key      string
		fallback time.Duration
		dst      *time.Duration
	}{
		{"DB_TIMEOUT", defaultDBTimeout, &cfg.DBTimeout},
		{"REQUEST_TIMEOUT", defaultReqTimeout, &cfg.RequestTimeout},
		{"READ_TIMEOUT", defaultReadTimeout, &cfg.ReadTimeout},
		{"WRITE_TIMEOUT", defaultWriteTimeout, &cfg.WriteTimeout},
		{"SHUTDOWN_TIMEOUT", defaultShutdownGrace, &cfg.ShutdownTimeout},
	}
	for _, d := range durations {
		*d.dst, err = durationEnv(d.key, d.fallback)
		if err != nil {
			return Config{}, err
		}
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func levelEnv(key string, fallback slog.Level) (slog.Level, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return 0, fmt.Errorf("config: invalid %s: %w", key, err)
	}

	return level, nil
}

func int32Env(key string, fallback int32) (int32, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s: %w", key, err)
	}
	return int32(v), nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s: %w", key, err)
	}
	return v, nil
}
