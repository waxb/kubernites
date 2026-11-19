package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type readyCase struct {
	progress, target, build string
}

var readySteps = []readyCase{
	{"ready: press BUILD", "", "keep"},
	{"ready: press CREW", CmdCrew, ""},
	{"ready: read about The Shift in the MANUAL", "cmd:grimoire", ""},
	{"ready: pull porter:1.1 from the REGISTRY", "cmd:soulvault", ""},
	{"ready: press PANEL and advance", "cmd:age", ""},
}

var readyPlaces = map[string]func(*HUD, *Frame){
	"the main grid":   func(*HUD, *Frame) {},
	"the BUILD menu":  func(h *HUD, _ *Frame) { h.PageID = PageBuild },
	"the SUMMON menu": func(h *HUD, _ *Frame) { h.PageID = PageSummon },
	"the MORE menu":   func(h *HUD, _ *Frame) { h.PageID = PageMore },
	"a Keep's card": func(_ *HUD, f *Frame) {
		f.Selection = Selection{Kind: "structure", ID: f.Site.Structures[0].ID, Structure: f.Site.Structures[0]}
	},
	"a hut's card": func(_ *HUD, f *Frame) {
		f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}
	},
	"the incident log":   func(h *HUD, _ *Frame) { h.OverlayID = OverlayChronicles },
	"the harbor's board": func(_ *HUD, f *Frame) { f.Selection = Selection{Kind: SelHaven, ID: "haven-0"} },
}

func readyLine(h *HUD, f Frame) string {
	for _, l := range h.ObjectiveLines(f) {
		if strings.HasPrefix(l.Text, readyMark) {
			return l.Text
		}
	}
	return ""
}

func TestReadyLineReachableFromEveryMenu(t *testing.T) {
	for _, step := range readySteps {
		for place, put := range readyPlaces {
			f := navFrame(t)
			f.State.Reveals = append(f.State.Reveals, "ageup", "chronicles", "soulvault", "grimoire")
			f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Do it", Progress: step.progress, Target: step.target, Build: step.build}}
			f.Guide.Target = step.target
			if step.build != "" {
				f.Guide.Target = "build:" + step.build
			}
			h := New(nil, (&fakeRunner{}).backend())
			h.SetStaging(false)
			put(h, &f)
			path := NavPath(h.readyGoal(f), h.NavState(f))
			if len(path) == 0 {
				continue
			}
			line := readyLine(h, f)
			_, back := backLabels[path[0]]
			back = back || strings.HasSuffix(path[0], ":close")
			named := map[bool]string{true: "press BACK, then", false: "X on th"}[path[0] == backStep]
			if path[0] == SelClose && !strings.Contains(line, closeLead(h.CardFor(f).Title)) {
				t.Errorf("%s in %s: %q does not name which card to close among several", step.progress, place, line)
			}
			if back && !strings.Contains(line, named) {
				t.Errorf("%s in %s: the line reads %q, but its first press is %s", step.progress, place, line, path[0])
			}
			inBuild := place == "the BUILD menu" && strings.HasPrefix(step.progress, "ready: press BUILD")
			if !back && !inBuild && line != f.QuoteObjective(step.progress) {
				t.Errorf("%s in %s: the line reads %q, want it as the game says it", step.progress, place, line)
			}
		}
	}
}

func TestAReadyLineToAPlaceOutOfViewStartsWithShowMe(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "augury", Text: "Read the Pod logs", Progress: "ready: click the Watchtower, then TOP PODS", Target: "structure:observatory-13"}}
	f.Guide.Target = "structure:observatory-13"
	f.GoalAt = &[2]float64{60, 60}
	f.Beacons = []ui.Rect{{X: Width + 40, Y: 200, W: 20, H: 40}}
	if line := readyLine(h, f); !strings.HasPrefix(line, "ready: press SHOW ME, then click the Watchtower") {
		t.Fatalf("the line names a place the player cannot see: %q", line)
	}
	f.Beacons = []ui.Rect{{X: 300, Y: 200, W: 20, H: 40}}
	if line := readyLine(h, f); line != f.QuoteObjective("ready: click the Watchtower, then TOP PODS") {
		t.Fatalf("a place in view still asks for SHOW ME: %q", line)
	}
}

func TestInsideTheBuildMenuTheReadyLineNamesTheBuildingNotBuild(t *testing.T) {
	for _, progress := range []string{"ready: press BUILD", "ready: press BUILD, then KEEP"} {
		f := navFrame(t)
		f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Raise a Keep", Progress: progress, Target: "cmd:build", Build: "keep"}}
		f.Guide.Target = "build:keep"
		h := New(nil, (&fakeRunner{}).backend())
		h.SetStaging(false)
		h.PageID = PageBuild
		line := readyLine(h, f)
		if strings.Contains(line, "press BUILD") || !strings.Contains(line, "press KEEP") {
			t.Errorf("%q with the BUILD menu open reads %q: BUILD is not on screen", progress, line)
		}
	}
}

func TestGotItOnACardWhoseButtonIsOnTheHomeGridKeepsThatGrid(t *testing.T) {
	f := crewFrame(crewSel(), "soulvault")
	f.Guide.Intro = &Intro{ID: "academy", Title: "The registry", Try: "Press " + view.ButtonToken + " to see its classes.", Target: "cmd:soulvault", Via: ViaOpen}
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	if !buttonIDs(h.CommandButtons(f))["cmd:soulvault"] {
		t.Fatal("the registry card points at a grid without REGISTRY")
	}
	h.PressIntro("ack", f)
	f.Guide.Intro = nil
	if ids := buttonIDs(h.CommandButtons(f)); !ids["cmd:soulvault"] {
		t.Fatalf("after GOT IT the shift card's grid hides the REGISTRY the card just named: %v", ids)
	}
}
