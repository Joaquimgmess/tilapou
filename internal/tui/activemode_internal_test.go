package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Quando o mapa nao cabe, a tela cai para o painel de numeros mas o modelo continua em modo
// mapa por dentro: as setas movem um boneco invisivel e o z cai no vazio. O jogador ve os
// numeros e as teclas respondem a um mapa que ninguem desenhou.
func TestComOMapaSemCaberAsTeclasFalamComOPainelQueEstaNaTela(t *testing.T) {
	t.Parallel()

	novo := func() Model {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height = 100, 32
		m.mode = ModeGameBoy

		return m
	}

	m := novo()
	if m.fitsGameBoy() {
		t.Fatal("100x32 comporta o mapa: este teste precisa do tamanho em que ele nao cabe")
	}

	depois, _ := m.onKey(tea.KeyPressMsg{Code: 'z', Text: "z"})
	aberto, ok := depois.(Model)
	if !ok {
		t.Fatal("onKey nao devolveu um Model")
	}

	if strings.Contains(aberto.message, "nao ha nada aqui") {
		t.Errorf("z num painel de numeros respondeu com a recusa do mapa: %q", aberto.message)
	}
	if aberto.menu == nil {
		t.Error("z num painel de numeros nao abriu o menu do tanque selecionado")
	}

	antes := novo()
	movido, _ := antes.onKey(tea.KeyPressMsg{Code: tea.KeyDown})
	desceu, ok := movido.(Model)
	if !ok {
		t.Fatal("onKey nao devolveu um Model")
	}

	if desceu.selected == antes.selected {
		t.Errorf("a seta para baixo nao moveu o cursor da tabela: continua em %d", desceu.selected)
	}
}

// O rodape e o menu nao anunciam o mapa numa tela que nao o comporta — e continuam
// anunciando onde ele cabe, que e o par que prova que a guarda nao apagou a funcao.
func TestMapaSoEAnunciadoOndeEleCabe(t *testing.T) {
	t.Parallel()

	novo := func(w, h int) Model {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height = w, h

		return m
	}

	apertado := novo(100, 32)
	if got := strings.Join(apertado.keyHints(), " "); strings.Contains(got, "tab mapa") {
		t.Errorf("o rodape promete o mapa em 100x32, onde ele nao cabe: %q", got)
	}

	folgado := novo(120, 40)
	if got := strings.Join(folgado.keyHints(), " "); !strings.Contains(got, "tab mapa") {
		t.Errorf("o rodape deixou de citar o mapa em 120x40, onde ele cabe: %q", got)
	}

	snap := sizedSnapshot()
	semMapa := tankMenu(snap, snap.Tanks[0], snap.Tanks[0].Batches[0], false)
	for _, item := range semMapa.items {
		if strings.Contains(item.label, "painel de numeros") {
			t.Error("o menu oferece ver o painel de numeros que ja esta na tela")
		}
	}

	comMapa := tankMenu(snap, snap.Tanks[0], snap.Tanks[0].Batches[0], true)
	achou := false
	for _, item := range comMapa.items {
		if strings.Contains(item.label, "painel de numeros") {
			achou = true
		}
	}
	if !achou {
		t.Error("o menu deixou de oferecer o painel de numeros onde o mapa esta na tela")
	}
}

// tab numa tela que nao comporta o mapa responde com a medida que falta, em vez de trocar
// para uma tela identica com as teclas mudas.
func TestTabRecusaComAMedidaQuandoOMapaNaoCabe(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 100, 32
	m.mode = ModeDashboard

	depois, _ := m.onCommand("tab")
	got, ok := depois.(Model)
	if !ok {
		t.Fatal("onCommand nao devolveu um Model")
	}

	if got.mode != ModeDashboard {
		t.Error("tab trocou para o mapa numa tela que nao o comporta")
	}
	if !strings.Contains(got.message, "88") {
		t.Errorf("tab recusou sem dizer a medida que falta: %q", got.message)
	}
}

// A recusa do tab nao pode depender de qual modo o modelo guarda por dentro: em 100x32 ele
// nasce em modo mapa, e guardar so o caso do painel deixava o primeiro tab passar reto,
// trocando o modo em silencio e apagando a linha que explica a medida que falta.
func TestTabRecusaNosDoisModosQuandoOMapaNaoCabe(t *testing.T) {
	t.Parallel()

	for _, mode := range []Mode{ModeGameBoy, ModeDashboard} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height = 100, 32
		m.mode = mode

		depois, _ := m.onCommand("tab")
		got, ok := depois.(Model)
		if !ok {
			t.Fatal("onCommand nao devolveu um Model")
		}

		if got.mode != mode {
			t.Errorf("com o modo %v o tab trocou para %v numa tela que nao comporta o mapa", mode, got.mode)
		}
		if !strings.Contains(got.message, "88") {
			t.Errorf("com o modo %v o tab nao disse a medida que falta: %q", mode, got.message)
		}
	}
}
