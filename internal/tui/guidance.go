package tui

import (
	"fmt"
	"strings"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	criticalOxygenUgL = 2_000
	feederIndex       = 0
	aeratorIndex      = 1
	minRestockKg      = 10
	minStock          = 100
	shortRunwayDays   = 20
	jumpKey           = "."
)

type advice struct {
	text   string
	urgent bool
	tank   uint32
	key    string
}

// headline turns the newest event past seen into a line the player reads. The events are
// what the farm did while nobody was watching, and without this they only ever existed in
// the database: the farm could go bankrupt and the screen would say nothing.
func headline(s client.Snapshot, seen uint64) (string, bool) {
	var (
		pick *client.Event
		best int
	)

	for i := range s.Events {
		e := &s.Events[i]
		if e.Seq <= seen {
			continue
		}

		// Por peso, e nao por recencia: a fome emite uma linha por tick e soterraria o
		// surto ou a falencia que decidem o lote.
		if weight := eventWeight(e.Kind); weight > best {
			pick, best = e, weight
		}
	}

	if pick == nil {
		return "", false
	}

	return eventHeadline(pick)
}

// How much an event deserves the one line the screen has for it.
const (
	weightNone = iota
	weightLoss
	weightThreat
	weightTurningPoint
)

func eventWeight(kind string) int {
	switch kind {
	case "bankrupt", "prestiged", "restarted":
		return weightTurningPoint
	case "disease", "disease_deaths":
		return weightThreat
	case "starvation_deaths", "hypoxia_deaths", "feed_exhausted":
		return weightLoss
	}

	return weightNone
}

func fishCount(fish int32) string {
	if fish == 1 {
		return "1 peixe"
	}

	return fmt.Sprintf("%d peixes", fish)
}

func died(fish int32) string {
	if fish == 1 {
		return "morreu"
	}

	return "morreram"
}

// newestEvent is the highest sequence the player has now been shown.
func newestEvent(s client.Snapshot, seen uint64) uint64 {
	for i := range s.Events {
		seen = max(seen, s.Events[i].Seq)
	}

	return seen
}

func eventHeadline(e *client.Event) (string, bool) {
	switch e.Kind {
	case "bankrupt":
		return "A fazenda quebrou: a divida de " + coins(e.CashCents) +
			" foi perdoada e voce recomeca com o lote inicial", true
	case "prestiged":
		return "Voce tilapou: a fazenda recomeca com as matrizes valendo para sempre", true
	case "restarted":
		return "A fazenda recomecou do zero", true
	case "disease":
		return fmt.Sprintf("Surto de doenca no tanque %d: trate com [z] ou aceite as perdas", e.Tank), true
	case "disease_deaths":
		return fmt.Sprintf("A doenca matou %s no tanque %d", fishCount(e.Fish), e.Tank), true
	case "starvation_deaths":
		return fmt.Sprintf("%s do tanque %d %s de fome", fishCount(e.Fish), e.Tank, died(e.Fish)), true
	case "hypoxia_deaths":
		return fmt.Sprintf("%s do tanque %d %s sem oxigenio", fishCount(e.Fish), e.Tank, died(e.Fish)), true
	case "feed_exhausted":
		return fmt.Sprintf("A racao do tanque %d acabou", e.Tank), true
	}

	return "", false
}

// adviceTank returns the tank the current advice is about, or zero when it is about the farm.
func adviceTank(s client.Snapshot) uint32 {
	found, ok := current(s)
	if !ok {
		return 0
	}

	return found.tank
}

// objective returns the advice for the farm, already worded for the tank in focus: a
// tank-scoped key is only offered when it would act on the tank the advice names.
func objective(s client.Snapshot, focused uint32) (text string, urgent bool) {
	if len(s.Tanks) == 0 {
		return "Compre um tanque com [t] para comecar", false
	}

	found, ok := current(s)
	if !ok {
		return farmGoal(s), false
	}
	if found.tank != 0 && found.tank != focused {
		return strings.Replace(found.text, "["+found.key+"]", "["+jumpKey+"]", 1) +
			" (o " + jumpKey + " leva ate ele)", found.urgent
	}

	return found.text, found.urgent
}

func current(s client.Snapshot) (advice, bool) {
	if len(s.Tanks) == 0 {
		return advice{}, false
	}

	checks := []func(client.Snapshot) (advice, bool){
		broke,
		suffocating,
		sickBatch,
		outOfFeed,
		shortRunway,
		crushingDebt,
		unfed,
		readyToHarvest,
		underStocked,
		affordableAutomation,
		prestigeReady,
	}

	for _, check := range checks {
		if found, ok := check(s); ok {
			return found, true
		}
	}

	return advice{}, false
}

