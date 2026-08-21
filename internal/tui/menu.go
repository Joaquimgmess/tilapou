package tui

import (
	"fmt"
	"strconv"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	fixedTankItems = 8
	thinPercent    = 30
	shedTitle      = "GALPAO"
	fullPercent    = 100
)

type menuItem struct {
	label   string
	hint    string
	enabled bool
	panel   bool
	status  string
	action  client.Action
}

type menu struct {
	title  string
	items  []menuItem
	cursor int
}

func (m *menu) move(delta int) {
	if len(m.items) == 0 {
		return
	}

	m.cursor = (m.cursor + delta + len(m.items)) % len(m.items)
}

func (m *menu) current() (menuItem, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return menuItem{}, false
	}

	return m.items[m.cursor], true
}

// tankMenu acts on one batch of the tank: the actions that carry a batch id use it, and the
// tank-wide ones ignore it.
func tankMenu(s api.Snapshot, t api.Tank, batch api.Batch, mapFits bool) *menu {
	items := make([]menuItem, 0, len(t.Upgrades)+fixedTankItems)
	items = append(items,
		feedItem(t),
		buyFeedItem(s, t),
		aeratorItem(t),
		harvestItem(t, batch),
		thinItem(t, batch),
		treatItem(t, batch),
		stockItem(s, t),
	)

	for _, upgrade := range t.Upgrades {
		items = append(items, upgradeItem(s, t, upgrade))
	}

	// Com o mapa sem caber, o painel de numeros ja e a tela: oferecer "ver" o que esta na
	// frente do jogador gasta uma linha do menu e nao leva a lugar nenhum.
	if mapFits {
		items = append(items, menuItem{
			label:   "Ver painel de numeros",
			hint:    "tecla tab",
			enabled: true,
			panel:   true,
		})
	}

	return &menu{title: fmt.Sprintf("TANQUE %d", t.ID), items: items}
}

func feedItem(t api.Tank) menuItem {
	item := menuItem{
		label:   "Servir o trato",
		enabled: t.FeedKg > 0 && t.Fish > 0,
		status:  "servindo o trato",
		action:  client.Action{Kind: "feed", Tank: t.ID},
	}

	switch {
	case t.Fish == 0:
		item.hint = "nao ha peixe aqui"
	case t.FeedKg == 0:
		item.hint = "sem racao no galpao do tanque"
	case t.ServedFor > 0:
		item.hint = "ja servido por mais " + minutes(t.ServedFor)
	default:
		item.hint = "os peixes estao esperando"
	}

	return item
}

func buyFeedItem(s api.Snapshot, t api.Tank) menuItem {
	price := feedPurchaseKg * s.Prices.FeedKgCents

	item := menuItem{
		label:   "Comprar 100 kg de racao",
		enabled: s.CashCents >= price,
		status:  "comprando racao",
		action:  client.Action{Kind: "buy_feed", Tank: t.ID, Amount: feedPurchaseKg},
		hint:    coins(price),
	}
	if !item.enabled {
		item.hint = "faltam " + coins(price-s.CashCents)
	}

	return item
}

func aeratorItem(t api.Tank) menuItem {
	label, hint := "Ligar o aerador", "gasta energia enquanto ligado"
	amount := int64(1)

	if t.Aerating {
		label, hint, amount = "Desligar o aerador", "esta ligado e gastando energia", 0
	}
	if t.OxygenUgL < criticalOxygenUgL && !t.Aerating {
		hint = "o oxigenio esta critico, ligue agora"
	}

	return menuItem{
		label:   label,
		hint:    hint,
		enabled: true,
		status:  "alternando o aerador",
		action:  client.Action{Kind: "aerate", Tank: t.ID, Amount: amount},
	}
}

func harvestItem(t api.Tank, batch api.Batch) menuItem {
	item := menuItem{
		label:   "Despescar o lote",
		enabled: batch.Fish > 0,
		status:  "despescando",
		action:  client.Action{Kind: "harvest", Tank: t.ID, Batch: batch.ID},
	}

	switch {
	case batch.Fish == 0:
		item.hint = "nao ha peixe aqui"
	case batch.Ready:
		item.hint = fmt.Sprintf("%d peixes de %d g, no ponto", batch.Fish, batch.MeanGrams)
	default:
		item.hint = fmt.Sprintf("so %d g: vender agora rende menos", batch.MeanGrams)
	}

	return item
}

func thinItem(t api.Tank, batch api.Batch) menuItem {
	count := int64(batch.Fish) * thinPercent / fullPercent
	revenue := count * batch.MeanGrams * batch.PriceKgCents / gramsPerKg

	item := menuItem{
		label:   fmt.Sprintf("Ralear %d%% do lote", thinPercent),
		enabled: count > 0,
		status:  "raleando o lote",
		action:  client.Action{Kind: "harvest", Tank: t.ID, Batch: batch.ID, Amount: count},
		hint:    fmt.Sprintf("vende %d peixes por ~%s e o resto continua crescendo", count, coins(revenue)),
	}
	if count <= 0 {
		item.hint = "nao ha peixe suficiente"
	}

	return item
}

