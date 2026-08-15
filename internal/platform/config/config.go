package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	LogLevel        string
	DBMaxConns      int32
	DBTimeout       time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LogLevel:    env("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL is required")
	}

	conns, err := intEnv("DB_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	cfg.DBMaxConns = int32(conns)

	durations := []struct {
		key      string
		fallback time.Duration
		dst      *time.Duration
	}{
		{"DB_TIMEOUT", 3 * time.Second, &cfg.DBTimeout},
		{"READ_TIMEOUT", 10 * time.Second, &cfg.ReadTimeout},
		{"WRITE_TIMEOUT", 15 * time.Second, &cfg.WriteTimeout},
		{"SHUTDOWN_TIMEOUT", 15 * time.Second, &cfg.ShutdownTimeout},
	}
	for _, d := range durations {
		v, err := durationEnv(d.key, d.fallback)
		if err != nil {
			return Config{}, err
		}
		*d.dst = v
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s: %w", key, err)
	}
	return v, nil
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