// creditRoom reports whether any tank can still borrow: with the limit maxed out, telling
// the player to take credit points at an option that does nothing.
func creditRoom(s client.Snapshot) bool {
	for i := range s.Tanks {
		if s.Tanks[i].LoanAdvice > 0 {
			return true
		}
	}

	return false
}

// thinAdvice is the way out when credit is gone: selling part of the batch buys feed for
// the rest. It returns false when there is no batch big enough to thin.
func thinAdvice(t *client.Tank) (string, bool) {
	count := int64(t.BatchFish) * thinPercent / fullPercent
	if count <= 0 || t.MeanGrams <= 0 || t.PriceKgCents <= 0 {
		return "", false
	}

	revenue := count * t.MeanGrams * t.PriceKgCents / gramsPerKg

	return fmt.Sprintf("Sem espaco no credito: raleie o tanque %d com [z] e venda %d peixes de %d g por ~%s",
		t.ID, count, t.MeanGrams, coins(revenue)), true
}

func farmGoal(s client.Snapshot) string {
	tank := s.Tanks[0]
	if tank.Fish == 0 {
		if s.CashCents < s.Prices.FingerlingCents*minStock {
			if !creditRoom(s) {
				return "Tanque vazio, sem grana e sem credito: so recomecando com [b]"
			}

			return "Tanque vazio e sem grana para povoar. Pegue um emprestimo com [g]"
		}

		return "O tanque esta vazio. Povoe com [s]"
	}
	if s.Prices.RatioPPM < s.Prices.ViablePPM {
		return "Racao cara demais para o preco do peixe: segure a despesca e evite povoar agora"
	}
	if tank.NextClassGrams > tank.MeanGrams {
		return fmt.Sprintf("Segurar ate %d g sobe o preco por quilo (esta em %d g)",
			tank.NextClassGrams, tank.MeanGrams)
	}

	return fmt.Sprintf("Engorde ate 800 g (esta em %d g) e sirva o trato antes de acabar", tank.MeanGrams)
}

func underStocked(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish == 0 || t.BreakEven <= 0 || int64(t.Fish) >= t.BreakEven {
			continue
		}

		missing := t.BreakEven - int64(t.Fish)
		if t.StockAdvice >= missing {
			return advice{text: fmt.Sprintf(
				"O tanque %d so paga a manutencao com %d peixes e tem %d. Povoe com [s]",
				t.ID, t.BreakEven, t.Fish), urgent: true, tank: t.ID, key: "s"}, true
		}

		if !creditRoom(s) {
			text, ok := thinAdvice(t)
			if !ok {
				continue
			}

			return advice{text: text, urgent: true, tank: t.ID, key: "z"}, true
		}

		return advice{text: fmt.Sprintf(
			"O tanque %d precisa de %d peixes para pagar a manutencao e o caixa so cobre %d. Pegue credito com [g]",
			t.ID, t.BreakEven, t.StockAdvice), urgent: true}, true
	}

	return advice{}, false
}

func broke(s client.Snapshot) (advice, bool) {
	if !s.Broke {
		return advice{}, false
	}

	return advice{text: "A fazenda quebrou: sem peixe, sem caixa e sem credito. Recomece do zero com [b]", urgent: true}, true
}

func suffocating(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.OxygenUgL < criticalOxygenUgL && !t.Aerating {
			return advice{text: fmt.Sprintf("URGENTE: o tanque %d esta sem oxigenio. Ligue o aerador com [a]", t.ID), urgent: true, tank: t.ID, key: "a"}, true
		}
	}

	return advice{}, false
}

func sickBatch(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Sick {
			return advice{text: fmt.Sprintf(
				"Doenca no tanque %d. Abra [z] e trate, ou aceite as perdas", t.ID), urgent: true, tank: t.ID, key: "z"}, true
		}
	}

	return advice{}, false
}

func shortRunway(s client.Snapshot) (advice, bool) {
	if s.RunwayDays < 0 || s.RunwayDays >= shortRunwayDays {
		return advice{}, false
	}

	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish == 0 || t.Decision.HoldDays <= s.RunwayDays {
			continue
		}

		if !creditRoom(s) {
			text, ok := thinAdvice(t)
			if !ok {
				continue
			}

			return advice{text: text, urgent: true, tank: t.ID, key: "z"}, true
		}

		return advice{text: fmt.Sprintf(
			"O caixa dura %d dias e o lote do tanque %d so fecha em %d. Pegue credito com [g]",
			s.RunwayDays, t.ID, t.Decision.HoldDays), urgent: true}, true
	}

	return advice{}, false
}

