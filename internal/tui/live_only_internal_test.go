//go:build live

//
// Os tres testes que falam com um daemon de verdade moram atras da tag live. Sem ela, um
// go test ./... limpo os pulava com SKIP verde: cobertura que nao cobre nada e pior que
// nenhuma, porque parece portao. Rode com: make test-live.
//
// Aqui dentro, TILAPOU_DAEMON vazio e erro de uso, e nao motivo de pular.

package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

func TestLiveSession(t *testing.T) {
	addr := os.Getenv("TILAPOU_DAEMON")
	if addr == "" {
		t.Fatal("TILAPOU_DAEMON vazio: dentro da tag live isto e erro de uso, e nao motivo de pular")
	}
	if err := qaDaemon(addr); err != nil {
		t.Fatalf("recusado: %v — o roteiro aperta teclas de verdade, e este e o save do jogador", err)
	}

	var model tea.Model = New(client.New(addr, 5*time.Second))
	d := &driver{t: t, model: model}
	d.run(model.Init())
	// Sem tamanho a sessao roda num terminal de 0x0 e o relatorio sai de uma tela que ninguem
	// ve. O tamanho e o mesmo dos outros testes de layout.
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.requireLive("1. abertura")

	var out strings.Builder
	abertura := d.frame("1. abertura")
	d.requireFrame("1. abertura", abertura, "TILAPOU")
	out.WriteString(abertura)

	for range 3 {
		d.press("up")
	}
	out.WriteString(d.frame("2. de frente para o viveiro"))

	d.press("z")
	menu := d.frame("3. menu do tanque")
	d.requireFrame("3. menu do tanque", menu, "TANQUE")
	out.WriteString(menu)

	d.press("z")
	out.WriteString(d.frame("4. servi o trato"))

	d.press("down")
	d.press("down")
	d.press("down")
	d.press("down")
	d.press("down")
	out.WriteString(d.frame("5. menu com cursor em outra opcao"))

	// O painel de numeros abre com tab: o roteiro apertava x e m, que nao trocam de tela, e o
	// quadro rotulado "painel de numeros" saia sendo o mapa.
	d.press("x")
	d.press("tab")
	painel := d.frame("6. painel de numeros")
	d.requireFrame("6. painel de numeros", painel, "MERCADO")
	out.WriteString(painel)

	fmt.Fprint(os.Stdout, out.String())

	if err := os.WriteFile("/tmp/live.txt", []byte(out.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProgression(t *testing.T) {
	addr := os.Getenv("TILAPOU_DAEMON")
	if addr == "" {
		t.Fatal("TILAPOU_DAEMON vazio: dentro da tag live isto e erro de uso, e nao motivo de pular")
	}
	if err := qaDaemon(addr); err != nil {
		t.Fatalf("recusado: %v — o roteiro aperta teclas de verdade, e este e o save do jogador", err)
	}

	// O roteiro prepara a propria entrada: dependendo do save que estiver la, ele vira
	// relatorio do save e nao teste do jogo — foi assim que ele passou verde com todos os
	// passos recusados.
	qaFreshFarm(t)

	var model tea.Model = New(client.New(addr, 10*time.Second))
	d := &driver{t: t, model: model}
	d.run(model.Init())
	d.requireLive("inicio")

	var out strings.Builder
	out.WriteString(d.line("inicio"))

	antes := d.snap()
	d.press("f")
	d.press("h")
	d.applied("vendi o lote herdado")
	d.cashRose("vendi o lote herdado", antes)
	if got := d.snap().Tanks[0].Fish; got != 0 {
		t.Fatalf("vendi o lote herdado e sobraram %d peixes no tanque", got)
	}
	out.WriteString(d.line("vendi o lote herdado"))

	antes = d.snap()
	d.press("1")
	d.press("2")
	d.applied("comprei comedouro e aerador")
	d.cashFell("comprei comedouro e aerador", antes)
	out.WriteString(d.line("comprei comedouro e aerador"))

	antes = d.snap()
	d.press("g")
	d.choose("Pegar emprestimo")
	d.press("x")
	d.applied("peguei o credito que a tela sugeriu")
	d.cashRose("peguei o credito que a tela sugeriu", antes)
	if depois := d.snap(); depois.Debt <= antes.Debt {
		t.Fatalf("peguei o credito e a divida foi de %d para %d", antes.Debt, depois.Debt)
	}
	out.WriteString(d.line("peguei o credito que a tela sugeriu"))

	querido := d.snap().Tanks[0].StockAdvice
	d.press("s")
	d.applied("povoei ate o equilibrio")
	d.stockedAtLeast("povoei ate o equilibrio", querido)
	out.WriteString(d.line("povoei ate o equilibrio"))

	for _, days := range []int{60, 60, 60} {
		antes = d.snap()
		qaJumpDays(t, days)
		d.press("r")
		d.moved(fmt.Sprintf("+%d dias", days), antes)
		out.WriteString(d.line(fmt.Sprintf("+%d dias", days)))
	}

	antes = d.snap()
	d.press("h")
	d.applied("despesquei")
	d.cashRose("despesquei", antes)
	out.WriteString(d.line("despesquei"))

	d.press("g")
	d.choose("Pagar divida")
	d.press("x")
	d.applied("quitei a divida")
	if depois := d.snap(); depois.Debt != 0 {
		t.Fatalf("quitei a divida: sobraram %d de divida", depois.Debt)
	}
	out.WriteString(d.line("quitei a divida"))

	querido = d.snap().Tanks[0].StockAdvice
	d.press("s")
	d.applied("povoei o ciclo seguinte")
	d.stockedAtLeast("povoei o ciclo seguinte", querido)
	out.WriteString(d.line("povoei o ciclo seguinte"))

	fmt.Fprint(os.Stdout, "\n"+out.String())
}
