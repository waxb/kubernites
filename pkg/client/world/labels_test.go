package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestALabelSlidesClearOfAPanelOverItsTail(t *testing.T) {
	box := ui.Rect{X: 100, Y: 50, W: 120, H: 12}
	card := ui.Rect{X: 200, Y: 30, W: 300, H: 200}
	got, ok := SettleLabel(box, 150, []ui.Rect{card})
	if !ok || Overlaps(got, card) || got.Y != box.Y || got.W != box.W || got.X > 150 || got.X+got.W < 150 {
		t.Fatalf("label %+v -> %+v (%v): it must clear the card and still sit over x=150", box, got, ok)
	}
	if got.X+got.W != card.X-labelGap {
		t.Errorf("the label slides only as far as it must: %+v", got)
	}
}

func TestALabelWhoseAnchorIsUnderAPanelHides(t *testing.T) {
	box := ui.Rect{X: 190, Y: 50, W: 120, H: 12}
	if got, ok := SettleLabel(box, 250, []ui.Rect{{X: 200, Y: 30, W: 300, H: 200}}); ok {
		t.Errorf("a label over a covered anchor shows at %+v", got)
	}
	if _, ok := SettleLabel(ui.Rect{X: 300, Y: 10, W: 80, H: 12}, 340, []ui.Rect{{X: 0, Y: 0, W: 960, H: 18}}); ok {
		t.Error("a label half under the top bar shows cut")
	}
}

func TestAClearLabelStaysPut(t *testing.T) {
	box := ui.Rect{X: 10, Y: 50, W: 120, H: 12}
	if got, ok := SettleLabel(box, 70, []ui.Rect{{X: 200, Y: 30, W: 300, H: 200}}); !ok || got != box {
		t.Errorf("a clear label moved to %+v (%v)", got, ok)
	}
}

func TestTheSignpostLabelStepsOutFromUnderACard(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Pods = append(sc.Pods, campers(12)...)
	r := NewRenderer(gfx.Load(art.Build()))
	r.HearthItems(cam, sc)
	sx, _, box := signpostBox(cam, sc, r.post)
	if box, ok := r.signpostLabel(cam, sc); !ok || box.W == 0 {
		t.Fatalf("no panels, no label: %+v", box)
	}
	card := ui.Rect{X: int(sx) + 20, Y: box.Y - 40, W: 300, H: 200}
	sc.Panels = []ui.Rect{card}
	moved, ok := r.signpostLabel(cam, sc)
	if !ok || Overlaps(moved, card) || moved.X > int(sx) || moved.X+moved.W < int(sx) {
		t.Fatalf("label %+v under card %+v moved to %+v (%v)", box, card, moved, ok)
	}
	for _, it := range r.HearthItems(cam, sc) {
		if it.Hit != nil && it.Hit.Kind == HitSignpost && !inside(moved, it.Hit.Rect) {
			t.Errorf("a click on the moved label %+v misses the signpost %+v", moved, it.Hit.Rect)
		}
	}
	sc.Panels = []ui.Rect{{X: int(sx) - 5, Y: box.Y - 40, W: 300, H: 200}}
	if got, ok := r.signpostLabel(cam, sc); ok {
		t.Errorf("the signpost is under the card, yet its label shows at %+v", got)
	}
}

func worldRuns(draw func()) []ui.TextRun {
	var runs []ui.TextRun
	stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
	defer stop()
	draw()
	return runs
}

func inside(r, b ui.Rect) bool {
	return r.X >= b.X && r.Y >= b.Y && r.X+r.W <= b.X+b.W && r.Y+r.H <= b.Y+b.H
}

