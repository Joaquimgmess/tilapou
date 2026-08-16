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
