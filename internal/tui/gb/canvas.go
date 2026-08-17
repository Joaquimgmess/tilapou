// Package gb draws GameBoy style screens.
package gb

import (
	"strconv"
	"strings"
)

// Shade is a palette shade, from the lightest (0) to the darkest (3).
type Shade uint8

// The four valid shades, from the lightest to the darkest.
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

// Hex gives the shade color as #RRGGBB, or "#000000" if the shade is invalid.
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

// Canvas is the pixel grid where sprites are drawn.
type Canvas struct {
	width  int
	height int
	pixels []Shade
}

// NewCanvas creates a canvas of width by height pixels; an odd height rounds up to the next even one.
func NewCanvas(width, height int) *Canvas {
	if height%2 != 0 {
		height++
	}

	return &Canvas{width: width, height: height, pixels: make([]Shade, width*height)}
}

// Set paints the pixel at (x, y); outside the canvas or an invalid shade does nothing.
func (c *Canvas) Set(x, y int, s Shade) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height || s >= shadeCount {
		return
	}

	c.pixels[y*c.width+x] = s
}

// Render gives the canvas as ANSI text, one terminal line per two pixels of height.
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
