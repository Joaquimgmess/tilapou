package tui

import (
	"fmt"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	criticalOxygenUgL = 2_000
	feederIndex       = 0
	aeratorIndex      = 1
	minRestockKg      = 10
	minStock          = 100
	shortRunwayDays   = 20
)

type advice struct {
	text   string
	urgent bool
}

func objective(s client.Snapshot) (text string, urgent bool) {
	if len(s.Tanks) == 0 {
		return "Compre um tanque com [t] para comecar", false
	}

	checks := []func(client.Snapshot) (advice, bool){
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
			return found.text, found.urgent
		}
	}

	tank := s.Tanks[0]
	if tank.Fish == 0 {
		if s.CashCents < s.Prices.FingerlingCents*minStock {
			return "Tanque vazio e sem grana para povoar. Pegue um emprestimo com [g]", true
		}

		return "O tanque esta vazio. Povoe com [s]", false
	}
	if s.Prices.RatioPPM < s.Prices.ViablePPM {
		return "Racao cara demais para o preco do peixe: segure a despesca e evite povoar agora", false
	}
	if tank.NextClassG > tank.MeanGrams {
		return fmt.Sprintf("Segurar ate %d g sobe o preco por quilo (esta em %d g)",
			tank.NextClassG, tank.MeanGrams), false
	}

	return fmt.Sprintf("Engorde ate 800 g (esta em %d g) e sirva o trato antes de acabar", tank.MeanGrams), false
}

func underStocked(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish == 0 || t.BreakEven <= 0 || int64(t.Fish) >= t.BreakEven {
			continue
		}

		missing := t.BreakEven - int64(t.Fish)
		if t.StockAdvice >= missing {
			return advice{fmt.Sprintf(
				"O tanque %d so paga a manutencao com %d peixes e tem %d. Povoe com [s]",
				t.ID, t.BreakEven, t.Fish), true}, true
		}

		return advice{fmt.Sprintf(
			"O tanque %d precisa de %d peixes para pagar a manutencao e o caixa so cobre %d. Pegue credito com [g]",
			t.ID, t.BreakEven, t.StockAdvice), true}, true
	}

	return advice{}, false
}

func suffocating(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.OxygenUgL < criticalOxygenUgL && !t.Aerating {
			return advice{fmt.Sprintf("URGENTE: o tanque %d esta sem oxigenio. Ligue o aerador com [a]", t.ID), true}, true
		}
	}

	return advice{}, false
}

func sickBatch(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Sick {
			return advice{fmt.Sprintf(
				"Doenca no tanque %d. Abra [z] e trate, ou aceite as perdas", t.ID), true}, true
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

		return advice{fmt.Sprintf(
			"O caixa dura %d dias e o lote do tanque %d so fecha em %d. Pegue credito com [g]",
			s.RunwayDays, t.ID, t.Decision.HoldDays), true}, true
	}

	return advice{}, false
}

func crushingDebt(s client.Snapshot) (advice, bool) {
	if s.Debt > 0 && s.CashCents == 0 {
		return advice{"Sem caixa e com divida: os juros estao crescendo. Venda peixe", true}, true
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
			return advice{fmt.Sprintf(
				"Sem racao e sem caixa. Abra [z] no tanque %d e raleie o lote para levantar dinheiro", t.ID), true}, true
		}

		return advice{fmt.Sprintf("O tanque %d ficou sem racao. Compre com [c]", t.ID), true}, true
	}

	return advice{}, false
}

func unfed(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish > 0 && t.ServedFor <= 0 && !owns(t, feederIndex) {
			return advice{fmt.Sprintf("Os peixes do tanque %d nao comem sozinhos. Sirva o trato com [f]", t.ID), true}, true
		}
	}

	return advice{}, false
}

func readyToHarvest(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Ready {
			return advice{fmt.Sprintf("O lote do tanque %d esta no ponto de abate. Despesque com [h]", t.ID), false}, true
		}
	}

	return advice{}, false
}

func affordableAutomation(s client.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if !owns(t, feederIndex) && affords(s, t, feederIndex) {
			return advice{"Da para comprar o comedouro com [1]: ele serve o trato sozinho", false}, true
		}
		if !owns(t, aeratorIndex) && affords(s, t, aeratorIndex) {
			return advice{"Da para comprar o aerador com [2]: ele liga sozinho na madrugada", false}, true
		}
	}

	return advice{}, false
}

func prestigeReady(s client.Snapshot) (advice, bool) {
	if s.PrestigeNow > s.Prestige {
		return advice{fmt.Sprintf("Voce pode tilapar com [p] e ganhar %d matrizes", s.PrestigeNow-s.Prestige), false}, true
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

var rejectMessages = map[string]string{
	"no_such_tank":        "esse tanque nao existe",
	"no_such_batch":       "nao ha lote nesse tanque",
	"not_enough_feed":     "sem racao no tanque: compre com [c]",
	"tank_full":           "o tanque ja tem lotes demais",
	"farm_full":           "a fazenda nao cabe mais tanques",
	"bad_amount":          "quantidade invalida",
	"too_dense":           "densidade estourada: esse tanque nao suporta tanto peixe",
	"already_owned":       "esse tanque ja tem essa automacao",
	"not_enough_lifetime": "ainda nao da para tilapar: fature mais primeiro",
	"unknown_kind":        "acao desconhecida",
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

	if message, ok := rejectMessages[outcome.Reason]; ok {
		return "Nao deu: " + message
	}

	return "Nao deu: " + outcome.Reason
}
