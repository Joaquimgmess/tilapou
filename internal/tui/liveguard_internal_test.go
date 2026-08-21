package tui

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// ownerPort e a porta do daemon do dono, com o save de verdade.
const ownerPort = 8099

var errOwnerDaemon = errors.New("o endereco e o daemon do dono")

// qaDaemon recusa o daemon do dono. O roteiro aperta teclas de verdade contra o endereco
// apontado, e QA_DATABASE so protege o salto de dias — sem esta guarda, um make test-live sem
// env escreve no save do jogador, que foi exatamente o que aconteceu.
//
// A porta sai de SplitHostPort e e comparada como NUMERO: com o corte no primeiro
// dois-pontos, http://[::1]:8099 lia ":1]:8099" e passava; comparando string, :08099 passava,
// porque o dialer normaliza o zero a esquerda e conecta no mesmo lugar. O daemon do dono
// escuta em [::]:8099, entao os dois enderecos chegam la.
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

	number, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("porta ilegivel em %q: %w", addr, err)
	}
	if number == ownerPort {
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
	for i := range m.menu.items {
		item := &m.menu.items[i]
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

// cashRose cobra que o caixa tenha subido no passo. Cada asercao dessas vira um galho no
// roteiro, e o roteiro inteiro estourou a complexidade que o lint aceita — entao elas moram
// aqui, com nome, e o roteiro fica legivel como roteiro.
func (d *driver) cashRose(step string, antes api.Snapshot) {
	d.t.Helper()

	if depois := d.snap(); depois.CashCents <= antes.CashCents {
		d.t.Fatalf("%s: o caixa nao subiu (%d -> %d)", step, antes.CashCents, depois.CashCents)
	}
}

// cashFell cobra que o caixa tenha caido: compra que nao cobra nada nao aconteceu.
func (d *driver) cashFell(step string, antes api.Snapshot) {
	d.t.Helper()

	if depois := d.snap(); depois.CashCents >= antes.CashCents {
		d.t.Fatalf("%s: o caixa nao caiu (%d -> %d)", step, antes.CashCents, depois.CashCents)
	}
}

// stockedAtLeast cobra que o tanque tenha recebido o que a tela sugeriu.
func (d *driver) stockedAtLeast(step string, querido int64) {
	d.t.Helper()

	if got := int64(d.snap().Tanks[0].Fish); got < querido {
		d.t.Fatalf("%s: a tela sugeriu %d peixes e o tanque ficou com %d", step, querido, got)
	}
}

// moved cobra que sessenta dias tenham mexido em alguma coisa. Nao se exige que o lote da
// frente cresca: com o peao de despesca ligado o ciclo fecha dentro da janela e o lote da
// frente passa a ser outro, mais novo. O que e defeito e nada mudar.
func (d *driver) moved(step string, antes api.Snapshot) {
	d.t.Helper()

	// O snapshot chega por pedido assincrono: um unico refresh pode ler o estado de antes do
	// salto. Insistir algumas vezes e o que o proprio jogo faz, e troca uma falha de corrida
	// por uma falha de verdade.
	depois := d.snap()
	for range 5 {
		if depois.Tick > antes.Tick {
			break
		}

		d.refresh()
		depois = d.snap()
	}

	if depois.Tick <= antes.Tick {
		d.t.Fatalf("%s: o relogio nao andou (%d -> %d)", step, antes.Tick, depois.Tick)
	}
	if cresceu(antes) == cresceu(depois) && depois.LifetimeCents == antes.LifetimeCents &&
		depois.CashCents == antes.CashCents && depois.Fish == antes.Fish {
		d.t.Fatalf("%s e nada mudou: %d g, %d de caixa, %d peixes, %d faturado",
			step, cresceu(depois), depois.CashCents, depois.Fish, depois.LifetimeCents)
	}
}

// cresceu e o peso do lote da frente, em gramas; zero quando nao ha lote.
func cresceu(s api.Snapshot) int64 {
	if len(s.Tanks) == 0 || len(s.Tanks[0].Batches) == 0 {
		return 0
	}

	return s.Tanks[0].Batches[0].MeanGrams
}

// snap e o estado de agora, para o passo seguinte comparar contra ele.
func (d *driver) snap() api.Snapshot {
	d.t.Helper()

	return d.model.(Model).snapshot
}
