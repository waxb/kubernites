package hud

import (
	"fmt"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func busyRealmFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.Pods = nil
	for i := range 31 {
		f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: fmt.Sprintf("porter-%d", i), State: model.HearthReady})
	}
	f.State.Freight.Kinds = []view.KindView{
		{Kind: "parcel", Name: "parcels", Open: true, DepotOpen: true, Depot: "parcels", Crew: 12, CrewName: "porters", ArrivalRate: 1, ServedRate: 1, Waiting: 1, Value: 2},
		{Kind: "ledger", Name: "ledgers", Open: true, DepotOpen: true, Depot: "ledgers", Crew: 9, CrewName: "clerks", ArrivalRate: .5, ServedRate: .5, Value: 3},
	}
	f.State.Freight.Seals = view.SealsView{Depot: "seals", DepotOpen: true, Crew: 4, CrewName: "sealwrights", ArrivalRate: .3, ServedRate: .3}
	f.State.Bottleneck = ""
	return f
}

func assertNothingCut(t *testing.T, name string, bounds ui.Rect, draw func()) {
	t.Helper()
	defer ui.SetNamer(nil)
	for _, mode := range namesModes() {
		mode.apply()
		runs := recordRuns(draw)
		if cut := truncated(runs); len(cut) > 0 {
			t.Errorf("%s [%s]: cut %q", name, mode.name, cut)
		}
		for _, msg := range textProblems(runs, bounds) {
			t.Errorf("%s [%s]: %s", name, mode.name, msg)
		}
	}
}

func TestRealmCardFitsABusyRealm(t *testing.T) {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h := New(nil, Backend{})
	f := busyRealmFrame(t)
	assertNothingCut(t, "realm card", CardBox, func() { h.drawRealmCard(dst, f) })
}

func TestDepotCardCountsOneBoxAsOne(t *testing.T) {
	f := busyRealmFrame(t)
	depot := view.Structure{Kind: view.DepotStructure("parcel"), Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	lines := DepotLines(depot, f)
	if got := lines[1].Text; got != "12 porters carry, 1 parcel waiting" {
		t.Fatalf("depot line %q", got)
	}
	f.State.Freight.Kinds[0].Waiting = 3
	if got := DepotLines(depot, f)[1].Text; got != "12 porters carry, 3 parcels waiting" {
		t.Fatalf("depot line %q", got)
	}
	f.State.Freight.Seals.Waiting = 1
	seals := view.Structure{Kind: view.DepotStructure("sealed"), Object: view.ObjectRef{Kind: "Service", Name: "seals"}}
	if got := DepotLines(seals, f)[1].Text; got != "4 sealwrights carry, 1 sealed box waiting" {
		t.Fatalf("seals line %q", got)
	}
	k8s := names.Kubernetes(64)
	f.State.Freight.Kinds[1].Waiting = 1
	ledgers := view.Structure{Kind: view.DepotStructure("ledger"), Object: view.ObjectRef{Kind: "Service", Name: "ledgers"}}
	if got := k8s(DepotLines(ledgers, f)[1].Text); got != "9 clerks carry, 1 database request waiting" {
		t.Fatalf("Kubernetes ledger line %q", got)
	}
	for plural, one := range map[string]string{"parcels": "parcel", "ledgers": "ledger", "dispatches": "dispatch", "sealed boxes": "sealed box", "crates": "crate", "x": "x"} {
		if got := singularOf(plural); got != one {
			t.Errorf("singularOf(%q) = %q, want %q", plural, got, one)
		}
	}
}

func TestLoomTitleClearsTheThreadCounter(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Reforge = view.ReforgeView{Threads: 0, RunThreads: 13, NextThreadAt: "a Load test weathered"}
	right := ageRight()
	for _, mode := range namesModes() {
		mode.apply()
		title, left := h.loomTitle(f, right)
		if title.X+title.W+loomTitleGap > left {
			t.Errorf("[%s] the title %+v runs into the counter at x=%d", mode.name, title, left)
		}
	}
}

func TestCoinsAndDeltasFlyUnderCardsAndWindows(t *testing.T) {
	at := map[string]int{}
	for i, l := range HudLayers {
		at[l.Name] = i
	}
	for _, flier := range []string{"coins", "deltas"} {
		if at[flier] < at["topbar"] {
			t.Errorf("%s must stay above the top bar they fly into", flier)
		}
		for _, window := range []string{"windows", "survey", "console", "overlay", "intro", "banner", "toasts", "tooltip"} {
			if at[flier] > at[window] {
				t.Errorf("%s draw over %s", flier, window)
			}
		}
	}
	names := make([]string, len(HudLayers))
	for i, l := range HudLayers {
		names[i] = l.Name
	}
	if !slices.Contains(names, "coins") || !slices.Contains(names, "deltas") {
		t.Fatalf("layers %v", names)
	}
}

func TestGoldDeltaStaysClearOfTheObjectiveCard(t *testing.T) {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "speak-get", Text: "Speak to the Stone", Command: "kubectl get nodes"}}
	h.Delta(MeterGold, 213)
	cards := h.topCards(f)
	if len(cards) == 0 {
		t.Fatal("the task card is up")
	}
	for _, r := range recordRuns(func() { h.drawDeltas(dst, f) }) {
		for _, c := range cards {
			if Overlaps(r.Rect, c) {
				t.Errorf("the delta %q at %+v floats over the card %+v", r.Text, r.Rect, c)
			}
		}
	}
	f.State.Objectives = nil
	if runs := recordRuns(func() { h.drawDeltas(dst, f) }); !hasRun(runs, "+213") {
		t.Fatalf("with no card the delta shows: %+v", runs)
	}
}
