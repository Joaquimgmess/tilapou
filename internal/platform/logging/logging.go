// Package logging builds the structured logger of the process. Zap is the handler
// underneath, and it does not leave this package: everyone else speaks slog.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

// Service is what every line carries, whatever the layer that wrote it.
type Service struct {
	Name    string
	Env     string
	Version string
}

type contextKey struct{}

// New creates the JSON logger on stdout, stamping every line with the service.
func New(level slog.Level, service Service) *slog.Logger {
	return NewTo(os.Stdout, level, service)
}

// NewTo writes to w, so a test can read the lines it emits.
func NewTo(w io.Writer, level slog.Level, service Service) *slog.Logger {
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		NameKey:        "logger",
		CallerKey:      zapcore.OmitKey,
		StacktraceKey:  zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	})

	core := zapcore.NewCore(encoder, zapcore.AddSync(w), zapLevel(level))

	return slog.New(zapslog.NewHandler(core)).With(
		slog.String("service", service.Name),
		slog.String("env", service.Env),
		slog.String("version", service.Version),
	)
}

func zapLevel(level slog.Level) zapcore.Level {
	switch {
	case level <= slog.LevelDebug:
		return zapcore.DebugLevel
	case level <= slog.LevelInfo:
		return zapcore.InfoLevel
	case level <= slog.LevelWarn:
		return zapcore.WarnLevel
	}

	return zapcore.ErrorLevel
}

// WithLogger stores logger in the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the logger from the context, or a silent one. It never falls back to
// slog.Default: a package-level logger is global state, and whoever needs to log is given
// a logger at construction.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}

	return Discard()
}

// Discard returns a logger that writes nothing, for the paths that never got one.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
