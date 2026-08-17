package tui

import (
	"strings"

	"github.com/Joaquimgmess/tilapou/internal/client"
	"github.com/Joaquimgmess/tilapou/internal/tui/gb"
)

const (
	maxPonds          = 2
	fishPerRow        = 1_000
	swimDivisor       = 2
	pathRowFromBottom = 2
	mapCols           = 11
	mapRows           = 7
	pondCols          = 4
	pondRows          = 3
	pondOriginX       = 1
	pondOriginY       = 1
	shedX             = 9
	shedY             = 4
)

type tileKind uint8

const (
	tileGrass tileKind = iota
	tilePath
	tileWater
	tileWaterTop
	tileFence
	tileShed
)

type farmMap struct {
	tiles [mapRows][mapCols]tileKind
}

func newFarmMap(tanks int) farmMap {
	var m farmMap

	for y := range mapRows {
		for x := range mapCols {
			m.tiles[y][x] = tileGrass
		}
	}

	for x := range mapCols {
		m.tiles[0][x] = tileFence
		m.tiles[mapRows-1][x] = tileFence
	}
	for y := range mapRows {
		m.tiles[y][0] = tileFence
		m.tiles[y][mapCols-1] = tileFence
	}

	for x := 1; x < mapCols-1; x++ {
		m.tiles[mapRows-pathRowFromBottom][x] = tilePath
	}

	m.tiles[shedY][shedX] = tileShed

	ponds := min(max(tanks, 1), maxPonds)
	for i := range ponds {
		originX := pondOriginX + i*(pondCols+1)
		for y := range pondRows {
			for x := range pondCols {
				kind := tileWater
				if y == 0 {
					kind = tileWaterTop
				}
				m.tiles[pondOriginY+y][originX+x] = kind
			}
		}
	}

	return m
}

func (m farmMap) blocked(x, y int) bool {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return true
	}

	switch m.tiles[y][x] {
	case tileWater, tileWaterTop, tileFence, tileShed:
		return true
	case tileGrass, tilePath:
		return false
	}

	return false
}

func (farmMap) pondAt(x, y int) (index int, found bool) {
	for i := range maxPonds {
		originX := pondOriginX + i*(pondCols+1)
		if x >= originX && x < originX+pondCols && y >= pondOriginY && y < pondOriginY+pondRows {
			return i, true
		}
	}

	return 0, false
}

type facing uint8

// Directions the avatar can look at.
const (
	facingDown facing = iota
	facingUp
	facingLeft
	facingRight
)

type avatar struct {
	x, y   int
	facing facing
}

func newAvatar() avatar {
	return avatar{x: 1, y: mapRows - pathRowFromBottom, facing: facingDown}
}

func (a avatar) sprite() gb.Sprite {
	switch a.facing {
	case facingUp:
		return gb.PlayerUp
	case facingLeft, facingRight:
		return gb.PlayerSide
	case facingDown:
		return gb.PlayerDown
	}

	return gb.PlayerDown
}

func (a avatar) ahead() (x, y int) {
	switch a.facing {
	case facingUp:
		return a.x, a.y - 1
	case facingDown:
		return a.x, a.y + 1
	case facingLeft:
		return a.x - 1, a.y
	case facingRight:
		return a.x + 1, a.y
	}

	return a.x + 1, a.y
}

func renderMap(m farmMap, a avatar, snapshot client.Snapshot, frame int) string {
	canvas := gb.NewCanvas(mapCols*gb.TileSize, mapRows*gb.TileSize)

	for y := range mapRows {
		for x := range mapCols {
			spriteFor(m.tiles[y][x]).Draw(canvas, x*gb.TileSize, y*gb.TileSize)
		}
	}

	drawPondContents(canvas, snapshot, frame)
	a.sprite().Draw(canvas, a.x*gb.TileSize, a.y*gb.TileSize)

	return canvas.Render()
}

func spriteFor(kind tileKind) gb.Sprite {
	switch kind {
	case tilePath:
		return gb.Path
	case tileWater:
		return gb.Water
	case tileWaterTop:
		return gb.WaterTop
	case tileFence:
		return gb.Fence
	case tileShed:
		return gb.Shed
	case tileGrass:
		return gb.Grass
	}

	return gb.Grass
}

func drawPondContents(canvas *gb.Canvas, snapshot client.Snapshot, frame int) {
	for i := range snapshot.Tanks {
		tank := &snapshot.Tanks[i]
		if i >= maxPonds {
			break
		}

		originX := pondOriginX + i*(pondCols+1)
		if tank.Aerating {
			gb.Aerator.Draw(canvas, (originX+pondCols-1)*gb.TileSize, pondOriginY*gb.TileSize)
		}

		swim := (frame / swimDivisor) % pondCols
		fishRows := min(int(tank.Fish/fishPerRow)+1, pondRows)
		for row := range fishRows {
			x := (originX + (swim+row)%pondCols) * gb.TileSize
			gb.Fish.Draw(canvas, x, (pondOriginY+row)*gb.TileSize)
		}
	}
}

func mapLegend(snapshot client.Snapshot) string {
	if len(snapshot.Tanks) == 0 {
		return "sem tanques"
	}

	parts := make([]string, 0, len(snapshot.Tanks))
	for i := range snapshot.Tanks {
		parts = append(parts, tankHeadline(snapshot.Tanks[i]))
	}

	return strings.Join(parts, "   ")
}
