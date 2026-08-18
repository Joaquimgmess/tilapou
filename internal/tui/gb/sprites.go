package gb

// TileSize is the tile side in pixels.
const TileSize = 8

// Sprite is a tile of TileSize by TileSize pixels.
type Sprite struct {
	rows [TileSize]string
}

// NewSprite builds a sprite of TileSize rows where each digit '0' to '3' is a Shade.
// Any other character stays transparent, without warning.
func NewSprite(rows [TileSize]string) Sprite {
	return Sprite{rows: rows}
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
		"03000030",
		"03000030",
		"33333333",
		"03000030",
		"03000030",
		"33333333",
		"03000030",
		"03000030",
	})

	Shed = NewSprite([TileSize]string{
		"00033000",
		"00333300",
		"03333330",
		"33333333",
		"31111113",
		"31002213",
		"31112213",
		"33332233",
	})

	Aerator = NewSprite([TileSize]string{
		"00300300",
		"03033030",
		"00333300",
		"33322333",
		"33322333",
		"00333300",
		"03033030",
		"00300300",
	})

	Fish = NewSprite([TileSize]string{
		"00000000",
		"00000000",
		"00033300",
		"30333330",
		"33333030",
		"30333330",
		"00033300",
		"00000000",
	})

	PlayerDown = NewSprite([TileSize]string{
		"00333300",
		"03333330",
		"01311310",
		"00111100",
		"12222221",
		"02222220",
		"00333300",
		"03300330",
	})

	PlayerUp = NewSprite([TileSize]string{
		"00333300",
		"03333330",
		"01111110",
		"00333300",
		"12222221",
		"02222220",
		"00333300",
		"03300330",
	})

	PlayerSide = NewSprite([TileSize]string{
		"00333300",
		"03333330",
		"00113100",
		"00111100",
		"00222211",
		"00222200",
		"00333300",
		"00330330",
	})
)
