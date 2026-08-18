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
	fenceThickness    = 1
	rowsPerCell       = 2
	minMapCols        = 11
	minMapRows        = 7
	maxMapCols        = 21
	maxMapRows        = 11
	pondCols          = 4
	pondRows          = 3
	pondOriginX       = 1
	pondOriginY       = 1
	shedFromRight     = 2
	shedFromBottom    = 3
)

type tileKind uint8

const (
	tileGrass tileKind = iota
	tilePath
	tileWater
	tileWaterTop
	tileWaterBottom
	tileWaterLeft
	tileWaterRight
	tileFence
	tileShed
)

type farmMap struct {
	tiles [maxMapRows][maxMapCols]tileKind
	cols  int
	rows  int
}

// mapSizeFor grows the farm with the terminal, between the smallest map that still tells
// the story and a cap that keeps a walk across it short.
func mapSizeFor(width, height int) (cols, rows int) {
	if width <= 0 || height <= 0 {
		return minMapCols, minMapRows
	}

	cols = min(max(width/gb.TileSize, minMapCols), maxMapCols)
	rows = min(max((height-gbChromeRows)*2/gb.TileSize, minMapRows), maxMapRows)

	return cols, rows
}

func newFarmMap(tanks, cols, rows int) farmMap {
	m := farmMap{cols: min(max(cols, minMapCols), maxMapCols), rows: min(max(rows, minMapRows), maxMapRows)}

	for y := range m.rows {
		for x := range m.cols {
			m.tiles[y][x] = tileGrass
		}
	}

	for x := range m.cols {
		m.tiles[0][x] = tileFence
		m.tiles[m.rows-1][x] = tileFence
	}
	for y := range m.rows {
		m.tiles[y][0] = tileFence
		m.tiles[y][m.cols-1] = tileFence
	}

	for x := 1; x < m.cols-1; x++ {
		m.tiles[m.rows-pathRowFromBottom][x] = tilePath
	}

	m.tiles[m.shedY()][m.shedX()] = tileShed

	ponds := min(max(tanks, 1), maxPonds)
	for i := range ponds {
		originX := pondOriginX + i*(pondCols+1)
		for y := range pondRows {
			for x := range pondCols {
				m.tiles[pondOriginY+y][originX+x] = waterTile(x, y)
			}
		}
	}

	return m
}

func (m farmMap) shedX() int {
	return m.cols - shedFromRight
}

func (m farmMap) shedY() int {
	return m.rows - shedFromBottom
}

func waterTile(x, y int) tileKind {
	switch {
	case y == 0:
		return tileWaterTop
	case y == pondRows-1:
		return tileWaterBottom
	case x == 0:
		return tileWaterLeft
	case x == pondCols-1:
		return tileWaterRight
	}

	return tileWater
}

func (m farmMap) blocked(x, y int) bool {
	if x < 0 || y < 0 || x >= m.cols || y >= m.rows {
		return true
	}

	switch m.tiles[y][x] {
	case tileWater, tileWaterTop, tileWaterBottom, tileWaterLeft, tileWaterRight, tileFence, tileShed:
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

func newAvatar(m farmMap) avatar {
	return avatar{x: 1, y: m.rows - pathRowFromBottom, facing: facingDown}
}

// clampedTo keeps the avatar inside a map that just changed size, on a tile it can stand on.
func (a avatar) clampedTo(m farmMap) avatar {
	a.x = min(max(a.x, fenceThickness), m.cols-fenceThickness-1)
	a.y = min(max(a.y, fenceThickness), m.rows-fenceThickness-1)

	if m.blocked(a.x, a.y) {
		a.x, a.y = 1, m.rows-pathRowFromBottom
	}

	return a
}

func (a avatar) sprite() gb.Sprite {
	switch a.facing {
	case facingUp:
		return gb.PlayerUp
	case facingLeft:
		return gb.PlayerSide.Flipped()
	case facingRight:
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

	return a.x, a.y
}

func renderMap(m farmMap, a avatar, snapshot client.Snapshot, frame int) string {
	canvas := gb.NewCanvas(m.cols*gb.TileSize, m.rows*gb.TileSize)

	for y := range m.rows {
		for x := range m.cols {
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
	case tileWaterBottom:
		return gb.WaterBottom
	case tileWaterLeft:
		return gb.WaterLeft
	case tileWaterRight:
		return gb.WaterRight
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
