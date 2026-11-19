package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheDemandRowPricesTheWorstStrandedCargo(t *testing.T) {
	f := fullFrame(t)
	u, ok := StrandedCargo(f)
	if !ok || u.Kind != "ledger" {
		t.Fatalf("stranded = %+v", u)
	}
	if got := segmentTexts(DemandSegments(u, 400)); got != "LEDGERS 0.50/S SPOILING | 4.8 GOLD/S ON THE FLOOR" {
		t.Fatalf("wide demand row = %q", got)
	}
	if got := segmentTexts(DemandSegments(u, BoardW-2*BoardPad)); got != "LEDGERS 0.50/S SPOILING | -"+ui.IconGold+"4.8/S" {
		t.Fatalf("board demand row = %q", got)
	}
	tight := segmentTexts(DemandSegments(u, 110))
	if !strings.Contains(tight, "-"+ui.IconGold+"4.8") {
		t.Fatalf("a narrow row keeps the figure: %q", tight)
	}
}

func TestTheDemandRowReadsInKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	u, _ := StrandedCargo(fullFrame(t))
	got := segmentTexts(DemandSegments(u, 400))
	if !strings.Contains(got, "DATABASE REQUESTS 0.50/S DROPPED") || !strings.Contains(got, "$4.8/S LOST") {
		t.Fatalf("names demand row = %q", got)
	}
	if tip := DemandTip(u); !strings.Contains(tip, "no Deployment serves") || strings.Contains(tip, "on the floor") || strings.Contains(tip, "stall") {
		t.Fatalf("names demand tip = %q", tip)
	}
}

func TestTheDemandRowSitsOnTheBoardAndPointsAtThePile(t *testing.T) {
	f := fullFrame(t)
	if !DemandShown(f) {
		t.Fatal("stranded ledgers show no demand row")
	}
	h := New(nil, Backend{})
	h.BoardOpen = true
	b, ok := HitID(h.Buttons(f), DemandRowID)
	if !ok {
		t.Fatal("no demand row button")
	}
	if !strings.Contains(b.Tip, "clerks") || !strings.Contains(b.Tip, "ledgers depot") {
		t.Fatalf("demand tip = %q", b.Tip)
	}
	intents := clickOn(t, h, f, DemandRowID)
	if _, ok := hasIntent(intents, IntentFocus); !ok {
		t.Fatalf("pressing the demand row = %+v", intents)
	}
}

func carriedCargo(kinds []view.KindView) []view.KindView {
	out := append([]view.KindView(nil), kinds...)
	for i := range out {
		out[i].DepotOpen, out[i].Crew, out[i].Losses = true, 2, view.LossView{}
		out[i].ServedRate = out[i].ArrivalRate
	}
	return out
}

func TestNoDemandRowWhileEveryCargoHasACrew(t *testing.T) {
	f := fullFrame(t)
	f.State.Freight.Kinds = carriedCargo(f.State.Freight.Kinds)
	if DemandShown(f) {
		t.Fatal("a demand row with every kind carried")
	}
	h := New(nil, Backend{})
	h.BoardOpen = true
	if _, ok := HitID(h.Buttons(f), DemandRowID); ok {
		t.Fatal("the demand row button stayed")
	}
	if b, ok := HitID(h.Buttons(f), FreightTip+"goal"); !ok || !strings.Contains(b.Tip, "HARBOR TIER") {
		t.Fatalf("the goal row came back = %+v", b)
	}
}

func TestAChipOfUnservedCargoShowsTheGoldLost(t *testing.T) {
	head := FreightHeadOf(fixtureFreight(), true)
	if got := KindChipLabel(head.Chips[1]); got != "-4.8" {
		t.Fatalf("ledger chip = %q", got)
	}
	if got := KindChipLabel(head.Chips[0]); got != "8" {
		t.Fatalf("a short crew that has spoiled nothing shows its waiting boxes, not a loss: %q", got)
	}
	if !strings.Contains(head.Chips[0].Tip, "Backing up: 0.20 a second more arrive than are carried; 0.96 gold a second is at risk") {
		t.Fatalf("a short crew's chip tip = %q", head.Chips[0].Tip)
	}
	if !strings.Contains(head.Chips[1].Tip, "4.8 gold a second on the floor") {
		t.Fatalf("ledger chip tip = %q", head.Chips[1].Tip)
	}
	calm := fixtureFreight()
	calm.Kinds[0].Waiting, calm.Kinds[0].Losses = 0, view.LossView{}
	if got := KindChipLabel(FreightHeadOf(calm, true).Chips[0]); got != "0" {
		t.Fatalf("a carried kind's chip = %q", got)
	}
}

