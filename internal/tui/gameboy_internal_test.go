package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestTheGameBoyFrameIsTheSizeTheGuardClaims(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 200, 80

	frame := m.render()
	rows := strings.Count(frame, "\n") + 1

	if rows != gbRows {
		t.Errorf("o quadro tem %d linhas e a guarda promete %d", rows, gbRows)
	}
	if cols := lipgloss.Width(frame); cols != m.width {
		t.Errorf("o quadro tem %d colunas numa tela de %d: sobra fundo do terminal em volta", cols, m.width)
	}

	// O aparelho fica centrado dentro do quadro, e e ele que a guarda dimensiona.
	device := 0
	for line := range strings.SplitSeq(ansi.Strip(frame), "\n") {
		device = max(device, lipgloss.Width(strings.TrimSpace(line)))
	}

	if device != gbCols {
		t.Errorf("o aparelho desenhado mede %d colunas e a guarda promete %d", device, gbCols)
	}
}

func TestATerminalTooSmallForTheMapFallsBackToTheNumbers(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 80, 24

	frame := m.render()
	if !strings.Contains(frame, "DECISAO") {
		t.Errorf("sem espaco para o mapa deveria cair nos numeros:\n%s", frame)
	}
	if !strings.Contains(ansi.Strip(frame), "mapa precisa de") {
		t.Errorf("a troca de tela precisa se explicar:\n%s", frame)
	}
}

// Linha curta demais nao apaga o resto do que estava ali: o terminal so reescreve as colunas
// que a linha nova ocupa, e o texto do frame anterior fica aparecendo depois do fim dela.
func TestNenhumaLinhaDoQuadroFicaCurtaDemais(t *testing.T) {
	t.Parallel()

	for _, size := range [][2]int{{200, 80}, {88, 35}, {120, 40}} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height = size[0], size[1]

		for _, message := range []string{
			"",
			"sem grana: custa 5247,03 TC e faltam 1702,19 TC",
			"ok",
		} {
			for _, open := range []bool{false, true} {
				m.message = message
				m.menu = nil
				if open {
					m.menu = tankMenu(m.snapshot, m.snapshot.Tanks[0], m.snapshot.Tanks[0].Batches[0], true)
				}

				frame := m.render()
				want := lipgloss.Width(frame)

				for i, line := range strings.Split(frame, "\n") {
					if got := lipgloss.Width(line); got != want {
						t.Errorf("em %dx%d com a mensagem %q, menu aberto=%v, a linha %d tem %d colunas de %d: o resto dela fica com o frame anterior",
							size[0], size[1], message, open, i, got, want)
					}
				}
			}
		}
	}
}

// fillTo e a rede: com a caixa e a barra ja saindo na largura certa, nada no quadro de hoje
// a exercita. Ela existe para a proxima string que encolher, entao e cobrada aqui direto.
func TestFillToCompletaALinhaCurta(t *testing.T) {
	t.Parallel()

	frame := fillTo(10, []string{"abc", "abcdefghij", "ab\ncdefg"})

	for i, line := range strings.Split(frame, "\n") {
		if got := lipgloss.Width(line); got != 10 {
			t.Errorf("a linha %d ficou com %d colunas de 10: %q", i, got, ansi.Strip(line))
		}
	}
}

// Quadro que encolhe nao apaga as linhas que ocupava: o terminal so reescreve as que o novo
// frame tem, e o rodape do anterior fica na tela. Fechar um menu de sete linhas e voltar ao
// painel de tres e exatamente esse caso.
func TestOQuadroTemSempreAMesmaAltura(t *testing.T) {
	t.Parallel()

	for _, size := range [][2]int{{200, 80}, {88, 35}, {120, 40}} {
		m := New(nil)
		m.snapshot = sizedSnapshot()
		m.width, m.height = size[0], size[1]

		m.menu, m.message = nil, ""
		closed := strings.Count(m.render(), "\n")

		// Todo menu que o jogo abre, com e sem mensagem: e a diferenca de altura entre dois
		// frames que fica na tela, e cada menu tem um tamanho.
		menus := map[string]*menu{
			"tanque": tankMenu(m.snapshot, m.snapshot.Tanks[0], m.snapshot.Tanks[0].Batches[0], true),
			"galpao": shedMenu(m.snapshot, m.snapshot.Tanks[0]),
		}

		for name, open := range menus {
			for _, message := range []string{
				"",
				"Sem grana: custa 5247,03 TC e faltam 1702,19 TC",
				// Recusa comprida: ela quebra em varias linhas dentro da caixa e e assim
				// que o corpo passa do que o mapa tem para devolver.
				strings.Repeat("o credito que sobra nao paga um ciclo inteiro: pague o que deve antes. ", 4),
			} {
				m.menu, m.message = open, message

				if got := strings.Count(m.render(), "\n"); got != closed {
					t.Errorf("em %dx%d o menu %q com a mensagem %q da %d linhas e o painel da %d: a diferenca fica na tela",
						size[0], size[1], name, message, got+1, closed+1)
				}
			}
		}
	}
}
