package tui

import (
	"fmt"
	"strconv"

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

func tankMenu(s client.Snapshot, t client.Tank) *menu {
	items := make([]menuItem, 0, len(t.Upgrades)+fixedTankItems)
	items = append(items,
		feedItem(t),
		buyFeedItem(s, t),
		aeratorItem(t),
		harvestItem(t),
		thinItem(s, t),
		treatItem(t),
		stockItem(s, t),
	)

	for _, upgrade := range t.Upgrades {
		items = append(items, upgradeItem(s, t, upgrade))
	}

	items = append(items, menuItem{
		label:   "Ver painel de numeros",
		hint:    "tecla m",
		enabled: true,
		panel:   true,
	})

	return &menu{title: fmt.Sprintf("TANQUE %d", t.ID), items: items}
}

func feedItem(t client.Tank) menuItem {
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

func buyFeedItem(s client.Snapshot, t client.Tank) menuItem {
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

func aeratorItem(t client.Tank) menuItem {
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

func harvestItem(t client.Tank) menuItem {
	item := menuItem{
		label:   "Despescar o lote",
		enabled: t.Fish > 0,
		status:  "despescando",
		action:  client.Action{Kind: "harvest", Tank: t.ID, Batch: t.BatchID},
	}

	switch {
	case t.Fish == 0:
		item.hint = "nao ha peixe aqui"
	case t.Ready:
		item.hint = fmt.Sprintf("%d peixes de %d g, no ponto", t.Fish, t.MeanGrams)
	default:
		item.hint = fmt.Sprintf("so %d g: vender agora rende menos", t.MeanGrams)
	}

	return item
}

func thinItem(s client.Snapshot, t client.Tank) menuItem {
	count := int64(t.Fish) * thinPercent / fullPercent
	revenue := count * t.MeanGrams * s.Prices.FishKgCents / gramsPerKg

	item := menuItem{
		label:   fmt.Sprintf("Ralear %d%% do lote", thinPercent),
		enabled: count > 0,
		status:  "raleando o lote",
		action:  client.Action{Kind: "harvest", Tank: t.ID, Batch: t.BatchID, Amount: count},
		hint:    fmt.Sprintf("vende %d peixes por ~%s e o resto continua crescendo", count, coins(revenue)),
	}
	if count <= 0 {
		item.hint = "nao ha peixe suficiente"
	}

	return item
}

func treatItem(t client.Tank) menuItem {
	item := menuItem{
		label:   "Tratar o lote",
		enabled: t.Sick,
		status:  "tratando o lote",
		action:  client.Action{Kind: "treat", Tank: t.ID},
		hint:    "nao ha doenca nesse tanque",
	}
	if t.Sick {
		item.hint = "cura agora, mas deixa portadores no tanque por um tempo"
	}

	return item
}

func stockItem(s client.Snapshot, t client.Tank) menuItem {
	room := t.Capacity - int64(t.Fish)
	amount := min(room, fingerlingsPerBuy)
	price := amount * s.Prices.FingerlingCents

	item := menuItem{
		label:   "Povoar com alevinos",
		enabled: room > 0 && s.CashCents >= price && amount > 0,
		status:  "povoando",
		action:  client.Action{Kind: "stock", Tank: t.ID, Amount: amount},
	}

	switch {
	case room <= 0:
		item.hint = "densidade no limite: " + strconv.FormatInt(t.Capacity, 10) + " peixes"
	case s.CashCents < price:
		item.hint = "faltam " + coins(price-s.CashCents)
	default:
		item.hint = fmt.Sprintf("%d alevinos por %s, cabem %d", amount, coins(price), room)
	}

	return item
}

func upgradeItem(s client.Snapshot, t client.Tank, upgrade client.Upgrade) menuItem {
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
		item.hint = coins(upgrade.CostCents) + " — " + automationHint(upgrade.Kind)
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

func shedMenu(s client.Snapshot, t client.Tank) *menu {
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

	items = append(items, creditItems(s)...)

	items = append(items, menuItem{
		label:   "Comprar outro viveiro",
		hint:    "amplia a fazenda",
		enabled: true,
		status:  "comprando um viveiro",
		action:  client.Action{Kind: "buy_tank", TankKind: "viveiro_escavado"},
	})

	return &menu{title: shedTitle, items: items}
}

func creditItems(s client.Snapshot) []menuItem {
	const loan = 500_000

	items := []menuItem{{
		label:   "Pegar emprestimo de " + coins(loan),
		hint:    "juros correm todo dia sobre o saldo",
		enabled: true,
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
