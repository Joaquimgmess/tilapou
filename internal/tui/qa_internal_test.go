package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/client"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

const (
	qaWidth       = 120
	qaHeight      = 40
	keyEnter      = "enter"
	ownerDatabase = "tilapou"
)

var (
	errNoQADatabase     = errors.New("QA_DATABASE nao esta definida")
	errOwnerDatabase    = errors.New("QA_DATABASE aponta para o banco do dono")
	errConnInfoDatabase = errors.New("QA_DATABASE tem de ser um nome de banco, e nao uma conninfo")

	// identifier e o que o Postgres aceita como nome sem aspas. Qualquer coisa fora disso —
	// espaco, '=', ':', '/' — abre a porta para conninfo.
	identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)
)

// qaDatabase recusa o banco do dono. O valor NAO e so um nome: ele vai para `psql -d`, que
// aceita conninfo — 'dbname=tilapou' e 'postgres://.../tilapou' passavam por uma comparacao
// de nome e caiam no save do jogador. Por isso se cobra um identificador, e nao uma string
// diferente de "tilapou".
func qaDatabase(name string) error {
	if name == "" {
		return errNoQADatabase
	}
	if !identifier.MatchString(name) {
		return fmt.Errorf("%w: %q nao e um nome de banco", errConnInfoDatabase, name)
	}
	if name == ownerDatabase {
		return errOwnerDatabase
	}

	return nil
}

func qaJumpDays(t *testing.T, days int) {
	t.Helper()

	name := os.Getenv("QA_DATABASE")
	if err := qaDatabase(name); err != nil {
		t.Fatalf("pular dias escreve no banco: %v (QA_DATABASE=%q)", err, name)
	}

	query := "UPDATE farms SET epoch = epoch - interval '" +
		strconv.Itoa(days*24*60) + " seconds'"

	cmd := exec.CommandContext(t.Context(), "docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "tilapou", "-d", name, "-c", query)
	cmd.Dir = "../.."

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pulando %d dias: %v: %s", days, err, out)
	}
}

func qaKey(name string) tea.KeyPressMsg {
	switch name {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case keyEnter:
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}

	return tea.KeyPressMsg{Code: rune(name[0]), Text: name}
}

func qaPlay(t *testing.T, d *driver, script string) {
	t.Helper()

	for step := range strings.SplitSeq(script, ",") {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		if days, ok := strings.CutPrefix(step, "d"); ok {
			if n, err := strconv.Atoi(days); err == nil {
				qaJumpDays(t, n)
				d.press("r")

				continue
			}
		}

		if step == "show" {
			fmt.Fprintf(os.Stdout, "\n%s\n", plain(d.model.(Model).render()))

			continue
		}

		var cmd tea.Cmd
		d.model, cmd = d.model.Update(qaKey(step))
		d.run(cmd)

		if !d.model.(Model).confirming {
			d.press("r")
		}
	}
}

func TestQAHarnessActuallySendsTheAction(t *testing.T) {
	t.Parallel()

	var posted []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := sizedSnapshot()
		if r.Method == http.MethodPost {
			var action client.Action
			_ = json.NewDecoder(r.Body).Decode(&action)
			posted = append(posted, action.Kind)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.refresh()

	qaPlay(t, d, "f,c,h")

	for _, want := range []string{"feed", "buy_feed", "harvest"} {
		if !slices.Contains(posted, want) {
			t.Errorf("o script mandou a tecla mas o daemon nao recebeu %q; recebeu %v", want, posted)
		}
	}
}

func TestQAScriptHandlesArrowsAndPrompts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		snap := sizedSnapshot()
		snap.PrestigeNow = 3
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.refresh()

	qaPlay(t, d, "down,up,left,right")

	qaPlay(t, d, "tab,p")
	if !d.model.(Model).confirming {
		t.Error("o refresh automatico cancelou o prompt antes do proximo passo do script")
	}
}

