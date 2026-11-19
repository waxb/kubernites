package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func porterState(served, arriving, waiting float64) view.GameState {
	st := view.GameState{}
	st.Crews = []view.CrewView{{Name: "porters", Image: "porter:1.0", Kind: "parcel", Depot: "parcels", Ready: 4, Replicas: 4}}
	st.Freight.Kinds = []view.KindView{{Kind: "parcel", Name: "parcels", Open: true, ArrivalRate: arriving, ServedRate: served, Waiting: waiting,
		Value: 2, DepotOpen: true, Crew: 4, CrewName: "porters", SizeMix: [4]float64{1}}}
	return st
}

func TestABetterClassNamesWhatItWouldEarnNow(t *testing.T) {
	st := porterState(1.2, 2, 8)
	got := ClassPayoff(st, "porter:2.0")
	if !strings.Contains(got, "x2 PARCELS PER RATION") || !strings.Contains(got, "+1.6 GOLD/S NOW") {
		t.Fatalf("porter:2.0 result = %q", got)
	}
}

func TestABetterClassForACrewThatKeepsUpFreesHands(t *testing.T) {
	st := porterState(2, 2, .2)
	got := ClassPayoff(st, "porter:2.0")
	if !strings.Contains(got, "x2 PARCELS PER RATION") || !strings.Contains(got, "HALF THE FOOD") {
		t.Fatalf("a kept-up crew's result = %q", got)
	}
}

func TestASturdierClassNamesHowMuchLessItFalls(t *testing.T) {
	got := ClassPayoff(porterState(1.2, 2, 8), "porter:1.1")
	if !strings.Contains(got, "FALLS 5x LESS OFTEN") {
		t.Fatalf("porter:1.1 result = %q", got)
	}
}

func TestAClassWithNoCrewOrNoGainSaysNothing(t *testing.T) {
	st := porterState(1.2, 2, 8)
	if got := ClassPayoff(st, "porter:1.0"); got != "" {
		t.Fatalf("the crew's own class = %q", got)
	}
	if got := ClassPayoff(st, "clerk:1.0"); got != "" {
		t.Fatalf("a class with no crew = %q", got)
	}
	if got := ClassPayoff(st, "porter:2.1"); got != "" {
		t.Fatalf("a crash-looping class promises = %q", got)
	}
}

func TestTheClassPayoffReadsInKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	got := ClassPayoff(porterState(1.2, 2, 8), "porter:2.0")
	if !strings.Contains(got, "x2 WEB REQUESTS PER CPU") || !strings.Contains(got, "+$1.6/S NOW") {
		t.Fatalf("names result = %q", got)
	}
	if got := ClassPayoff(porterState(1.2, 2, 8), "porter:1.1"); !strings.Contains(got, "RESTARTS 5x LESS OFTEN") {
		t.Fatalf("names sturdier result = %q", got)
	}
}

func academyFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	st := porterState(1.2, 2, 8)
	f.State.Crews, f.State.Freight = st.Crews, st.Freight
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "porter:2.0", Name: "Porters, class of 2.0", Description: "Twin-strap masters.", Cost: 450, Visible: true})
	return f
}

func TestTheAcademyShowsBeforeAndAfter(t *testing.T) {
	f := academyFrame(t)
	h := New(nil, Backend{})
	h.OverlayID = OverlaySoulvault
	b, ok := HitID(h.AcademyButtons(f), "vault:porter:2.0")
	if !ok || !strings.Contains(b.Tip, "+1.6 GOLD/S NOW") {
		t.Fatalf("PULL tip = %+v", b)
	}
	if got := AcademyPayoff(f, f.State.Soulstones[len(f.State.Soulstones)-1]); !strings.Contains(got, "x2 PARCELS PER RATION") {
		t.Fatalf("registry row result = %q", got)
	}
}

func TestNextNamesWhatAClassWouldEarn(t *testing.T) {
	f := academyFrame(t)
	n := &view.NextView{Kind: view.NextClass, ID: "porter:2.0", Label: "Porters, class of 2.0", Gold: 450}
	if tip := NextTip(n, f.State); !strings.Contains(tip, "+1.6 GOLD/S NOW") {
		t.Fatalf("NEXT class tip = %q", tip)
	}
}

func TestADearClassThatEarnsLittleSaysHowLongItTakesToPayBack(t *testing.T) {
	got := ClassPayoff(porterState(2, 2.025, 8), "porter:latest")
	if !strings.Contains(got, "+0.05 GOLD/S NOW, PAYS BACK IN 15") || !strings.Contains(got, " H; OR THE SAME LOAD ON 88% LESS FOOD") {
		t.Fatalf("a 27K class earning 0.05 a second reads %q", got)
	}
	if fast := ClassPayoff(porterState(1.5, 4, 8), "porter:2.0"); strings.Contains(fast, "PAYS BACK") {
		t.Fatalf("a class that settles within minutes still names a settling time: %q", fast)
	}
}
