package tui

import "testing"

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
