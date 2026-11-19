package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

var dispatches = view.TreasureView{Kind: "dispatch", Boxes: "dispatches", Crew: "couriers"}

func TestTheTreasureBannerNamesTheCargoNoCrewCarries(t *testing.T) {
	title, cause := TreasureBanner(dispatches)
	if title != "NEW CARGO: DISPATCHES" || !strings.HasPrefix(cause, "dispatches wait at sea until their depot stands") || !strings.Contains(cause, "couriers") {
		t.Fatalf("banner = %q / %q", title, cause)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if _, cause := TreasureBanner(dispatches); !strings.Contains(cause, "no Deployment") {
		t.Fatalf("NAMES cause = %q", cause)
	}
}

func TestATreasureOpensWhenItStopsWaiting(t *testing.T) {
	var ch treasureChests
	ch, fresh, opened := ch.step([]view.TreasureView{dispatches}, 10)
	if len(fresh) != 1 || len(opened) != 0 {
		t.Fatalf("first sight: fresh %v opened %v", fresh, opened)
	}
	ch, fresh, _ = ch.step([]view.TreasureView{dispatches}, 11)
	if len(fresh) != 0 {
		t.Fatalf("the banner repeats: %v", fresh)
	}
	if got := ch.marks(11); !slices.Equal(got, []world.ChestMark{{Kind: "dispatch"}}) {
		t.Fatalf("waiting marks = %+v", got)
	}
	ch, _, opened = ch.step(nil, 20)
	if !slices.Equal(opened, []string{"dispatch"}) {
		t.Fatalf("opened = %v", opened)
	}
	if got := ch.marks(21.5); len(got) != 1 || !got[0].Open || got[0].Age != 1.5 {
		t.Fatalf("opening marks = %+v", got)
	}
	if got := ch.marks(20 + chestOpenSeconds + 1); len(got) != 0 {
		t.Fatalf("the open chest lingers: %+v", got)
	}
}

func TestAnOldSaveDoesNotBannerItsWaitingTreasureAsNew(t *testing.T) {
	var ch treasureChests
	ch = ch.prime([]view.TreasureView{dispatches})
	if _, fresh, _ := ch.step([]view.TreasureView{dispatches}, 5); len(fresh) != 0 {
		t.Fatalf("a loaded save bannered %v", fresh)
	}
}
