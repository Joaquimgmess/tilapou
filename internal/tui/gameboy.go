package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

const (
	menuKeys = "setas escolhem   j/k move   z confirma   x fecha"
)

var (
	screenStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Lightest))).
			Foreground(lipgloss.Color(gb.Hex(gb.Darkest))).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(gb.Hex(gb.Dark))).
			Background(lipgloss.Color(gb.Hex(gb.Lightest))).
			Foreground(lipgloss.Color(gb.Hex(gb.Darkest))).
			Padding(0, 1)

	hudStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Dark))).
			Foreground(lipgloss.Color(gb.Hex(gb.Lightest))).
			Bold(true).
			Padding(0, 1)

	urgentStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Darkest))).
			Foreground(lipgloss.Color(gb.Hex(gb.Lightest))).
			Bold(true).
			Padding(0, 1)

	goalStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Light))).
			Foreground(lipgloss.Color(gb.Hex(gb.Darkest))).
			Padding(0, 1)

	pickedStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)

const (
	dialogueRows = 4
	gbChromeRows = 7
	gbCols       = minMapCols * gb.TileSize
	gbRows       = minMapRows*gb.TileSize/2 + gbChromeRows
)

// mapScreenRows is how many terminal rows are left for the map after the chrome.
// animation congela o quadro da animacao quando o dado ja e velho demais: peixe nadando com
// numero parado e a tela dizendo que esta viva quando o que ela mostra ja nao vale.
func (m Model) animation() int {
	if m.staleTicks >= staleFreeze {
		return 0
	}

	return m.frame
}

// mapSpace e quantas linhas o mapa desenha e quantas sobram de vidro para o quadro nao
// encolher quando o menu fecha.
type mapSpace struct {
	drawn int
	glass int
}

// mapRows fixa a altura do quadro na altura dele em repouso — mapa inteiro mais o cromo sem
// menu — e reparte o que sobra entre mapa e vidro.
func (m Model) mapRows(width, chrome int) mapSpace {
	drawn := m.farm.rows * gb.TileSize / rowsPerCell

	total := drawn + m.chromeAtRest(width)
	if m.height > 0 {
		total = min(total, m.height)
	}

	room := max(total-chrome, 1)

	return mapSpace{drawn: min(drawn, room), glass: max(room-drawn, 0)}
}

// chromeAtRest mede o cromo com o menu fechado, que e a altura que o aparelho tem de manter.
func (m Model) chromeAtRest(width int) int {
	// Sem menu e sem mensagem: uma recusa comprida aumentaria a propria altura de
	// referencia, e o aparelho cresceria junto com o texto em vez de cortar o texto.
	rest := m
	rest.menu, rest.message = nil, ""

	return lipgloss.Height(hudStyle.Width(width).Render(" ")) +
		lipgloss.Height(goalStyle.Width(width).Render(" ")) +
		lipgloss.Height(boxStyle.Width(width).Render(rest.dialogue())) +
		lipgloss.Height(rest.renderGameBoyKeys(width))
}

func cropTo(frame string, rows int) string {
	lines := strings.Split(frame, "\n")
	if rows >= len(lines) {
		return frame
	}
	if rows < 1 {
		rows = 1
	}

	return strings.Join(lines[:rows], "\n")
}

func (m Model) fitsGameBoy() bool {
	if m.width <= 0 {
		return true
	}

	return m.width >= gbCols && m.height >= gbRows
}

