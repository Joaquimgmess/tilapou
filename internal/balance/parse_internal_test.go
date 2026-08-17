package balance

import (
	"errors"
	"strings"
	"testing"
)

func balanceTOML(t *testing.T) string {
	t.Helper()

	raw, err := files.ReadFile("balance.toml")
	if err != nil {
		t.Fatalf("lendo o balance embutido: %v", err)
	}

	return string(raw)
}

func TestParseAceitaOBalanceEmbutido(t *testing.T) {
	t.Parallel()

	if _, err := parse(balanceTOML(t)); err != nil {
		t.Fatalf("o balance embutido nao passa no parse: %v", err)
	}
}

func TestParseRecusaTOMLQuebrado(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		mutate func(string) string
		want   error
	}{
		"chave que ninguem le": {
			mutate: func(s string) string { return s + "\nchave_inventada = 1\n" },
			want:   ErrUnusedKeys,
		},
		"secao que ninguem le": {
			mutate: func(s string) string { return s + "\n[secao_inventada]\nx = 1\n" },
			want:   ErrUnusedKeys,
		},
		"tipo de tanque fora do enum": {
			mutate: func(s string) string {
				return strings.Replace(s, `tipo = "viveiro_escavado"`, `tipo = "aquario"`, 1)
			},
			want: ErrUnknownTankKind,
		},
		"automacao fora do enum": {
			mutate: func(s string) string {
				return strings.Replace(s, `nome = "comedouro"`, `nome = "trator"`, 1)
			},
			want: ErrUnknownAutomation,
		},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			raw := tc.mutate(balanceTOML(t))
			if raw == balanceTOML(t) {
				t.Fatal("a mutacao nao mudou o TOML: o teste nao esta exercitando nada")
			}

			_, err := parse(raw)
			if !errors.Is(err, tc.want) {
				t.Errorf("parse devolveu %v, queria %v", err, tc.want)
			}
		})
	}
}
