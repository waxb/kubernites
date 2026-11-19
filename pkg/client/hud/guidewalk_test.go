package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func walkFrame(t *testing.T, target string) Frame {
	t.Helper()
	f := navFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Do it", Target: target}}
	f.Guide.Target = target
	return f
}

func TestTheWalkClicksThroughPagesAndWindows(t *testing.T) {
	for _, target := range []string{"build:keep", "cmd:grimoire", "vault:porter:1.1", "cmd:siege"} {
		f := walkFrame(t, target)
		f.State.Reveals = append(f.State.Reveals, SiegeReveal)
		h := New(nil, (&fakeRunner{}).backend())
		h.OverlayID = OverlayCouncil
		w := h.WalkGuide(f)
		if !w.OK() || w.Goal != target || w.Steps[len(w.Steps)-1] != target {
			t.Errorf("%s: %+v", target, w)
		}
		if h.Overlay() != OverlayCouncil || h.Page() != PageMain {
			t.Errorf("%s: the walk leaves the HUD as it found it", target)
		}
	}
}

func TestTheWalkNamesWhatBlocksTheStep(t *testing.T) {
	f := walkFrame(t, "cmd:build")
	h := New(nil, (&fakeRunner{}).backend())
	b, _ := HitID(h.Buttons(f), "cmd:build")
	h.ToastList = []Toast{{Text: "a toast in the way", Start: 0}}
	if w := h.WalkGuide(f); !w.OK() {
		t.Fatalf("toasts step aside from the guide: %+v", w)
	}
	h.ToastList = nil
	h.Juice.Banner, h.Juice.BannerAt = "A BANNER", h.Now
	if w := h.WalkGuide(f); !w.OK() {
		t.Fatalf("banners step aside from the guide: %+v", w)
	}
	if !OnScreen(b.Rect) || OnScreen(ui.Rect{X: -4, Y: 3, W: 10, H: 10}) || OnScreen(ui.Rect{}) {
		t.Error("onScreen")
	}
}

func TestTheWalkReportsAnUnreachableGoal(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := walkFrame(t, "cmd:nowhere")
	if w := h.WalkGuide(f); w.OK() || !strings.Contains(w.Problem, "nothing on screen") {
		t.Errorf("an unknown command: %+v", w)
	}
	f = walkFrame(t, "pod:default/ghost")
	if w := h.WalkGuide(f); w.OK() || !strings.Contains(w.Problem, "place on the map") {
		t.Errorf("a pod with no place: %+v", w)
	}
	f.GoalAt = &[2]float64{3, 4}
	if w := h.WalkGuide(f); !w.OK() {
		t.Errorf("a pod the camera can glide to: %+v", w)
	}
	f = walkFrame(t, "cmd:age")
	f.State.AgeUp.Available = false
	h.OverlayID = OverlayAge
	if w := h.WalkGuide(f); !w.OK() {
		t.Errorf("already there: %+v", w)
	}
}

func TestTheWalkRefusesADisabledOrHiddenStep(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := walkFrame(t, "build:codex")
	if w := h.WalkGuide(f); w.OK() || !strings.Contains(w.Problem, "disabled") {
		t.Errorf("a disabled build: %+v", w)
	}
	if got := h.ClickProblem(GuideStep{ID: "cmd:build", Rect: ui.Rect{X: 2, Y: 2, W: 4, H: 4}}, "cmd:build", f); !strings.Contains(got, "lies under") {
		t.Errorf("a step drawn under another button: %q", got)
	}
	if got := h.ClickProblem(GuideStep{ID: "surface:x", Rect: ui.Rect{X: 300, Y: 300, W: 4, H: 4}}, "surface:x", f); !strings.Contains(got, "no button") {
		t.Errorf("a middle step that is no button: %q", got)
	}
	h.OverlayID = OverlayGrimoire
	if what, ok := h.CoveredBy(h.OverlayRect().Offset(-5, 0), f); !ok || !strings.Contains(what, "window") {
		t.Errorf("a window over the card covers it: %q %v", what, ok)
	}
}

func TestStepProblemsNameOffScreenAndCoveredSteps(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := walkFrame(t, "cmd:build")
	if got := h.StepProblem(GuideStep{ID: "x", Rect: ui.Rect{X: Width - 2, Y: 3, W: 10, H: 10}}, "x", f); !strings.Contains(got, "off the screen") {
		t.Errorf("off screen: %q", got)
	}
	if got := h.StepProblem(GuideStep{ID: "pod:x", Map: true}, "pod:x", f); got != "" {
		t.Errorf("a beacon on the map: %q", got)
	}
	h.Now = 5
	h.PushToast("covering", false)
	r := h.ToastRects(f)[0]
	if got := h.StepProblem(GuideStep{ID: "x", Rect: r}, "x", f); !strings.Contains(got, "toast") {
		t.Errorf("under a toast: %q", got)
	}
	h.ToastList = nil
	h.Juice.Banner, h.Juice.BannerAt = "A BANNER", h.Now
	b, _ := h.BannerRect(f)
	if got := h.StepProblem(GuideStep{ID: "x", Rect: b}, "x", f); !strings.Contains(got, "banner") {
		t.Errorf("under a banner: %q", got)
	}
}

func TestTheSummonPageKeepsTheFirstPodsCellLit(t *testing.T) {
	f := walkFrame(t, CmdSummonID)
	h := New(nil, (&fakeRunner{}).backend())
	h.PageID = PageSummon
	cells := SoulButtons(f, "summon:", "Summon")
	if len(cells) == 0 {
		t.Fatal("the fixture owns no image")
	}
	if got := h.StepID(f); got != cells[0].ID {
		t.Fatalf("on the summon page the guide points at %q, want %q", got, cells[0].ID)
	}
}

func TestCrewPlacesNeedANamespacedName(t *testing.T) {
	for target, want := range map[string]bool{"crew:default/porters": true, "crew:porters": false, "crew:default/": false, "pod:x": false} {
		if _, ok := CrewPlace(target); ok != want {
			t.Errorf("%s: %v", target, ok)
		}
	}
}
