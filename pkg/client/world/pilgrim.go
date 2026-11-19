package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	beaconAmp   = 2.5
	beaconHz    = 1.6
	beaconLift  = 24
	bubbleClear = 16
	beaconWidth = 9
)

func offset(v, dir Vec, d float64) Vec {
	return Vec{v.X + dir.X*d, v.Y + dir.Y*d}
}

func BeaconBob(t float64) float64 {
	return beaconAmp * math.Sin(2*math.Pi*beaconHz*t)
}

func beaconHeight(zoom float64) float64 {
	if zoom >= detailZoom {
		return beaconLift + bubbleClear
	}
	return beaconLift
}

func drawBeacon(dst *ebiten.Image, sx, sy, zoom float64, sc Scene) {
	bob := 0.0
	if !sc.Reduce {
		bob = BeaconBob(sc.RealTime())
	}
	k := max(2, int(zoom+.5)+1)
	x, y := int(sx), int(sy-beaconHeight(zoom)*zoom+bob*zoom)-beaconWidth*k/2
	parts := []ui.Rect{{X: x - k*3/2, Y: y - 4*k, W: 3 * k, H: 4 * k}}
	for row := 0; row*2 < beaconWidth; row++ {
		w := (beaconWidth - row*2) * k
		parts = append(parts, ui.Rect{X: x - w/2, Y: y + row*k, W: w, H: k})
	}
	for _, p := range parts {
		ui.Fill(dst, p.Offset(k/2+1, k/2+1), ui.Palette.Shadow)
	}
	for _, p := range parts {
		ui.Fill(dst, p.Inset(-1), ui.Palette.Ink)
	}
	for _, p := range parts {
		ui.Fill(dst, p, ui.Palette.GoldBright)
	}
}
