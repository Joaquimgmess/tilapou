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

// freshFarm le a fazenda e diz se ela ja e a nova. Erro de rede aqui e o daemon ainda
// remontando, e nao defeito: quem decide desistir e o laco de espera.
func freshFarm(t *testing.T, poller *http.Client, addr string) bool {
	t.Helper()

	req, err := httpRequest(t, addr+"/v1/farm")
	if err != nil {
		return false
	}

	resp, err := poller.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	var snap api.Snapshot
	if json.NewDecoder(resp.Body).Decode(&snap) != nil {
		return false
	}

	return snap.FarmID != "" && snap.Tick < int64(sim.TicksPerDay)
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

	for range 20 {
		if freshFarm(t, poller, addr) {
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatal("o daemon continua servindo a fazenda velha depois do banco ser limpo: ele guarda a sessao em memoria, entao reinicie o daemon antes de rodar o portao")
}
