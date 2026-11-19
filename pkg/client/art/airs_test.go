package art

import (
	"image/color"
	"testing"
)

func probeSpec(spec roleSpec) roleSpec {
	spec.skin = ramp{rgb(1, 2, 3), rgb(4, 5, 6), rgb(7, 8, 9), rgb(10, 11, 12)}
	spec.hair = rgb(spec.hair.R, 14, 15)
	return spec
}

func TestSkinAndHairAreTheOnlyPixelsInTheirRamps(t *testing.T) {
	table := map[color.NRGBA]bool{}
	for _, c := range AirColours() {
		table[c] = true
	}
	for _, role := range AiredRoles {
		for v := 0; v < WalkerLooks*WalkerTiers(role); v++ {
			spec := lookSpec(role, v%WalkerLooks, 1+v/WalkerLooks)
			for _, action := range append(append([]string(nil), WalkerActions...), HoldAction) {
				for frame := 0; frame < WalkerFrames; frame++ {
					plain, probe := newCanvas(walkerFrameW, walkerFrameH), newCanvas(walkerFrameW, walkerFrameH)
					paintPerson(plain, spec, action, frame)
					paintPerson(probe, probeSpec(spec), action, frame)
					for y := 0; y < walkerFrameH; y++ {
						for x := 0; x < walkerFrameW; x++ {
							a, b := plain.img.NRGBAAt(x, y), probe.img.NRGBAAt(x, y)
							if a.A < 255 {
								continue
							}
							switch {
							case a != b && !table[a]:
								t.Fatalf("%s look %d %s/%d: skin or hair pixel %v at %d,%d is off the ramps", role, v, action, frame, a, x, y)
							case a == b && table[a]:
								t.Fatalf("%s look %d %s/%d: pixel %v at %d,%d shares a skin or hair colour", role, v, action, frame, a, x, y)
							}
						}
					}
				}
			}
		}
	}
}
