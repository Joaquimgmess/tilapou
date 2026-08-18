package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const blockNoCredit = "no_credit"

var offeredKey = regexp.MustCompile(`\[(\w|\.)\]`)

func adviceCases() map[string]func(client.Snapshot) client.Snapshot {
	return map[string]func(client.Snapshot) client.Snapshot{
		"sem oxigenio": func(s client.Snapshot) client.Snapshot {
			s.Tanks[1].OxygenUgL, s.Tanks[1].Aerating = 900, false

			return s
		},
		"doente": func(s client.Snapshot) client.Snapshot {
			s.Tanks[1].Sick = true

			return s
		},
		"sem racao": func(s client.Snapshot) client.Snapshot {
			s.Tanks[1].FeedKg = 0

			return s
		},
		"sem trato": func(s client.Snapshot) client.Snapshot {
			s.Tanks[1].ServedFor = 0

			return s
		},
		"no ponto de abate": func(s client.Snapshot) client.Snapshot {
			s.Tanks[1].Ready = true

			return s
		},
		"automacao ao alcance": func(s client.Snapshot) client.Snapshot {
			s.Tanks[0].Upgrades = everyUpgrade()
			s.Tanks[0].Upgrades[feederIndex].Owned = true
			s.Tanks[0].Upgrades[aeratorIndex].Owned = true
			s.Tanks[1].Upgrades = everyUpgrade()
			s.CashCents = 10_000_000

			return s
		},
		"abaixo do break-even": func(s client.Snapshot) client.Snapshot {
			s.Tanks[1].Fish, s.Tanks[1].BreakEven, s.Tanks[1].StockAdvice = 10, 1_800, 5_000

			return s
		},
	}
}

func tankScopedAdvice(t *testing.T) client.Snapshot {
	t.Helper()

	s := sizedSnapshot()
	s.Tanks[0].OxygenUgL, s.Tanks[0].FeedKg, s.Tanks[0].ServedFor = 6_000, 400, 240
	s.Tanks[1].OxygenUgL, s.Tanks[1].FeedKg, s.Tanks[1].ServedFor = 6_000, 400, 240
	s.Tanks[1].Fish, s.Tanks[1].BreakEven = 1_400, 100
	s.RunwayDays = shortRunwayDays * 10

	return s
}

func TestConselhoNuncaOfereceTeclaQueAgeEmOutroTanque(t *testing.T) {
	t.Parallel()

	for name, mutate := range adviceCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			snap := mutate(tankScopedAdvice(t))

			found, ok := current(snap)
			if !ok || found.tank == 0 {
				t.Fatalf("o cenario nao disparou conselho de tanque: %+v", found)
			}
			if found.tank == snap.Tanks[0].ID {
				t.Fatal("o cenario precisa apontar para um tanque que nao e o focado")
			}

			if named := fmt.Sprintf("tanque %d", found.tank); !strings.Contains(found.text, named) {
				t.Errorf("o conselho %q fala do tanque %d sem nomea-lo: o jogador nao sabe para onde o %s leva",
					found.text, found.tank, jumpKey)
			}

			text, _ := objective(snap, snap.Tanks[0].ID)
			for _, key := range offeredKey.FindAllStringSubmatch(text, -1) {
				if key[1] != jumpKey {
					t.Errorf("o conselho %q oferece [%s] falando do tanque %d com o %d em foco",
						text, key[1], found.tank, snap.Tanks[0].ID)
				}
			}

			focused, _ := objective(snap, found.tank)
			if focused != found.text {
				t.Errorf("com o tanque %d em foco o conselho virou %q, queria %q", found.tank, focused, found.text)
			}
		})
	}
}

