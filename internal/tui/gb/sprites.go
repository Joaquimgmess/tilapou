package gb

const TileSize = 8

type Sprite struct {
	rows [TileSize]string
}

func NewSprite(rows [TileSize]string) Sprite {
	return Sprite{rows: rows}
}

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
		"22222222",
		"23333332",
		"23222232",
		"23233323",
		"23233323",
		"23222232",
		"23333332",
		"22222222",
	})

	Water = NewSprite([TileSize]string{
		"22222222",
		"22322232",
		"23222322",
		"22222222",
		"22232223",
		"23222232",
		"22222222",
		"22322232",
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
		"30000003",
		"30322303",
		"30322303",
		"33333333",
	})

	Aerator = NewSprite([TileSize]string{
		"00033000",
		"00303000",
		"03333300",
		"33000330",
		"03333300",
		"00303000",
		"00033000",
		"00000000",
	})

	Fish = NewSprite([TileSize]string{
		"00000000",
		"00033000",
		"00333300",
		"33333330",
		"33033330",
		"00333300",
		"00033000",
		"00000000",
	})

	PlayerDown = NewSprite([TileSize]string{
		"00333000",
		"03222300",
		"03232300",
		"03222300",
		"00333000",
		"03333300",
		"03000300",
		"03000300",
	})

	PlayerUp = NewSprite([TileSize]string{
		"00333000",
		"03333300",
		"03333300",
		"03333300",
		"00333000",
		"03333300",
		"03000300",
		"03000300",
	})

	PlayerSide = NewSprite([TileSize]string{
		"00333000",
		"03222300",
		"03223300",
		"03222300",
		"00333000",
		"00333300",
		"00300300",
		"00300300",
	})
)