func TestPularDiasSoAceitaBancoDeTeste(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		db   string
		ok   bool
	}{
		{"banco de teste passa", "tilapou_qa", true},
		{"sem variavel definida nao passa", "", false},
		{"o banco do dono nunca passa", ownerDatabase, false},
	} {
		if err := qaDatabase(tc.db); (err == nil) != tc.ok {
			t.Errorf("%s: qaDatabase(%q) = %v", tc.name, tc.db, err)
		}
	}
}

func TestApertarZNoSegundoViveiroTrocaOTanqueSelecionado(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sizedSnapshot())
	}))
	t.Cleanup(server.Close)

	d := &driver{t: t, model: New(client.New(server.URL, time.Second))}
	d.model, _ = d.model.Update(tea.WindowSizeMsg{Width: qaWidth, Height: qaHeight})
	d.refresh()

	if got := d.model.(Model).mode; got != ModeGameBoy {
		t.Fatalf("o jogo abre no modo %v, queria a tela GameBoy", got)
	}
	if got := d.model.(Model).selected; got != 0 {
		t.Fatalf("o jogo comeca com o tanque %d selecionado, queria 0", got)
	}

	for range pondCols + 1 {
		d.press(keyRight)
	}
	d.press(keyUp)
	d.press("z")

	m, ok := d.model.(Model)
	if !ok {
		t.Fatal("o driver perdeu o Model")
	}
	if m.selected != 1 {
		t.Fatalf("apertar z de frente para o segundo viveiro deixou selected em %d, queria 1", m.selected)
	}
	if m.menu == nil {
		t.Fatal("apertar z de frente para o viveiro nao abriu o menu do tanque")
	}
}

// qaFreshFarm recomeca a fazenda do banco de QA para o roteiro nao depender do save que
// estiver la. Roteiro que herda estado alheio e relatorio do save, e nao teste do jogo.
func qaFreshFarm(t *testing.T) {
	t.Helper()

	name := os.Getenv("QA_DATABASE")
	if err := qaDatabase(name); err != nil {
		t.Fatalf("recomecar a fazenda escreve no banco: %v (QA_DATABASE=%q)", err, name)
	}
	// Antes de escrever, pergunta ao daemon em que banco ELE esta: a variavel e o que alguem
	// digitou, e quem escreve e ele. Recusar aqui e o que impede a limpeza de cair no save do
	// dono por env errada.
	if err := checkDatabase(name, daemonHealth(t)); err != nil {
		t.Fatalf("recomecar a fazenda: %v", err)
	}

	cmd := exec.CommandContext(t.Context(), "docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "tilapou", "-d", name, "-c", "DELETE FROM farm_events; DELETE FROM farm_actions; DELETE FROM farms;")
	cmd.Dir = "../.."

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("recomecando a fazenda: %v: %s", err, out)
	}

	// O daemon mantem a fazenda em memoria: apagar a linha por baixo dele nao reinicia a
	// sessao, e a proxima leitura pode devolver o estado velho — com o relogio adiantado, ele
	// entra num catch-up que estoura o prazo do cliente e o erro que aparece e "deadline
	// exceeded", que nao diz nada. Esperar aqui troca o misterio por um diagnostico.
	waitFreshFarm(t)
}

var errDatabaseMismatch = errors.New("o daemon nao confirma o banco de teste")

// healthRead e o que a leitura do /healthz devolveu. Guardar o status e o erro separados e o
// que permite a recusa dizer a causa: quatro estados diferentes caiam na mesma frase, e so
// um deles era o que ela afirmava.
type healthRead struct {
	status   int
	database string
	err      error
}

// checkDatabase decide se da para escrever, e diz por que nao quando nao da.
func checkDatabase(want string, got healthRead) error {
	switch {
	case got.err != nil || got.status == 0:
		return fmt.Errorf("%w: o daemon nao respondeu", errDatabaseMismatch)
	case got.status != http.StatusOK:
		return fmt.Errorf("%w: o daemon respondeu %d no /healthz", errDatabaseMismatch, got.status)
	case got.database == "":
		return fmt.Errorf("%w: o daemon nao publica o banco (build anterior ao /healthz com banco?)", errDatabaseMismatch)
	case got.database != want:
		return fmt.Errorf("%w: o daemon escreve em %q e o teste usaria %q", errDatabaseMismatch, got.database, want)
	}

	return nil
}

