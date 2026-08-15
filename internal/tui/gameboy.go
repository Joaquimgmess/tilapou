package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

const boxPadding = 2

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
)

func (m Model) renderGameBoy() string {
	snapshot := m.snapshot

	hud := hudStyle.Render(fmt.Sprintf("TILAPOU   %s   %d peixes   dia %d  %02dh",
		coins(snapshot.CashCents), snapshot.Fish, snapshot.Tick/(hoursPerDay*minutesPerHour), snapshot.Hour))

	screen := renderMap(m.farm, m.you, snapshot, m.frame)
	dialogue := boxStyle.Render(m.dialogue())

	return strings.Join([]string{hud, screen, dialogue, m.renderGameBoyKeys()}, "\n")
}

func (m Model) dialogue() string {
	if m.message != "" {
		return m.message
	}

	x, y := m.you.ahead()
	if index, ok := m.farm.pondAt(x, y); ok && index < len(m.snapshot.Tanks) {
		return tankHeadline(m.snapshot.Tanks[index]) + "\n" + tankAdvice(m.snapshot.Tanks[index])
	}
	if x == shedX && y == shedY {
		return "GALPAO DE RACAO\naperte z para comprar 100 kg"
	}

	return mapLegend(m.snapshot) + "\nande com as setas, z para interagir"
}

func tankHeadline(t client.Tank) string {
	return fmt.Sprintf("TANQUE %d: %d peixes de %d g", t.ID, t.Fish, t.MeanGrams)
}

func tankAdvice(t client.Tank) string {
	switch {
	case t.OxygenUgL < oxygenLowUgL && !t.Aerating:
		return "a agua esta sufocando! aperte a para ligar o aerador"
	case t.FeedKg == 0:
		return "a racao acabou, va ate o galpao"
	case t.Ready:
		return "os peixes estao no ponto: aperte h para despescar"
	default:
		return fmt.Sprintf("O2 %d ug/L   racao %d kg   %d kg/m3", t.OxygenUgL, t.FeedKg, t.DensityKg)
	}
}

func (m Model) renderGameBoyKeys() string {
	owned := 0
	for _, u := range m.snapshot.Upgrades {
		if u.Owned {
			owned++
		}
	}

	return screenStyle.Render("setas mover  z agir  a aerador  h despescar  1-5 automacao (" +
		strconv.Itoa(owned) + "/5)  m painel  q sair")
}

func (m Model) interact() (updated Model, target string) {
	x, y := m.you.ahead()

	if index, ok := m.farm.pondAt(x, y); ok && index < len(m.snapshot.Tanks) {
		return m, "tank:" + strconv.Itoa(index)
	}
	if x == shedX && y == shedY {
		return m, "feed"
	}

	return m, ""
}

func (m Model) move(dx, dy int, facing byte) Model {
	m.you.facing = facing

	if !m.farm.blocked(m.you.x+dx, m.you.y+dy) {
		m.you.x += dx
		m.you.y += dy
	}

	m.view = ""

	return m
}
