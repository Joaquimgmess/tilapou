package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	minutesPerHour   = 60
	hoursPerDay      = 24
	oxygenBarWidth   = 24
	oxygenFullUgL    = 9000
	oxygenLowUgL     = 3000
	eventsShown      = 8
	milliPerUnit     = 1000
	centsPerCoin     = 100
	panelWidth       = 58
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
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(panelWidth)
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("ate mais, piscicultor\n")
	}

	content := m.view
	if content == "" {
		content = m.render()
	}

	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "tilapou"

	return view
}

func (m Model) render() string {
	if m.err == nil && m.snapshot.FarmID != "" && m.mode == ModeGameBoy {
		return m.renderGameBoy()
	}
	if m.err != nil {
		return dangerStyle.Render("daemon fora do ar: "+m.err.Error()) +
			dimStyle.Render("\n\nsuba com: tilapou serve\n[r] tentar de novo   [q] sair\n")
	}
	if m.snapshot.FarmID == "" {
		return dimStyle.Render("conectando no daemon...\n")
	}

	sections := []string{
		m.renderHeader(),
		m.renderMarket(),
		m.renderTanks(),
		m.renderUpgrades(),
		m.renderEvents(),
		m.renderFooter(),
	}

	return strings.Join(sections, "\n") + "\n"
}

func (m Model) renderHeader() string {
	s := m.snapshot
	day := s.Tick / (hoursPerDay * minutesPerHour)
	minute := s.Tick % minutesPerHour

	clock := fmt.Sprintf("dia %d  %02d:%02d  %.1f C", day, s.Hour, minute, float64(s.TempMilliC)/milliPerUnit)

	line := fmt.Sprintf("%s   %s %s   %s %s   %s %s",
		titleStyle.Render("TILAPOU"),
		labelStyle.Render("caixa"), valueStyle.Render(coins(s.CashCents)),
		labelStyle.Render("biomassa"), valueStyle.Render(fmt.Sprintf("%d kg", s.BiomassG/gramsPerKg)),
		labelStyle.Render("peixes"), valueStyle.Render(strconv.Itoa(int(s.Fish))),
	)

	return line + "\n" + dimStyle.Render(clock)
}

func ratio(ppm int64) string {
	return fmt.Sprintf("%d,%02d", ppm/UnitPPMValue, (ppm%UnitPPMValue)/ppmPerCentesimal)
}

func (m Model) renderMarket() string {
	p := m.snapshot.Prices

	verdict := okStyle.Render("da lucro")
	if p.RatioPPM < p.ViablePPM {
		verdict = dangerStyle.Render("inviavel: nao vale a pena produzir")
	}

	line := fmt.Sprintf("%s peixe %s/kg   racao %s/kg   equivalencia %s  %s",
		labelStyle.Render("mercado"), coins(p.FishKgCents), coins(p.FeedKgCents),
		valueStyle.Render(ratio(p.RatioPPM)), verdict)

	cycle := m.snapshot.LastCycle
	if cycle.Fish == 0 {
		return line
	}

	margin := okStyle.Render("+" + coins(cycle.MarginTC))
	if cycle.MarginTC < 0 {
		margin = dangerStyle.Render(coins(cycle.MarginTC))
	}

	return line + "\n" + fmt.Sprintf("%s %d peixes, %d kg   custo %s/kg   venda %s/kg   margem %s   CAA %s",
		labelStyle.Render("ultimo ciclo"), cycle.Fish, cycle.MassG/gramsPerKg,
		coins(cycle.CostPerKg), coins(cycle.PricePerKg), margin, ratio(cycle.FCRPPM))
}

