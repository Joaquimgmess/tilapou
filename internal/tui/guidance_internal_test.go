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

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/client"
)

const blockNoCredit = "no_credit"

var offeredKey = regexp.MustCompile(`\[(\w|\.)\]`)

func adviceCases() map[string]func(api.Snapshot) api.Snapshot {
	return map[string]func(api.Snapshot) api.Snapshot{
		"sem oxigenio": func(s api.Snapshot) api.Snapshot {
			s.Tanks[1].OxygenUgL, s.Tanks[1].Aerating = 900, false

			return s
		},
		"doente": func(s api.Snapshot) api.Snapshot {
			s.Tanks[1].Batches[0].Sick = true

			return s
		},
		"sem racao": func(s api.Snapshot) api.Snapshot {
			s.Tanks[1].FeedKg = 0

			return s
		},
		"sem trato": func(s api.Snapshot) api.Snapshot {
			s.Tanks[1].ServedFor = 0

			return s
		},
		"no ponto de abate": func(s api.Snapshot) api.Snapshot {
			s.Tanks[1].Batches[0].Ready = true

			return s
		},
		"automacao ao alcance": func(s api.Snapshot) api.Snapshot {
			s.Tanks[0].Upgrades = everyUpgrade()
			s.Tanks[0].Upgrades[feederIndex].Owned = true
			s.Tanks[0].Upgrades[aeratorIndex].Owned = true
			s.Tanks[1].Upgrades = everyUpgrade()
			s.CashCents = 10_000_000

			return s
		},
		"abaixo do break-even": func(s api.Snapshot) api.Snapshot {
			s.Tanks[1].Fish, s.Tanks[1].BreakEven, s.Tanks[1].StockAdvice = 10, 1_800, 5_000

			return s
		},
	}
}

func tankScopedAdvice(t *testing.T) api.Snapshot {
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
	d.refresh()

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
			d.refresh()

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

func semCreditoNenhum(s api.Snapshot) api.Snapshot {
	for i := range s.Tanks {
		s.Tanks[i].LoanAdvice, s.Tanks[i].LoanBlock = 0, blockNoCredit
	}

	return s
}

func TestConselhoNaoMandaPegarCreditoComOLimiteEstourado(t *testing.T) {
	t.Parallel()

	casos := map[string]func(api.Snapshot) api.Snapshot{
		"abaixo do break-even": func(s api.Snapshot) api.Snapshot {
			s.Tanks[0].Fish, s.Tanks[0].BreakEven, s.Tanks[0].StockAdvice = 10, 1_800, 0
			s.Tanks[1].Fish, s.Tanks[1].BreakEven = 1_400, 100

			return s
		},
		"folego menor que o ciclo": func(s api.Snapshot) api.Snapshot {
			s.RunwayDays = 1
			s.Tanks[0].Batches[0].Decision.HoldDays = 60

			return s
		},
		"tanque vazio e sem grana": func(s api.Snapshot) api.Snapshot {
			s.Tanks[0].Fish, s.Tanks[0].Batches[0].Fish = 0, 0
			s.Tanks[1].Fish, s.Tanks[1].Batches[0].Fish = 0, 0
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
	snap.NextTankCents = 524_703

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

func TestAcontecimentoNovoVeioParaATela(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		event api.Event
		want  string
	}{
		"falencia":       {event: api.Event{Seq: 9, Kind: "bankrupt", CashCents: 4_500_000}, want: "quebrou"},
		"doenca":         {event: api.Event{Seq: 9, Kind: "disease", Tank: 2}, want: "tanque 2"},
		"mortandade":     {event: api.Event{Seq: 9, Kind: "starvation_ended", Tank: 1, Fish: 449}, want: "449 peixes"},
		"morte de um so": {event: api.Event{Seq: 9, Kind: "starvation_ended", Tank: 1, Fish: 1}, want: "1 peixe morreu"},
		"fome comecando": {event: api.Event{Seq: 9, Kind: "starvation_began", Tank: 1, Fish: 1}, want: "comecou a morrer de fome"},
		"tilapada":       {event: api.Event{Seq: 9, Kind: "prestiged"}, want: "tilap"},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			snap := sizedSnapshot()
			snap.Events = []api.Event{tc.event}

			told, ok := headline(snap, 0)
			if !ok {
				t.Fatalf("o acontecimento %q nao virou aviso nenhum", tc.event.Kind)
			}
			if !strings.Contains(told, tc.want) {
				t.Errorf("o aviso de %q e %q, sem dizer %q", tc.event.Kind, told, tc.want)
			}
		})
	}
}

func TestAcontecimentoJaVistoNaoVoltaAAvisar(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Events = []api.Event{{Seq: 7, Kind: "bankrupt"}}

	if _, ok := headline(snap, 7); ok {
		t.Error("o mesmo acontecimento avisou de novo depois de ja ter sido mostrado")
	}
	if _, ok := headline(snap, 6); !ok {
		t.Error("um acontecimento mais novo que o ultimo visto nao avisou")
	}
}

func TestOHistoricoDaAberturaNaoViraNovidade(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Events = []api.Event{{Seq: 3, Kind: "bankrupt"}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.refresh()

	if said := d.model.(Model).message; strings.Contains(said, "quebrou") {
		t.Errorf("o historico da abertura foi anunciado como novidade: %q", said)
	}

	snap.Events = []api.Event{{Seq: 4, Kind: "bankrupt", CashCents: 100}, {Seq: 3, Kind: "bankrupt"}}
	d.refresh()

	if said := d.model.(Model).message; !strings.Contains(said, "quebrou") {
		t.Errorf("o acontecimento novo nao foi anunciado: %q", said)
	}
}

func TestConfirmacaoSobreviveAoProximoPasso(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	for _, key := range []string{keyUp, keyDown, keyLeft, keyRight, jumpKey} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
			d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
			d.refresh()

			d.press("f")

			said := d.model.(Model).message
			if said == "" {
				t.Fatal("servir o trato nao confirmou nada")
			}

			d.press(key)

			if got := d.model.(Model).message; got != said {
				t.Errorf("apertar %q no mesmo quadro apagou a confirmacao %q, sobrou %q", key, said, got)
			}
		})
	}
}

