package view

import (
	"strings"
)

type Rect struct {
	X0, Y0, X1, Y1 int `json:"-"`
}

func Footprint(s Structure) Rect {
	return Rect{X0: s.X, Y0: s.Y, X1: s.X + s.W - 1, Y1: s.Y + s.H - 1}
}

func (a Rect) Gap(b Rect) int {
	dx := max(0, max(b.X0-a.X1, a.X0-b.X1))
	dy := max(0, max(b.Y0-a.Y1, a.Y0-b.Y1))
	return max(dx, dy)
}

func (a Rect) Contains(x, y int) bool {
	return x >= a.X0 && x <= a.X1 && y >= a.Y0 && y <= a.Y1
}

func SiteBuildable(t TileKind, placement string) bool {
	switch t {
	case TileGrass, TileMeadow, TileSand:
		return true
	case TileForum:
		return placement == PlaceForum
	}
	return false
}

func KeepDef(def BuildDef) bool {
	return def.Kind == StructKeep || strings.HasSuffix(def.Kind, "-"+StructKeep)
}

func Abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
