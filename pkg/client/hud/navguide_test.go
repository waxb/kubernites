package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func navIntroFrame(t *testing.T, target string) Frame {
	t.Helper()
	f := navFrame(t)
	f.Guide = Guide{Intro: &Intro{ID: "x", Title: "Summon", Try: "Press SUMMON", Target: target, Via: ViaOpen}}
	return f
}

func area(rects []ui.Rect) int {
	total := 0
	for _, r := range rects {
		total += r.W * r.H
	}
	return total
}

func TestDimRectsLeaveAClearHole(t *testing.T) {
	screen := ui.Rect{W: Width, H: Height}
	cases := []ui.Rect{{X: 100, Y: 100, W: 40, H: 20}, {X: -10, Y: -10, W: 40, H: 40}, {X: 940, Y: 520, W: 60, H: 60}, {X: 0, Y: 0, W: Width, H: 10}}
	for _, hole := range cases {
		rects := DimRects(screen, hole, true)
		for _, r := range rects {
			if overlapArea(r, hole) > 0 {
				t.Errorf("dim %+v covers the hole %+v", r, hole)
			}
		}
		if got, want := area(rects)+overlapArea(screen, hole), screen.W*screen.H; got != want {
			t.Errorf("hole %+v: dim plus hole cover %d of %d pixels", hole, got, want)
		}
	}
	if rects := DimRects(screen, ui.Rect{X: 2000, Y: 2000, W: 5, H: 5}, true); len(rects) != 1 || rects[0] != screen {
		t.Errorf("an off-screen hole dims everything: %+v", rects)
	}
	if rects := DimRects(screen, cases[0], false); len(rects) != 1 {
		t.Errorf("no hole: %+v", rects)
	}
}

func assertClearHole(t *testing.T, h *HUD, f Frame, want string) {
	t.Helper()
	s, ok := h.Step(f)
	if !ok || s.ID != want {
		t.Fatalf("step = %+v %v, want %s", s, ok, want)
	}
	hole, ok := h.IntroHole(f)
	if !ok || hole != s.Rect {
		t.Fatalf("the intro's hole %+v is not over the step %+v", hole, s.Rect)
	}
	for _, r := range DimRects(ui.Rect{W: Width, H: Height}, hole.Inset(-HolePad), true) {
		if overlapArea(r, s.Rect) > 0 {
			t.Fatalf("the dim %+v covers the current step %+v", r, s.Rect)
		}
	}
	if Overlaps(h.IntroCard(f), s.Rect) {
		t.Fatalf("the intro card %+v covers the current step %+v", h.IntroCard(f), s.Rect)
	}
}

func TestIntroHoleFollowsTheNavigationPath(t *testing.T) {
	f := navIntroFrame(t, "cmd:summon")
	h := New(nil, (&fakeRunner{}).backend())
	h.PageID = PageBuild
	assertClearHole(t, h, f, "cmd:back")
	clickOn(t, h, f, "cmd:back")
	assertClearHole(t, h, f, "cmd:summon")
}

func TestIntroShowsOverAWindowAndPointsAtItsClose(t *testing.T) {
	f := navIntroFrame(t, "cmd:summon")
	for _, o := range []string{OverlayGrimoire, OverlayCouncil, OverlayMenu, OverlayStone, OverlayLedger, OverlayAge, OverlaySiege, OverlayReforge} {
		h := New(nil, (&fakeRunner{}).backend())
		h.OverlayID = o
		if !h.IntroShown(f) {
			t.Fatalf("%s hides the intro", o)
		}
		want := closeID(o)
		assertClearHole(t, h, f, want)
		x, y := h.introHole0(f)
		if b, ok := HitButton(h.Buttons(f), x, y); !ok || b.ID != want {
			t.Fatalf("%s: clicking the hole hits %q, want %q", o, b.ID, want)
		}
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.OverlayID = OverlayWelcome
	if h.IntroShown(f) {
		t.Fatal("the Welcome keeps the intro back")
	}
}

func (h *HUD) introHole0(f Frame) (int, int) {
	r, _ := h.IntroHole(f)
	return r.X + r.W/2, r.Y + r.H/2
}

func TestObjectiveBeaconWalksThePath(t *testing.T) {
	f := navFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "raise-keep", Progress: "ready: press BUILD"}}
	h := New(nil, (&fakeRunner{}).backend())
	h.OverlayID = OverlayGrimoire
	if got := h.StepID(f); got != "overlay:close" {
		t.Fatalf("under the manual the step is %q", got)
	}
	clickOn(t, h, f, "overlay:close")
	if got := h.StepID(f); got != "cmd:build" {
		t.Fatalf("then %q", got)
	}
	clickOn(t, h, f, "cmd:build")
	if got := h.StepID(f); got != "build:keep" {
		t.Fatalf("then %q", got)
	}
	targets := h.GuideTargets(f)
	if len(targets) == 0 || !targets[len(targets)-1].Contains(CellRect(0).X+2, CellRect(0).Y+2) {
		t.Fatalf("guide targets %+v miss the Keep cell", targets)
	}
}

func TestMapStepsWaitForTheWindowToClose(t *testing.T) {
	f := navFrame(t)
	f.Guide.Target = "pod:default/porter-a"
	f.Beacons = []ui.Rect{{X: 300, Y: 200, W: 20, H: 30}}
	h := New(nil, (&fakeRunner{}).backend())
	h.OverlayID = OverlayCouncil
	if h.MapStep(f) || h.StepID(f) != "overlay:close" {
		t.Fatalf("with the Council open the step is %q", h.StepID(f))
	}
	h.OverlayID = OverlayNone
	s, ok := h.Step(f)
	if !ok || !s.Map || s.Rect != f.Beacons[0] || !h.MapStep(f) || h.StepID(f) != "" {
		t.Fatalf("map step = %+v %v", s, ok)
	}
}

func TestTheGuideDrawsAboveEveryLayerButTheHoverTips(t *testing.T) {
	last := []string{}
	for _, l := range HudLayers[len(HudLayers)-3:] {
		last = append(last, l.Name)
	}
	if !slices.Equal(last, []string{GuideLayer, "tooltip", "worldtip"}) {
		t.Fatalf("the last HUD layers are %v, want the guide under the hover tips", last)
	}
	names := map[string]bool{}
	for _, l := range HudLayers {
		if names[l.Name] {
			t.Fatalf("layer %q twice", l.Name)
		}
		names[l.Name] = true
	}
	for _, need := range []string{"overlay", "intro", "tooltip", "toasts"} {
		if !names[need] {
			t.Fatalf("no %s layer", need)
		}
	}
}

func TestReforgeStepsStayInsideItsBox(t *testing.T) {
	for _, target := range []string{"cmd:build", "age:advance", "reforge:confirm", MeterGold, "pod:x"} {
		f := navFrame(t)
		f.Guide.Target = target
		f.State.Objectives = []view.ObjectiveView{{ID: "o1"}}
		h := New(nil, (&fakeRunner{}).backend())
		h.OverlayID = OverlayReforge
		s, ok := h.Step(f)
		if !ok || !ReforgeBox.Inset(-1).Contains(s.Rect.X, s.Rect.Y) || !ReforgeBox.Inset(-1).Contains(s.Rect.X+s.Rect.W-1, s.Rect.Y+s.Rect.H-1) {
			t.Errorf("%s: the step %+v under the rebuild panel leaves its box %+v", target, s, ReforgeBox)
		}
	}
}
