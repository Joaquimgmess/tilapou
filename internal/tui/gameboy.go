package tui

import (
	"fmt"
	"strconv"
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
			Padding(0, 1).
			Width(mapCols*gb.TileSize - boxPadding)

	hudStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Dark))).
			Foreground(lipgloss.Color(gb.Hex(gb.Lightest))).
			Bold(true).
			Padding(0, 1).
			Width(mapCols * gb.TileSize)

	urgentStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Darkest))).
			Foreground(lipgloss.Color(gb.Hex(gb.Lightest))).
			Bold(true).
			Padding(0, 1).
			Width(mapCols * gb.TileSize)

	goalStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(gb.Hex(gb.Light))).
			Foreground(lipgloss.Color(gb.Hex(gb.Darkest))).
			Padding(0, 1).
			Width(mapCols * gb.TileSize)

	pickedStyle   = lipgloss.NewStyle().Bold(true).Underline(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true).Bold(true)
)

const (
	dialogueRows = 4
	gbChromeRows = 7
	gbCols       = mapCols * gb.TileSize
	gbRows       = mapRows*gb.TileSize/2 + gbChromeRows
)

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

	hud := hudStyle.Render(fmt.Sprintf("TILAPOU   %s%s   peixe %s/kg   equiv %s   dia %d",
		coins(snapshot.CashCents), debt, coins(snapshot.Prices.FishKgCents),
		ratio(snapshot.Prices.RatioPPM), snapshot.Tick/(hoursPerDay*minutesPerHour)))

	goal, urgent := objective(snapshot)
	banner := goalStyle.Render("OBJETIVO: " + goal)
	if urgent {
		banner = urgentStyle.Render("! " + goal)
	}

	body := boxStyle.Render(m.dialogue())
	if m.menu != nil {
		body = boxStyle.Render(renderMenu(m.menu))
	}

	return strings.Join([]string{
		hud,
		banner,
		cropTo(renderMap(m.farm, m.you, snapshot, m.frame),
			gbRows-gbChromeRows+dialogueRows-lipgloss.Height(body)),
		body,
		m.renderGameBoyKeys(),
	}, "\n")
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
	if x == shedX && y == shedY {
		return "GALPAO DE RACAO\n" + dimStyle.Render("[z] abre as compras")
	}

	return mapLegend(m.snapshot) + "\n" +
		dimStyle.Render("ande com as setas ate o viveiro ou o galpao, z abre as opcoes")
}

func tankHeadline(t client.Tank) string {
	return fmt.Sprintf("TANQUE %d: %d peixes de %d g", t.ID, t.Fish, t.MeanGrams)
}

func tankAdvice(t client.Tank) string {
	switch {
	case t.OxygenUgL < criticalOxygenUgL && !t.Aerating:
		return "a agua esta sufocando! ligue o aerador"
	case t.FeedKg == 0:
		return "a racao acabou, va ate o galpao"
	case t.ServedFor <= 0:
		return "os peixes estao sem trato servido"
	case t.Ready:
		return "os peixes estao no ponto de abate"
	}

	next := ""
	if t.NextClassG > 0 && t.NextClassG > t.MeanGrams {
		next = fmt.Sprintf("   proxima classe em %d g", t.NextClassG)
	}

	return fmt.Sprintf("%s/kg agora   racao %d kg   trato por %s%s",
		coins(t.PriceKgCents), t.FeedKg, minutes(t.ServedFor), next)
}

func (m Model) renderGameBoyKeys() string {
	if m.menu != nil {
		return screenStyle.Render(menuKeys)
	}

	return screenStyle.Render("setas andam  z opcoes  f trato  c racao  a aerador  h despescar  tab numeros  q sai")
}

func (m Model) interact() (updated Model, target string) {
	x, y := m.you.ahead()

	if index, ok := m.farm.pondAt(x, y); ok && index < len(m.snapshot.Tanks) {
		return m, "tank:" + strconv.Itoa(index)
	}
	if x == shedX && y == shedY {
		return m, "shed"
	}

	return m, ""
}

func (m Model) move(dx, dy int, facing byte) Model {
	if m.menu != nil {
		m.menu.move(dy)

		return m
	}

	m.you.facing = facing
	if !m.farm.blocked(m.you.x+dx, m.you.y+dy) {
		m.you.x += dx
		m.you.y += dy
	}
	m.message = ""

	return m
}
