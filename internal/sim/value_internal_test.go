package sim

import "testing"

// O oraculo e o caixa: o valor exibido tem de ser o que a despesca credita, e nao uma formula
// recopiada. O teste que existia recopiava a linha da tela, entao ele nao podia falhar — e foi
// por isso que a divergencia viveu.
func TestOValorBrutoEOQueODespescaCredita(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	// A faixa em que a tela congelava: entre dois quilos inteiros, a conta truncada devolve o
	// mesmo numero para lotes diferentes.
	for fish := FishCount(370); fish <= 380; fish++ {
		s := stockedFarm(t, 81)
		s.Cash = 0
		s.Tanks[0].Batches[0].Fish = fish
		s.Tanks[0].Batches[0].MeanMass = 24 * MicrogramsPerGram

		tank := &s.Tanks[0]
		batch := &tank.Batches[0]
		price := b.PriceFor(batch.MeanMass, s.Tick)
		esperado := GrossValue(price, batch.Biomass())

		sell(&s, b, tank, batch, fish, s.Tick, &eventSink{})

		if s.Cash != esperado {
			t.Errorf("%d peixes: a despesca creditou %d e o valor bruto diz %d", fish, s.Cash, esperado)
		}
	}
}

// E o degrau: dois lotes diferentes nao podem valer o mesmo numero. Era a coluna congelada
// que o @qa mediu, de 370 a 380 peixes imprimindo 7887 c sempre.
func TestOValorBrutoNaoCongelaEntreQuilos(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	price := b.PriceFor(24*MicrogramsPerGram, 0)

	menor := GrossValue(price, Micrograms(370)*24*MicrogramsPerGram)
	maior := GrossValue(price, Micrograms(380)*24*MicrogramsPerGram)

	if menor == maior {
		t.Errorf("370 e 380 peixes valem o mesmo (%d): a conta esta truncando em quilos inteiros", menor)
	}
}

// Com contrato, o que a decisao usa tem de ser o que o caixa recebe: o bonus vive na venda, e
// ignora-lo na decisao so errava para BAIXO — a direcao que marca fazenda viva como quebrada.
func TestOValorDoTanqueComContratoBateComOCaixa(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	s := stockedFarm(t, 82)
	s.Cash = 0
	tank := &s.Tanks[0]
	tank.grant(AutoContract)

	prometido := TankPayout(b, tank, s.Tick)

	batch := &tank.Batches[0]
	sell(&s, b, tank, batch, batch.Fish, s.Tick, &eventSink{})

	if s.Cash != prometido {
		t.Errorf("com contrato a despesca creditou %d e a decisao contava com %d", s.Cash, prometido)
	}

	// E o par: sem contrato os dois numeros continuam iguais entre si, mas menores.
	semContrato := stockedFarm(t, 82)
	semContrato.Cash = 0
	outro := &semContrato.Tanks[0]

	semBonus := TankPayout(b, outro, semContrato.Tick)
	if semBonus >= prometido {
		t.Errorf("o contrato nao rendeu nada: com %d, sem %d", prometido, semBonus)
	}

	sell(&semContrato, b, outro, &outro.Batches[0], outro.Batches[0].Fish, semContrato.Tick, &eventSink{})
	if semContrato.Cash != semBonus {
		t.Errorf("sem contrato a despesca creditou %d e a decisao contava com %d", semContrato.Cash, semBonus)
	}
}
