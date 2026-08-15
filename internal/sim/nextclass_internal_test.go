package sim

import "testing"

func TestNextClassPointsAtTheEntryOfTheBetterBand(t *testing.T) {
	t.Parallel()

	b := testBalance(t)

	for _, tc := range []struct {
		name  string
		mass  Micrograms
		entry Micrograms
		ok    bool
	}{
		{"faixa de 88 por cento sobe para 100 logo acima de 600 g", 450 * MicrogramsPerGram, 600*MicrogramsPerGram + 1, true},
		{"faixa de 100 por cento sobe para 112 logo acima de 900 g", 800 * MicrogramsPerGram, 900*MicrogramsPerGram + 1, true},
		{"logo acima de 400 g sobe para 88 por cento", 306 * MicrogramsPerGram, 400*MicrogramsPerGram + 1, true},
		{"acima de 900 g ja esta na melhor faixa", 950 * MicrogramsPerGram, 0, false},
		{"peixe grande demais perde valor e nao tem proxima", 1_500 * MicrogramsPerGram, 0, false},
	} {
		entry, gain, ok := b.NextClass(tc.mass)
		if ok != tc.ok || entry != tc.entry {
			t.Errorf("%s: NextClass(%d) = (%d, %d, %v), queria (%d, _, %v)",
				tc.name, tc.mass, entry, gain, ok, tc.entry, tc.ok)

			continue
		}
		if ok && b.ClassPPM(entry) <= b.ClassPPM(tc.mass) {
			t.Errorf("%s: o preco em %d nao e melhor que em %d", tc.name, entry, tc.mass)
		}
	}
}
