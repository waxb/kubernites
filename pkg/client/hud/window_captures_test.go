package hud

import "testing"

func TestEveryOpenPanelCapturesThePointer(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	r, ok := WorkerWindow(h, f)
	if !ok {
		t.Fatal("the worker's panel is not open")
	}
	x, y := r.X+r.W/2, r.Y+3
	if !h.Captures(x, y, f) {
		t.Errorf("the worker's panel at %+v does not capture the pointer at %d,%d: the world behind it takes hovers, and a click there drags the panel while the novice meant the hut behind it (novice seed 11 at 3:02: porter-sxwvx's hut under the worker panel of porter-g7vsw, enrolled only at 5:38)", r, x, y)
	}
}