func (m Model) renderGameBoy() string {
	snapshot := m.snapshot

	debt := ""
	if snapshot.Debt > 0 {
		debt = "   divida " + coins(snapshot.Debt)
	}

	width := m.farm.cols * gb.TileSize

	hud := hudStyle.Width(width).Render(fmt.Sprintf("TILAPOU   %s%s   peixe %s/kg   equiv %s   dia %d",
		coins(snapshot.CashCents), debt, coins(snapshot.Prices.FishKgCents),
		ratio(snapshot.Prices.RatioPPM), snapshot.Tick/(hoursPerDay*minutesPerHour)))

	goal, urgent := objective(snapshot, m.tankID())
	banner := goalStyle.Width(width).Render("OBJETIVO: " + goal)
	if urgent {
		banner = urgentStyle.Width(width).Render("! " + goal)
	}

	box := boxStyle.Width(width)

	body := box.Render(m.dialogue())
	if m.menu != nil {
		body = box.Render(m.menuWithReply())
	}

	keys := m.renderGameBoyKeys(width)

	// O menu e conteudo, e conteudo nao muda o tamanho do aparelho: ele come linha do mapa e
	// devolve ao fechar. Sem a altura em repouso o quadro encolheria junto com o menu, e o
	// rodape do frame anterior ficaria na tela.
	//
	chrome := lipgloss.Height(hud) + lipgloss.Height(banner) + lipgloss.Height(keys) +
		lipgloss.Height(body)

	rows := m.mapRows(width, chrome)

	blocks := []string{hud, banner, cropTo(renderMap(m.farm, m.you, snapshot, m.animation()), rows.drawn)}
	if rows.glass > 0 {
		blocks = append(blocks, strings.Repeat("\n", rows.glass-1))
	}

	frame := fillTo(width, append(blocks, body, keys))

	// O mapa tem teto de tamanho, entao numa tela grande sobra faixa com o fundo do
	// terminal em volta: centralizar e pintar mantem a ilusao de um aparelho.
	if m.width <= 0 || m.width <= width {
		return frame
	}

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, frame,
		lipgloss.WithWhitespaceStyle(marginStyle))
}

// fillTo cola os blocos num quadro opaco de width colunas. Linha curta nao apaga o resto do
// que estava ali: o terminal so reescreve as colunas que ela ocupa, e o texto do frame
// anterior continua aparecendo depois do fim dela. Preenche com o fundo do vidro, e nao com o
// da margem, que cortaria a tela com a cor do chassi.
func fillTo(width int, blocks []string) string {
	// Sem o padding do screenStyle, que alargaria em duas colunas o preenchimento que existe
	// justamente para a linha bater na largura.
	glass := screenStyle.UnsetPadding()

	lines := strings.Split(strings.Join(blocks, "\n"), "\n")
	for i, line := range lines {
		if short := width - lipgloss.Width(line); short > 0 {
			lines[i] = line + glass.Render(strings.Repeat(" ", short))
		}
	}

	return strings.Join(lines, "\n")
}

// menuWithReply keeps the refusal visible while the menu is open: without it the answer
// lands in the dialogue box behind the menu and the key looks like it did nothing.
func (m Model) menuWithReply() string {
	rendered := renderMenu(m.menu)
	if m.message == "" {
		return rendered
	}

	return dangerStyle.Render(m.message) + "\n" + rendered
}

func renderMenu(current *menu) string {
	lines := make([]string, 0, len(current.items)+1)
	lines = append(lines, current.title)

	for i := range current.items {
		item := &current.items[i]
		mark, label := "  ", item.label
		if i == current.cursor {
			mark, label = "> ", pickedStyle.Render(item.label)
		}
		if !item.enabled {
			label = dimStyle.Render(item.label)
		}
		lines = append(lines, mark+label+dimStyle.Render("  "+item.hint))
	}

	return strings.Join(lines, "\n")
}

func (m Model) dialogue() string {
	if m.message != "" {
		return m.message + "\n" + dimStyle.Render("z abre as opcoes de onde voce esta")
	}

	x, y := m.you.ahead()
	if index, ok := m.farm.pondAt(x, y); ok && index < len(m.snapshot.Tanks) {
		tank := m.snapshot.Tanks[index]

		return tankHeadline(tank) + "\n" + tankAdvice(tank) + dimStyle.Render("   [z] opcoes")
	}
	if x == m.farm.shedX() && y == m.farm.shedY() {
		return "GALPAO DE RACAO\n" + dimStyle.Render("[z] abre as compras")
	}

	return mapLegend(m.snapshot) + "\n" +
		dimStyle.Render("ande com as setas ate o viveiro ou o galpao, z abre as opcoes")
}

