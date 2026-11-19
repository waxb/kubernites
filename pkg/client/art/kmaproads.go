package art

import (
	"image/color"
	"strings"
)

var KMapRoadMasks = []string{"n", "e", "s", "w", "ne", "ns", "nw", "es", "ew", "sw", "nes", "new", "nsw", "esw", "nesw"}

const (
	roadLo = 5
	roadHi = 10
)

func KMapRoadMask(n, e, s, w bool) string {
	out := ""
	for i, on := range []bool{n, e, s, w} {
		if on {
			out += string("nesw"[i])
		}
	}
	return out
}

func onRoad(mask string, x, y int) bool {
	inX, inY := x >= roadLo && x <= roadHi, y >= roadLo && y <= roadHi
	switch {
	case inX && y <= roadHi && strings.Contains(mask, "n"):
		return true
	case inX && y >= roadLo && strings.Contains(mask, "s"):
		return true
	case inY && x >= roadLo && strings.Contains(mask, "e"):
		return true
	case inY && x <= roadHi && strings.Contains(mask, "w"):
		return true
	}
	return inX && inY
}

func roadEdge(mask string, x, y int) bool {
	return !onRoad(mask, x-1, y) || !onRoad(mask, x+1, y) || !onRoad(mask, x, y-1) || !onRoad(mask, x, y+1)
}

func paintRoadTile(c *canvas, state, mask string) {
	for y := 0; y < KMapCell; y++ {
		for x := 0; x < KMapCell; x++ {
			if !onRoad(mask, x, y) {
				continue
			}
			if col, ok := roadPixel(state, mask, x, y); ok {
				c.dot(x, y, col)
			}
		}
	}
}

func roadPixel(state, mask string, x, y int) (color.NRGBA, bool) {
	edge := roadEdge(mask, x, y)
	switch state {
	case "locked":
		vert, horz := strings.ContainsAny(mask, "ns"), strings.ContainsAny(mask, "ew")
		centre := mask != "ns" && mask != "ew" && (x == 7 || x == 8) && (y == 7 || y == 8)
		along := (vert && (x == 7 || x == 8) && y%4 < 2) || (horz && (y == 7 || y == 8) && x%4 < 2)
		if centre || (along && !edge) {
			return mapInkLt, true
		}
		return color.NRGBA{}, false
	case "open":
		if edge {
			return mapInk, true
		}
		if (x+y)%4 == 0 && (x/4+y/4)%2 == 0 {
			return dirtR[1], true
		}
		return dirtR[2], true
	}
	if edge {
		return rgb(70, 50, 34), true
	}
	if (x%4 == 0) != (y%4 == 0) {
		return stoneR[1], true
	}
	if (x+y)%8 == 2 {
		return tkGold, true
	}
	return stoneR[2], true
}
