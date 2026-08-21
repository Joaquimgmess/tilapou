package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// O painel do mercado fala de PRECO do dia; resultado do lote e do painel ao lado. Dizer "da
// lucro" ali punha os dois a discordar no mesmo quadro: a equivalencia dizia lucro enquanto a
// DECISAO dizia ABAIXO DO CUSTO e o lote perdia dinheiro. As duas afirmacoes eram verdadeiras
// e mediam coisas diferentes — o que estava errado era o rotulo.
func TestOMercadoNaoDaVeredictoSobreOLote(t *testing.T) {
	t.Parallel()

	for _, ratio := range []int64{2_320_000, 1_100_000} {
		got := ansi.Strip(viability(ratio, 1_250_000))

		for _, avoid := range []string{"lucro", "inviavel"} {
			if strings.Contains(got, avoid) {
				t.Errorf("com equivalencia %d o painel do mercado da veredicto: %q", ratio, got)
			}
		}
		// O piso ja chega no cliente e hoje e gasto so para produzir um adjetivo: mostra-lo
		// ensina o limiar e tira a leitura da dependencia da cor.
		if !strings.Contains(got, "1,25") {
			t.Errorf("com equivalencia %d o painel nao mostra o piso: %q", ratio, got)
		}
	}
}

// A DECISAO continua dando o veredicto do lote: e ela que tem o custo do peixe.
func TestADecisaoContinuaDizendoQuandoOLotePerde(t *testing.T) {
	t.Parallel()

	m := New(nil)
	m.snapshot = sizedSnapshot()
	m.width, m.height = 120, 40
	m.snapshot.Tanks[0].Batches[0].Decision.BreakEvenPerKg = 900
	m.snapshot.Tanks[0].Batches[0].PriceKgCents = 678

	if frame := ansi.Strip(m.renderDashboard()); !strings.Contains(frame, "ABAIXO DO CUSTO") {
		t.Errorf("com o preco abaixo do custo a DECISAO deixou de avisar:\n%s", frame)
	}
}
