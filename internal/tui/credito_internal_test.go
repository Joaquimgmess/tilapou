package tui

import (
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// A linha do credito mostra o custo e para de afirmar resultado: "cobre os N peixes que
// faltam para pagar a manutencao" e promessa, e o ciclo que ela vende pode fechar no
// prejuizo. O jogador ve principal, quanto devolve no fim e a margem projetada, e decide.
func TestALinhaDoCreditoMostraOCustoENaoPrometeResultado(t *testing.T) {
	t.Parallel()

	tank := api.Tank{
		ID: 1, Fish: 0, BreakEven: 824, StockAdvice: 0,
		LoanAdvice: 188_336, LoanFish: 400, LoanBlock: api.LoanOpen,
		LoanOwed: 224_000, CycleDays: 189, CycleMargin: 223_932,
	}
	snap := api.Snapshot{CashCents: 50_000, Fish: 0}

	got := loanHint(snap, tank)

	for _, want := range []string{"1883,36", "2240,00", "189 d", "2239,32"} {
		if !strings.Contains(got, want) {
			t.Errorf("a linha do credito nao mostra %q: %q", want, got)
		}
	}
	for _, avoid := range []string{"cobre", "Pegue"} {
		if strings.Contains(got, avoid) {
			t.Errorf("a linha do credito ainda afirma resultado com %q: %q", avoid, got)
		}
	}
}

// Ciclo sem margem projetada nao pode aparecer com um numero de margem que nao existe.
func TestALinhaDoCreditoDizQuandoOCicloNaoProjetaMargem(t *testing.T) {
	t.Parallel()

	tank := api.Tank{
		// LoanFish cobre o que falta: a ressalva "da para N dos M peixes" nao entra, e a linha
		// e a do custo. O @game-design decidiu que aquela ressalva fica como esta.
		ID: 1, BreakEven: 200, LoanAdvice: 100_000, LoanFish: 200, LoanBlock: api.LoanOpen,
		LoanOwed: 118_900, CycleDays: 189, CycleMargin: -446_403,
	}

	got := loanHint(api.Snapshot{CashCents: 50_000}, tank)
	if !strings.Contains(got, "nao projeta margem") {
		t.Errorf("com margem negativa a linha nao avisa: %q", got)
	}
	if strings.Contains(got, "-4464,03") {
		t.Errorf("a linha exibe a margem negativa como se fosse um ganho: %q", got)
	}
}

// O objetivo do topo e o conselho do tanque nao podem discordar na mesma tela: o topo
// adivinhava o motivo por StockAdvice <= 0, e por isso mandava pegar emprestimo enquanto a
// linha do tanque, ja consertada, dizia quanto faltava e apontava outra saida.
func TestObjetivoDoTopoEConselhoDoTanqueContamAMesmaHistoria(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome string
		snap api.Snapshot
	}{
		{
			nome: "caixa positivo que nao fecha o ciclo, credito aberto",
			snap: api.Snapshot{
				CashCents: 199_845,
				Tanks: []api.Tank{{
					ID: 1, StockBlock: api.StockNoCycle, StockShort: 30_355,
					LoanBlock: api.LoanOpen, LoanAdvice: 500_000,
				}},
			},
		},
		{
			nome: "caixa zero, credito fechado, fazenda sem peixe",
			snap: api.Snapshot{
				CashCents: 0,
				Tanks: []api.Tank{{
					ID: 1, StockBlock: api.StockNoCash, LoanBlock: api.LoanNoCycle,
				}},
			},
		},
	}

	for _, caso := range casos {
		topo := farmGoal(caso.snap)
		linha := emptyTankAdvice(caso.snap, caso.snap.Tanks[0])

		for _, key := range []string{"[g]", "[h]", "[b]"} {
			if strings.Contains(topo, key) != strings.Contains(linha, key) {
				t.Errorf("%s: o topo diz %q e a linha do tanque diz %q — discordam sobre %s",
					caso.nome, topo, linha, key)
			}
		}

		// Afirmar "sem grana" com dinheiro na barra de cima e o mesmo defeito que a linha do
		// tanque deixou de ter: o topo tem de sair do motivo, e nao de StockAdvice <= 0.
		if caso.snap.CashCents > 0 && strings.Contains(topo, "sem grana") {
			t.Errorf("%s: o topo afirma 'sem grana' com %s no caixa: %q",
				caso.nome, coins(caso.snap.CashCents), topo)
		}
	}
}
