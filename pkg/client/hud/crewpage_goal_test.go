package hud

import (
	"strings"
	"testing"
)

func modeFrame(t *testing.T) Frame {
	t.Helper()
	f := chooserFrame(t)
	f.Selection = crewSel()
	f.State.Reveals = append(f.State.Reveals, allReveals...)
	f.State.Unlocks = map[string]bool{HornUnlock: true}
	return f
}

func imageStepFrame(t *testing.T) Frame {
	t.Helper()
	f := modeFrame(t)
	f = guided(f, "crew:default/porters", "crew:mode:image")
	f.State.Objectives[0].Command = "kubectl set image deployment/porters porter=porter:1.1"
	return f
}

func TestTheGuideLeadsFromSetImageToTheClassCell(t *testing.T) {
	f := imageStepFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.Update(Input{}, f)
	if got := h.StepID(f); got != "crew:mode:image" {
		t.Fatalf("on the crew's commands the guide points at SET IMAGE, got %q", got)
	}
	clickOn(t, h, f, "crew:mode:image")
	if got := h.StepID(f); got != "crew:image:porter:1.1" {
		t.Fatalf("on the image page the guide points at the class, got %q", got)
	}
	if w := h.WalkGuide(f); !w.OK() || !strings.HasPrefix(w.Steps[len(w.Steps)-1], "crew:image:") {
		t.Fatalf("the walk ends on a class cell: %+v", w)
	}
}

func TestTheGuideNeverPointsBackFromAModePage(t *testing.T) {
	for _, button := range []string{"crew:mode:image", "crew:mode:resources", "crew:mode:horn"} {
		f := guided(modeFrame(t), "crew:default/porters", button)
		h := New(nil, (&fakeRunner{}).backend())
		h.Update(Input{}, f)
		if _, ok := HitID(h.CommandButtons(f), button); !ok {
			continue
		}
		clickOn(t, h, f, button)
		if got := h.StepID(f); got == "" || got == crewBack || got == "cmd:back" {
			t.Errorf("%s: on its own page the guide points at %q", button, got)
		}
	}
}
