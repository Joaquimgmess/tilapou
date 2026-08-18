package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/client"
)

func TestPainelUsaATelaInteira(t *testing.T) {
	t.Parallel()

	for _, size := range [][2]int{{120, 40}, {160, 50}, {200, 60}} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height, m.mode = size[0], size[1], ModeDashboard

		frame := m.render()
		if got := lipgloss.Height(frame); got != size[1] {
			t.Errorf("em %dx%d o painel desenhou %d linhas, sobrando %d mortas",
				size[0], size[1], got, size[1]-got)
		}

		for line := range strings.SplitSeq(frame, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Errorf("em %dx%d uma linha ficou com %d colunas", size[0], size[1], lipgloss.Width(line))

				break
			}
		}
	}
}

func TestDecisaoNaoQuebraAFraseComTelaLarga(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height, m.mode = 120, 40, ModeDashboard

	// A frase inteira tem de caber numa linha so: com largura fixa ela quebra entre o
	// "para pagar a" e o "manutencao", enquanto sobram colunas a direita.
	for line := range strings.SplitSeq(m.render(), "\n") {
		if strings.Contains(line, "minimo") && strings.Contains(line, "manutencao") {
			return
		}
	}

	t.Error("a linha de lotacao quebrou em duas mesmo com colunas livres a direita")
}

func TestOMenuCabeNoTamanhoMinimo(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Tanks[0].Upgrades = everyUpgrade()

	menus := map[string]*menu{
		"tanque": tankMenu(snap, snap.Tanks[0], snap.Tanks[0].Batches[0]),
		"galpao": shedMenu(snap, snap.Tanks[0]),
	}

	for name, overlay := range menus {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, size := range [][2]int{{gbCols, gbRows}, {gbCols, gbRows + 3}, {100, 38}, {120, 40}} {
				m := New(nil)
				m.snapshot = snap
				m.width, m.height, m.mode = size[0], size[1], ModeGameBoy
				m.menu = overlay
				// A recusa entra acima do menu, e e nesse quadro que o topo sumia.
				m = m.say("Sem grana: custa 5247,03 TC e faltam 1702,19 TC")

				frame := m.render()
				if got := lipgloss.Height(frame); got > size[1] {
					t.Errorf("em %dx%d o quadro saiu com %d linhas: o topo do menu rola para fora da tela",
						size[0], size[1], got)
				}

				// O item sob o cursor sublinha por palavra e injeta ANSI entre elas, entao a
				// comparacao e sobre o texto limpo.
				plain := ansi.Strip(frame)
				for _, want := range []string{overlay.title, overlay.items[0].label, "┏"} {
					if !strings.Contains(plain, want) {
						t.Errorf("em %dx%d o menu perdeu %q:\n%s", size[0], size[1], want, plain)
					}
				}
			}
		})
	}
}

func snapshotComDoisLotes() api.Snapshot {
	snap := sizedSnapshot()
	tank := &snap.Tanks[0]
	tank.Fish, tank.BatchCount = 423, 2
	tank.Batches[0].Fish, tank.Batches[0].MeanGrams = 276, 450
	tank.Batches = append(tank.Batches, api.Batch{ID: 7, Fish: 147, MeanGrams: 120})

	return snap
}

func TestOsDoisLotesDoTanqueViramDuasLinhas(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = snapshotComDoisLotes()
	m.width, m.height, m.mode = 120, 40, ModeDashboard

	plain := ansi.Strip(m.render())
	for _, want := range []string{"T1-L3", "T1-L7", "276", "147"} {
		if !strings.Contains(plain, want) {
			t.Errorf("a tabela nao mostra %q: o painel esconde lote\n%s", want, plain)
		}
	}
}

func TestJAndaPorLoteDentroDoTanque(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = snapshotComDoisLotes()
	m.width, m.height, m.mode = 120, 40, ModeDashboard

	first, firstTank := m.batchID(), m.tankID()

	m = m.selectDelta(1)

	second, secondTank := m.batchID(), m.tankID()
	if second == first {
		t.Errorf("j nao trocou de lote: continua no %d", first)
	}
	if secondTank != firstTank {
		t.Errorf("j saiu do tanque %d para o %d em vez de andar entre os lotes", firstTank, secondTank)
	}
}