func TestPuloDoConselhoSelecionaOTanqueQueEleNomeia(t *testing.T) {
	t.Parallel()

	snap := adviceCases()["sem racao"](tankScopedAdvice(t))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.run(d.model.(Model).fetch())

	if got := d.model.(Model).selected; got != 0 {
		t.Fatalf("o cenario comeca com o tanque de indice %d selecionado, queria 0", got)
	}

	d.press(jumpKey)

	m, ok := d.model.(Model)
	if !ok {
		t.Fatal("o driver perdeu o Model")
	}
	if m.snapshot.Tanks[m.selected].ID != adviceTank(snap) {
		t.Errorf("o pulo deixou o tanque %d selecionado, o conselho fala do %d",
			m.snapshot.Tanks[m.selected].ID, adviceTank(snap))
	}
}

func TestTodaAcaoDeTanqueDizEmQualTanqueCaiu(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	for _, key := range []string{"f", "c", "a", "h"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
			d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
			d.run(d.model.(Model).fetch())

			m, ok := d.model.(Model)
			if !ok {
				t.Fatal("o driver perdeu o Model")
			}
			d.press(key)

			said := d.model.(Model).message
			want := fmt.Sprintf("no tanque %d", m.snapshot.Tanks[m.selected].ID)

			if !strings.Contains(said, want) {
				t.Errorf("a tecla %q confirmou com %q, sem dizer %q", key, said, want)
			}
		})
	}
}

func semCreditoNenhum(s client.Snapshot) client.Snapshot {
	for i := range s.Tanks {
		s.Tanks[i].LoanAdvice, s.Tanks[i].LoanBlock = 0, blockNoCredit
	}

	return s
}

func TestConselhoNaoMandaPegarCreditoComOLimiteEstourado(t *testing.T) {
	t.Parallel()

	casos := map[string]func(client.Snapshot) client.Snapshot{
		"abaixo do break-even": func(s client.Snapshot) client.Snapshot {
			s.Tanks[0].Fish, s.Tanks[0].BreakEven, s.Tanks[0].StockAdvice = 10, 1_800, 0
			s.Tanks[1].Fish, s.Tanks[1].BreakEven = 1_400, 100

			return s
		},
		"folego menor que o ciclo": func(s client.Snapshot) client.Snapshot {
			s.RunwayDays = 1
			s.Tanks[0].Decision.HoldDays = 60

			return s
		},
		"tanque vazio e sem grana": func(s client.Snapshot) client.Snapshot {
			s.Tanks[0].Fish, s.Tanks[0].BatchFish = 0, 0
			s.Tanks[1].Fish, s.Tanks[1].BatchFish = 0, 0
			s.CashCents = 0

			return s
		},
	}

	for name, mutate := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			snap := semCreditoNenhum(mutate(tankScopedAdvice(t)))

			text, _ := objective(snap, snap.Tanks[0].ID)
			if strings.Contains(text, "[g]") {
				t.Errorf("o conselho manda pegar credito com o limite estourado: %q", text)
			}
		})
	}
}

func TestComprarViveiroSemGranaDizOPrecoEOQueFalta(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.CashCents = 1_000
	snap.NextTankCents = 5_247_03

	shed := shedMenu(snap, snap.Tanks[0])

	var found *menuItem
	for i := range shed.items {
		if strings.Contains(shed.items[i].label, "viveiro") {
			found = &shed.items[i]
		}
	}

	if found == nil {
		t.Fatal("o galpao nao oferece comprar viveiro")
	}
	if !strings.Contains(found.label+found.hint, coins(snap.NextTankCents)) {
		t.Errorf("a opcao nao diz o preco: %q / %q", found.label, found.hint)
	}
	if found.enabled {
		t.Error("a opcao esta clicavel sem caixa para pagar")
	}
}

func TestRecusaApareceComOMenuAberto(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height, m.mode = 120, 40, ModeGameBoy
	m.menu = shedMenu(m.snapshot, m.snapshot.Tanks[0])
	m = m.say("Sem grana: custa 5247,03 TC e faltam 1702,19 TC")

	if !strings.Contains(m.render(), "faltam 1702,19 TC") {
		t.Error("a recusa nao aparece enquanto o menu esta aberto: o jogador aperta e nada responde")
	}
}
