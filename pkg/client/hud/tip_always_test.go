package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestAHoverTipIsDrawnInItsBestPlaceWhenNoPlaceIsClear(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.WorldBoxes = []ui.Rect{{X: 0, Y: TopH, W: Width, H: ConsoleY - TopH}}
	h.Update(Input{MX: 400, MY: 300}, f)
	tip := "a tip with no place clear of the world's markers"
	h.Hover, h.Hovering = Button{ID: "info:live:listed", Rect: ui.Rect{X: 400, Y: 300, W: 60, H: ui.LineHeight}, Tip: tip}, true
	f = h.Stage(f)
	r, shown := h.TipRect(f)
	if !shown {
		t.Fatal("the hover tip is not drawn where no place is clear")
	}
	if r.X < 0 || r.Y < 0 || r.X+r.W > Width || r.Y+r.H > Height {
		t.Errorf("the tip %+v leaves the view", r)
	}
	found := false
	for _, s := range h.Surfaces(f) {
		found = found || s.Slot == SlotTip && strings.HasPrefix(s.Key, "tooltip:") && s.Crowded
	}
	if !found {
		t.Error("the stage holds no crowded surface for the drawn tip")
	}
	if faults := CheckAttention(AttnLog{Frames: []AttnFrame{{Surfaces: h.Surfaces(f)}}}, AttnRules{OverlapOnly: true}); len(faults) > 0 {
		t.Errorf("a tip with no clear place counts as an overlap: %v", faults)
	}
	var drawn []ui.TextRun
	stop := ui.RecordText(func(run ui.TextRun) { drawn = append(drawn, run) })
	dst := ebiten.NewImage(Width, Height)
	h.drawTooltip(dst, f)
	stop()
	dst.Deallocate()
	inside := false
	for _, run := range drawn {
		inside = inside || strings.Contains(tip, run.Text) && run.Rect.X >= r.X && run.Rect.Y >= r.Y && run.Rect.X < r.X+r.W && run.Rect.Y < r.Y+r.H
	}
	if !inside {
		t.Errorf("the tip's words are not drawn in its place %+v: %+v", r, drawn)
	}

	h.Hovering = false
	f.Tip, f.TipAt = tip, [2]int{400, 300}
	wr, ok := h.WorldTipRect(f)
	if !ok {
		t.Fatal("the world's hover tip is not drawn where no place is clear")
	}
	if wr.X < 0 || wr.Y < TopH || wr.X+wr.W > Width || wr.Y+wr.H > ConsoleY {
		t.Errorf("the world tip %+v leaves the map", wr)
	}
}
