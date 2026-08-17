// Package logging monta o logger estruturado do processo.
package logging

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

// New cria um logger JSON em stdout.
func New(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

// WithLogger guarda logger no context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext devolve o logger do context, ou slog.Default.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
