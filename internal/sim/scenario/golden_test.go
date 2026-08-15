package scenario_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/sim"
	"github.com/Joaquimgmess/tilapou/internal/sim/scenario"
)

var update = flag.Bool("update", false, "reescreve os arquivos golden")

func loadBalance(t *testing.T) *sim.Balance {
	t.Helper()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("balance.Load() error = %v", err)
	}

	return &b
}

func TestScenariosMatchGolden(t *testing.T) {
	t.Parallel()

	b := loadBalance(t)

	for _, s := range scenario.All() {
		t.Run(s.Name, func(t *testing.T) {
			t.Parallel()

			result, err := scenario.Run(s, b)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			got := result.Render()
			path := filepath.Join("testdata", s.Name+".golden")

			if *update {
				if werr := os.WriteFile(path, []byte(got), 0o600); werr != nil {
					t.Fatalf("gravando golden: %v", werr)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("lendo golden (rode com -update): %v", err)
			}
			if got != string(want) {
				t.Errorf("cenario %s divergiu do golden\n--- obtido ---\n%s", s.Name, got)
			}
		})
	}
}

func TestScenariosSurvivePartitioning(t *testing.T) {
	t.Parallel()

	b := loadBalance(t)

	for _, s := range scenario.All() {
		t.Run(s.Name, func(t *testing.T) {
			t.Parallel()

			whole, err := scenario.RunWhole(s, b)
			if err != nil {
				t.Fatalf("RunWhole() error = %v", err)
			}

			daily, err := scenario.Run(s, b)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if whole != daily.Final {
				t.Error("rodar de uma vez difere de rodar dia a dia")
			}
		})
	}
}

func TestDensityScenarioDivergesFromAeratedOne(t *testing.T) {
	t.Parallel()

	b := loadBalance(t)

	crowded, ok := scenario.ByName("densidade-contra-oxigenio")
	if !ok {
		t.Fatal("cenario densidade-contra-oxigenio sumiu")
	}
	aerated, ok := scenario.ByName("aerador-salva")
	if !ok {
		t.Fatal("cenario aerador-salva sumiu")
	}

	without, err := scenario.RunWhole(crowded, b)
	if err != nil {
		t.Fatalf("RunWhole() error = %v", err)
	}
	with, err := scenario.RunWhole(aerated, b)
	if err != nil {
		t.Fatalf("RunWhole() error = %v", err)
	}

	if with.Fish() <= without.Fish() {
		t.Errorf("aerador nao mudou o resultado: com=%d sem=%d, o trade-off virou cosmetico",
			with.Fish(), without.Fish())
	}
}
