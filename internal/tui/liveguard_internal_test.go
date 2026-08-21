package tui

import "strings"

// Estes dois testes falam com um processo externo, e o cache do go test nao sabe disso: sem
// -count=1 ele devolve (cached) e o relatorio vira o de uma sessao antiga. Quem os roda como
// portao de task tem de passar -count=1.
//
// requireLive falha quando a sessao nao chegou a falar com o daemon. Sem esta guarda,
// TestLiveSession e TestProgression ficam verdes com o daemon fora do ar, e sao o
// portao que este backlog usa para dar task por pronta.
func (d *driver) requireLive(step string) {
	d.t.Helper()

	m, ok := d.model.(Model)
	if !ok {
		d.t.Fatalf("%s: o driver nao esta com um Model", step)
	}

	if m.err != nil {
		d.t.Fatalf("%s: a sessao nao falou com o daemon: %v", step, m.err)
	}
	if m.snapshot.FarmID == "" {
		d.t.Fatalf("%s: o daemon nao devolveu fazenda nenhuma", step)
	}
	if len(m.snapshot.Tanks) == 0 {
		d.t.Fatalf("%s: a fazenda voltou sem tanque", step)
	}
}

// requireFrame cobra que o quadro do passo mostre o que o rotulo promete, e nunca a tela de
// daemon fora do ar: relatorio que imprime "connection refused" e passa verde e pior que
// nenhum teste, porque parece cobertura.
func (d *driver) requireFrame(step, frame string, wants ...string) {
	d.t.Helper()

	if strings.Contains(frame, "daemon fora do ar") || strings.Contains(frame, "conectando no daemon") {
		d.t.Fatalf("%s: o quadro e a tela de daemon fora do ar:\n%s", step, frame)
	}

	for _, want := range wants {
		if !strings.Contains(frame, want) {
			d.t.Errorf("%s: o quadro nao mostra %q:\n%s", step, want, frame)
		}
	}
}
