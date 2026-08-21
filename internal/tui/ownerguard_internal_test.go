package tui

import (
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

// A :8099 e o daemon do dono, com o save de verdade. Nenhum teste pode escrever nele: as
// teclas do roteiro vao para o daemon apontado, e o QA_DATABASE so protege o salto de dias.
// Guarda declarada em prompt nao segura; esta segura, e por isso mora no codigo.
func TestOPortaoRecusaODaemonDoDono(t *testing.T) {
	t.Parallel()

	for _, addr := range []string{
		"http://localhost:8099",
		"http://127.0.0.1:8099/",
		"localhost:8099",
		// O daemon do dono escuta em [::]:8099, entao o literal IPv6 chega la. Cortar no
		// primeiro dois-pontos lia a porta como ":1]:8099" e deixava passar.
		"http://[::1]:8099",
		"http://[::1]:8099/",
		"[::1]:8099",
		"http://localhost:8099/x",
		// Porta com zero a esquerda: url.Parse devolve "08099", o dialer normaliza e conecta
		// no mesmo lugar. Comparar string deixava passar.
		"http://localhost:08099",
		"http://[::1]:08099/",
	} {
		if err := qaDaemon(addr); err == nil {
			t.Errorf("o portao aceitou %q, que e o daemon do dono", addr)
		}
	}

	for _, addr := range []string{
		"http://localhost:8098",
		"http://localhost:8106",
		"http://[::1]:8098",
		"http://127.0.0.1:8106/",
	} {
		if err := qaDaemon(addr); err != nil {
			t.Errorf("o portao recusou %q, que e daemon de teste: %v", addr, err)
		}
	}
}

// O nome do banco chega ao cliente de linha de comando como destino, e esse destino aceita
// conninfo: comparar com "tilapou" deixava passar 'dbname=tilapou' e a URL inteira, e a
// escrita do salto de dias caia no save do dono.
func TestOPortaoRecusaConnInfoNoNomeDoBanco(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"tilapou",
		"dbname=tilapou",
		"postgres://tilapou:tilapou@localhost/tilapou",
		"tilapou_qa dbname=tilapou",
		"host=localhost dbname=tilapou",
	} {
		if err := qaDatabase(name); err == nil {
			t.Errorf("o portao aceitou %q, que alcanca o banco do dono", name)
		}
	}

	for _, name := range []string{"tilapou_qa", "tilapou_qa7", "qarun2"} {
		if err := qaDatabase(name); err != nil {
			t.Errorf("o portao recusou %q, que e banco de teste: %v", name, err)
		}
	}
}

// A espera pela fazenda nova precisa separar dois estados que ela hoje confunde: daemon fora
// do ar e daemon servindo a fazenda velha. Engolir os dois no mesmo silencio faz a falha
// dizer "reinicie o daemon" quando nao ha daemon nenhum — a mensagem manda caçar o problema
// errado, e foi o que aconteceu com o @qa.
func TestAEsperaSeparaDaemonForaDoArDeFazendaVelha(t *testing.T) {
	t.Parallel()

	fora := freshness(t, nil, errNoDaemon)
	if fora != daemonDown {
		t.Errorf("sem daemon a espera classificou como %v", fora)
	}

	velha := freshness(t, &api.Snapshot{FarmID: "x", Tick: 100 * 24 * 60}, nil)
	if velha != farmStale {
		t.Errorf("com a fazenda velha a espera classificou como %v", velha)
	}

	nova := freshness(t, &api.Snapshot{FarmID: "x", Tick: 3}, nil)
	if nova != farmFresh {
		t.Errorf("com a fazenda nova a espera classificou como %v", nova)
	}
}

// A guarda do banco passa a vir do daemon: e ele quem sabe em que banco escreve, e o QA_DATABASE
// e so o que alguem digitou. Quando os dois discordam, quem manda e o daemon — foi por uma
// variavel errada que o save do dono ficou a um passo quatro vezes.
func TestOPortaoConfereOBancoQueODaemonDiz(t *testing.T) {
	t.Parallel()

	if err := sameDatabase("tilapou_qa", "tilapou_qa"); err != nil {
		t.Errorf("bancos iguais foram recusados: %v", err)
	}

	if err := sameDatabase("tilapou_qa", "tilapou"); err == nil {
		t.Error("o portao aceitou o daemon escrevendo no banco do dono com QA_DATABASE de teste")
	}

	// Daemon que nao publica o banco nao serve de guarda: e build velho, e o teste tem de
	// dizer isso em vez de seguir achando que conferiu.
	if err := sameDatabase("tilapou_qa", ""); err == nil {
		t.Error("o portao aceitou um daemon que nao diz em que banco escreve")
	}
}

