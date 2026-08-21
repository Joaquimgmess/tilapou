package sim

import "testing"

// O evento de racao acabada registra um fato, e nao uma amostra do estado: emitindo a cada
// janela enquanto o silo estiver vazio, um tanque em seca enche o log sozinho e empurra para
// fora dele o que o jogador precisa ler — o @qa mediu 40 de 40 eventos sendo este, com a
// propria falencia fora da janela.
func TestRacaoAcabadaEmiteUmaVezPorTransicao(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 91)
	s.Cash = 10_000_000
	s.Tanks[0].FeedStock = 200 * MicrogramsPerKilogram
	s.Tanks[0].ServedUntil = Tick(maxInt32)

	conta := func(out Output) int {
		n := 0
		for _, e := range out.Events {
			if e.Kind == EventFeedExhausted {
				n++
			}
		}

		return n
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 30*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("o ciclo nao avancou: %v", err)
	}

	if got := conta(out); got != 1 {
		t.Errorf("a racao acabou uma vez e o log recebeu %d avisos: o evento esta amostrando o estado em vez de registrar o fato", got)
	}
}
