package world

import (
	"math"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestEachKindHasItsOwnYardOnTheQuay(t *testing.T) {
	sc, _ := harborScene(t)
	parcels, ok := YardFocus(sc.Site, "parcel")
	ledgers, ok2 := YardFocus(sc.Site, "ledger")
	quay, _ := HarborFocus(sc.Site)
	if !ok || !ok2 || parcels == ledgers {
		t.Fatalf("yards parcel %v ledger %v", parcels, ledgers)
	}
	if math.Hypot(ledgers.X-quay.X, ledgers.Y-quay.Y) > 6 {
		t.Fatalf("the ledger yard %v is far from the quay %v", ledgers, quay)
	}
	if v, ok := YardFocus(sc.Site, "sealed"); !ok || v != quay {
		t.Fatalf("a kind with no yard falls back to the quay: %v", v)
	}
}

func TestTheUnservedLabelReadsTheGoldLostInBothNames(t *testing.T) {
	defer ui.SetNamer(nil)
	u := model.Unserved{Kind: "ledger", Name: "ledgers", Gold: 7.68}
	if got := UnservedLabel(u); got != "LEDGERS SPOILING: -7.7 GOLD/S" {
		t.Fatalf("story label = %q", got)
	}
	ui.SetNamer(names.Kubernetes(64))
	if got := UnservedLabel(u); got != "DATABASE REQUESTS DROPPED: -$7.7/S" {
		t.Fatalf("names label = %q", got)
	}
}

func TestAnUnservedYardWearsARedPrice(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	if marks := r.UnservedMarks(cam, sc); len(marks) != 0 {
		t.Fatalf("a calm quay = %+v", marks)
	}
	kinds := append([]view.KindView(nil), sc.Freight.Kinds...)
	kinds[1].ArrivalRate, kinds[1].Value, kinds[1].Losses = 3.2, 12.5, view.LossView{NoDepot: 3.2}
	sc.Freight.Kinds = kinds
	marks := r.UnservedMarks(cam, sc)
	if len(marks) != 1 || marks[0].Kind != "ledger" || marks[0].Text != "LEDGERS SPOILING: -40 GOLD/S" || marks[0].Box.W == 0 {
		t.Fatalf("marks = %+v", marks)
	}
}

func TestThePriceStandsOverThePortWhereTheCargoPiles(t *testing.T) {
	h := &Harbor{Yards: []Yard{{Haven: 0, Kind: 0, Boxes: []QuayBox{{}}}, {Haven: 2, Kind: 1, Overflow: 3}}}
	if got := h.pileHaven(1); got != 2 {
		t.Fatalf("ledgers pile at port %d", got)
	}
	if got := h.pileHaven(3); got != 0 {
		t.Fatalf("an empty kind falls back to port %d", got)
	}
}

func TestAPriceUnderAPanelMovesAsideAndStaysInSight(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	kinds := append([]view.KindView(nil), sc.Freight.Kinds...)
	kinds[1].ArrivalRate, kinds[1].Value, kinds[1].Losses = 3.2, 12.5, view.LossView{NoDepot: 3.2}
	sc.Freight.Kinds = kinds
	free := r.UnservedMarks(cam, sc)
	if len(free) != 1 {
		t.Fatalf("marks = %+v", free)
	}
	sc.Panels = []ui.Rect{free[0].Box.Inset(-2)}
	moved := r.UnservedMarks(cam, sc)
	if len(moved) != 1 || Overlaps(moved[0].Box, sc.Panels[0]) {
		t.Fatalf("the label stays under the panel: %+v", moved)
	}
}

func TestAPriceStepsAsideFromTheGuidesBeacon(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	kinds := append([]view.KindView(nil), sc.Freight.Kinds...)
	kinds[1].ArrivalRate, kinds[1].Value, kinds[1].Losses = 3.2, 12.5, view.LossView{NoDepot: 3.2}
	sc.Freight.Kinds = kinds
	at, _ := YardFocus(sc.Site, "ledger")
	sc.Goal = &at
	beacons := r.BeaconRects(cam, sc)
	if len(beacons) == 0 {
		t.Fatal("no beacon at the ledger yard")
	}
	marks := r.UnservedMarks(cam, sc)
	if len(marks) != 1 {
		t.Fatalf("the label vanished beside the beacon: %+v", marks)
	}
	for _, m := range marks {
		for _, b := range beacons {
			if Overlaps(m.Box, b) {
				t.Fatalf("the label %+v sits under the beacon %+v", m.Box, b)
			}
		}
	}
}

func TestAQueueWorthNothingNamesNoGold(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	for _, u := range []model.Unserved{{Kind: "parcel", Name: "parcels", Queued: true}, {Kind: "parcel", Name: "parcels"}} {
		if got := UnservedLabel(u); strings.Contains(got, "$") || strings.Contains(got, "GOLD") || !strings.HasPrefix(got, "WEB REQUESTS ") {
			t.Errorf("a queue with Chapter set reads %q", got)
		}
	}
}

func TestAChapterQuayWearsNoPrice(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	kinds := append([]view.KindView(nil), sc.Freight.Kinds...)
	kinds[1].ArrivalRate, kinds[1].Value, kinds[1].Losses = 3.2, 12.5, view.LossView{NoDepot: 3.2}
	sc.Freight.Kinds = kinds
	sc.Chapter = true
	if marks := r.UnservedMarks(cam, sc); len(marks) != 0 {
		t.Fatalf("the quay with Chapter set = %+v", marks)
	}
}

func TestAChapterQuayShowsNoQuayFullLabel(t *testing.T) {
	sc := Scene{}
	if !quayFullShown(sc, true) || quayFullShown(sc, false) {
		t.Fatal("the quay-full label ignores its flag")
	}
	sc.Chapter = true
	if quayFullShown(sc, true) {
		t.Fatal("the quay with Chapter set shows QUAY FULL")
	}
}
