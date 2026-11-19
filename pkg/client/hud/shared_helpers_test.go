package hud

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"testing"
)

func calamityCase(t *testing.T) (*HUD, Frame) {
	h := New(nil, Backend{})
	f := settled(h, calamityFrame(t, noDepot, oom))
	return h, f
}

func containsRect(outer, inner ui.Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y && inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}
