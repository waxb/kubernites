package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	beaconTarget = 10
	beaconMargin = 2
)

type beaconSpot struct {
	sx, sy, zoom float64
}

func beaconUnit(zoom float64) int {
	return max(2, int(zoom+.5)+1)
}

func BeaconRect(sx, sy, zoom float64) ui.Rect {
	k := beaconUnit(zoom)
	lift := beaconHeight(zoom) * zoom
	amp := beaconAmp * zoom
	top := int(sy-lift-amp) - beaconWidth*k/2 - 4*k - beaconMargin
	bottom := int(sy-lift+amp) - beaconWidth*k/2 + 5*k + k/2 + beaconMargin + 1
	half := beaconWidth*k/2 + beaconMargin
	beacon := ui.Rect{X: int(sx) - half, Y: top, W: 2*half + k/2 + 1, H: bottom - top}
	reach := int(math.Ceil(beaconTarget * zoom))
	spot := ui.Rect{X: int(sx) - reach, Y: int(sy) - 2*reach, W: 2 * reach, H: 2*reach + beaconMargin}
	return union(beacon, spot)
}

func (r *Renderer) beaconSpots(cam render.Camera, sc Scene) []beaconSpot {
	out := []beaconSpot{}
	zoom := cam.Scale()
	if sc.Goal != nil && sc.Ghost == nil {
		sx, sy := ScreenPointOf(cam, sc.Site, *sc.Goal)
		out = append(out, beaconSpot{sx, sy, zoom})
	}
	if spot, ok := r.plotSpot(cam, sc); ok {
		out = append(out, spot)
	}
	if front, lift, ok := r.HarborOf().front(); ok && sc.Beckon {
		sx, sy := ScreenPointOf(cam, sc.Site, front)
		out = append(out, beaconSpot{sx, sy - lift*zoom, zoom})
	}
	return out
}

func (r *Renderer) plotSpot(cam render.Camera, sc Scene) (beaconSpot, bool) {
	if sc.Plot == nil {
		return beaconSpot{}, false
	}
	lo, hi := Footprint(GhostStructure(sc.Site, *sc.Plot))
	ground := Vec{float64(lo.X+hi.X)/2 + .5, float64(lo.Y+hi.Y)/2 + .5}
	sx, sy := ScreenPointOf(cam, sc.Site, ground)
	return beaconSpot{sx, sy, cam.Scale()}, true
}

func (r *Renderer) BeaconRects(cam render.Camera, sc Scene) []ui.Rect {
	out := []ui.Rect{}
	for _, b := range r.beaconSpots(cam, sc) {
		out = append(out, BeaconRect(b.sx, b.sy, b.zoom))
	}
	return out
}

func (r *Renderer) DrawBeacons(dst *ebiten.Image, cam render.Camera, sc Scene) {
	for _, b := range r.beaconSpots(cam, sc) {
		drawBeacon(dst, b.sx, b.sy, b.zoom, sc)
	}
}