func (m Model) renderTanks() string {
	if len(m.snapshot.Tanks) == 0 {
		return dimStyle.Render("nenhum tanque")
	}

	panels := make([]string, 0, len(m.snapshot.Tanks))
	for i := range m.snapshot.Tanks {
		panels = append(panels, panelStyle.Render(renderTank(&m.snapshot.Tanks[i])))
	}

	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

func renderTank(t *client.Tank) string {
	trato := dangerStyle.Render("sem trato servido")
	if t.ServedFor > 0 {
		trato = okStyle.Render("trato por " + minutes(t.ServedFor))
	}

	aerator := dimStyle.Render("aerador off")
	if t.Aerating {
		aerator = okStyle.Render("aerador ON")
	}

	ready := ""
	if t.Ready {
		ready = okStyle.Render("  PRONTO PARA DESPESCA")
	}

	return fmt.Sprintf("%s %s%s\n%s %s/%d   %s %s   %s %s\n%s %s   %s   %s\n%s",
		titleStyle.Render(fmt.Sprintf("tanque %d", t.ID)), dimStyle.Render(t.Kind), ready,
		labelStyle.Render("peixes"), valueStyle.Render(strconv.Itoa(int(t.Fish))), t.Capacity,
		labelStyle.Render("peso"), valueStyle.Render(fmt.Sprintf("%d g", t.MeanGrams)),
		labelStyle.Render("dens"), valueStyle.Render(density(t.DensityMilli)),
		labelStyle.Render("racao"), valueStyle.Render(fmt.Sprintf("%d kg", t.FeedKg)),
		trato,
		aerator,
		oxygenBar(t.OxygenUgL),
	)
}

func oxygenBar(level int32) string {
	filled := int(int64(level) * oxygenBarWidth / oxygenFullUgL)
	filled = max(0, min(filled, oxygenBarWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", oxygenBarWidth-filled)
	text := fmt.Sprintf("O2 %s %d ug/L", bar, level)

	if level < oxygenLowUgL {
		return dangerStyle.Render(text)
	}

	return okStyle.Render(text)
}

func (m Model) renderUpgrades() string {
	tank, ok := m.tank()
	if !ok || len(tank.Upgrades) == 0 {
		return ""
	}

	parts := make([]string, 0, len(tank.Upgrades))
	for i, u := range tank.Upgrades {
		key := "[" + strconv.Itoa(i+1) + "] "
		switch {
		case u.Owned:
			parts = append(parts, okStyle.Render(key+u.Kind+" ok"))
		case m.snapshot.CashCents >= u.CostCents:
			parts = append(parts, valueStyle.Render(key+u.Kind+" "+coins(u.CostCents)))
		default:
			parts = append(parts, dimStyle.Render(key+u.Kind+" "+coins(u.CostCents)))
		}
	}

	line := labelStyle.Render(fmt.Sprintf("automacao do tanque %d  ", tank.ID)) +
		strings.Join(parts, dimStyle.Render("  "))

	if m.snapshot.PrestigeNow > m.snapshot.Prestige {
		line += "\n" + okStyle.Render(fmt.Sprintf("[p] tilapar agora rende %d matrizes (voce tem %d)",
			m.snapshot.PrestigeNow, m.snapshot.Prestige))
	}

	return line
}

func (m Model) renderEvents() string {
	if len(m.snapshot.Events) == 0 {
		return dimStyle.Render("sem eventos ainda")
	}

	lines := []string{labelStyle.Render("eventos recentes")}
	for i, e := range m.snapshot.Events {
		if i >= eventsShown {
			break
		}
		lines = append(lines, "  "+dimStyle.Render(describe(e)))
	}

	return strings.Join(lines, "\n")
}

func describe(e client.Event) string {
	switch e.Kind {
	case "hypoxia_deaths":
		return fmt.Sprintf("%d peixes morreram por falta de oxigenio", e.Fish)
	case "starvation_deaths":
		return fmt.Sprintf("%d peixes morreram de fome", e.Fish)
	case "harvest":
		return fmt.Sprintf("despesca de %d peixes, %d kg, %s", e.Fish, e.MassG/gramsPerKg, coins(e.CashTC))
	case "feed_bought":
		return fmt.Sprintf("comprou %d kg de racao por %s", e.MassG/gramsPerKg, coins(e.CashTC))
	case "feed_exhausted":
		return "a racao acabou no tanque " + strconv.FormatUint(uint64(e.Tank), 10)
	case "growth":
		return fmt.Sprintf("o lote ganhou %d g", e.MassG)
	case "stocked":
		return fmt.Sprintf("povoou %d alevinos por %s", e.Fish, coins(e.CashTC))
	case "tank_bought":
		return "comprou um tanque por " + coins(e.CashTC)
	case "upgrade_bought":
		return "automacao comprada por " + coins(e.CashTC)
	case "prestiged":
		return fmt.Sprintf("voce tilapou: +%d matrizes", e.Fish)
	case "action_rejected":
		return "acao recusada: " + e.Reason
	default:
		return e.Kind
	}
}

func (m Model) renderFooter() string {
	goal, urgent := objective(m.snapshot)
	banner := okStyle.Render("objetivo: " + goal)
	if urgent {
		banner = dangerStyle.Render("! " + goal)
	}

	keys := dimStyle.Render(
		"[f] trato  [c] racao  [a] aerador  [h] despescar  [s] povoar  [t] tanque  [1-5] automacao  [tab] trocar  [m] mapa  [q] sair")

	if m.message == "" {
		return banner + "\n" + keys
	}

	return banner + "\n" + keys + "\n" + labelStyle.Render(m.message)
}

const densityDecimal = 100

func density(milli int64) string {
	return fmt.Sprintf("%d,%d kg/m3", milli/milliPerUnit, (milli%milliPerUnit)/densityDecimal)
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