func treatItem(t api.Tank, batch api.Batch) menuItem {
	item := menuItem{
		label:   "Tratar o lote",
		enabled: batch.Sick,
		status:  "tratando o lote",
		action:  client.Action{Kind: "treat", Tank: t.ID},
		hint:    "nao ha doenca nesse tanque",
	}
	if batch.Sick {
		item.hint = "cura agora, mas deixa portadores no tanque por um tempo"
	}

	return item
}

func loanHint(s api.Snapshot, t api.Tank) string {
	// A saida sai do estado, e nao de um palpite sobre o caixa: mandar pagar divida com caixa
	// zerado, ou vender peixe sem peixe, e mandar fazer o que a tela ao lado ja nega.
	pagar := raiseCash(s)

	switch t.LoanBlock {
	case api.LoanOpen:
	case api.LoanNoCredit:
		return "sem espaco no limite de credito: " + pagar
	case api.LoanNoRoom:
		return fmt.Sprintf("o tanque %d nao aceita mais peixe: nao ha o que financiar", t.ID)
	case api.LoanNoNeed:
		return fmt.Sprintf("o caixa ja cobre o que falta no tanque %d", t.ID)
	case api.LoanNoCycle:
		return "o credito que sobra nao paga o ciclo: " + pagar
	}

	// Prometer que cobre o que falta e mentira quando o dinheiro nao chega la: quem sabe
	// quantos peixes ele povoa e o daemon, que ja desconta o custo fixo do ciclo.
	short := t.BreakEven - int64(t.Fish) - t.StockAdvice
	if short > 0 && t.LoanFish > 0 && t.LoanFish < short {
		// Os dois juntos dao 99 colunas num campo de 81. Quem sai e a margem, que e do ciclo
		// e nao muda com o principal; o juro fica, porque e o numero que decide a jogada.
		return fmt.Sprintf("da para %d de %d; %s", t.LoanFish, short, creditOwed(t))
	}

	return creditCost(t)
}

// creditCost e o custo do emprestimo em tres numeros, na ordem em que a decisao se forma: o
// que entra hoje, o que volta no fim do ciclo e a margem que o ciclo projeta. Nenhum verbo de
// recomendacao: o texto mostra o custo e a conclusao e do jogador (decision-012).
func creditCost(t api.Tank) string {
	// O principal nao se repete aqui: o rotulo do item ja diz "Pegar emprestimo de X", e a
	// repeticao custava as colunas que o juro precisa — e o juro e o numero que decide.
	custo := creditOwed(t)

	// "o ciclo projeta", e nao "margem projetada": a margem e do ciclo e nao muda com o
	// principal. Sem sujeito e colada em "volta X", o olho empresta o sujeito da frase
	// anterior — o emprestimo — e le como lucro daquele negocio.
	if t.CycleMargin <= 0 {
		return custo + "; o ciclo nao projeta margem"
	}

	return custo + "; o ciclo projeta " + coins(t.CycleMargin)
}

// raiseCash aponta a saida que o estado sustenta, na ordem em que ela existe: pagar o que se
// deve so quando ha divida E caixa, vender peixe so quando ha peixe, e o recomeco quando nao
// sobra nem uma coisa nem outra. Cada palpite a mais aqui ja virou um defeito proprio.
func raiseCash(s api.Snapshot) string {
	switch {
	case s.Debt > 0 && s.CashCents > 0:
		return "pague o que deve antes"
	case s.Fish > 0:
		return "venda peixe com [h] antes"
	case s.CashCents > 0:
		return "junte caixa antes"
	default:
		return "sem peixe para vender, so [b]"
	}
}

// creditOwed e quanto o emprestimo devolve no fim e quanto disso e juro, em TC e em %.
func creditOwed(t api.Tank) string {
	juro := t.LoanOwed - t.LoanAdvice

	return fmt.Sprintf("volta %s em %d d; juro %s (%s)",
		coins(t.LoanOwed), t.CycleDays, coins(juro), shareOf(juro, t.LoanAdvice))
}

// shareOf formata a fatia de base em pontos percentuais com uma casa. Separado do percent do
// painel, que recebe PPM e arredonda para inteiro: aqui a casa decimal muda a leitura.
func shareOf(part, base int64) string {
	if base <= 0 {
		return "-"
	}

	// Decimos de ponto percentual: a casa decimal muda a leitura de "18,9%" para "19%", e a
	// diferenca entre juro e margem cabe justamente ali.
	const tenthsPerPercent = 10

	tenths := part * centiUnit * tenthsPerPercent / base

	return fmt.Sprintf("%d,%d%%", tenths/tenthsPerPercent, tenths%tenthsPerPercent)
}

