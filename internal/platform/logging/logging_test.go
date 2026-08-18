package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
)

func lines(t *testing.T, raw []byte) []map[string]any {
	t.Helper()

	var out []map[string]any

	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}

		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("linha de log nao e JSON: %q: %v", line, err)
		}
		out = append(out, entry)
	}

	return out
}

func TestTodaLinhaCarregaOsCamposObrigatorios(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := logging.NewTo(&buf, slog.LevelDebug, logging.Service{Name: "tilapou", Env: "test", Version: "1.2.3"})
	logger.Info("farm advanced", slog.String("farm_id", "abc"))
	logger.Warn("advance truncated")

	entries := lines(t, buf.Bytes())
	if len(entries) != 2 {
		t.Fatalf("saiu %d linhas, queria 2", len(entries))
	}

	for _, entry := range entries {
		for _, key := range []string{"ts", "level", "msg", "service", "env", "version"} {
			if _, ok := entry[key]; !ok {
				t.Errorf("linha sem o campo obrigatorio %q: %v", key, entry)
			}
		}
	}
}

func TestNomeDeCampoEmSnakeCaseComUnidadeNumerica(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := logging.NewTo(&buf, slog.LevelDebug, logging.Service{Name: "tilapou", Env: "test", Version: "1.2.3"})
	logger.Info("http request",
		slog.String("request_id", "r1"),
		slog.Int64("duration_ms", 12),
		slog.Int64("cash_cents", 4_999),
		slog.Int64("mass_grams", 306),
	)

	for _, entry := range lines(t, buf.Bytes()) {
		for key, value := range entry {
			if key != strings.ToLower(key) || strings.Contains(key, "-") {
				t.Errorf("campo %q nao esta em snake_case", key)
			}

			for _, unit := range []string{"_ms", "_cents", "_grams", "_ppm"} {
				if !strings.HasSuffix(key, unit) {
					continue
				}
				if _, ok := value.(float64); !ok {
					t.Errorf("campo %q tem sufixo de unidade mas o valor nao e numero: %v", key, value)
				}
			}
		}
	}
}

func TestContextoSemLoggerNaoCaiEmGlobal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	logging.FromContext(t.Context()).Error("nao deveria sair em lugar nenhum")

	if buf.Len() != 0 {
		t.Errorf("o logger de contexto vazio escreveu no default global: %s", buf.String())
	}
}
