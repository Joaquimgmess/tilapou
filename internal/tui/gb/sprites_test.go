package gb_test

import (
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

func shades(t *testing.T, s gb.Sprite) [gb.TileSize][gb.TileSize]gb.Shade {
	t.Helper()

	canvas := gb.NewCanvas(gb.TileSize, gb.TileSize)
	s.Draw(canvas, 0, 0)

	var grid [gb.TileSize][gb.TileSize]gb.Shade
	for y := range gb.TileSize {
		for x := range gb.TileSize {
			grid[y][x] = canvas.At(x, y)
		}
	}

	return grid
}

func TestFlippedEspelhaAsColunas(t *testing.T) {
	t.Parallel()

	origem := shades(t, gb.PlayerSide)
	espelho := shades(t, gb.PlayerSide.Flipped())

	for y := range gb.TileSize {
		for x := range gb.TileSize {
			if got, want := espelho[y][x], origem[y][gb.TileSize-1-x]; got != want {
				t.Fatalf("pixel (%d,%d) do espelho e %d, queria %d", x, y, got, want)
			}
		}
	}
}

func TestOsDoisLadosNaoDesenhamOMesmo(t *testing.T) {
	t.Parallel()

	if shades(t, gb.PlayerSide) == shades(t, gb.PlayerSide.Flipped()) {
		t.Error("o sprite de lado e simetrico: andar para a esquerda desenha igual a direita")
	}
}

func TestFlipDuasVezesVoltaAoOriginal(t *testing.T) {
	t.Parallel()

	if shades(t, gb.PlayerSide.Flipped().Flipped()) != shades(t, gb.PlayerSide) {
		t.Error("espelhar duas vezes nao devolveu o sprite original")
	}
}
