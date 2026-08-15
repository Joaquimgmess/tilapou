package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	milliUnit   = 1_000
	deciUnit    = 10
	centiUnit   = 100
	panelInset  = 2
	lowRunway   = 10
	wideWidth   = 100
	minWidth    = 80
	minHeight   = 20
	decisionCol = 57
	marketCol   = 40
	sparkPoints = 15
	sparkFloor  = 2
)

var sparkLevels = []rune("▁▂▃▄▅▆▇█")

func (m Model) renderDashboard() string {
	if m.width > 0 && (m.width < minWidth || m.height < minHeight) {
		return m.renderTooSmall()
	}

	rows := []string{
		m.renderTopBar(),
		m.renderGoal(),
		m.renderBatches(),
	}

	wide := m.effectiveWidth() >= wideWidth

	switch {
	case m.menu != nil:
		rows = append(rows, panel(m.effectiveWidth()-panelInset, renderMenu(m.menu)))
	case wide:
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			panel(decisionCol, m.renderDecision()),
			panel(marketCol, m.renderMarket()),
		))
	default:
		rows = append(rows, panel(m.effectiveWidth()-panelInset, m.renderDecision()),
			panel(m.effectiveWidth()-panelInset, m.renderMarket()))
	}

	rows = append(rows, m.renderKeys())

	return strings.Join(rows, "\n")
}

func (m Model) renderTooSmall() string {
	return dangerStyle.Render(fmt.Sprintf(
		"terminal de %dx%d e pequeno demais\nfaltam %d colunas e %d linhas para o painel",
		m.width, m.height, max(minWidth-m.width, 0), max(minHeight-m.height, 0)))
}

func (m Model) renderTopBar() string {
	s := m.snapshot

	parts := []string{
		titleStyle.Render("TILAPOU"),
		labelStyle.Render("caixa ") + valueStyle.Render(coins(s.CashCents)),
	}

	if s.Debt > 0 {
		parts = append(parts, dangerStyle.Render("divida "+coins(s.Debt))+
			dimStyle.Render(" juros "+coins(s.InterestDay)+"/d"))
	}
	if s.RunwayDays >= 0 {
		folego := labelStyle.Render("folego ") + valueStyle.Render(strconv.FormatInt(s.RunwayDays, 10)+" d")
		if s.RunwayDays < lowRunway {
			folego = dangerStyle.Render("folego " + strconv.FormatInt(s.RunwayDays, 10) + " d")
		}
		parts = append(parts, folego)
	}

	if m.effectiveWidth() >= wideWidth {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("d%d %02dh %.1f C",
			s.Tick/(hoursPerDay*minutesPerHour), s.Hour, float64(s.TempMilliC)/milliUnit)))
	}
	if m.staleTicks > 1 {
		parts = append(parts, dangerStyle.Render(fmt.Sprintf("! %ds sem resposta", m.staleTicks)))
	}

	return strings.Join(parts, "  ")
}

func (m Model) renderGoal() string {
	if m.mode == ModeGameBoy && !m.fitsGameBoy() {
		return dangerStyle.Render(fmt.Sprintf(
			"o mapa precisa de %dx%d e o terminal tem %dx%d, entao ficam os numeros",
			gbCols, gbRows, m.width, m.height))
	}
	if m.message != "" {
		return valueStyle.Render(clipTo(m.message, m.effectiveWidth()))
	}

	goal, urgent := objective(m.snapshot)
	if urgent {
		return dangerStyle.Render("! " + clipTo(goal, m.effectiveWidth()-panelInset))
	}

	return okStyle.Render("> " + clipTo(goal, m.effectiveWidth()-panelInset))
}

func clipTo(text string, width int) string {
	if width <= 1 || len(text) <= width {
		return text
	}

	return text[:width-1] + "~"
}

