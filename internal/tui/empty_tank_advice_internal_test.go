package tui

import (
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// O conselho do tanque vazio sai do motivo tipado que o dominio calculou. Antes ele
// readivinhava por StockAdvice <= 0, e por isso afirmava "sem caixa" com caixa no bolso e
// apontava tecla que o jogo recusa.
func TestConselhoDoTanqueVazioSaiDoMotivoENaoDoPalpite(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome    string
		snap    api.Snapshot
		tank    api.Tank
		quer    []string
		naoQuer []string
	}{
		{
			nome: "povoar aberto",
			snap: api.Snapshot{CashCents: 500_000, Fish: 2_000},
			tank: api.Tank{ID: 1, StockBlock: api.StockOpen, StockAdvice: 300},
			quer: []string{"[s]"},
		},
		{
			nome:    "caixa positivo que nao fecha o ciclo",
			snap:    api.Snapshot{CashCents: 49_995, Fish: 2_000},
			tank:    api.Tank{ID: 1, StockBlock: api.StockNoCycle, StockShort: 120_000},
			quer:    []string{"1200,00", "outro tanque"},
			naoQuer: []string{"sem caixa"},
		},
		{
			nome:    "caixa zero com credito aberto",
			snap:    api.Snapshot{CashCents: 0, Fish: 2_000},
			tank:    api.Tank{ID: 1, StockBlock: api.StockNoCash, LoanBlock: api.LoanOpen, LoanAdvice: 30_000},
			quer:    []string{"sem caixa", "[g]"},
			naoQuer: []string{"[h]", "[b]"},
		},
		{
			nome:    "caixa zero, credito fechado, peixe na fazenda",
			snap:    api.Snapshot{CashCents: 0, Fish: 2_000},
			tank:    api.Tank{ID: 1, StockBlock: api.StockNoCash, LoanBlock: api.LoanNoCycle},
			quer:    []string{"outro tanque"},
			naoQuer: []string{"[g]", "[b]", "[h]"},
		},
		{
			nome:    "caixa zero, credito fechado, fazenda sem peixe",
			snap:    api.Snapshot{CashCents: 0, Fish: 0},
			tank:    api.Tank{ID: 1, StockBlock: api.StockNoCash, LoanBlock: api.LoanNoCycle},
			quer:    []string{"[b]"},
			naoQuer: []string{"[h]", "[g]"},
		},
		{
			nome:    "tanque cheio",
			snap:    api.Snapshot{CashCents: 500_000, Fish: 5_000},
			tank:    api.Tank{ID: 1, StockBlock: api.StockNoRoom},
			quer:    []string{"cheio", "[h]"},
			naoQuer: []string{"sem caixa"},
		},
		{
			nome:    "lotes no limite",
			snap:    api.Snapshot{CashCents: 500_000, Fish: 5_000},
			tank:    api.Tank{ID: 1, StockBlock: api.StockNoBatch},
			quer:    []string{"lote", "[h]"},
			naoQuer: []string{"sem caixa"},
		},
	}

	for _, caso := range casos {
		got := emptyTankAdvice(caso.snap, caso.tank)

		for _, want := range caso.quer {
			if !strings.Contains(got, want) {
				t.Errorf("%s: o conselho nao diz %q: %q", caso.nome, want, got)
			}
		}
		for _, avoid := range caso.naoQuer {
			if strings.Contains(got, avoid) {
				t.Errorf("%s: o conselho diz %q, que nao vale neste estado: %q", caso.nome, avoid, got)
			}
		}
	}
}

// A dica do galpao escolhe a saida pelo estado, e nao por um palpite sobre o caixa: mandar
// pagar divida com caixa zerado, ou vender peixe sem peixe, e mandar fazer o que a tela ao
// lado ja nega.
func TestDicaDoGalpaoApontaSaidaQueResponde(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome    string
		snap    api.Snapshot
		quer    string
		naoQuer string
	}{
		{nome: "com caixa e divida", snap: api.Snapshot{CashCents: 100_000, Debt: 700_000, Fish: 2_000}, quer: "pague o que deve", naoQuer: "vendendo peixe"},
		{nome: "sem caixa, com peixe", snap: api.Snapshot{CashCents: 0, Debt: 700_000, Fish: 2_000}, quer: "venda peixe", naoQuer: "pague o que deve"},
		{nome: "sem caixa, sem peixe", snap: api.Snapshot{CashCents: 0, Debt: 700_000, Fish: 0}, quer: "[b]", naoQuer: "pague o que deve"},
	}

	for _, caso := range casos {
		tank := api.Tank{ID: 1, LoanBlock: api.LoanNoCycle}

		got := loanHint(caso.snap, tank)
		if !strings.Contains(got, caso.quer) {
			t.Errorf("%s: a dica nao diz %q: %q", caso.nome, caso.quer, got)
		}
		if strings.Contains(got, caso.naoQuer) {
			t.Errorf("%s: a dica diz %q, que nao vale neste estado: %q", caso.nome, caso.naoQuer, got)
		}
	}
}

// O objetivo do topo nao pode mandar vender peixe numa fazenda sem peixe — era a contradicao
// que sobrava na mesma tela em que o conselho do tanque ja apontava o recomeco.
func TestObjetivoDoTopoNaoMandaVenderPeixeQueNaoExiste(t *testing.T) {
	t.Parallel()

	semPeixe, ok := crushingDebt(api.Snapshot{Debt: 700_000, CashCents: 0, Fish: 0})
	if !ok {
		t.Fatal("com divida e caixa zero o objetivo nao apareceu")
	}
	if strings.Contains(semPeixe.text, "Venda peixe") {
		t.Errorf("sem um peixe na fazenda o objetivo manda vender peixe: %q", semPeixe.text)
	}

	comPeixe, ok := crushingDebt(api.Snapshot{Debt: 700_000, CashCents: 0, Fish: 2_000})
	if !ok {
		t.Fatal("com divida, caixa zero e peixe o objetivo nao apareceu")
	}
	if !strings.Contains(comPeixe.text, "Venda peixe") {
		t.Errorf("com peixe na fazenda o objetivo deixou de apontar a despesca: %q", comPeixe.text)
	}
}

// Mandar pagar divida sem divida e mandar fazer o que nao existe: com Debt zero o galpao
// mostrava "pague o que deve antes" e o menu nem tinha o item de pagar. A dica so pode
// apontar a saida que o estado sustenta.
func TestDicaDoGalpaoNaoMandaPagarDividaQuandoNaoHaDivida(t *testing.T) {
	t.Parallel()

	semDivida := api.Snapshot{CashCents: 100_000, Debt: 0, Fish: 2_000}
	comDivida := api.Snapshot{CashCents: 100_000, Debt: 700_000, Fish: 2_000}
	tank := api.Tank{ID: 1, LoanBlock: api.LoanNoCycle}

	if got := loanHint(semDivida, tank); strings.Contains(got, "pague o que deve") {
		t.Errorf("sem divida a dica manda pagar o que deve: %q", got)
	}
	if got := loanHint(comDivida, tank); !strings.Contains(got, "pague o que deve") {
		t.Errorf("com divida e caixa a dica deixou de apontar o pagamento: %q", got)
	}
}
