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

const sparkLevels = "▁▂▃▄▅▆▇█"

func (m Model) renderDashboard() string {
	if m.width > 0 && (m.width < minWidth || m.height < minHeight) {
		return m.renderTooSmall()
	}

	rows := []string{
		m.renderTopBar(),
		m.renderGoal(),
	}

	// A explicacao fica onde o mapa estaria, e nao na linha do conselho nem no cabecalho:
	// nos dois ela disputava espaco com o que o jogador precisa para agir.
	if m.mode == ModeGameBoy && !m.fitsGameBoy() {
		rows = append(rows, dimStyle.Render(clipTo(
			fmt.Sprintf("o mapa precisa de %dx%d e o terminal tem %dx%d, entao ficam os numeros",
				gbCols, gbRows, m.width, m.height), m.effectiveWidth()-panelInset)))
	}

	rows = append(rows, m.renderBatches())

	wide := m.effectiveWidth() >= wideWidth

	switch {
	case m.menu != nil:
		rows = append(rows, panel(m.effectiveWidth()-panelInset, renderMenu(m.menu)))
	case wide:
		// A decisao fica com o que sobra: com largura fixa ela quebra a frase enquanto
		// dezenas de colunas ficam vazias a direita.
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			panel(max(m.effectiveWidth()-marketCol-panelInset, decisionCol), m.renderDecision()),
			panel(marketCol, m.renderMarket()),
		))
	default:
		rows = append(rows, panel(m.effectiveWidth()-panelInset, m.renderDecision()),
			panel(m.effectiveWidth()-panelInset, m.renderMarket()))
	}

	return m.withKeysAtTheBottom(rows)
}