func TestOAvisoEhDoAcontecimentoMaisNovo(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	// Como chega da API: do mais novo para o mais velho, e a fome enche o historico.
	snap.Events = []api.Event{
		{Seq: 12, Kind: "disease", Tank: 2},
		{Seq: 11, Kind: "starvation_ended", Tank: 1, Fish: 3},
		{Seq: 10, Kind: "starvation_deaths", Tank: 1, Fish: 2},
	}

	told, ok := headline(snap, 9)
	if !ok {
		t.Fatal("nenhum acontecimento virou aviso")
	}
	if !strings.Contains(told, "Surto") {
		t.Errorf("o aviso foi %q: o surto ficou escondido atras da fome mais velha", told)
	}
}

func TestOSurtoNaoSeAfogaNaEnxurradaDeFome(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Events = []api.Event{{Seq: 500, Kind: "disease", Tank: 2}}

	// A fome emite uma linha por tick e enterra o surto que veio antes.
	for seq := uint64(499); seq > 400; seq-- {
		snap.Events = append([]api.Event{{Seq: seq + 100, Kind: "starvation_deaths", Tank: 1, Fish: 1}},
			snap.Events...)
	}

	told, ok := headline(snap, 400)
	if !ok {
		t.Fatal("nenhum acontecimento virou aviso")
	}
	if !strings.Contains(told, "Surto") {
		t.Errorf("o aviso foi %q: o surto que decide o lote perdeu para a enesima linha de fome", told)
	}
}

func TestFazendaSemPeixeEComCaixaMandaPovoar(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.CashCents = 398_900

	for i := range snap.Tanks {
		snap.Tanks[i].Fish, snap.Tanks[i].Batches[0].Fish = 0, 0
		snap.Tanks[i].Upgrades = everyUpgrade()
		snap.Tanks[i].StockAdvice, snap.Tanks[i].StockBlock = 500, api.StockOpen
	}

	told, _ := objective(snap, snap.Tanks[0].ID)
	if !strings.Contains(told, "[s]") {
		t.Errorf("com caixa e nenhum peixe, o conselho foi %q em vez de mandar povoar", told)
	}
}

func TestZNoVazioDizQueNaoTemNada(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.refresh()

	// De costas para o caminho, sem viveiro nem galpao na frente.
	d.press(keyDown)
	d.press("z")

	m, ok := d.model.(Model)
	if !ok {
		t.Fatal("o driver perdeu o Model")
	}
	if m.menu != nil {
		t.Fatal("o z abriu menu de frente para o vazio")
	}
	if m.message == "" {
		t.Error("o z no vazio nao respondeu nada, e a caixa promete que ele abre as opcoes")
	}
}

// O objetivo e a tecla precisam sair da mesma conta: mandar povoar quando povoar recusa deixa
// o jogador apertando uma tecla que o proprio jogo acabou de anunciar.
func TestOObjetivoNaoMandaPovoarQuandoOConselhoEZero(t *testing.T) {
	t.Parallel()

	// Caixa alto o bastante para os alevinos, mas nao para o custo fixo do ciclo: e assim
	// que o conselho volta zero com o caixa parecendo cheio.
	s := api.Snapshot{
		CashCents: 79_998,
		Prices:    api.Prices{FingerlingCents: 80},
		Tanks: []api.Tank{{
			ID: 1, Fish: 0, Capacity: 5_000, StockAdvice: 0,
		}},
	}

	if goal := farmGoal(s); strings.Contains(goal, "Povoe com [s]") {
		t.Errorf("com o conselho em zero o objetivo ainda manda povoar: %q", goal)
	}
}
