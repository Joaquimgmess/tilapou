package tui

import (
	"fmt"
	"strings"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

const (
	criticalOxygenUgL = 2_000
	feederIndex       = 0
	aeratorIndex      = 1
	minRestockKg      = 10
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
func headline(s api.Snapshot, seen uint64) (string, bool) {
	var (
		pick *api.Event
		best int
	)

	for i := range s.Events {
		e := &s.Events[i]
		if e.Seq <= seen {
			continue
		}

		// Por peso, e nao por recencia: um evento repetido soterraria o surto ou a falencia
		// que decidem o lote.
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
	case "starvation_began", "starvation_ended", "hypoxia_deaths", "feed_exhausted":
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
func newestEvent(s api.Snapshot, seen uint64) uint64 {
	for i := range s.Events {
		seen = max(seen, s.Events[i].Seq)
	}

	return seen
}

func eventHeadline(e *api.Event) (string, bool) {
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
	case "starvation_began":
		return fmt.Sprintf("O lote do tanque %d comecou a morrer de fome", e.Tank), true
	case "starvation_ended":
		// O total so existe aqui: a abertura sai com o morto do primeiro tick, e e este numero
		// que diz o tamanho da perda.
		return fmt.Sprintf("A fome no tanque %d acabou: %s %s", e.Tank, fishCount(e.Fish), died(e.Fish)), true
	case "hypoxia_deaths":
		return fmt.Sprintf("%s do tanque %d %s sem oxigenio", fishCount(e.Fish), e.Tank, died(e.Fish)), true
	case "feed_exhausted":
		return fmt.Sprintf("A racao do tanque %d acabou", e.Tank), true
	}

	return "", false
}

// adviceTank returns the tank the current advice is about, or zero when it is about the farm.
func adviceTank(s api.Snapshot) uint32 {
	found, ok := current(s)
	if !ok {
		return 0
	}

	return found.tank
}

// objective returns the advice for the farm, already worded for the tank in focus: a
// tank-scoped key is only offered when it would act on the tank the advice names.
func objective(s api.Snapshot, focused uint32) (text string, urgent bool) {
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

func current(s api.Snapshot) (advice, bool) {
	if len(s.Tanks) == 0 {
		return advice{}, false
	}

	checks := []func(api.Snapshot) (advice, bool){
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
// creditFor diz se o galpao empresta neste tanque agora.
func creditFor(t api.Tank) bool {
	return t.LoanBlock == api.LoanOpen && t.LoanAdvice > 0
}

func creditRoom(s api.Snapshot) bool {
	for i := range s.Tanks {
		if s.Tanks[i].LoanAdvice > 0 {
			return true
		}
	}

	return false
}

// anyBatch reports whether some batch in the tank answers yes: the alert is about the tank,
// and one sick or ready batch is enough to raise it.
func anyBatch(t *api.Tank, yes func(*api.Batch) bool) bool {
	for i := range t.Batches {
		if yes(&t.Batches[i]) {
			return true
		}
	}

	return false
}

// thinAdvice is the way out when credit is gone: selling part of the batch buys feed for
// the rest. It returns false when there is no batch big enough to thin.
func thinAdvice(t *api.Tank) (string, bool) {
	if len(t.Batches) == 0 {
		return "", false
	}

	batch := &t.Batches[0]

	count := int64(batch.Fish) * thinPercent / fullPercent
	if count <= 0 || batch.MeanGrams <= 0 || batch.PriceKgCents <= 0 {
		return "", false
	}

	revenue := count * batch.MeanGrams * batch.PriceKgCents / gramsPerKg

	return fmt.Sprintf("Sem espaco no credito: raleie o tanque %d com [z] e venda %d peixes de %d g por ~%s",
		t.ID, count, batch.MeanGrams, coins(revenue)), true
}

func farmGoal(s api.Snapshot) string {
	tank := s.Tanks[0]
	if tank.Fish == 0 {
		// O motivo vem tipado do dominio, como na linha do tanque: deduzir por StockAdvice
		// <= 0 fazia o topo afirmar "sem grana" com dinheiro na barra de cima, contradizendo
		// a linha logo abaixo na mesma tela.
		switch tank.StockBlock {
		case api.StockOpen:
			return "O tanque esta vazio. Povoe com [s]"
		case api.StockNoTank, api.StockNoRoom, api.StockNoBatch:
		case api.StockNoCycle:
			return "Tanque vazio: " + emptyTankAdvice(s, tank)
		case api.StockShortFeed:
			// Os dois numeros ja vem prontos no payload: quanto falta de racao e quanto o
			// galpao empresta. Sem o credito citado, o jogador nao via o emprestimo que paga
			// a racao inteira.
			if creditFor(tank) {
				return fmt.Sprintf("Tanque vazio: faltam %s de racao. Credito em [g] paga, depois [s]",
					coins(tank.StockShort))
			}

			return "Tanque vazio: da para povoar com [s], mas o caixa nao paga a racao"
		case api.StockNoCash:
			if !creditRoom(s) {
				return "Tanque vazio, sem caixa e sem credito: so recomecando com [b]"
			}

			return "Tanque vazio e sem caixa para povoar. Veja o credito com [g]"
		}

		return "O tanque esta vazio"
	}
	if s.Prices.RatioPPM < s.Prices.ViablePPM {
		return "Racao cara demais para o preco do peixe: segure a despesca e evite povoar agora"
	}
	front := tank.Batches[0]
	if front.NextClassGrams > front.MeanGrams {
		return fmt.Sprintf("Segurar ate %d g sobe o preco por quilo (esta em %d g)",
			front.NextClassGrams, front.MeanGrams)
	}

	return fmt.Sprintf("Engorde ate o ponto de abate (esta em %d g) e sirva o trato antes de acabar",
		front.MeanGrams)
}

func underStocked(s api.Snapshot) (advice, bool) {
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
			"O tanque %d paga a manutencao com %d peixes; o caixa compra %d. [g] mostra o credito com juro e margem",
			t.ID, t.BreakEven, t.StockAdvice), urgent: true}, true
	}

	return advice{}, false
}

func broke(s api.Snapshot) (advice, bool) {
	if !s.Broke {
		return advice{}, false
	}

	// Com prestigio a colher a saida e tilapar: reconstroi a mesma fazenda e ainda devolve os
	// pontos que a partida rendeu. Oferecer o recomeco aqui seria oferecer a porta pior.
	// "recomeca do zero" e o verbo que o jogador ja associa ao [b]: sem ele, o [p] e lido como
	// algo guardado para depois, e a diferenca (as matrizes) e a unica coisa que o [b] nao tem.
	if s.PrestigeNow > s.Prestige {
		return advice{text: fmt.Sprintf(
			"A fazenda quebrou: tilapar com [p] recomeca do zero e ainda da %d matrizes",
			s.PrestigeNow-s.Prestige), urgent: true}, true
	}

	// Diz o que e verdade e nada mais: enumerar motivos obriga a verificar cada um, e a lista
	// antiga afirmava "sem peixe" com 500 peixes vivos e "sem credito" com o galpao aberto na
	// mesma tela.
	return advice{text: "A fazenda quebrou: nao resta jogada possivel. Recomece do zero com [b]", urgent: true}, true
}

func suffocating(s api.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish == 0 {
			continue
		}
		if t.OxygenUgL < criticalOxygenUgL && !t.Aerating {
			return advice{text: fmt.Sprintf("URGENTE: o tanque %d esta sem oxigenio. Ligue o aerador com [a]", t.ID), urgent: true, tank: t.ID, key: "a"}, true
		}
	}

	return advice{}, false
}

func sickBatch(s api.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if anyBatch(t, func(b *api.Batch) bool { return b.Sick }) {
			return advice{text: fmt.Sprintf(
				"Doenca no tanque %d. Abra [z] e trate, ou aceite as perdas", t.ID), urgent: true, tank: t.ID, key: "z"}, true
		}
	}

	return advice{}, false
}

func shortRunway(s api.Snapshot) (advice, bool) {
	if s.RunwayDays < 0 || s.RunwayDays >= shortRunwayDays {
		return advice{}, false
	}

	for i := range s.Tanks {
		t := &s.Tanks[i]
		front, ok := frontBatch(*t)
		if !ok || t.Fish == 0 || front.Decision.HoldDays <= s.RunwayDays {
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
			s.RunwayDays, t.ID, front.Decision.HoldDays), urgent: true}, true
	}

	return advice{}, false
}

