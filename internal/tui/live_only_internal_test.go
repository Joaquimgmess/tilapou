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

	var model tea.Model = New(client.New(addr, 10*time.Second))
	d := &driver{t: t, model: model}
	d.run(model.Init())
	d.requireLive("inicio")

	var out strings.Builder
	out.WriteString(d.line("inicio"))

	d.press("f")
	d.press("h")
	out.WriteString(d.line("vendi o lote herdado"))

	d.press("1")
	d.press("2")
	out.WriteString(d.line("comprei comedouro e aerador"))

	d.press("g")
	for range 2 {
		d.press("down")
	}
	d.press("z")
	d.press("x")
	out.WriteString(d.line("peguei o credito que a tela sugeriu"))

	d.press("s")
	out.WriteString(d.line("povoei ate o equilibrio"))

	for _, days := range []int{60, 60, 60} {
		qaJumpDays(t, days)
		out.WriteString(d.line(fmt.Sprintf("+%d dias", days)))
	}

	d.press("h")
	d.requireLive("despesquei")
	out.WriteString(d.line("despesquei"))

	d.press("g")
	for range 3 {
		d.press("down")
	}
	d.press("z")
	d.press("x")
	out.WriteString(d.line("quitei a divida"))

	d.press("s")
	d.requireLive("povoei o ciclo seguinte")
	out.WriteString(d.line("povoei o ciclo seguinte"))

	fmt.Fprint(os.Stdout, "\n"+out.String())
}

func TestQASession(t *testing.T) {
	t.Parallel()

	addr := os.Getenv("TILAPOU_DAEMON")
	if addr == "" {
		t.Fatal("TILAPOU_DAEMON vazio: dentro da tag live isto e erro de uso, e nao motivo de pular")
	}

	d := &driver{t: t, model: New(client.New(addr, 10*time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.refresh()

	qaPlay(t, d, os.Getenv("QA_SCRIPT"))

	fmt.Fprintf(os.Stdout, "\n%s\n", plain(d.model.(Model).render()))
}