// sameDatabase confere o banco que o daemon PUBLICA contra o que o teste pretende usar. A
// guarda mora do lado que sabe: QA_DATABASE e o que alguem digitou, e o daemon e quem escreve.
func sameDatabase(want, daemon string) error {
	return checkDatabase(want, healthRead{status: http.StatusOK, database: daemon})
}

// farmState diz em qual dos tres estados a leitura caiu. Sem isso, daemon fora do ar e daemon
// servindo a fazenda velha viravam o mesmo silencio, e a falha mandava reiniciar um daemon
// que nao estava rodando.
type farmState uint8

const (
	daemonDown farmState = iota
	farmStale
	farmFresh
)

var errNoDaemon = errors.New("daemon nao respondeu")

func (f farmState) String() string {
	switch f {
	case daemonDown:
		return "daemon fora do ar"
	case farmStale:
		return "fazenda velha"
	case farmFresh:
		return "fazenda nova"
	}

	return "desconhecido"
}

// freshness classifica a leitura. Recebe o resultado ja lido para o teste poder cobrar os
// tres ramos sem daemon nenhum.
func freshness(t *testing.T, snap *api.Snapshot, err error) farmState {
	t.Helper()

	if err != nil || snap == nil {
		return daemonDown
	}
	if snap.FarmID == "" || snap.Tick >= int64(sim.TicksPerDay) {
		return farmStale
	}

	return farmFresh
}

// daemonHealth pergunta ao daemon em que banco ele escreve, guardando o que deu errado.
func daemonHealth(t *testing.T) healthRead {
	t.Helper()

	req, err := httpRequest(t, os.Getenv("TILAPOU_DAEMON")+"/healthz")
	if err != nil {
		return healthRead{err: err}
	}

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return healthRead{err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	var health struct {
		Database string `json:"database"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&health); decodeErr != nil {
		return healthRead{status: resp.StatusCode, err: decodeErr}
	}

	return healthRead{status: resp.StatusCode, database: health.Database}
}

// freshFarm le a fazenda e classifica o que veio.
func freshFarm(t *testing.T, poller *http.Client, addr string) farmState {
	t.Helper()

	req, err := httpRequest(t, addr+"/v1/farm")
	if err != nil {
		return freshness(t, nil, err)
	}

	resp, err := poller.Do(req)
	if err != nil {
		return freshness(t, nil, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var snap api.Snapshot
	if decodeErr := json.NewDecoder(resp.Body).Decode(&snap); decodeErr != nil {
		return freshness(t, nil, decodeErr)
	}

	return freshness(t, &snap, nil)
}

func httpRequest(t *testing.T, url string) (*http.Request, error) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("montando o pedido para %s: %w", url, err)
	}

	return req, nil
}

// waitFreshFarm espera o daemon devolver uma fazenda nova depois do banco ser limpo.
func waitFreshFarm(t *testing.T) {
	t.Helper()

	addr := os.Getenv("TILAPOU_DAEMON")
	poller := &http.Client{Timeout: 5 * time.Second}

	ultimo := daemonDown
	for range 20 {
		ultimo = freshFarm(t, poller, addr)
		if ultimo == farmFresh {
			return
		}

		time.Sleep(time.Second)
	}

	if ultimo == daemonDown {
		t.Fatalf("nao ha daemon respondendo em %s: o portao precisa de um de pe, e nao e ele que sobe", addr)
	}

	t.Fatal("o daemon continua servindo a fazenda velha depois do banco ser limpo: ele guarda a sessao em memoria, entao reinicie o daemon antes de rodar o portao")
}