func (m Model) renderBatches() string {
	if len(m.snapshot.Tanks) == 0 {
		return dimStyle.Render("nenhum tanque")
	}

	wide := m.effectiveWidth() >= wideWidth

	header := fmt.Sprintf("%-7s %6s %6s %11s %11s  %s", "LOTE", "PEIXES", "PESO", "VALOR", "MARGEM", "ESTADO")
	if wide {
		header = fmt.Sprintf("%-7s %6s %6s %11s %11s %12s  %s",
			"LOTE", "PEIXES", "PESO", "VALOR", "MARGEM", "PROX CLASSE", "ESTADO")
	}

	lines := []string{labelStyle.Render(header)}

	for i := range m.snapshot.Tanks {
		t := &m.snapshot.Tanks[i]
		state, alert := tankState(t)

		row := fmt.Sprintf("%-7s %6d %4d g %11s %11s  %s",
			fmt.Sprintf("T%d-L%d", t.ID, t.BatchID), t.Fish, t.MeanGrams,
			coins(t.ValueCents), signedPlain(t.MarginCents), state)

		if wide {
			row = fmt.Sprintf("%-7s %6d %4d g %11s %11s %12s  %s",
				fmt.Sprintf("T%d-L%d", t.ID, t.BatchID), t.Fish, t.MeanGrams,
				coins(t.ValueCents), signedPlain(t.MarginCents), nextClass(t), state)
		}

		lines = append(lines, m.decorateRow(i, row, alert))
	}

	return strings.Join(lines, "\n")
}

func (m Model) decorateRow(index int, row string, alert bool) string {
	if index == m.selected {
		return selectedStyle.Render("> " + row)
	}
	if alert {
		return "  " + dangerStyle.Render(row)
	}

	return "  " + row
}

func nextClass(t *client.Tank) string {
	if t.NextClassG <= t.MeanGrams {
		return "no topo"
	}

	return fmt.Sprintf("%d g +%s", t.NextClassG, percent(t.NextClassGain))
}

func tankState(t *client.Tank) (label string, alert bool) {
	switch {
	case t.Sick:
		return "DOENTE", true
	case t.OxygenUgL < criticalOxygenUgL && !t.Aerating:
		return "O2 " + strconv.FormatInt(int64(t.OxygenUgL), 10), true
	case t.FeedKg == 0:
		return "SEM RACAO", true
	case t.ServedFor <= 0:
		return "SEM TRATO", true
	case t.Decision.DaysOfFeed > 0:
		return fmt.Sprintf("racao %d d", t.Decision.DaysOfFeed), false
	}

	return "ok", false
}

func (m Model) effectiveWidth() int {
	if m.width <= 0 {
		return wideWidth
	}

	return m.width
}

func (m Model) renderDecision() string {
	tank, ok := m.tank()
	if !ok {
		return dimStyle.Render("sem lote selecionado")
	}

	d := tank.Decision
	lines := []string{
		labelStyle.Render(fmt.Sprintf("DECISAO T%d-L%d", tank.ID, tank.BatchID)),
		fmt.Sprintf("%s %s/kg (classe %s)   %s %s g/d",
			labelStyle.Render("preco"), coins(tank.PriceKgCents), percent(tank.ClassPPM),
			labelStyle.Render("ganho"), grams(d.GainPerDayMg)),
		fmt.Sprintf("%s %s kg/d por %s   %s %s/kg",
			labelStyle.Render("racao"), kilos(d.FeedPerDayG), coins(d.FeedCostPerDay),
			labelStyle.Render("custo"), coins(tank.CostPerKg)),
		"",
		fmt.Sprintf("  vender agora    %12s  %s", coins(d.SellNowCents), signedCoins(d.SellNowMargin)),
	}

	if d.HoldToGrams > 0 && d.HoldReached {
		better := ""
		if d.HoldMargin > d.SellNowMargin {
			better = okStyle.Render("  <<")
		}
		lines = append(lines,
			fmt.Sprintf("  segurar %4d g  %12s  %s%s",
				d.HoldToGrams, coins(d.HoldCents), signedCoins(d.HoldMargin), better),
			dimStyle.Render(fmt.Sprintf("    %d dias, ja descontados %s de racao",
				d.HoldDays, coins(d.HoldFeedCents))))
	}

	return strings.Join(append(lines, renderBreakEven(tank), renderStocking(tank)), "\n")
}

func renderBreakEven(tank client.Tank) string {
	price := tank.PriceKgCents
	breakEven := tank.Decision.BreakEvenPerKg

	if breakEven <= 0 {
		return dimStyle.Render("break-even: lote sem custo acumulado")
	}

	folga := (price - breakEven) * milliUnit / breakEven

	verdict := okStyle.Render(fmt.Sprintf("folga %s%d,%d%%", sign(folga), abs(folga)/deciUnit, abs(folga)%deciUnit))
	if price < breakEven {
		verdict = dangerStyle.Render("ABAIXO DO CUSTO")
	}

	return fmt.Sprintf("%s %s  %s %s  %s",
		labelStyle.Render("break-even"), coins(breakEven),
		labelStyle.Render("mercado"), coins(price), verdict)
}

