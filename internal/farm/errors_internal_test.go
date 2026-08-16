package farm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
)

var errDatabase = errors.New(`update farm 8f3a: pq: relation "farms" does not exist`)

func TestADatabaseErrorIsLoggedAndNeverReachesTheClient(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	ctx := logging.WithLogger(context.Background(),
		slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelError})))

	out := reportError(ctx, "get-farm", errDatabase)

	if strings.Contains(out.Error(), "pq:") || strings.Contains(out.Error(), "farms") {
		t.Errorf("o erro que sobe para o cliente carrega o texto interno: %q", out)
	}
	if !strings.Contains(logged.String(), "does not exist") {
		t.Errorf("o erro nao foi para o log do daemon: %q", logged.String())
	}
	if !strings.Contains(logged.String(), "get-farm") {
		t.Errorf("o log nao diz qual operacao falhou: %q", logged.String())
	}
}

func TestErrosDeUsoContinuamExplicando(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), slog.New(slog.DiscardHandler))

	out := reportError(ctx, "buy_upgrade", ErrMissingAuto)
	if !strings.Contains(out.Error(), "auto") {
		t.Errorf("o 422 perdeu a explicacao: %q", out)
	}
}
