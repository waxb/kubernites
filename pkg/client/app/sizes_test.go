package app

import (
	"fmt"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
)

func TestEveryWindowSizeDrawsTheOneLayoutScaledSoItsChecksHoldAtEachSize(t *testing.T) {
	g := &Game{}
	for _, size := range [][2]int{{1280, 800}, {1920, 1080}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			if w, h := g.Layout(size[0], size[1]); w != hud.Width || h != hud.Height {
				t.Fatalf("a %dx%d window lays out %dx%d, not the one %dx%d layout", size[0], size[1], w, h, hud.Width, hud.Height)
			}
			g.Final = FinalState{Fit: FitGeoM(size[0], size[1]), Ok: true}
			for _, corner := range [][2]int{{0, 0}, {hud.Width - 1, hud.Height - 1}} {
				x, y := g.Final.Fit.Apply(float64(corner[0])+.5, float64(corner[1])+.5)
				if lx, ly := g.ToLogical(x, y); lx != corner[0] || ly != corner[1] {
					t.Errorf("the layout's point %v lands back on %d,%d at %dx%d", corner, lx, ly, size[0], size[1])
				}
			}
			s := FitScale(size[0], size[1])
			if w, h := hud.Width*s, hud.Height*s; w > float64(size[0]) || h > float64(size[1]) {
				t.Errorf("the scaled layout %.0fx%.0f spills out of the %dx%d window", w, h, size[0], size[1])
			}
			g.Final = FinalState{}
		})
	}
}
