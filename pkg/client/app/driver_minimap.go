package app

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
)

const MinimapID = "minimap"

var errMinimapAway = errors.New("the minimap is not on screen yet")

func minimapShown(f hud.Frame) bool {
	return f.Site.Width > 0 && f.Shown(hud.SurfaceMinimap)
}

func (d *Driver) minimapTarget(id string, f hud.Frame) (int, int, bool, error) {
	rest, ok := strings.CutPrefix(id, MinimapID)
	if !ok || (rest != "" && rest[0] != ':') {
		return 0, 0, false, nil
	}
	if !minimapShown(f) {
		return 0, 0, true, errMinimapAway
	}
	x, y := hud.Minimap.X+hud.Minimap.W/2, hud.Minimap.Y+hud.Minimap.H/2
	if rest != "" {
		gx, gy, ok := tileOf(rest[1:])
		if !ok {
			return 0, 0, true, fmt.Errorf("%q is not a minimap target: use minimap for its middle or minimap:<x>,<y> for a tile", id)
		}
		fx, fy := hud.GridToMinimap(f.Site, float64(gx), float64(gy))
		x, y = int(fx), int(fy)
	}
	if !hud.Minimap.Inset(5).Contains(x, y) {
		return 0, 0, true, errors.New("that tile is not on the minimap")
	}
	return x, y, true, nil
}

func tileOf(text string) (int, int, bool) {
	xs, ys, ok := strings.Cut(text, ",")
	if !ok {
		return 0, 0, false
	}
	x, errX := strconv.Atoi(xs)
	y, errY := strconv.Atoi(ys)
	return x, y, errX == nil && errY == nil
}

func minimapButtons(f hud.Frame) []LookButton {
	if !minimapShown(f) {
		return nil
	}
	return []LookButton{{ID: MinimapID, Label: "(the minimap: click to jump the view; minimap:<x>,<y> clicks a tile)", Region: "minimap", Enabled: true}}
}