func renderStocking(tank client.Tank) string {
	if tank.BreakEven <= 0 {
		return ""
	}

	line := fmt.Sprintf("%s %d de %d peixes  %s %d para pagar a manutencao",
		labelStyle.Render("lotacao"), tank.Fish, tank.Capacity,
		labelStyle.Render("minimo"), tank.BreakEven)

	if int64(tank.Fish) < tank.BreakEven {
		return line + dangerStyle.Render("  faltam "+
			strconv.FormatInt(tank.BreakEven-int64(tank.Fish), 10))
	}

	return line + okStyle.Render("  no azul")
}

func (m Model) renderMarket() string {
	p := m.snapshot.Prices
	s := m.snapshot.Series

	lines := []string{
		labelStyle.Render("MERCADO") + dimStyle.Render(history(len(s.FishKgCents))),
		fmt.Sprintf("%s %s/kg %s", labelStyle.Render("peixe"), coins(p.FishKgCents), sparkline(s.FishKgCents)),
		fmt.Sprintf("%s %s/kg %s", labelStyle.Render("racao"), coins(p.FeedKgCents), sparkline(s.FeedKgCents)),
		fmt.Sprintf("%s %s  %s", labelStyle.Render("equivalencia"), valueStyle.Render(ratio(p.RatioPPM)),
			viability(p.RatioPPM, p.ViablePPM)),
	}

	cycle := m.snapshot.LastCycle
	if cycle.Fish > 0 {
		lines = append(lines, "",
			labelStyle.Render("ULTIMO CICLO"),
			fmt.Sprintf("%d peixes, %d kg", cycle.Fish, cycle.MassG/gramsPerKg),
			fmt.Sprintf("custo %s/kg  venda %s/kg", coins(cycle.CostPerKg), coins(cycle.PricePerKg)),
			fmt.Sprintf("margem %s  CAA %s", signedCoins(cycle.MarginTC), ratio(cycle.FCRPPM)))
	}

	return strings.Join(lines, "\n")
}

func history(points int) string {
	if points < sparkFloor {
		return "  sem historico"
	}

	return fmt.Sprintf("  %d d", points)
}

func viability(ratio, viable int64) string {
	if ratio < viable {
		return dangerStyle.Render("inviavel")
	}

	return okStyle.Render("da lucro")
}

func (m Model) renderKeys() string {
	if m.menu != nil {
		return dimStyle.Render(menuKeys)
	}
	if m.effectiveWidth() < wideWidth {
		return dimStyle.Render("j/k lote  z opcoes  g galpao  f trato  c racao  h despescar  tab mapa  q sair")
	}

	return dimStyle.Render(
		"j/k lote  z opcoes  g galpao  f trato  c racao  a aerador  s povoar  h despescar  tab mapa  q sair")
}

func sparkline(values []int64) string {
	if len(values) < sparkFloor {
		return ""
	}

	if len(values) > sparkPoints {
		values = values[len(values)-sparkPoints:]
	}

	low, high := values[0], values[0]
	for _, v := range values {
		low, high = min(low, v), max(high, v)
	}

	span := high - low
	var out strings.Builder

	for _, v := range values {
		level := 0
		if span > 0 {
			level = int((v - low) * int64(len(sparkLevels)-1) / span)
		}
		_, _ = out.WriteRune(sparkLevels[level])
	}

	return out.String()
}

func panel(width int, content string) string {
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(content)
}

func signedPlain(cents int64) string {
	if cents < 0 {
		return "-" + coins(-cents)
	}

	return "+" + coins(cents)
}

func signedCoins(cents int64) string {
	if cents < 0 {
		return dangerStyle.Render("-" + coins(-cents))
	}

	return okStyle.Render("+" + coins(cents))
}

func percent(ppm int64) string {
	return strconv.FormatInt(ppm/10_000, 10) + "%"
}

func grams(milligrams int64) string {
	return fmt.Sprintf("%d,%d", milligrams/milliUnit, (milligrams%milliUnit)/centiUnit)
}

func kilos(g int64) string {
	return fmt.Sprintf("%d,%d", g/milliUnit, (g%milliUnit)/centiUnit)
}

func sign(v int64) string {
	if v < 0 {
		return "-"
	}

	return "+"
}
