package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func watchFrame(runes ...string) Frame {
	f := crewFrame(crewSel(), allReveals...)
	f.State.Grimoire = []view.GrimoirePage{{ID: view.WatchPage, Ranks: []view.RankView{{Runes: runes, Learned: true}}}}
	f.State.Watch = &view.WatchView{Crews: []view.CrewWatchView{{Crew: "porters", Container: "porter"}}}
	return f
}

func slotIDs(slots []gridSlot) []string {
	out := []string{}
	for _, s := range slots {
		out = append(out, s.button.ID)
	}
	return out
}

func TestTheWatchTakesTheYamlCellOnceItsRuneIsKnown(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	plain := slotIDs(h.crewSlots(crewFrame(crewSel(), allReveals...)))
	if slices.Contains(plain, view.WatchButton) || !slices.Contains(plain, "crew:yaml") {
		t.Fatalf("before the Watch the grid is %v", plain)
	}
	got := slotIDs(h.crewSlots(watchFrame("readinessProbe")))
	if !slices.Contains(got, view.WatchButton) || slices.Contains(got, "crew:yaml") {
		t.Fatalf("with the Shutters the grid is %v", got)
	}
}

func TestTheWatchModeTogglesTheShuttersAndTheBell(t *testing.T) {
	f := watchFrame("readinessProbe")
	h := New(nil, (&fakeRunner{}).backend())
	h.pressCrew("probes", f)
	ids := slotIDs(h.crewSlots(f))
	if !slices.Contains(ids, "crew:probes:readiness") || slices.Contains(ids, "crew:probes:liveness") || !slices.Contains(ids, "crew:yaml") {
		t.Fatalf("the Watch mode shows %v", ids)
	}
	it, ok := hasIntent(h.pressCrew("probes:readiness", f), IntentAction)
	want := view.Action{Action: view.ActCrewProbes, Name: "porters", Crew: view.CrewOrder{Container: "porter", Probe: view.ProbeReadiness,
		Period: view.WatchPeriod, Threshold: view.WatchThreshold, Timeout: view.WatchPatience}}
	if !ok || it.Action.Action != want.Action || it.Action.Name != want.Name || it.Action.Crew != want.Crew {
		t.Fatalf("SHUTTERS sent %+v", it.Action)
	}
	f.State.Watch.Crews[0].Readiness = &view.ProbeSetting{Path: "/ready"}
	it, _ = hasIntent(h.pressCrew("probes:readiness", f), IntentAction)
	if !it.Action.Crew.Off {
		t.Fatalf("pressing lit SHUTTERS does not take them off: %+v", it.Action.Crew)
	}
	f = watchFrame("readinessProbe", "livenessProbe")
	h.pressCrew("probes", f)
	if ids := slotIDs(h.crewSlots(f)); !slices.Contains(ids, "crew:probes:liveness") {
		t.Fatalf("the Bell is missing once its rune is known: %v", ids)
	}
}

func TestTheWatchSpeaksBothModes(t *testing.T) {
	f := watchFrame("readinessProbe", "livenessProbe")
	labels := func() []string {
		out := []string{}
		for _, b := range probeButtons(crewChain(), f) {
			out = append(out, ui.Name(b.Label))
		}
		return out
	}
	if got := labels(); !slices.Equal(got, []string{"SHUTTERS: OFF", "BELL: OFF"}) {
		t.Fatalf("story labels %v", got)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if got := labels(); !slices.Equal(got, []string{"READINESS: OFF", "LIVENESS: OFF"}) {
		t.Fatalf("NAMES labels %v", got)
	}
}