func TestTheHarborCardNamesTheMoneyLost(t *testing.T) {
	f := fullFrame(t)
	c := HarborCard(f)
	found := false
	for _, l := range c.Lines {
		if strings.Contains(l.Text, "4.8 GOLD/S ON THE FLOOR") {
			found = l.Clr == ui.Palette.RedText
		}
	}
	if !found {
		t.Fatalf("harbor card lines = %+v", c.Lines)
	}
}

func TestWhereAStallGoesIsOnTheMapButItsButtonIsNot(t *testing.T) {
	if !MapTarget("stallspot:ledger") {
		t.Fatal("where the ledgers stall goes is not a place on the map")
	}
	if MapTarget(StallTarget("ledger")) {
		t.Fatal("the grid's stall button reads as a place on the map")
	}
}

func cardSays(c Card, text string) bool {
	for _, l := range c.Lines {
		if strings.Contains(l.Text, text) {
			return true
		}
	}
	return false
}

func TestAPortCardPricesOnlyTheCargoItBrings(t *testing.T) {
	f := fullFrame(t)
	cogs := view.PortView{ID: "haven-0", Name: "Gate Harbor", Open: true, Level: 1, MaxLevel: 4, Kinds: []string{"parcel"}}
	if c := PortCard(f, cogs); cardSays(c, "LEDGERS") {
		t.Fatalf("a port of cogs lists the ledgers it never brings: %+v", c.Lines)
	}
	caravels := cogs
	caravels.Level, caravels.Kinds = 2, []string{"parcel", "ledger"}
	if !cardSays(PortCard(f, caravels), "LEDGERS: 0.50/S SPOILING, 4.8 GOLD/S ON THE FLOOR") {
		t.Fatalf("a port of caravels hides the ledgers it lands: %+v", PortCard(f, caravels).Lines)
	}
}

func TestAChapterPricesNoRequestAndDropsNone(t *testing.T) {
	f := fullFrame(t)
	f.State.Tutorial = true
	if _, ok := StrandedCargo(f); ok {
		t.Fatal("the tutorial frame shows a demand row")
	}
	for _, c := range []Card{HarborCard(f), PortCard(f, view.PortView{ID: "haven-0", Name: "Gate Harbor", Open: true, Level: 2, Kinds: []string{"parcel", "ledger"}})} {
		for _, l := range c.Lines {
			if strings.Contains(l.Text, "GOLD/S") || strings.Contains(l.Text, "SPOILING") || strings.Contains(l.Text, "LOST") {
				t.Fatalf("a card in the tutorial frame reads %q", l.Text)
			}
		}
	}
}

func TestAStrandedPileThatHasNotSpoiledYetReadsBackingUp(t *testing.T) {
	u := model.Unserved{Kind: "ledger", Name: "ledgers", Crew: "clerks", Depot: "ledgers", Cause: model.UnservedNoDepot, Rate: 3.2, Gold: 40, Queued: true}
	for _, text := range []string{segmentTexts(DemandSegments(u, 400)), DemandText(u), DemandTip(u)} {
		if strings.Contains(strings.ToUpper(text), "SPOIL") && !strings.Contains(strings.ToUpper(text), "BACKING UP") || !strings.Contains(strings.ToUpper(text), "BACKING UP") {
			t.Errorf("ledgers piling up on the quay, none lost yet, read %q (berths capture: the harbor's LOST 2.3/S beside the card's SPOILING 3.9/S)", text)
		}
	}
}