func TestNoWorldLabelIsCutByAPanel(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Pods = append(sc.Pods, campers(12)...)
	sc.Causes = []view.CauseTag{{Pod: "default/camp-00", Tag: view.TagUnhoused, Text: "no room"},
		{Pod: "default/camp-01", Tag: view.TagThrottled, Node: "lone-tower", Text: "slow"}}
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	docked := []ui.Rect{{X: 0, Y: 0, W: ViewWidth, H: 18}, {X: 0, Y: 408, W: ViewWidth, H: 132}}
	seen := 0
	for x := -200; x < ViewWidth; x += 37 {
		for y := 0; y < 400; y += 53 {
			sc.Panels = append([]ui.Rect{{X: x, Y: y, W: 300, H: 120}}, docked...)
			runs := worldRuns(func() { r.Draw(dst, cam, sc); r.DrawCauseTags(dst, cam, sc) })
			seen += len(runs)
			for _, run := range runs {
				for _, p := range sc.Panels {
					if Overlaps(run.Rect, p) && !inside(run.Rect, p) {
						t.Errorf("%q at %+v is cut by the panel %+v", run.Text, run.Rect, p)
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("the scene drew no labels")
	}
}

func TestACauseTagNeverCoversTheSignpostLabel(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Pods = append(sc.Pods, campers(12)...)
	sc.Causes = []view.CauseTag{{Pod: "default/camp-11", Tag: view.TagUnhoused, Text: "no room"},
		{Pod: "default/camp-00", Tag: view.TagOOMKilled, Text: "oom"}}
	r := NewRenderer(gfx.Load(art.Build()))
	r.HearthItems(cam, sc)
	post, ok := r.signpostLabel(cam, sc)
	if !ok {
		t.Fatal("no signpost label")
	}
	marks := r.CauseMarks(cam, sc)
	for _, m := range marks {
		if Overlaps(m.Rect, post) {
			t.Errorf("%s at %+v covers the signpost label %+v", m.Label, m.Rect, post)
		}
	}
	if len(marks) == 0 {
		t.Error("every tag hid")
	}
}

func TestTheCauseTipsFollowTheTagsAsDrawn(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Pods = append(sc.Pods, campers(12)...)
	sc.Causes = []view.CauseTag{{Pod: "default/camp-00", Tag: view.TagOOMKilled, Text: "oom"}}
	r := NewRenderer(gfx.Load(art.Build()))
	if len(r.DrawnCauseMarks()) != 0 {
		t.Fatal("marks before any frame")
	}
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	home := r.CauseMarks(cam, sc)
	if len(home) != 1 {
		t.Fatalf("marks = %+v", home)
	}
	sc.Panels = []ui.Rect{{X: home[0].Tail + 2, Y: home[0].Rect.Y - 20, W: 300, H: 60}}
	r.DrawCauseTags(dst, cam, sc)
	drawn := r.DrawnCauseMarks()
	if len(drawn) != 1 || drawn[0].Rect == home[0].Rect || Overlaps(drawn[0].Rect, sc.Panels[0]) {
		t.Errorf("the tip sits at %+v, the tag was drawn clear of %+v (home %+v)", drawn, sc.Panels[0], home[0].Rect)
	}
}

func TestALabelNeverLeavesTheScreen(t *testing.T) {
	for _, box := range []ui.Rect{{X: 900, Y: 200, W: 80, H: 12}, {X: -30, Y: 200, W: 80, H: 12}, {X: 930, Y: 200, W: 80, H: 12}} {
		got, ok := SettleLabel(box, box.X+box.W/2, []ui.Rect{{X: 400, Y: 30, W: 100, H: 100}})
		if !ok || got.X < 0 || got.X+got.W > ViewWidth || got.W != box.W || got.Y != box.Y {
			t.Errorf("label %+v -> %+v (%v): it must stay inside the %dpx screen", box, got, ok, ViewWidth)
		}
	}
}

func anyOverlap(boxes []ui.Rect) (int, int, bool) {
	for i := range boxes {
		for j := i + 1; j < len(boxes); j++ {
			if Overlaps(boxes[i], boxes[j]) {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}

func TestLabelsStackAboveEachOtherAndStayNearTheirAnchor(t *testing.T) {
	placed := []ui.Rect{}
	for i := 0; i < 4; i++ {
		box := ui.Rect{X: 300, Y: 200, W: 120, H: 12}
		got, ok := StackLabel(box, 360, nil, placed)
		if !ok || Abs(got.Y-box.Y) > 5*(box.H+labelGap) || got.X > 360 || got.X+got.W < 360 {
			t.Fatalf("label %d = %+v (%v)", i, got, ok)
		}
		placed = append(placed, got)
	}
	if i, j, bad := anyOverlap(placed); bad {
		t.Errorf("labels %d and %d overlap: %+v", i, j, placed)
	}
}

func TestAStackedLabelStillClearsPanelsAndTheScreen(t *testing.T) {
	panel := ui.Rect{X: 0, Y: 0, W: 960, H: 20}
	placed := []ui.Rect{{X: 300, Y: 22, W: 120, H: 12}}
	got, ok := StackLabel(ui.Rect{X: 300, Y: 22, W: 120, H: 12}, 360, []ui.Rect{panel}, placed)
	if !ok || Overlaps(got, panel) || Overlaps(got, placed[0]) || got.Y+got.H > ViewHeight {
		t.Errorf("stack = %+v (%v)", got, ok)
	}
}

func TestRolloutAndLossLabelsAtOnePortNeverOverlap(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	kinds := append([]view.KindView(nil), sc.Freight.Kinds...)
	for i := range kinds {
		kinds[i].ArrivalRate, kinds[i].Value, kinds[i].Losses = 3.2, 12.5, view.LossView{NoDepot: 3.2}
	}
	sc.Freight.Kinds = kinds
	sc.Hint = true
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	if len(r.placed) < 3 {
		t.Fatalf("expected the loss labels and the hint to be placed: %+v", r.placed)
	}
	if i, j, bad := anyOverlap(r.placed); bad {
		t.Errorf("labels %d (%+v) and %d (%+v) overlap", i, r.placed[i], j, r.placed[j])
	}
}

func TestALensLabelAtTheScreenEdgeStaysWhole(t *testing.T) {
	for _, sx := range []float64{5, 480, 950, 959} {
		if box := lensLabelBox(sx, 300, 130); box.X < 0 || box.X+box.W > ViewWidth {
			t.Errorf("the label at x=%v leaves the screen: %+v", sx, box)
		}
	}
}
