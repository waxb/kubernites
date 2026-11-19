package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestAnAlertCountBadgeStaysOnTheScreen(t *testing.T) {
	h := New(nil, Backend{})
	for _, r := range []ui.Rect{{X: Width - 20, Y: 200, W: 18, H: 18}, {X: 0, Y: 200, W: 18, H: 18}} {
		b := h.countBadgeRect(5, r)
		if b.X < 0 || b.X+b.W > Width {
			t.Errorf("the count badge of an arrow at %+v spans %+v, past the screen's edge (berths: \"x5 ALER\")", r, b)
		}
	}
}
