package hud

import (
	"strings"
	"testing"
)

func TestTheKnobFollowsTheDragAndIgnoresAHover(t *testing.T) {
	f := boardFrame()
	h, v := openBoard(t, f)
	h.Board.Drag, h.Board.Pick, h.Board.Top = DragKnob, 5, v.Top
	v.Target = 5
	if got := h.Board.follow(v, h.Board.clock+.016, false).shown; got != 5 {
		t.Fatalf("while dragging the knob lags at %v", got)
	}
	h.Board.Drag = ""
	v.Target = v.Chain.Desired + 1
	start := h.Board.shown
	if got := h.Board.follow(v, h.Board.clock+.5, false).shown; got != start && got != float64(v.Chain.Desired) {
		t.Fatalf("hovering +1 moves the knob from %v to %v", start, got)
	}
}

func TestTheRedMarkSaysWhatItMeans(t *testing.T) {
	f := boardFrame()
	_, v := openBoard(t, f)
	if v.Wall() >= v.Top {
		t.Skip("no red mark in this frame")
	}
	if tip := sliderTip(v); !strings.Contains(tip, "Pending") {
		t.Fatalf("slider tip = %q", tip)
	}
}
