package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func earlyStallFrame(t *testing.T, introduced ...string) Frame {
	t.Helper()
	f := offerFrame(t, openParcels)
	f.State.Crews = []view.CrewView{{Name: "porters", Replicas: 2}}
	f.Guide = Guide{Staged: true, Introduced: append([]string{"cmd:serve", "cmd:build"}, introduced...)}
	return f
}

func TestStallCellsWaitForTheServicesLesson(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, naming := range []bool{false, true} {
		ui.SetNamer(nil)
		if naming {
			ui.SetNamer(names.Kubernetes(64))
		}
		early := earlyStallFrame(t)
		b, ok := HitID(stallButtons(early), StallTarget("parcel"))
		if !ok || !b.Disabled || !strings.Contains(b.Tip, "Opens once three") || strings.Contains(b.Tip, "lesson") || strings.Contains(b.Tip, "Stone") {
			t.Errorf("names %v: before the Services step the stall is %+v", naming, b)
		}
		ready := earlyStallFrame(t, CmdDepot)
		if b, ok := HitID(stallButtons(ready), StallTarget("parcel")); !ok || b.Disabled {
			t.Errorf("names %v: once the Services step is reached the stall stays shut: %+v", naming, b)
		}
	}
}

func TestTheGuideNeverPointsAtAStallThatIsNotThere(t *testing.T) {
	f := earlyStallFrame(t, CmdDepot)
	f.State.Crews = nil
	if goal := stallGoal(CmdDepot, f); goal != "" {
		t.Fatalf("with no stall to press the guide points at %q", goal)
	}
}

func TestTheLadderHidesTheGuildhallBesideTheParcelsStall(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := earlyStallFrame(t, CmdDepot)
	f.State.Crews = nil
	f.State.Buildables = []view.Buildable{{Kind: view.StructGuildhall, Name: "Parcels Depot", Reason: view.LadderDepotGate, Locked: true}}
	for _, b := range h.buildPage(f) {
		if b.ID == "build:"+view.StructGuildhall {
			t.Fatalf("a second PARCELS DEPOT tile stands on the hand ladder: %+v", b)
		}
	}
}