func crushingDebt(s api.Snapshot) (advice, bool) {
	if s.Debt <= 0 || s.CashCents != 0 {
		return advice{}, false
	}
	// Fazenda quebrada tem dono proprio na lista de conselhos: falar aqui tambem daria dois
	// conselhos para a mesma tela.
	if s.Broke {
		return advice{}, false
	}
	// Com credito aberto a saida e o galpao, e nao o recomeco — que neste estado responde que
	// a fazenda nao quebrou. Guardar so divida e caixa fazia isto disparar com 0,07 TC.
	if creditRoom(s) {
		return advice{text: "Sem caixa e com divida: o credito e a saida, veja com [g]", urgent: true}, true
	}

	// Mandar vender peixe sem peixe e o mesmo defeito do conselho do tanque vazio, uma linha
	// acima na mesma tela: a saida tem de ser a que responde neste estado.
	if s.Fish > 0 {
		return advice{text: "Sem caixa e com divida: os juros crescem. Venda peixe com [h]", urgent: true}, true
	}

	// Sem enumerar de novo: o que vale dizer e que nao ha jogada, e isso o Broke ja sabe.
	return advice{text: "Sem caixa e com divida, e nao resta jogada: recomece do zero com [b]", urgent: true}, true
}

func outOfFeed(s api.Snapshot) (advice, bool) {
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

func unfed(s api.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if t.Fish > 0 && t.ServedFor <= 0 && !owns(t, feederIndex) {
			return advice{text: fmt.Sprintf("Os peixes do tanque %d nao comem sozinhos. Sirva o trato com [f]", t.ID), urgent: true, tank: t.ID, key: "f"}, true
		}
	}

	return advice{}, false
}

