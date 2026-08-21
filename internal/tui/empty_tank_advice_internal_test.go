package tui

import (
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// O conselho do tanque vazio nao pode apontar uma tecla que abre tela de recusa: com o
// galpao bloqueado o [g] leva a "Emprestimo indisponivel", e a saida oferecida ao lado
// ("pague o que deve") e desmentida pelo item vizinho, que diz que da para pagar 0,00.
func TestConselhoDoTanqueVazioSoApontaCreditoQuandoOGalpaoAceita(t *testing.T) {
	t.Parallel()

	for _, block := range []string{"no_credit", "no_room", "no_need", "no_cycle"} {
		tank := api.Tank{ID: 1, StockAdvice: 0, LoanAdvice: 0, LoanBlock: block}

		if got := emptyTankAdvice(tank); strings.Contains(got, "[g]") {
			t.Errorf("com o galpao bloqueado por %s o conselho manda apertar [g]: %q", block, got)
		}
	}

	aberto := api.Tank{ID: 1, StockAdvice: 0, LoanAdvice: 30_000, LoanBlock: "open"}
	if got := emptyTankAdvice(aberto); !strings.Contains(got, "[g]") {
		t.Errorf("com o galpao aceitando o conselho deixou de apontar o credito: %q", got)
	}
}

// A dica do galpao nao pode mandar pagar a divida quando nao ha caixa para pagar: as duas
// unicas saidas da tela ficam impossiveis ao mesmo tempo.
func TestDicaDoGalpaoNaoMandaPagarDividaSemCaixa(t *testing.T) {
	t.Parallel()

	snap := sizedSnapshot()
	snap.CashCents = 0
	tank := snap.Tanks[0]

	for _, block := range []string{"no_credit", "no_cycle"} {
		tank.LoanBlock = block

		if got := loanHint(snap, tank); strings.Contains(got, "pague o que deve") {
			t.Errorf("com caixa 0 e bloqueio %s a dica manda pagar a divida: %q", block, got)
		}
	}
}
