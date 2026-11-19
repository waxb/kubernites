package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func darkBellFrame() Frame {
	f := watchFrame("livenessProbe")
	f.State.Age = view.AgeDark
	f.State.Grimoire = []view.GrimoirePage{{ID: view.BellPage, Ranks: []view.RankView{{Runes: []string{"livenessProbe"}, Learned: true}}}}
	return f
}

func TestTheBellAloneOpensWatchInTheDarkCrewWindow(t *testing.T) {
	f := darkBellFrame()
	h := New(nil, (&fakeRunner{}).backend())
	if ids := slotIDs(h.crewSlots(f)); !slices.Contains(ids, view.WatchButton) {
		t.Fatalf("with the Bell learned the Dark crew window has no WATCH: %v", ids)
	}
	if tip := watchOrder(f).Tip; strings.Contains(tip, "Shutters") || !strings.Contains(tip, "Watchman's Bell") {
		t.Fatalf("WATCH promises what the first view cannot know: %q", tip)
	}
	h.pressCrew("probes", f)
	ids := slotIDs(h.crewSlots(f))
	if !slices.Contains(ids, "crew:probes:liveness") || slices.Contains(ids, "crew:probes:readiness") {
		t.Fatalf("the Dark WATCH shows %v", ids)
	}
	if legend := crewLegend(f.Crews[0], CrewModeProbes, f); legend != "WATCH: the Bell on every hut of the crew" {
		t.Fatalf("the Dark WATCH legend reads %q", legend)
	}
	f.State.Watch.Crews[0].Delay = 4
	it, ok := hasIntent(h.pressCrew("probes:liveness", f), IntentAction)
	if !ok || it.Action.Crew.Probe != view.ProbeLiveness || it.Action.Crew.Off || it.Action.Crew.Timeout < view.WatchPatience || it.Action.Crew.Delay != 4 {
		t.Fatalf("BELL sent %+v: the crew window's bell must wait out the warm-up as the game's offers do", it.Action)
	}
}

func TestTheDarkClassBannerCreditsTheRollOut(t *testing.T) {
	f := fullFrame(t)
	f.State.Age = view.AgeDark
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "porter:2.0", Owned: true, Visible: true, Description: "Twice the parcels of 1.0 for the same food."})
	done := model.RolloutDone{Crew: "porters", Image: "porter:2.0", From: "porter:1.1", Huts: 12, Gain: .5}
	if _, cause := RolloutBannerIn(f, done); !strings.Contains(cause, "rebuilt a few at a time while the rest carried: the crew's roll-out") || !strings.Contains(cause, "Twice the parcels") {
		t.Fatalf("the class 2.0 banner reads %q", cause)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if _, cause := RolloutBannerIn(f, done); !strings.Contains(cause, "a rolling update") {
		t.Fatalf("in NAMES mode the class 2.0 banner reads %q", cause)
	}
	ui.SetNamer(nil)
	f.State.Age = view.AgeFeudal
	if _, cause := RolloutBannerIn(f, done); strings.Contains(cause, "the crew's roll-out") {
		t.Fatalf("the first view's credit leaks into the second view's banner: %q", cause)
	}
}

func TestTheGuideLightsTheBellOnceWatchIsOpen(t *testing.T) {
	f := darkBellFrame()
	f.Guide.Button = view.WatchButton
	h := New(nil, (&fakeRunner{}).backend())
	if got := h.chooserGoal(view.WatchButton, f); got != view.WatchButton {
		t.Fatalf("before WATCH is open the guide lights %q", got)
	}
	h.pressCrew("probes", f)
	if got := h.chooserGoal(view.WatchButton, f); got != "crew:probes:liveness" {
		t.Fatalf("with WATCH open the guide lights %q, so it walks BACK and loops", got)
	}
	both := watchFrame("readinessProbe", "livenessProbe")
	both.State.Watch.Crews[0].Readiness = &view.ProbeSetting{Path: "/ready"}
	h.pressCrew("probes", both)
	if got := h.chooserGoal(view.WatchButton, both); got != "crew:probes:liveness" {
		t.Fatalf("with the Shutters on, the guide lights %q, not the Bell still off", got)
	}
}

func TestTheWatchSwitchIsReachedInsideWatchNotByWalkingBack(t *testing.T) {
	inWatch := NavState{Object: true, Crew: true, Chain: true, Mode: CrewModeProbes}
	if got := NavPath("crew:probes:liveness", inWatch); len(got) != 1 || got[0] != "crew:probes:liveness" {
		t.Fatalf("with WATCH open the path to BELL is %v", got)
	}
	onChain := NavState{Object: true, Crew: true, Chain: true, Mode: CrewModeChain}
	if got := NavPath("crew:probes:liveness", onChain); len(got) != 2 || got[0] != view.WatchButton {
		t.Fatalf("from the crew's chain the path to BELL is %v, want WATCH first", got)
	}
}

func TestTheBellPathReadsTheRealButtonsInBothNamings(t *testing.T) {
	f := darkBellFrame()
	h := New(nil, (&fakeRunner{}).backend())
	f.Selection = Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")}
	if got := NavPath(view.WatchButton, h.NavState(f)); len(got) == 0 || got[0] != ChainButton {
		t.Fatalf("from a porter hut's card the guide walks %v, want CREW first", got)
	}
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		path := QuoteButton("CREW, "+ButtonToken+" and BELL", view.WatchButton)
		chain, _ := h.chainCell(f)
		bell := ui.Bridge(probeKinds[1].story, probeKinds[1].k8s)
		want := chain.Label + ", " + ui.Bridge("WATCH", "PROBES") + " and " + bell
		if got := ui.Name(path); got != want {
			t.Fatalf("the Bell step reads %q, the buttons read %q", got, want)
		}
	}
}

func TestTheBellSpotCardNamesWatchNotTheHighlightedButton(t *testing.T) {
	f := darkBellFrame()
	f.Selection = Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")}
	f.Guide.Target, f.Guide.Button = "crew:default/porters", view.WatchButton
	f.State.Objectives = []view.ObjectiveView{{ID: view.BellObjective, Text: "Hang the watchman's bell on the porters",
		Progress: "ready: hang the bell on the porters: click a porter hut, then CREW, " + ButtonToken + " and BELL"}}
	h := New(nil, (&fakeRunner{}).backend())
	if _, in := h.objectiveGuideCard(f, ui.Rect{X: 600, Y: 400, W: 40, H: 30}); !strings.Contains(in.Try, "CREW, WATCH and BELL") {
		t.Fatalf("with CREW lit the guide card reads %q: the highlighted button is CREW, not WATCH", in.Try)
	}
}
