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

	// E o que nao pode sobrar: mais de um aviso pela mesma seca, que era o que cegava o log.
	if abriu+fechou > 2 {
		t.Errorf("a seca emitiu %d eventos: ela continua sendo amostrada, e nao registrada", abriu+fechou)
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

// Despescar no meio da seca nao pode engolir o episodio: o lote sai com os mortos dentro, e
// o unico registro que fica e a abertura dizendo "1 peixe". Quem perdeu o lote nunca ve o
// tamanho da perda.
func TestDespescarNoMeioDaSecaFechaOEpisodio(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 94)
	s.Cash = 10_000_000
	s.Tanks[0].Batches[0].Fish = 2_000
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].ServedUntil = Tick(maxInt32)

	// Seis dias: a seca ja matou muita coisa e o lote ainda esta vivo. Passando disso ele
	// acaba sozinho, o episodio fecha por conta propria e o teste deixaria de medir a
	// despesca — que e o caminho em que o episodio sumia.
	out, err := Advance(Input{State: s, Until: s.Tick + 6*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando a seca: %v", err)
	}

	morreram := out.State.Tanks[0].Batches[0].StarvationEpisodeDeaths
	if morreram <= 0 {
		t.Fatal("ninguem morreu de fome: o teste precisa da seca de verdade")
	}

	colhido := out.State
	depois, err := Advance(Input{State: colhido, Until: colhido.Tick + 1, Balance: b,
		Actions: []Action{{
			ID: 1, Kind: ActionHarvest, Tank: colhido.Tanks[0].ID,
			Batch: colhido.Tanks[0].Batches[0].ID, At: colhido.Tick,
		}}})
	if err != nil {
		t.Fatalf("despescando: %v", err)
	}

	for _, e := range depois.Events {
		if e.Kind == EventStarvationEnded && e.Fish == morreram {
			return
		}
	}

	t.Errorf("a despesca levou o lote com %d mortos de fome dentro e nao fechou o episodio", morreram)
}

// O fechamento cobre o intervalo do episodio: dizer que 3271 peixes morreram em um tick e
// mentir sobre quando, e o feed do jogo e a memoria do que aconteceu.
func TestOFechamentoDaSecaCobreOIntervalo(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 95)
	s.Cash = 10_000_000
	s.Tanks[0].Batches[0].Fish = 2_000
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].ServedUntil = Tick(maxInt32)

	out, err := Advance(Input{State: s, Until: s.Tick + 8*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando a seca: %v", err)
	}

	var abertura Tick
	for _, e := range out.Events {
		if e.Kind == EventStarvationBegan {
			abertura = e.From
		}
	}

	fed := out.State
	fed.Tanks[0].FeedStock = 500 * MicrogramsPerKilogram
	fed.Tanks[0].ServedUntil = Tick(maxInt32)

	depois, err := Advance(Input{State: fed, Until: fed.Tick + TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando com racao: %v", err)
	}

	for _, e := range depois.Events {
		if e.Kind != EventStarvationEnded {
			continue
		}
		if e.From != abertura {
			t.Errorf("o fechamento diz que a seca comecou no tick %d e ela comecou em %d", e.From, abertura)
		}
		if e.To <= e.From {
			t.Errorf("o fechamento cobre %d..%d: um intervalo de um tick para uma seca de dias", e.From, e.To)
		}

		return
	}

	t.Fatal("a seca nao fechou")
}

// O lote pode acabar por OUTRA causa no meio da seca — hipoxia, doenca — e o episodio tem de
// fechar do mesmo jeito. Este ramo existia e nenhum teste o cobria: o @engenheiro mutou o
// fechamento e a suite ficou verde, com o ramo vivo e medido.
func TestOutraCausaTerminandoOLoteFechaOEpisodio(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := stockedFarm(t, 96)
	s.Cash = 10_000_000
	s.Tanks[0].Batches[0].Fish = 400
	s.Tanks[0].FeedStock = 0
	s.Tanks[0].ServedUntil = Tick(maxInt32)

	// Seis dias de seca deixam o episodio aberto com o lote ainda vivo.
	out, err := Advance(Input{State: s, Until: s.Tick + 6*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando a seca: %v", err)
	}

	aberto := out.State
	if aberto.Tanks[0].Batches[0].StarvationEpisodeDeaths <= 0 {
		t.Fatal("o episodio nao abriu: o teste precisa da seca em curso")
	}

	// A agua sufoca o que sobrou: o lote acaba sem ser pela fome nem pela despesca.
	aberto.Tanks[0].Oxygen = 0
	aberto.Tanks[0].Aerating = false

	depois, err := Advance(Input{State: aberto, Until: aberto.Tick + 5*TicksPerDay, Balance: b})
	if err != nil {
		t.Fatalf("avancando a hipoxia: %v", err)
	}

	if depois.State.Tanks[0].BatchCount > 0 && !depois.State.Tanks[0].Batches[0].Empty() {
		t.Skip("o lote sobreviveu a hipoxia: este teste precisa dele acabando por outra causa")
	}

	for _, e := range depois.Events {
		if e.Kind == EventStarvationEnded {
			return
		}
	}

	t.Error("o lote acabou por outra causa e o episodio de fome nunca fechou: os mortos somem com ele")
}
