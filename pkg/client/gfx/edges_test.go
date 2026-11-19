package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestLibraryEdges(t *testing.T) {
	var nilLib *Library
	if _, _, ok := nilLib.Lookup("x"); ok {
		t.Error("nil library looks nothing up")
	}
	if got := PortraitScale(96, 48, 48, 48); got != 0.5 {
		t.Errorf("oversized portrait scale = %v", got)
	}
	if got := PortraitScale(0, 0, 48, 48); got != 48 {
		t.Errorf("empty portrait scale = %v", got)
	}
	parts := NineParts(art.Region{X: 0, Y: 0, W: 10, H: 10, Slice: 5}, ui.Rect{W: 10, H: 10})
	if len(parts) != 4 {
		t.Errorf("a box exactly twice the slice has 4 parts, got %d", len(parts))
	}
}
