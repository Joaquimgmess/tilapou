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

func TestTanqueNovoJaNasceComOxigenio(t *testing.T) {
	t.Parallel()

	b := testBalance(t)
	s := NewState(1, 0, 0)
	s.Tick = 300 * TicksPerDay

	id, ok := s.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if !ok {
		t.Fatal("sem tanque")
	}

	tank := s.tank(id)
	if tank.Oxygen == 0 {
		t.Error("o tanque novo nasce com oxigenio zero: a tela acusa falta de ar que nao existe")
	}
	if got, want := tank.Oxygen, oxygenAt(b, tank, s.Tick, s.Zone); got != want {
		t.Errorf("o tanque novo nasceu com %d de oxigenio, o ambiente da %d", got, want)
	}
}
