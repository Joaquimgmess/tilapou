package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Fatores de conversao da formatacao; UnitPPMValue e uma unidade em partes por milhao.
const (
	minutesPerHour   = 60
	hoursPerDay      = 24
	milliPerUnit     = 1000
	centsPerCoin     = 100
	gramsPerKg       = 1000
	UnitPPMValue     = 1_000_000
	ppmPerCentesimal = 10_000
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BAC0F"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	valueStyle  = lipgloss.NewStyle().Bold(true)
	dangerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D64545"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

// View monta a tela a partir do ultimo snapshot.
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("ate mais, piscicultor\n")
	}

	view := tea.NewView(m.render())
	view.AltScreen = true
	view.WindowTitle = "tilapou"

	return view
}

func (m Model) render() string {
	if m.snapshot.FarmID == "" {
		if m.err != nil {
			return dangerStyle.Render("daemon fora do ar: "+m.err.Error()) +
				dimStyle.Render("\n\nsuba com: make up\n[r] tentar de novo   [q] sair\n")
		}

		return dimStyle.Render("conectando no daemon...\n")
	}
	if m.mode == ModeGameBoy && m.fitsGameBoy() {
		return m.renderGameBoy()
	}

	return m.renderDashboard()
}

func ratio(ppm int64) string {
	return fmt.Sprintf("%d,%02d", ppm/UnitPPMValue, (ppm%UnitPPMValue)/ppmPerCentesimal)
}

func coins(cents int64) string {
	return fmt.Sprintf("%d,%02d TC", cents/centsPerCoin, abs(cents%centsPerCoin))
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}

	return v
}