func TestOTanqueVazioContinuaSelecionavel(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.Tanks[1].Fish, snap.Tanks[1].BatchCount, snap.Tanks[1].Batches = 0, 0, nil

	m := New(nil)
	m.snapshot = snap
	m.width, m.height, m.mode = 120, 40, ModeDashboard

	rows := m.rows()
	if len(rows) != 2 {
		t.Fatalf("um tanque com lote e um vazio geraram %d linhas, queria 2", len(rows))
	}

	m.selected = 1
	if got := m.tankID(); got != snap.Tanks[1].ID {
		t.Errorf("a linha do tanque vazio seleciona o tanque %d, queria %d", got, snap.Tanks[1].ID)
	}
	if _, ok := m.batch(); ok {
		t.Error("a linha do tanque vazio devolveu um lote")
	}
}

func TestATeclaDoAlertaApareceNaBarraQuandoServe(t *testing.T) {
	t.Parallel()

	comAlerta := sizedSnapshot()
	comAlerta.Tanks[1].Batches[0].Sick = true

	// Fazenda saudavel: o conselho vira de fazenda e nao aponta tanque nenhum.
	semAlerta := sizedSnapshot()
	semAlerta.Tanks = semAlerta.Tanks[:1]
	semAlerta.Tanks[0].OxygenUgL, semAlerta.Tanks[0].FeedKg, semAlerta.Tanks[0].ServedFor = 6_000, 400, 240
	semAlerta.Tanks[0].BreakEven = 100

	for _, mode := range []Mode{ModeGameBoy, ModeDashboard} {
		m := New(nil)
		m.snapshot = comAlerta
		m.width, m.height, m.mode = 120, 40, mode
		m.selected = 0

		if !strings.Contains(ansi.Strip(m.keyBar()), jumpKey+" alerta") {
			t.Errorf("modo %d: o alerta fala de outro tanque e a barra nao anuncia o %q: %q",
				mode, jumpKey, ansi.Strip(m.keyBar()))
		}
		if got := lipgloss.Width(m.render()); got > m.width {
			t.Errorf("modo %d: a barra com o %q estourou a largura, %d de %d", mode, jumpKey, got, m.width)
		}

		m.snapshot = semAlerta
		m.selected = 0

		if strings.Contains(ansi.Strip(m.keyBar()), jumpKey+" alerta") {
			t.Errorf("modo %d: a barra anuncia o %q sem alerta de outro tanque", mode, jumpKey)
		}
	}
}

func TestEsbarrarJaOlhandoParaOObstaculoResponde(t *testing.T) {
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

	// Primeira seta vira o avatar para o viveiro; a segunda ja o encontra virado.
	d.press(keyUp)

	before, ok := d.model.(Model)
	if !ok {
		t.Fatal("o driver perdeu o Model")
	}

	d.press(keyUp)

	after, ok := d.model.(Model)
	if !ok {
		t.Fatal("o driver perdeu o Model depois do segundo passo")
	}
	if after.you != before.you {
		t.Fatal("o cenario precisa de um obstaculo: o avatar andou")
	}
	if after.message == "" {
		t.Error("esbarrar de frente para o obstaculo nao respondeu nada")
	}
}

func TestOMapaFicaCentradoNaTelaGrande(t *testing.T) {
	t.Parallel()

	for _, size := range [][2]int{{200, 60}, {240, 80}} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height, m.mode = size[0], size[1], ModeGameBoy
		m.farm = m.resizedFarm(len(m.snapshot.Tanks))
		m.you = m.you.clampedTo(m.farm)

		frame := m.render()
		if got := lipgloss.Width(frame); got != size[0] {
			t.Errorf("em %dx%d o quadro tem %d colunas: sobram %d com o fundo do terminal",
				size[0], size[1], got, size[0]-got)
		}
	}
}
