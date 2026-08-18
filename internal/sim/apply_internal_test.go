package sim

import "testing"

// Acao represada volta minutos depois e seria aplicada contra uma fazenda que ja mudou: uma
// despesca decidida com outro preco na tela vira dinheiro que o jogador nao escolheu.
func TestAcaoDecididaContraUmMundoVelhoERecusada(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	casos := map[string]struct {
		seen Tick
		want RejectReason
	}{
		"sem tick informado":  {seen: 0, want: RejectNone},
		"tela do tick atual":  {seen: 100, want: RejectNone},
		"tela de agora ha um": {seen: 100 - staleViewTicks, want: RejectNone},
		"tela de minutos atras": {
			seen: 100 - staleViewTicks - 1, want: RejectStaleView,
		},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := NewState(1, 0, 100)
			s.Cash = 50_000_000

			if _, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres); !ok {
				t.Fatal("sem tanque")
			}

			sink := &eventSink{}
			reason, _ := apply(&s, b, Action{
				ID: 1, Kind: ActionAerate, Tank: 1, Amount: 1, SeenAt: tc.seen,
			}, 100, sink)

			if reason != tc.want {
				t.Errorf("acao com a tela do tick %d no tick 100 deu %v, queria %v", tc.seen, reason, tc.want)
			}
		})
	}
}