// A frase de recusa nao pode ser um balde: o @qa mostrou quatro causas caindo em "o daemon
// nao publica o banco (build velho?)" — sem daemon, daemon antigo, daemon com erro, e daemon
// de HEAD com a URL sem banco. So a segunda e verdade, e nas outras a frase manda cacar o
// problema errado.
func TestARecusaDoBancoDizACausaCerta(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome    string
		leitura healthRead
		quer    string
	}{
		{"sem daemon", healthRead{}, "nao respondeu"},
		{"daemon com erro", healthRead{status: 500}, "respondeu 500"},
		{"daemon sem o campo", healthRead{status: 200}, "nao publica"},
		{"banco diferente", healthRead{status: 200, database: "tilapou"}, "escreve em"},
		// 200 com corpo que nao e JSON: o daemon responde, e dizer "nao respondeu" manda
		// cacar o problema errado. O proprio daemon serve 200 text/html em /docs.
		{"200 com corpo ilegivel", healthRead{status: 200, err: errJSONQuebrado}, "nao entendi"},
	}

	for _, caso := range casos {
		err := checkDatabase("tilapou_qa", caso.leitura)
		if err == nil {
			t.Errorf("%s: o portao aceitou", caso.nome)

			continue
		}
		if !strings.Contains(err.Error(), caso.quer) {
			t.Errorf("%s: a recusa diz %q, e nao menciona %q", caso.nome, err, caso.quer)
		}
	}

	if err := checkDatabase("tilapou_qa", healthRead{status: 200, database: "tilapou_qa"}); err != nil {
		t.Errorf("banco igual foi recusado: %v", err)
	}
}

// Duas causas que ainda caiam no balde errado, as duas achadas medindo:
//
//  1. status zero era testado ANTES do erro, entao daemon de pe com endereco sem esquema
//     virava "o daemon nao respondeu" — mandando reiniciar um daemon que esta no ar.
//  2. o decoder aceitava lixo depois do JSON, entao um corpo meio valido passava pela guarda
//     inteira. Isso nao e frase errada: e a fazenda sendo apagada por engano.
func TestARecusaSeparaTransporteDeCorpoQuebrado(t *testing.T) {
	t.Parallel()

	comErro := healthRead{status: 0, err: errJSONQuebrado}
	if err := checkDatabase("tilapou_qa", comErro); err == nil ||
		!strings.Contains(err.Error(), "nao entendi") {
		t.Errorf("erro de transporte com status zero virou %v", err)
	}

	semNada := healthRead{}
	if err := checkDatabase("tilapou_qa", semNada); err == nil ||
		!strings.Contains(err.Error(), "nao respondeu") {
		t.Errorf("daemon fora do ar virou %v", err)
	}
}

// Corpo com JSON valido seguido de lixo tem de ser recusado: o decoder de fluxo devolvia o
// primeiro valor e seguia em frente, entao a guarda inteira passava com um corpo que ninguem
// escreveu de caso — e ai a limpeza apaga a fazenda que estiver la.
func TestOPortaoRecusaCorpoComLixoDepoisDoJSON(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"lixo depois":  `{"database":"tilapou_qa"} LIXO NAO JSON`,
		"html":         `<!doctype html><title>docs</title>`,
		"vazio":        ``,
		"json cortado": `{"database":"tilap`,
	}

	for nome, corpo := range casos {
		if _, err := decodeHealth(strings.NewReader(corpo)); err == nil {
			t.Errorf("%s: o portao aceitou o corpo %q", nome, corpo)
		}
	}

	got, err := decodeHealth(strings.NewReader(`{"database":"tilapou_qa"}`))
	if err != nil || got != "tilapou_qa" {
		t.Errorf("corpo valido devolveu %q, %v", got, err)
	}
}

// O teto de leitura nao pode truncar em silencio: com o lixo depois do byte do teto, o
// prefixo cortado e JSON valido, a guarda passa inteira e a limpeza apaga a fazenda. O @qa
// cravou o limiar no byte — 65510 de enchimento recusava, 65511 passava.
func TestOPortaoRecusaCorpoQueEstouraOTeto(t *testing.T) {
	t.Parallel()

	valido := `{"database":"tilapou_qa"}`
	enchimento := strings.Repeat(" ", healthBodyLimit-len(valido)) + "LIXO"

	if _, err := decodeHealth(strings.NewReader(valido + enchimento)); err == nil {
		t.Error("o portao aceitou um corpo que passa do teto: o lixo do fim foi jogado fora sem ser olhado")
	}

	// E o par: corpo que cabe no teto continua passando, senao a guarda recusaria tudo.
	if _, err := decodeHealth(strings.NewReader(valido)); err != nil {
		t.Errorf("corpo valido dentro do teto foi recusado: %v", err)
	}
}
