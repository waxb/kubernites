package art

import (
	"image/color"
	"testing"
)

func countColours(a Atlas, r Region, want map[color.NRGBA]bool) int {
	n := 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if want[a.Image.NRGBAAt(x, y)] {
				n++
			}
		}
	}
	return n
}

func TestTheQoSLadderReadsShackCottageManor(t *testing.T) {
	a := built().Buildings
	for tier := 1; tier <= 3; tier++ {
		shack := contentHeight(a, a.Regions[hearthQoSName(tier, "ready", "vagrant")])
		cottage := contentHeight(a, a.Regions[hearthQoSName(tier, "ready", "freeman")])
		manor := contentHeight(a, a.Regions[hearthQoSName(tier, "ready", "landed")])
		if !(shack < cottage && cottage < manor) {
			t.Errorf("tier %d heights: shack %d, cottage %d, manor %d", tier, shack, cottage, manor)
		}
	}
}

func TestTheBestEffortShackHasPlankWallsADoorAndAPipe(t *testing.T) {
	a := built().Buildings
	walls := map[color.NRGBA]bool{shackWood[1]: true, shackWood[2]: true}
	for tier := 1; tier <= 3; tier++ {
		r := a.Regions[hearthQoSName(tier, "ready", "vagrant")]
		if n := countColours(a, r, walls); n < 40 {
			t.Errorf("tier %d shack shows only %d plank pixels", tier, n)
		}
		if n := countColours(a, r, map[color.NRGBA]bool{shackWood[0]: true}); n < 20 {
			t.Errorf("tier %d shack shows only %d plank seams", tier, n)
		}
		if n := countColours(a, r, map[color.NRGBA]bool{inkSft: true}); n < 6 {
			t.Errorf("tier %d shack has no door opening (%d)", tier, n)
		}
		if n := countColours(a, r, map[color.NRGBA]bool{greyR[1]: true, greyR[2]: true}); n < 6 {
			t.Errorf("tier %d shack has no stovepipe (%d)", tier, n)
		}
	}
}
