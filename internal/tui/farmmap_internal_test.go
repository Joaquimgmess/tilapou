package tui

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

func TestOMapaCresceComATelaEParaNoTeto(t *testing.T) {
	t.Parallel()

	casos := map[string]struct {
		width, height int
		cols, rows    int
	}{
		"tela minima":       {width: minMapCols * gb.TileSize, height: minMapRows*gb.TileSize/2 + gbChromeRows, cols: minMapCols, rows: minMapRows},
		"tela apertada":     {width: 40, height: 10, cols: minMapCols, rows: minMapRows},
		"tela comum":        {width: 120, height: 40, cols: 15, rows: 8},
		"tela grande":       {width: 400, height: 200, cols: maxMapCols, rows: maxMapRows},
		"tamanho ainda nao": {width: 0, height: 0, cols: minMapCols, rows: minMapRows},
	}

	for name, tc := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cols, rows := mapSizeFor(tc.width, tc.height)
			if cols != tc.cols || rows != tc.rows {
				t.Errorf("mapSizeFor(%d, %d) = %dx%d, queria %dx%d", tc.width, tc.height, cols, rows, tc.cols, tc.rows)
			}
		})
	}
}

func TestOMapaContinuaJogavelEmQualquerTamanho(t *testing.T) {
	t.Parallel()

	for cols := minMapCols; cols <= maxMapCols; cols++ {
		for rows := minMapRows; rows <= maxMapRows; rows++ {
			m := newFarmMap(maxPonds, cols, rows)

			if !m.blocked(m.shedX(), m.shedY()) {
				t.Fatalf("%dx%d: o galpao nao esta bloqueando", cols, rows)
			}
			if _, ok := m.pondAt(m.shedX(), m.shedY()); ok {
				t.Fatalf("%dx%d: o galpao caiu em cima de um viveiro", cols, rows)
			}

			a := newAvatar(m)
			if m.blocked(a.x, a.y) {
				t.Fatalf("%dx%d: o avatar comeca dentro de um tile bloqueado", cols, rows)
			}
			if a.x >= m.cols || a.y >= m.rows {
				t.Fatalf("%dx%d: o avatar comeca fora do mapa em (%d,%d)", cols, rows, a.x, a.y)
			}
		}
	}
}

func TestAvatarVoltaParaDentroQuandoOMapaEncolhe(t *testing.T) {
	t.Parallel()

	grande := newFarmMap(1, maxMapCols, maxMapRows)
	a := newAvatar(grande)
	a.x, a.y = maxMapCols-2, maxMapRows-2

	pequeno := newFarmMap(1, minMapCols, minMapRows)

	got := a.clampedTo(pequeno)
	if got.x >= pequeno.cols || got.y >= pequeno.rows {
		t.Fatalf("o avatar ficou em (%d,%d), fora do mapa %dx%d", got.x, got.y, pequeno.cols, pequeno.rows)
	}
	if pequeno.blocked(got.x, got.y) {
		t.Errorf("o avatar parou num tile bloqueado em (%d,%d)", got.x, got.y)
	}
}