func stockBlocked(t api.Tank) string {
	tank := strconv.FormatInt(int64(t.ID), 10)

	switch {
	case t.Capacity-int64(t.Fish) <= 0:
		return "O tanque " + tank + " ja esta no limite de densidade: " +
			strconv.FormatInt(t.Capacity, 10) + " peixes"
	case t.MaxBatches > 0 && t.BatchCount >= t.MaxBatches:
		return "O tanque " + tank + " ja tem os " + strconv.FormatInt(int64(t.MaxBatches), 10) +
			" lotes que cabem. Despesque um antes de povoar de novo"
	}

	return "Sem grana para povoar: o caixa nao paga o alevino mais a racao ate a despesca"
}

func stockItem(s api.Snapshot, t api.Tank) menuItem {
	amount := t.StockAdvice
	room := t.Capacity - int64(t.Fish)

	item := menuItem{
		label:   "Povoar com alevinos",
		enabled: amount > 0,
		status:  "povoando",
		action:  client.Action{Kind: "stock", Tank: t.ID, Amount: amount},
	}

	if amount <= 0 {
		item.hint = stockBlocked(t)

		return item
	}
	item.hint = fmt.Sprintf("%d alevinos por %s, cabem %d e o caixa banca a racao deles",
		amount, coins(amount*s.Prices.FingerlingCents), room)

	return item
}

func upgradeItem(s api.Snapshot, t api.Tank, upgrade api.Upgrade) menuItem {
	item := menuItem{
		label:   "Instalar " + upgrade.Kind,
		enabled: !upgrade.Owned && s.CashCents >= upgrade.CostCents,
		status:  "instalando " + upgrade.Kind,
		action:  client.Action{Kind: "buy_upgrade", Tank: t.ID, Auto: upgrade.Kind},
	}

	switch {
	case upgrade.Owned:
		item.label = upgrade.Kind + " instalado"
		item.hint = automationHint(upgrade.Kind)
	case s.CashCents < upgrade.CostCents:
		item.hint = "faltam " + coins(upgrade.CostCents-s.CashCents)
	default:
		item.hint = coins(upgrade.CostCents) + ", " + automationHint(upgrade.Kind)
	}

	return item
}

func automationHint(kind string) string {
	switch kind {
	case "comedouro":
		return "serve o trato e repoe a racao sozinho"
	case "aerador":
		return "liga sozinho quando o oxigenio cai"
	case "peao":
		return "despesca sozinho no ponto de abate"
	case "tecnico":
		return "ajusta o trato e acelera o ganho"
	case "contrato":
		return "vende com preco melhor"
	}

	return "automacao"
}

func shedMenu(s api.Snapshot, t api.Tank) *menu {
	loads := []int64{feedPurchaseKg, feedPurchaseKg * 5}

	items := make([]menuItem, 0, len(loads)+1)
	for _, kilos := range loads {
		price := kilos * s.Prices.FeedKgCents
		item := menuItem{
			label:   fmt.Sprintf("Comprar %d kg de racao", kilos),
			hint:    coins(price),
			enabled: s.CashCents >= price,
			status:  "comprando racao",
			action:  client.Action{Kind: "buy_feed", Tank: t.ID, Amount: kilos},
		}
		if !item.enabled {
			item.hint = "faltam " + coins(price-s.CashCents)
		}
		items = append(items, item)
	}

	items = append(items, creditItems(s, t)...)

	items = append(items, tankItem(s))

	return &menu{title: shedTitle, items: items}
}

func tankItem(s api.Snapshot) menuItem {
	price := s.NextTankCents
	hint := "amplia a fazenda, e cada tanque sai mais caro que o anterior"

	if short := price - s.CashCents; short > 0 {
		hint = "faltam " + coins(short)
	}

	return menuItem{
		label:   "Comprar outro viveiro por " + coins(price),
		hint:    hint,
		enabled: price > 0 && s.CashCents >= price,
		status:  "comprando um viveiro",
		action:  client.Action{Kind: "buy_tank", TankKind: "viveiro_escavado"},
	}
}

func creditItems(s api.Snapshot, t api.Tank) []menuItem {
	loan := t.LoanAdvice

	// Sem espaco no limite a linha vira o motivo: "Pegar emprestimo de 0,00" e uma opcao
	// que nunca faz nada, mas sumir com ela deixaria a recusa sem explicacao.
	label := "Pegar emprestimo de " + coins(loan)
	if loan <= 0 {
		label = "Emprestimo indisponivel"
	}

	items := []menuItem{{
		label:   label,
		hint:    loanHint(s, t),
		enabled: loan > 0,
		status:  "pegando emprestimo",
		action:  client.Action{Kind: "borrow", Amount: loan},
	}}

	if s.Debt > 0 {
		pay := min(s.Debt, s.CashCents)
		items = append(items, menuItem{
			label:   "Pagar divida",
			hint:    "deve " + coins(s.Debt) + ", da para pagar " + coins(pay),
			enabled: pay > 0,
			status:  "pagando a divida",
			action:  client.Action{Kind: "repay", Amount: pay},
		})
	}

	return items
}

func minutes(ticks int64) string {
	if ticks >= minutesPerHour {
		return strconv.FormatInt(ticks/minutesPerHour, 10) + " h"
	}

	return strconv.FormatInt(ticks, 10) + " min"
}
