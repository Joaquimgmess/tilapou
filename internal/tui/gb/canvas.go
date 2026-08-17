// Package gb desenha telas no estilo GameBoy.
package gb

import (
	"strconv"
	"strings"
)

// Shade e um tom da paleta, do mais claro (0) ao mais escuro (3).
type Shade uint8

// Os quatro tons validos, do mais claro ao mais escuro.
const (
	Lightest Shade = iota
	Light
	Dark
	Darkest
	shadeCount
)

type rgb struct {
	r, g, b uint8
}

var palette = [shadeCount]rgb{
	Lightest: {0xE0, 0xF8, 0xD0},
	Light:    {0x88, 0xC0, 0x70},
	Dark:     {0x34, 0x68, 0x56},
	Darkest:  {0x08, 0x18, 0x20},
}

const bytesPerCell = 24

var pairs = buildPairs()

func buildPairs() [shadeCount][shadeCount]string {
	var table [shadeCount][shadeCount]string

	for top := range shadeCount {
		for bottom := range shadeCount {
			t, b := palette[top], palette[bottom]

			var sb strings.Builder
			sb.WriteString("\x1b[38;2;")
			writeRGB(&sb, t)
			sb.WriteString(";48;2;")
			writeRGB(&sb, b)
			sb.WriteString("m▀")

			table[top][bottom] = sb.String()
		}
	}

	return table
}

func writeRGB(sb *strings.Builder, c rgb) {
	sb.WriteString(strconv.Itoa(int(c.r)))
	sb.WriteByte(';')
	sb.WriteString(strconv.Itoa(int(c.g)))
	sb.WriteByte(';')
	sb.WriteString(strconv.Itoa(int(c.b)))
}

// Hex da a cor do tom em #RRGGBB, ou "#000000" se o tom for invalido.
func Hex(s Shade) string {
	if s >= shadeCount {
		return "#000000"
	}

	c := palette[s]
	digits := "0123456789ABCDEF"

	return string([]byte{
		'#',
		digits[c.r>>4], digits[c.r&0xF],
		digits[c.g>>4], digits[c.g&0xF],
		digits[c.b>>4], digits[c.b&0xF],
	})
}

// Canvas e a grade de pixels onde os sprites sao desenhados.
type Canvas struct {
	width  int
	height int
	pixels []Shade
}

// NewCanvas cria um canvas de width por height pixels; height impar sobe para o par seguinte.
func NewCanvas(width, height int) *Canvas {
	if height%2 != 0 {
		height++
	}

	return &Canvas{width: width, height: height, pixels: make([]Shade, width*height)}
}

// Set pinta o pixel em (x, y); fora do canvas ou tom invalido nao faz nada.
func (c *Canvas) Set(x, y int, s Shade) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height || s >= shadeCount {
		return
	}

	c.pixels[y*c.width+x] = s
}

// Render da o canvas como texto ANSI, uma linha de terminal a cada dois pixels de altura.
func (c *Canvas) Render() string {
	var sb strings.Builder
	sb.Grow(c.width * c.height / 2 * bytesPerCell)

	for y := 0; y < c.height; y += 2 {
		for x := range c.width {
			sb.WriteString(pairs[c.pixels[y*c.width+x]][c.pixels[(y+1)*c.width+x]])
		}
		sb.WriteString("\x1b[m")
		if y+2 < c.height {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}