func crushingDebt(s client.Snapshot) (advice, bool) {
	if s.Debt > 0 && s.CashCents == 0 {
		return advice{text: "Sem caixa e com divida: os juros estao crescendo. Venda peixe", urgent: true}, true
	}

	return advice{}, false
}

func outOfFeed(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish == 0 || t.FeedKg > 0 {
			continue
		}
		if s.CashCents < s.Prices.FeedKgCents*minRestockKg {
			return advice{text: fmt.Sprintf(
				"Sem racao e sem caixa. Abra [z] no tanque %d e raleie o lote para levantar dinheiro", t.ID), urgent: true, tank: t.ID, key: "z"}, true
		}

		return advice{text: fmt.Sprintf("O tanque %d ficou sem racao. Compre com [c]", t.ID), urgent: true, tank: t.ID, key: "c"}, true
	}

	return advice{}, false
}

func unfed(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish > 0 && t.ServedFor <= 0 && !owns(t, feederIndex) {
			return advice{text: fmt.Sprintf("Os peixes do tanque %d nao comem sozinhos. Sirva o trato com [f]", t.ID), urgent: true, tank: t.ID, key: "f"}, true
		}
	}

	return advice{}, false
}

func readyToHarvest(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Ready {
			return advice{text: fmt.Sprintf("O lote do tanque %d esta no ponto de abate. Despesque com [h]", t.ID), tank: t.ID, key: "h"}, true
		}
	}

	return advice{}, false
}

func affordableAutomation(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if !owns(t, feederIndex) && affords(s, t, feederIndex) {
			return advice{text: fmt.Sprintf(
				"Da para comprar o comedouro do tanque %d com [1]: ele serve o trato sozinho", t.ID), tank: t.ID, key: "1"}, true
		}
		if !owns(t, aeratorIndex) && affords(s, t, aeratorIndex) {
			return advice{text: fmt.Sprintf(
				"Da para comprar o aerador do tanque %d com [2]: ele liga sozinho na madrugada", t.ID), tank: t.ID, key: "2"}, true
		}
	}

	return advice{}, false
}

func prestigeReady(s client.Snapshot) (advice, bool) {
	if s.PrestigeNow > s.Prestige {
		return advice{text: fmt.Sprintf("Voce pode tilapar com [p] e ganhar %d matrizes", s.PrestigeNow-s.Prestige)}, true
	}

	return advice{}, false
}

func owns(t *client.Tank, index int) bool {
	if index < 0 || index >= len(t.Upgrades) {
		return false
	}

	return t.Upgrades[index].Owned
}

func affords(s client.Snapshot, t *client.Tank, index int) bool {
	if index < 0 || index >= len(t.Upgrades) {
		return false
	}

	return s.CashCents >= t.Upgrades[index].CostCents
}

func rejectMessage(reason string) (string, bool) {
	switch reason {
	case "not_broke":
		return "a fazenda ainda tem como se virar", true
	case "credit_limit":
		return "o emprestimo passa do limite de credito: pague o que deve antes", true
	case "no_debt":
		return "nao ha divida para pagar", true
	case "nothing_sick":
		return "nao ha doenca nesse tanque", true
	case "no_such_tank":
		return "esse tanque nao existe", true
	case "no_such_batch":
		return "nao ha lote nesse tanque", true
	case "not_enough_feed":
		return "sem racao no tanque: compre com [c]", true
	case "tank_full":
		return "o tanque ja tem lotes demais", true
	case "farm_full":
		return "a fazenda nao cabe mais tanques", true
	case "bad_amount":
		return "quantidade invalida", true
	case "too_dense":
		return "densidade estourada: esse tanque nao suporta tanto peixe", true
	case "already_owned":
		return "esse tanque ja tem essa automacao", true
	case "not_enough_lifetime":
		return "ainda nao da para tilapar: fature mais primeiro", true
	case "unknown_kind":
		return "acao desconhecida", true
	}

	return "", false
}

func explain(outcome *client.Outcome, cash int64) string {
	if outcome == nil || outcome.Applied {
		return ""
	}

	if outcome.Reason == "not_enough_cash" {
		if outcome.NeededCash > 0 {
			return fmt.Sprintf("Sem grana: custa %s e faltam %s",
				coins(outcome.NeededCash), coins(outcome.NeededCash-cash))
		}

		return "Sem grana para isso"
	}

	if message, ok := rejectMessage(outcome.Reason); ok {
		return "Nao deu: " + message
	}

	return "Nao deu: " + outcome.Reason
}