func readyToHarvest(s api.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		if anyBatch(t, func(b *api.Batch) bool { return b.Ready }) {
			return advice{text: fmt.Sprintf("O lote do tanque %d esta no ponto de abate. Despesque com [h]", t.ID), tank: t.ID, key: "h"}, true
		}
	}

	return advice{}, false
}

func affordableAutomation(s api.Snapshot) (advice, bool) {
	for i := range s.Tanks {
		t := &s.Tanks[i]
		// Automacao em tanque vazio nao muda rotina nenhuma: primeiro tem de haver peixe.
		if t.Fish == 0 {
			continue
		}
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

func prestigeReady(s api.Snapshot) (advice, bool) {
	if s.PrestigeNow > s.Prestige {
		return advice{text: fmt.Sprintf("Voce pode tilapar com [p] e ganhar %d matrizes", s.PrestigeNow-s.Prestige)}, true
	}

	return advice{}, false
}

func owns(t *api.Tank, index int) bool {
	if index < 0 || index >= len(t.Upgrades) {
		return false
	}

	return t.Upgrades[index].Owned
}

func affords(s api.Snapshot, t *api.Tank, index int) bool {
	if index < 0 || index >= len(t.Upgrades) {
		return false
	}

	return s.CashCents >= t.Upgrades[index].CostCents
}

// rejectMessageMore continua a tabela de rejectMessage: uma so estourava o limite de galhos.
func rejectMessageMore(reason string) (string, bool) {
	switch reason {
	case "stale_view":
		return "a tela ja estava velha quando isso chegou: olhe os numeros e decida de novo", true
	case "too_dense":
		return "densidade estourada: esse tanque nao suporta tanto peixe", true
	case "already_owned":
		return "esse tanque ja tem essa automacao", true
	case "not_enough_lifetime":
		return "ainda nao da para tilapar: fature mais primeiro", true
	}

	return "", false
}

func rejectMessage(reason string) (string, bool) {
	if message, ok := rejectMessageMore(reason); ok {
		return message, true
	}

	switch reason {
	case "not_broke":
		return "a fazenda ainda tem como se virar", true
	case "prestige_first":
		return "ha prestigio a colher: [p] recomeca do zero igual, e ainda da as matrizes", true
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
	case "unknown_kind":
		return "acao desconhecida", true
	}

	return "", false
}

func explain(outcome *api.Outcome, cash int64) string {
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
