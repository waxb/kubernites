package hud

import (
	"testing"
)

func TestFollowingTheGuideToTheStoneOrTheScriptoriumOpensIt(t *testing.T) {
	f := navFrame(t)
	for _, c := range []struct{ surface, overlay string }{{SurfaceStone, OverlayStone}, {SurfaceScriptorium, OverlayScriptorium}} {
		for _, s := range navStarts() {
			h := startHUD(s)
			opened := false
			var clicked []string
			for range MaxNavDepth + 2 {
				path := NavPath(c.surface, h.NavState(f))
				if len(path) == 0 {
					opened = h.OverlayID == c.overlay
					break
				}
				r, ok := h.StepRect(path[0], f)
				if !ok {
					t.Fatalf("%s from %+v: step %s is not on screen", c.surface, s, path[0])
				}
				clicked = append(clicked, path[0])
				h.Update(Input{MX: r.X + r.W/2, MY: r.Y + r.H/2, Click: true}, f)
				h.Update(Input{MX: -50, MY: -50}, f)
			}
			if !opened {
				t.Errorf("%s from %+v: the guide lit %v and the %s never opened", c.surface, s, clicked, c.overlay)
			}
		}
	}
}

func TestAStoneStepWithACommandLightsTheCommand(t *testing.T) {
	f := stoneObjective(t, "kubectl get nodes")
	h := New(nil, (&fakeRunner{}).backend())
	h.TabID = TabKube
	if got := h.chooserGoal(SurfaceStone, f); got != ObjSpeak {
		t.Fatalf("the step says to click its command, the guide aims at %q", got)
	}
	if _, ok := h.StepRect(ObjSpeak, f); !ok {
		t.Fatal("the guide cannot find the command chip on screen")
	}
	h.OverlayID = OverlayStone
	if got := NavPath(h.chooserGoal(SurfaceStone, f), h.NavState(f)); len(got) != 0 {
		t.Fatalf("with the Stone open the guide still points at %v", got)
	}
}

func TestTheGuideAimsAtTheStoneWhileTheCommandChipIsNotDrawn(t *testing.T) {
	f := stoneObjective(t, "kubectl get nodes")
	h := New(nil, (&fakeRunner{}).backend())
	h.TabID = TabKube
	h.OverlayID = OverlayBook
	if h.ObjectiveDrawn(f) || f.SpeakCommand() == "" {
		t.Fatal("the fixture should know the command and hide the task card that draws its chip")
	}
	if got := h.chooserGoal(SurfaceStone, f); got != SurfaceStone {
		t.Fatalf("with no command chip on screen the guide aims at %q, which nothing answers", got)
	}
}
