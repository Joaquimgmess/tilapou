package gb_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

// dump renders one digit per pixel, so an assertion can spell out the block it expects.
func dump(c *gb.Canvas, w, h int) string {
	var sb strings.Builder

	for y := range h {
		for x := range w {
			sb.WriteByte(byte('0' + c.At(x, y)))
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func TestRenderPoeOPixelDeCimaNoForeground(t *testing.T) {
	t.Parallel()

	c := gb.NewCanvas(1, 2)
	c.Set(0, 0, gb.Darkest)
	c.Set(0, 1, gb.Light)

	rendered := gb.NewCanvas(1, 2).Render()
	if rendered == c.Render() {
		t.Fatal("pintar os dois pixels da celula nao mudou nada no render")
	}

	top, bottom := hexRGB(t, gb.Hex(gb.Darkest)), hexRGB(t, gb.Hex(gb.Light))
	if !strings.Contains(c.Render(), "38;2;"+top) {
		t.Errorf("o pixel de cima nao virou foreground: %q", c.Render())
	}
	if !strings.Contains(c.Render(), "48;2;"+bottom) {
		t.Errorf("o pixel de baixo nao virou background: %q", c.Render())
	}
}

// hexRGB turns "#E0F8D0" into "224;248;208", which is how the escape sequence spells it.
func hexRGB(t *testing.T, hex string) string {
	t.Helper()

	if len(hex) != 7 {
		t.Fatalf("hex fora do formato: %q", hex)
	}

	out := make([]string, 0, 3)
	for i := 1; i < 7; i += 2 {
		v, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err != nil {
			t.Fatalf("componente %q nao e hexadecimal: %v", hex[i:i+2], err)
		}
		out = append(out, strconv.FormatUint(v, 10))
	}

	return strings.Join(out, ";")
}

func TestCanvasDeAlturaImparArredondaParaCima(t *testing.T) {
	t.Parallel()

	casos := map[int]int{2: 1, 4: 2, 5: 3, 6: 3}

	for height, wantLines := range casos {
		c := gb.NewCanvas(3, height)
		if got := strings.Count(c.Render(), "\n") + 1; got != wantLines {
			t.Errorf("canvas de altura %d rendeu %d linhas, queria %d", height, got, wantLines)
		}
	}
}

func TestDrawRecortaNaBordaSemEstourar(t *testing.T) {
	t.Parallel()

	c := gb.NewCanvas(4, 4)
	for _, at := range [][2]int{{-3, -3}, {2, 2}, {-99, -99}, {99, 99}} {
		gb.Fence.Draw(c, at[0], at[1])
	}

	if got := c.At(99, 99); got != gb.Lightest {
		t.Errorf("At fora do canvas devolveu %d, queria o tom mais claro", got)
	}
}

func TestPontoDeixaOCenarioAparecerEZeroPinta(t *testing.T) {
	t.Parallel()

	c := gb.NewCanvas(gb.TileSize, gb.TileSize)
	for y := range gb.TileSize {
		for x := range gb.TileSize {
			c.Set(x, y, gb.Dark)
		}
	}

	var rows [gb.TileSize]string
	rows[0] = "0......."
	for i := 1; i < gb.TileSize; i++ {
		rows[i] = "........"
	}
	gb.NewSprite(rows).Draw(c, 0, 0)

	if got := c.At(0, 0); got != gb.Lightest {
		t.Errorf("o pixel '0' devia pintar o tom mais claro, veio %d", got)
	}
	if got := c.At(1, 0); got != gb.Dark {
		t.Errorf("o pixel '.' devia deixar o cenario aparecer, veio %d", got)
	}
}

func TestShadeForaDoEnumNaoPintaENaoEstoura(t *testing.T) {
	t.Parallel()

	c := gb.NewCanvas(2, 2)
	c.Set(0, 0, gb.Shade(9))

	if got := c.At(0, 0); got != gb.Lightest {
		t.Errorf("um tom fora do enum pintou %d", got)
	}
	if got := gb.Hex(gb.Shade(9)); got != "#000000" {
		t.Errorf("Hex fora do enum devolveu %q", got)
	}
}

func TestDumpMostraOTileDesenhado(t *testing.T) {
	t.Parallel()

	c := gb.NewCanvas(gb.TileSize, gb.TileSize)
	gb.Fish.Draw(c, 0, 0)

	// A linha do olho: corpo escuro com o pixel claro no meio. O ultimo e transparente,
	// e o canvas em branco ja e o tom mais claro, entao o dump mostra 0 nos dois casos.
	const want = "33333030"

	lines := strings.Split(dump(c, gb.TileSize, gb.TileSize), "\n")
	if lines[4] != want {
		t.Errorf("a linha do olho da tilapia e %q, queria %q", lines[4], want)
	}
}