func tankHeadline(t api.Tank) string {
	front, ok := frontBatch(t)
	if !ok {
		return fmt.Sprintf("TANQUE %d: vazio", t.ID)
	}

	lotes := ""
	if len(t.Batches) > 1 {
		lotes = fmt.Sprintf(" em %d lotes", len(t.Batches))
	}

	return fmt.Sprintf("TANQUE %d: %d peixes de %d g%s", t.ID, t.Fish, front.MeanGrams, lotes)
}

// frontBatch is the batch the map talks about: the tank has one line, so it speaks for the
// oldest batch, which is the one closest to being sold.
func frontBatch(t api.Tank) (api.Batch, bool) {
	if len(t.Batches) == 0 {
		return api.Batch{}, false
	}

	return t.Batches[0], true
}

func tankAdvice(t api.Tank) string {
	front, ok := frontBatch(t)
	if !ok {
		return "tanque vazio: " + emptyTankAdvice(t)
	}

	switch {
	case t.OxygenUgL < criticalOxygenUgL && !t.Aerating:
		return "a agua esta sufocando! ligue o aerador"
	case t.FeedKg == 0:
		return "a racao acabou, va ate o galpao"
	case t.ServedFor <= 0:
		return "os peixes estao sem trato servido"
	case front.Ready:
		return "os peixes estao no ponto de abate"
	}

	next := ""
	if front.NextClassGrams > 0 && front.NextClassGrams > front.MeanGrams {
		next = fmt.Sprintf("   proxima classe em %d g", front.NextClassGrams)
	}

	return fmt.Sprintf("%s/kg agora   racao %d kg   trato por %s%s",
		coins(front.PriceKgCents), t.FeedKg, minutes(t.ServedFor), next)
}

func (m Model) renderGameBoyKeys(width int) string {
	if m.menu != nil {
		return screenStyle.Width(width).Render(menuKeys)
	}

	// A barra do mapa cabe em 88 colunas contadas: com o alerta em outro tanque, o pulo
	// entra no lugar do que o jogador descobre sozinho.
	if m.jumpKeyHint() != "" {
		return screenStyle.Width(width).Render("setas andam  z opcoes  f trato  c racao  a aerador  h despesca  . alerta  q sai")
	}

	return screenStyle.Width(width).Render("setas andam  z opcoes  f trato  c racao  a aerador  h despescar  tab numeros  q sai")
}

type targetKind uint8

// What the avatar has in front of it.
const (
	targetNone targetKind = iota
	targetTank
	targetShed
)

func (m Model) target() (index int, kind targetKind) {
	x, y := m.you.ahead()

	if pond, ok := m.farm.pondAt(x, y); ok && pond < len(m.snapshot.Tanks) {
		return pond, targetTank
	}
	if x == m.farm.shedX() && y == m.farm.shedY() {
		return 0, targetShed
	}

	return 0, targetNone
}

func (m Model) move(dx, dy int, look facing) Model {
	if m.menu != nil {
		m.menu.move(dy)

		return m
	}

	turned := m.you.facing != look
	m.you.facing = look

	if !m.farm.blocked(m.you.x+dx, m.you.y+dy) {
		m.you.x += dx
		m.you.y += dy

		return m.clearStale()
	}

	// Virar ja e resposta; esbarrar de frente para o obstaculo nao mexia um pixel.
	if turned {
		return m.clearStale()
	}

	return m.say(blockedMessage(m.farm, m.you.x+dx, m.you.y+dy))
}

func blockedMessage(farm farmMap, x, y int) string {
	if _, ok := farm.pondAt(x, y); ok {
		return "nao da para atravessar o viveiro: ande em volta"
	}
	if x == farm.shedX() && y == farm.shedY() {
		return "o galpao esta fechado desse lado: entre pela frente"
	}

	return "a cerca fecha aqui"
}
