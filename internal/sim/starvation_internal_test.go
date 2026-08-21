package sim

import "testing"

// A seca e um EPISODIO, e nao uma amostra por janela: com 4000 peixes morrendo de fome o log
// virava 40 de 40 avisos iguais e empurrava para fora dele a propria falencia. Dois eventos
// por seca — a abertura, para o jogador nao ficar cego durante ela, e o fechamento, com o
// total do episodio.
func TestSecaEmiteAberturaEFechamento(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 92)
	s.Cash = 10_000_000
	s.Tanks[0].Batches[0].Fish = 4_000
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].ServedUntil = Tick(maxInt32)

	out, err := Advance(Input{State: s, Until: s.Tick + 20*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando: %v", err)
	}

	conta := func(kind EventKind) int {
		n := 0
		for _, e := range out.Events {
			if e.Kind == kind {
				n++
			}
		}

		return n
	}

	abriu, fechou := conta(EventStarvationBegan), conta(EventStarvationEnded)

	if abriu != 1 {
		t.Errorf("a seca abriu %d vezes, e ela comecou uma so", abriu)
	}
	if fechou > 1 {
		t.Errorf("a seca fechou %d vezes", fechou)
	}

	// E o que nao pode sobrar: o aviso por janela, que era o que cegava o log.
	if porJanela := conta(EventStarvationDeaths); porJanela > 0 {
		t.Errorf("sobraram %d avisos por janela: o evento continua amostrando o estado", porJanela)
	}
}

// Voltar a comer fecha o episodio com o total, e comecar a passar fome de novo abre outro: o
// rearme e o proprio prato de comida.
func TestVoltarAComerFechaOEpisodioComOTotal(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 93)
	s.Cash = 10_000_000
	s.Tanks[0].Batches[0].Fish = 2_000
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].ServedUntil = Tick(maxInt32)

	out, err := Advance(Input{State: s, Until: s.Tick + 10*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando a seca: %v", err)
	}

	morreram := out.State.Tanks[0].Batches[0].StarvationEpisodeDeaths
	if morreram <= 0 {
		t.Fatal("ninguem morreu de fome: o teste precisa da seca de verdade")
	}

	fed := out.State
	fed.Tanks[0].FeedStock = 500 * MicrogramsPerKilogram
	fed.Tanks[0].ServedUntil = Tick(maxInt32)

	depois, err := Advance(Input{State: fed, Until: fed.Tick + TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando com racao: %v", err)
	}

	var fechamento *Event
	for i := range depois.Events {
		if depois.Events[i].Kind == EventStarvationEnded {
			fechamento = &depois.Events[i]
		}
	}

	if fechamento == nil {
		t.Fatal("voltar a comer nao fechou o episodio de fome")
	}
	if fechamento.Fish != morreram {
		t.Errorf("o fechamento diz %d mortos e o episodio contou %d", fechamento.Fish, morreram)
	}
	if depois.State.Tanks[0].Batches[0].StarvationEpisodeDeaths != 0 {
		t.Error("o contador do episodio nao zerou: a proxima seca comecaria somando a anterior")
	}
}
