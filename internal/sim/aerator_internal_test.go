package sim

import "testing"

// Quem comprou o aerador recebia o interruptor mais cego: o automatico reescrevia Aerating
// todo tick, entao o [a] durava um segundo. Nenhuma outra automacao desfaz o jogador — o
// comedouro e o peao so acrescentam. A regra e do @game-design (decision-018): o manual vence
// ate a histerese fechar.
func TestOManualVenceOAutomaticoAteAHistereseFechar(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 97)
	s.Cash = 10_000_000

	tank := &s.Tanks[0]
	tank.grant(AutoAerator)
	// Agua boa: o automatico nao quer aerar, entao ligar e uma escolha do jogador.
	tank.Oxygen = b.Water.AeratorOff + 1_000
	tank.Aerating = false

	reason, _ := applyAerate(&s, Action{Kind: ActionAerate, Tank: tank.ID, Amount: 1})
	if reason != RejectNone {
		t.Fatalf("o jogo recusou ligar o aerador: %v", reason)
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 10, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	if !out.State.Tanks[0].Aerating {
		t.Error("o automatico desfez o [a] do jogador no tick seguinte: quem pagou pelo upgrade fica sem o interruptor")
	}

	// A histerese fecha quando o automatico quer o MESMO valor. Aqui a regra e exercitada
	// direto: avancar ticks nao serve, porque o aerador ligado levanta o oxigenio e o proprio
	// cenario se desfaz antes de a conta ser feita.
	comOverride := *out.State.tank(tank.ID)
	comOverride.Oxygen = b.Water.AeratorOn / 2

	fazenda := out.State
	aerate(&fazenda, b, &comOverride)

	if !comOverride.Aerating {
		t.Error("com o oxigenio no fundo o aerador desligou")
	}
	if comOverride.AeratorManual != aeratorAuto {
		t.Error("a histerese fechou e o override continuou de pe: ele viraria permanente")
	}
}

// E o outro lado: desligar tambem e escolha, e o automatico so retoma quando quiser o mesmo.
func TestDesligarNoManualTambemVence(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 98)
	s.Cash = 10_000_000

	tank := &s.Tanks[0]
	tank.grant(AutoAerator)
	// Agua ruim: o automatico quer aerar, e o jogador desliga assim mesmo.
	tank.Oxygen = b.Water.AeratorOn / 2
	tank.Aerating = true

	reason, _ := applyAerate(&s, Action{Kind: ActionAerate, Tank: tank.ID, Amount: 0})
	if reason != RejectNone {
		t.Fatalf("o jogo recusou desligar o aerador: %v", reason)
	}

	out, err := Advance(Input{State: s, Until: s.Tick + 10, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	if out.State.Tanks[0].Aerating {
		t.Error("o automatico religou por cima do jogador no tick seguinte")
	}
}
