package main

import "testing"

// O nome do banco tem de sair da mesma leitura que o driver faz. Lendo so o PATH, a URL
// abaixo publica "tilapou_qa7" e o pgx escreve em "tilapou" — foi o quinto caminho para o
// save do jogador, e ele racha o portao em dois: a limpeza vai para um banco e as teclas do
// roteiro para outro, com tudo verde.
func TestONomeDoBancoSaiDoQueODriverUsa(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"postgres://u:p@localhost:5433/tilapou_qa?sslmode=disable":                 "tilapou_qa",
		"postgres://u:p@localhost:5433/tilapou_qa7?sslmode=disable&dbname=tilapou": "tilapou",
		"postgres://u:p@localhost:5433/?dbname=tilapou_qa":                         "tilapou_qa",
		"postgres://u:p@localhost:5433/tilapou":                                    "tilapou",
	}

	for url, want := range casos {
		if got := databaseName(url); got != want {
			t.Errorf("%s: o daemon publica %q e o driver usa %q", url, got, want)
		}
	}
}