// withKeysAtTheBottom pushes the key bar to the last line, so the eye knows where the
// information ends instead of finding it stranded in the middle of an empty screen.
func (m Model) withKeysAtTheBottom(rows []string) string {
	body := strings.Join(rows, "\n")
	keys := m.renderKeys()

	blank := m.height - lipgloss.Height(body) - lipgloss.Height(keys)
	if m.height <= 0 || blank <= 0 {
		return body + "\n" + keys
	}

	return body + strings.Repeat("\n", blank+1) + keys
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

	// Da esquerda para a direita, do mais urgente ao menos: e do fim que a barra abre mao
	// quando a tela e estreita, entao o que nao pode sumir vem antes.
	if m.staleTicks > 1 {
		// A saida junto com o problema: sem isso o jogador le que travou e nao le o que fazer.
		parts = append(parts, dangerStyle.Render(fmt.Sprintf("! %ds sem resposta — r tenta, q sai",
			m.staleTicks)))
	}
	if wait := m.waitMark(); wait != "" {
		parts = append(parts, wait)
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

	// Mesma regra da barra de teclas: quebrar em duas linhas come uma linha do painel, entao
	// o que esta no fim sai ate caber. O nome e o caixa ficam.
	for len(parts) > 2 {
		bar := barStyle.Width(m.effectiveWidth()).Render(strings.Join(parts, dimStyle.Render(" | ")))
		if lipgloss.Height(bar) == 1 {
			return bar
		}

		parts = parts[:len(parts)-1]
	}

	return barStyle.Width(m.effectiveWidth()).Render(strings.Join(parts, dimStyle.Render(" | ")))
}

// waitMark e o sinal de que ha pedido no ar. So aparece quando a espera passa de um tick:
// piscar em toda resposta rapida seria ruido, e o que ele existe para explicar e a demora.
func (m Model) waitMark() string {
	if m.flying() < 1 {
		return ""
	}

	return dimStyle.Render(string(waitFrames[m.frame%len(waitFrames)]))
}

// emptyTankAdvice diz o que fazer com um tanque vazio. Sai do conselho de lotacao, e nao de
// uma regra propria: e o mesmo numero que a tecla consome, entao a tela nao manda apertar o
// que o jogo vai recusar.
func emptyTankAdvice(t client.Tank) string {
	if t.StockAdvice <= 0 {
		return "sem caixa para o ciclo: veja o credito com [g]"
	}

	return "povoe com [s]"
}

// rule renders a section title followed by a line filling the width.
func rule(title string, width int) string {
	head := labelStyle.Render(title + " ")
	fill := width - lipgloss.Width(head)
	if fill < 1 {
		return head
	}

	return head + dimStyle.Render(strings.Repeat("─", fill))
}

func (m Model) renderGoal() string {
	if m.message != "" {
		return valueStyle.Render(clipTo(m.message, m.effectiveWidth()))
	}

	goal, urgent := objective(m.snapshot, m.tankID())
	if urgent {
		return dangerStyle.Render("! " + clipTo(goal, m.effectiveWidth()-panelInset))
	}

	return okStyle.Render("> " + clipTo(goal, m.effectiveWidth()-panelInset))
}

func clipTo(text string, width int) string {
	if width <= 1 || lipgloss.Width(text) <= width {
		return text
	}

	var kept strings.Builder

	used := 0
	for _, r := range text {
		w := lipgloss.Width(string(r))
		if used+w > width-1 {
			break
		}
		used += w
		_, _ = kept.WriteRune(r)
	}

	return kept.String() + "~"
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

	lines := []string{labelStyle.Render(header) + dimStyle.Render(
		strings.Repeat(" ", max(m.effectiveWidth()-lipgloss.Width(header)-panelInset, 0)))}

	for i, r := range m.rows() {
		t := &m.snapshot.Tanks[r.tank]

		if r.batch < 0 {
			lines = append(lines, m.decorateRow(i,
				fmt.Sprintf("%-7s %6s %6s %11s %11s  %s",
					fmt.Sprintf("T%d", t.ID), "-", "-", "-", "-", "vazio, "+emptyTankAdvice(*t)), false))

			continue
		}

		batch := &t.Batches[r.batch]
		state, alert := rowState(t, batch)

		line := fmt.Sprintf("%-7s %6d %4d g %11s %11s  %s",
			fmt.Sprintf("T%d-L%d", t.ID, batch.ID), batch.Fish, batch.MeanGrams,
			coins(batch.ValueCents), signedPlain(batch.MarginCents), state)

		if wide {
			line = fmt.Sprintf("%-7s %6d %4d g %11s %11s %12s  %s",
				fmt.Sprintf("T%d-L%d", t.ID, batch.ID), batch.Fish, batch.MeanGrams,
				coins(batch.ValueCents), signedPlain(batch.MarginCents), nextClass(batch), state)
		}

		lines = append(lines, m.decorateRow(i, line, alert))
	}

	return strings.Join(lines, "\n")
}

func (m Model) decorateRow(index int, row string, alert bool) string {
	if index == m.selected {
		return okStyle.Render("▌") + " " + valueStyle.Render(row)
	}
	if alert {
		return "  " + dangerStyle.Render(row)
	}

	return "  " + dimStyle.Render(row)
}

func nextClass(batch *client.Batch) string {
	if batch.NextClassGrams <= batch.MeanGrams {
		return "no topo"
	}

	return fmt.Sprintf("%d g +%s", batch.NextClassGrams, percent(batch.NextClassGain))
}

// rowState mixes the two levels the row shows: doenca e do lote, oxigenio e racao sao do
// tanque. Sem peixe no lote nada disso e urgencia.
func rowState(t *client.Tank, batch *client.Batch) (label string, alert bool) {
	switch {
	case batch.Fish == 0:
		return "vazio", false
	case batch.Sick:
		return "DOENTE", true
	case t.OxygenUgL < criticalOxygenUgL && !t.Aerating:
		return "O2 " + strconv.FormatInt(int64(t.OxygenUgL), 10), true
	case t.FeedKg == 0:
		return "SEM RACAO", true
	case t.ServedFor <= 0:
		return "SEM TRATO", true
	case batch.Decision.DaysOfFeed > 0:
		return fmt.Sprintf("racao %d d", batch.Decision.DaysOfFeed), false
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
		return dimStyle.Render("sem tanque selecionado")
	}

	batch, ok := m.batch()
	if !ok {
		return rule(fmt.Sprintf("DECISAO T%d", tank.ID), decisionCol-panelInset) + "\n" +
			dimStyle.Render("sem lote neste tanque: "+emptyTankAdvice(tank))
	}

	d := batch.Decision
	lines := []string{
		rule(fmt.Sprintf("DECISAO T%d-L%d", tank.ID, batch.ID), decisionCol-panelInset),
		fmt.Sprintf("%s %s/kg (classe %s)   %s %s g/d",
			labelStyle.Render("preco"), coins(batch.PriceKgCents), percent(batch.ClassPPM),
			labelStyle.Render("ganho"), grams(d.GainPerDayMg)),
		fmt.Sprintf("%s %s kg/d   %s %s/d   %s %s/kg",
			labelStyle.Render("racao"), kilos(d.FeedPerDayG),
			labelStyle.Render("gasto"), coins(d.CostPerDay),
			labelStyle.Render("custo"), coins(batch.CostPerKg)),
		"",
		fmt.Sprintf("  vender agora    %12s  %s", coins(d.SellNowCents), signedCoins(d.SellNowMargin)),
	}

	if d.HoldToGrams > 0 && d.HoldReached {
		better := ""
		if holdWins(d) {
			better = okStyle.Render("  <<")
		}
		lines = append(lines,
			fmt.Sprintf("  segurar %4d g  %12s  %s%s",
				d.HoldToGrams, coins(d.HoldCents), signedCoins(d.HoldMargin), better),
			dimStyle.Render(fmt.Sprintf("    %d dias, ja descontados %s de gasto",
				d.HoldDays, coins(d.HoldCostCents))))
	}

	return strings.Join(append(lines, renderBreakEven(batch), renderStocking(tank)), "\n")
}

func renderBreakEven(batch client.Batch) string {
	price := batch.PriceKgCents
	breakEven := batch.Decision.BreakEvenPerKg

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

	// Lotacao acima do break-even nao garante dinheiro: o que decide e a margem do lote.
	if margin := tankMargin(tank); margin < 0 {
		return line + dangerStyle.Render("  mas o lote perde "+coins(-margin))
	}

	return line + okStyle.Render("  no azul")
}

// jumpKeyHint anuncia o pulo so quando ele faz alguma coisa: a tecla nasce escondida
// dentro do texto do conselho, e um jogador com um tanque so nunca a encontraria.
func (m Model) jumpKeyHint() string {
	target := adviceTank(m.snapshot)
	if target == 0 || target == m.tankID() {
		return ""
	}

	return "  " + jumpKey + " alerta"
}

// keyBar is the row of keys at the bottom of whichever screen is showing.
func (m Model) keyBar() string {
	if m.mode == ModeGameBoy {
		// Zero deixa a barra no tamanho do texto: quem precisa dela preenchida ate a borda
		// e o quadro do aparelho, que passa a largura dele.
		return m.renderGameBoyKeys(0)
	}

	return m.renderKeys()
}

func tankMargin(tank client.Tank) int64 {
	var total int64
	for i := range tank.Batches {
		total += tank.Batches[i].MarginCents
	}

	return total
}

// holdWins says whether holding beats selling now. It compares gain per day and not total,
// because waiting weeks for a bit more loses to starting the cycle over — but only after
// checking there is a gain at all: on two losses, per-day would elect the slower loss.
// A tie goes to selling: money today is worth more than the same money weeks of feed later.
func holdWins(d client.Decision) bool {
	gain := d.HoldMargin - d.SellNowMargin
	if gain <= 0 {
		return false
	}
	if d.HoldDays <= 0 {
		return true
	}

	return gain/d.HoldDays > d.SellNowMargin/max(d.CycleDays, 1)
}

func (m Model) renderMarket() string {
	p := m.snapshot.Prices
	s := m.snapshot.Series

	lines := []string{
		rule("MERCADO"+history(len(s.FishKgCents)), marketCol-panelInset),
		fmt.Sprintf("%s %s/kg %s", labelStyle.Render("peixe"), coins(p.FishKgCents), sparkline(s.FishKgCents)),
		fmt.Sprintf("%s %s/kg %s", labelStyle.Render("racao"), coins(p.FeedKgCents), sparkline(s.FeedKgCents)),
		fmt.Sprintf("%s %s  %s", labelStyle.Render("equivalencia"), valueStyle.Render(ratio(p.RatioPPM)),
			viability(p.RatioPPM, p.ViablePPM)),
	}

	cycle := m.snapshot.LastCycle
	if cycle.Fish > 0 {
		lines = append(lines, "",
			labelStyle.Render("ULTIMO CICLO"),
			fmt.Sprintf("%d peixes, %d kg", cycle.Fish, cycle.MassGrams/gramsPerKg),
			fmt.Sprintf("custo %s/kg  venda %s/kg", coins(cycle.CostPerKg), coins(cycle.PricePerKg)),
			fmt.Sprintf("margem %s  CAA %s", signedCoins(cycle.MarginCents), ratio(cycle.FCRPPM)))
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
		return keyStyle.Width(m.effectiveWidth()).Render(menuKeys)
	}

	// A linha quebra em duas quando nao cabe, e o rodape passa a comer uma linha do painel.
	// Em vez de contar colunas na mao, as dicas menos urgentes saem ate caber; a de sair
	// fica sempre, porque e a unica que o jogador precisa achar em qualquer estado.
	hints := m.keyHints()
	for len(hints) > 0 {
		bar := keyStyle.Width(m.effectiveWidth()).Render(strings.Join(append(hints, "q sair"), "  "))
		if lipgloss.Height(bar) == 1 {
			return bar
		}

		hints = hints[:len(hints)-1]
	}

	return keyStyle.Width(m.effectiveWidth()).Render("q sair")
}

// keyHints lista as teclas da mais urgente para a menos, porque e do fim que a barra abre mao
// quando a tela e estreita. As condicionais nascem escondidas e aparecem na hora em que
// agem: listar sempre custa a coluna de quem esta jogando agora.
func (m Model) keyHints() []string {
	hints := []string{"j/k lote", "z opcoes", "g galpao", "f trato", "c racao"}

	if m.snapshot.Broke {
		// Fazenda quebrada nao tem o que povoar nem despescar: a saida dela e a unica tecla
		// que importa, e hoje o jogo so a cita no texto de objetivo.
		return append(hints, "b recomecar")
	}

	hints = append(hints, "h despescar")
	if target := adviceTank(m.snapshot); target != 0 && target != m.tankID() {
		hints = append(hints, jumpKey+" alerta")
	}
	if m.snapshot.PrestigeNow > m.snapshot.Prestige {
		hints = append(hints, "p tilapar")
	}

	return append(hints, "tab mapa", "s povoar", "a aerador")
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
	levels := []rune(sparkLevels)

	var out strings.Builder

	for _, v := range values {
		level := 0
		if span > 0 {
			level = int((v - low) * int64(len(levels)-1) / span)
		}
		_, _ = out.WriteRune(levels[level])
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
	return strconv.FormatInt(ppm/ppmPerCentesimal, 10) + "%"
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
