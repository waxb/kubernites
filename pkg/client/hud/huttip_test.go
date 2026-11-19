package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAHutsHoverCardIsAtMostSixAlignedLinesInBothModes(t *testing.T) {
	h := New(nil, Backend{})
	p := model.PodView{Name: "porter-2v44t", Image: "porter:1.0", Node: "lone-tower", State: model.HearthReady,
		Requests: []model.ContainerRequest{{CPUMilli: 200, MemoryBytes: 64 << 20}}}
	m := view.PodMetrics{CPUMilli: 40, MemoryBytes: 40 << 20, ServedRate: .4}
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		} else {
			ui.SetNamer(nil)
		}
		lines, _ := h.layoutTip(HutTip(p, m, "", ""))
		if len(lines) > 6 {
			t.Errorf("[%s] %d lines: %q", mode, len(lines), lines)
		}
		if !strings.HasPrefix(lines[0], "porter-2v44t - ") || !strings.Contains(lines[0], ui.Bridge("Burning: up and ready", "Running")) {
			t.Errorf("[%s] title %q", mode, lines[0])
		}
		col := strings.Index(lines[1], "porter:1.0")
		for _, l := range lines[1:4] {
			if len(l) <= col || l[col-1] != ' ' {
				t.Errorf("[%s] the value column is not aligned: %q", mode, l)
			}
		}
	}
	ui.SetNamer(nil)
}

func TestATroubledHutsHoverCardCarriesWhyAndFixLines(t *testing.T) {
	h := New(nil, Backend{})
	p := model.PodView{Name: "porter-x", Image: "porter:2.1", Node: "stone-keep", State: model.HearthGuttering}
	lines, styles := h.layoutTip(HutTip(p, view.PodMetrics{}, "it crashes at every start", "Roll the porters back"))
	why, fix := false, false
	for i, l := range lines {
		why = why || strings.HasPrefix(l, "Why: ") && styles[i].clr == ui.Palette.Gold
		fix = fix || strings.HasPrefix(l, "Fix: ") && styles[i].clr == ui.Palette.Gold
	}
	if !why || !fix {
		t.Errorf("no highlighted Why and Fix lines: %q", lines)
	}
}

func TestAHutsHoverCardStaysInsideTheViewWhateverEdgeTheHutIsOn(t *testing.T) {
	h := New(nil, Backend{})
	p := model.PodView{Name: "porters-5dfdd7746f-r269r", Image: "porter:2.1", Node: "stone-keep-2", State: model.HearthGuttering,
		Requests: []model.ContainerRequest{{CPUMilli: 200, MemoryBytes: 64 << 20}}}
	tip := HutTip(p, view.PodMetrics{}, "it crashes at every start", "Roll the porters back to porter:1.0")
	lines, _ := h.layoutTip(tip)
	widest := 0
	for _, l := range lines {
		widest = max(widest, Literal.Width(l, 1))
	}
	f := fullFrame(t)
	spots := map[string]ui.Rect{
		"left": {X: 0, Y: 200, W: 16, H: 16}, "right": {X: Width - 16, Y: 200, W: 16, H: 16},
		"top": {X: 400, Y: TopH, W: 16, H: 16}, "bottom": {X: 400, Y: ConsoleY - 16, W: 16, H: 16},
		"top right": {X: Width - 16, Y: TopH, W: 16, H: 16}, "bottom right": {X: Width - 16, Y: ConsoleY - 16, W: 16, H: 16},
	}
	shown := 0
	for name, at := range spots {
		h.Hover, h.Hovering = Button{ID: "hut:" + p.Name, Rect: at, Tip: tip}, true
		r := h.TooltipRect(lines, f)
		if r.X < 0 || r.Y < 0 || r.X+r.W > Width || r.Y+r.H > Height {
			t.Errorf("%s: the card %+v leaves the %dx%d view", name, r, Width, Height)
		}
		if r.W < widest+10 {
			t.Errorf("%s: the card is %d wide for a %d wide line", name, r.W, widest)
		}
		world := f
		world.Tip, world.TipAt = tip, [2]int{at.X + 8, at.Y + 8}
		wr, ok := h.WorldTipRect(world)
		if ok {
			shown++
		}
		if ok && (wr.X < 0 || wr.Y < 0 || wr.X+wr.W > Width || wr.Y+wr.H > Height || wr.W < widest+10) {
			t.Errorf("%s: the world card %+v is cut (widest line %d)", name, wr, widest)
		}
	}
	if shown != len(spots) {
		t.Errorf("the world card showed at only %d of %d edges: a tip is always drawn", shown, len(spots))
	}
}
