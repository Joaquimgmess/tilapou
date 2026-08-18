package gb

// TileSize is the tile side in pixels.
const TileSize = 8

// Sprite is a tile of TileSize by TileSize pixels.
type Sprite struct {
	rows [TileSize]string
}

// NewSprite builds a sprite of TileSize rows where each digit '0' to '3' is a Shade and
// '.' leaves the pixel transparent. Note that '0' paints the lightest shade, it does not
// clear: padding a sprite with '0' draws a pale block over the scenery.
func NewSprite(rows [TileSize]string) Sprite {
	return Sprite{rows: rows}
}

// Flipped returns the sprite mirrored horizontally, for the side that faces the other way.
func (s Sprite) Flipped() Sprite {
	var flipped Sprite

	for i, row := range s.rows {
		runes := []rune(row)
		for a, b := 0, len(runes)-1; a < b; a, b = a+1, b-1 {
			runes[a], runes[b] = runes[b], runes[a]
		}
		flipped.rows[i] = string(runes)
	}

	return flipped
}

// Draw draws the sprite with its top left corner at (x, y).
func (s Sprite) Draw(c *Canvas, x, y int) {
	for dy, row := range s.rows {
		for dx, ch := range row {
			if ch < '0' || ch > '3' {
				continue
			}
			c.Set(x+dx, y+dy, Shade(ch-'0'))
		}
	}
}

// Scenery and avatar tiles.
var (
	Grass = NewSprite([TileSize]string{
		"11111111",
		"11121111",
		"11111111",
		"11111211",
		"11211111",
		"11111111",
		"12111121",
		"11111111",
	})

	Path = NewSprite([TileSize]string{
		"00000000",
		"00010000",
		"00000000",
		"00000100",
		"01000000",
		"00000000",
		"00001000",
		"00000000",
	})

	WaterTop = NewSprite([TileSize]string{
		"33333333",
		"23333332",
		"22222222",
		"22112222",
		"22222222",
		"22222211",
		"22222222",
		"11222222",
	})

	Water = NewSprite([TileSize]string{
		"22222222",
		"22112222",
		"22222222",
		"22222211",
		"22222222",
		"11222222",
		"22222222",
		"22221122",
	})

	WaterBottom = NewSprite([TileSize]string{
		"22112222",
		"22222222",
		"22222211",
		"22222222",
		"11222222",
		"22222222",
		"23333332",
		"33333333",
	})

	WaterLeft = NewSprite([TileSize]string{
		"32222222",
		"32112222",
		"32222222",
		"32222211",
		"32222222",
		"31222222",
		"32222222",
		"32221122",
	})

	WaterRight = NewSprite([TileSize]string{
		"22222223",
		"22112223",
		"22222223",
		"22222213",
		"22222223",
		"11222223",
		"22222223",
		"22221123",
	})

	Fence = NewSprite([TileSize]string{
		".3....3.",
		".3....3.",
		"33333333",
		".3....3.",
		".3....3.",
		"33333333",
		".3....3.",
		".3....3.",
	})

	Shed = NewSprite([TileSize]string{
		"...33...",
		"..3333..",
		".333333.",
		"33333333",
		"31111113",
		"31002213",
		"31112213",
		"33332233",
	})

	Aerator = NewSprite([TileSize]string{
		"..3..3..",
		".3.33.3.",
		"..3333..",
		"33322333",
		"33322333",
		"..3333..",
		".3.33.3.",
		"..3..3..",
	})

	Fish = NewSprite([TileSize]string{
		"........",
		"........",
		"...333..",
		"3.33333.",
		"3333303.",
		"3.33333.",
		"...333..",
		"........",
	})

	PlayerDown = NewSprite([TileSize]string{
		"..3333..",
		".333333.",
		".330033.",
		".300003.",
		"32222223",
		".322223.",
		"..3333..",
		".33..33.",
	})

	PlayerUp = NewSprite([TileSize]string{
		"..3333..",
		".333333.",
		".300003.",
		".330033.",
		"32222223",
		".322223.",
		"..3333..",
		".33..33.",
	})

	PlayerSide = NewSprite([TileSize]string{
		"..3333..",
		".333333.",
		"..30033.",
		"..30003.",
		".3222233",
		"..32223.",
		"..3333..",
		"..33.33.",
	})
)
