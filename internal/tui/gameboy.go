package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

const (
	boxPadding = 2
	menuKeys   = "setas escolhem   j/k move   z confirma   x fecha"
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
func (m Model) mapScreenRows(chrome int) int {
	drawn := m.farm.rows * gb.TileSize / rowsPerCell
	if m.height <= 0 {
		return drawn
	}

	return max(min(drawn, m.height-chrome), 1)
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

	box := boxStyle.Width(width - boxPadding)

	body := box.Render(m.dialogue())
	if m.menu != nil {
		body = box.Render(m.menuWithReply())
	}

	chrome := lipgloss.Height(hud) + lipgloss.Height(banner) + lipgloss.Height(body) +
		lipgloss.Height(m.renderGameBoyKeys())

	return strings.Join([]string{
		hud,
		banner,
		cropTo(renderMap(m.farm, m.you, snapshot, m.frame), m.mapScreenRows(chrome)),
		body,
		m.renderGameBoyKeys(),
	}, "\n")
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

func tankHeadline(t client.Tank) string {
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
func frontBatch(t client.Tank) (client.Batch, bool) {
	if len(t.Batches) == 0 {
		return client.Batch{}, false
	}

	return t.Batches[0], true
}

func tankAdvice(t client.Tank) string {
	front, ok := frontBatch(t)
	if !ok {
		return "tanque vazio: povoe com [s]"
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

func (m Model) renderGameBoyKeys() string {
	if m.menu != nil {
		return screenStyle.Render(menuKeys)
	}

	return screenStyle.Render("setas andam  z opcoes  f trato  c racao  a aerador  h despescar  tab numeros  q sai")
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

	m.you.facing = look
	if !m.farm.blocked(m.you.x+dx, m.you.y+dy) {
		m.you.x += dx
		m.you.y += dy
	}
	return m.clearStale()
}
