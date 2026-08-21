package tui

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// ownerPort e a porta do daemon do dono, com o save de verdade.
const ownerPort = "8099"

var errOwnerDaemon = errors.New("o endereco e o daemon do dono")

// qaDaemon recusa o daemon do dono. O roteiro aperta teclas de verdade contra o endereco
// apontado, e QA_DATABASE so protege o salto de dias — sem esta guarda, um make test-live sem
// env escreve no save do jogador, que foi exatamente o que aconteceu.
//
// A porta sai de SplitHostPort, e nao de um corte no primeiro dois-pontos: com o corte,
// http://[::1]:8099 lia a porta como ":1]:8099" e passava — e o daemon do dono escuta em
// [::]:8099, entao esse endereco chega la.
func qaDaemon(addr string) error {
	raw := addr
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("endereco ilegivel %q: %w", addr, err)
	}

	port := parsed.Port()
	if port == "" {
		_, hostPort, splitErr := net.SplitHostPort(parsed.Host)
		if splitErr == nil {
			port = hostPort
		}
	}

	if port == ownerPort {
		return fmt.Errorf("%w: %s", errOwnerDaemon, addr)
	}

	return nil
}

// requireLive falha quando a sessao nao chegou a falar com o daemon. Sem esta guarda,
// TestLiveSession e TestProgression ficam verdes com o daemon fora do ar, e sao o
// portao que este backlog usa para dar task por pronta.
func (d *driver) requireLive(step string) {
	d.t.Helper()

	m, ok := d.model.(Model)
	if !ok {
		d.t.Fatalf("%s: o driver nao esta com um Model", step)
	}

	// O que se cobra e ter falado com o daemon, e nao a acao ter sido aceita: recusa de jogo
	// (despescar lote que nao esta no ponto) e resposta legitima e prova que houve conversa.
	// Daemon fora do ar nao devolve fazenda nenhuma, e ai o portao fecha.
	if m.snapshot.FarmID == "" {
		d.t.Fatalf("%s: o daemon nao devolveu fazenda nenhuma (erro: %v)", step, m.err)
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

// choose escolhe o item do menu pelo ROTULO, e nao pela posicao: navegar com N setas amarra o
// roteiro a ordem do menu, que muda com o estado — foi assim que um passo rotulado "quitei a
// divida" povoou 50 peixes sem ninguem perceber.
func (d *driver) choose(label string) {
	d.t.Helper()

	m, ok := d.model.(Model)
	if !ok || m.menu == nil {
		d.t.Fatalf("choose(%q) sem menu aberto", label)
	}

	rotulos := make([]string, 0, len(m.menu.items))
	for i, item := range m.menu.items {
		rotulos = append(rotulos, item.label)
		if !strings.Contains(item.label, label) {
			continue
		}
		if !item.enabled {
			d.t.Fatalf("o item %q esta desabilitado: %s", item.label, item.hint)
		}

		for range i - m.menu.cursor {
			d.press("down")
		}
		for range m.menu.cursor - i {
			d.press("up")
		}
		d.press("z")

		return
	}

	d.t.Fatalf("nenhum item do menu casa com %q; a tela oferece: %s", label, strings.Join(rotulos, " | "))
}

// applied cobra que a ultima acao tenha sido aceita pelo daemon. Passo recusado com rotulo de
// passo feito e o defeito que este portao existe para nao ter.
func (d *driver) applied(step string) {
	d.t.Helper()

	d.requireLive(step)

	out := d.model.(Model).snapshot.LastOutcome
	if out == nil {
		d.t.Fatalf("%s: o daemon nao registrou acao nenhuma", step)
	}
	if !out.Applied {
		d.t.Fatalf("%s: a acao foi recusada (%s), e o rotulo diz que ela aconteceu", step, out.Reason)
	}
}

// snap e o estado de agora, para o passo seguinte comparar contra ele.
func (d *driver) snap() api.Snapshot {
	d.t.Helper()

	return d.model.(Model).snapshot
}
