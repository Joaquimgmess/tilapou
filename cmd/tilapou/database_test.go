package main

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// O nome do banco tem de sair da CONEXAO, e nao da string: enquanto a fonte for o texto da
// URL, todo parametro que nao esteja nele abre caminho — PGDATABASE fazia o daemon conectar
// no banco do dono publicando database vazio, e PGHOST/PGPORT trocam ate o servidor sem mudar
// uma letra do que a tela mostra.
func TestONomeDoBancoSaiDaConexaoENaoDaString(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://u:p@localhost:5433/tilapou_qa?sslmode=disable")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got := cfg.ConnConfig.Database; got != "tilapou_qa" {
		t.Fatalf("a conexao diz %q", got)
	}

	// A env vence a URL, e e por isso que ler a string nao serve de guarda.
	t.Setenv("PGDATABASE", "tilapou")

	comEnv, err := pgxpool.ParseConfig("postgres://u:p@localhost:5433/?sslmode=disable")
	if err != nil {
		t.Fatalf("parse com env: %v", err)
	}

	if got := comEnv.ConnConfig.Database; got != "tilapou" {
		t.Errorf("com PGDATABASE=tilapou a conexao diz %q: a guarda nao veria o banco do dono", got)
	}
}
