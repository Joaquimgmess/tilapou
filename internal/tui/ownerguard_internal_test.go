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
	} {
		if err := qaDaemon(addr); err == nil {
			t.Errorf("o portao aceitou %q, que e o daemon do dono", addr)
		}
	}

	for _, addr := range []string{
		"http://localhost:8098",
		"http://localhost:8106",
	} {
		if err := qaDaemon(addr); err != nil {
			t.Errorf("o portao recusou %q, que e daemon de teste: %v", addr, err)
		}
	}
}
