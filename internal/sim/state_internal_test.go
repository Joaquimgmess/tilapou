package sim

import "testing"

func TestTodoTankKindTemNomeQueVoltaParaOMesmoTipo(t *testing.T) {
	t.Parallel()

	for kind := range tankKindCount {
		name := kind.String()
		if name == "" || name == invalidName {
			t.Errorf("o tipo de tanque %d saiu sem nome: a API manda kind vazio e a compra e recusada", kind)

			continue
		}

		back, ok := TankKindNamed(name)
		if !ok || back != kind {
			t.Errorf("%q voltou como o tipo %d, queria %d", name, back, kind)
		}
	}
}

func TestOSegundoTanqueDoMesmoTipoCustaMais(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	base := b.Tanks[TankEarthPond].BaseCost

	first := ladderCost(base, 0, b.Progression.CostFactorPPM)
	second := ladderCost(base, 1, b.Progression.CostFactorPPM)

	if first != base {
		t.Fatalf("o primeiro tanque custa %d, queria o preco de tabela %d", first, base)
	}
	if second <= first {
		t.Fatalf("o segundo tanque custa %d, queria mais do que o primeiro (%d)", second, first)
	}
}
