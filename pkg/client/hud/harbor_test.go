package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func harborFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.Selection = Selection{Kind: SelHaven, ID: "haven-0"}
	f.State.Freight.HarborView = view.HarborView{Tier: 2, Multiplier: 4, Fresh: 9500, NextTier: 200000, TierProgress: .05}
	f.State.Freight.Kinds = carriedCargo(f.State.Freight.Kinds)
	f.State.Ports = fixturePorts()
	return f
}

func fixturePorts() []view.PortView {
	gate := view.PortView{ID: "haven-0", Name: "Gate Harbor", Open: true, Level: 2, MaxLevel: 4, Ship: "caravel", Ships: "caravels", Multiplier: 4,
		Listeners: []string{"HTTP", "TCP"}, Berths: 1, MaxBerths: 5, BerthCost: 100, UpgradeCost: 200000,
		Next:    &view.PortNextView{Level: 3, Ship: "galleon", Ships: "galleons", Listeners: []string{"HTTP", "TCP", "HTTPS"}, Multiplier: 16},
		Effects: view.PortEffects{Berth: "+25% ships at Gate Harbor for 100 MiB (x1.25 now, x1.56 after)", Upgrade: "Galleons bring dispatches"},
		Gates:   view.PortGates{Open: "Gate Harbor is already open", Upgrade: "upgrading Gate Harbor needs 200000 MiB (you have 1500)"}}
	north := view.PortView{ID: "haven-1", Name: "North Haven", MaxLevel: 4, OpenCost: 150, Next: &view.PortNextView{Level: 1, Ships: "cogs"},
		Effects: view.PortEffects{Open: "Its own cogs"}, Gates: view.PortGates{Upgrade: "North Haven is closed: open it first", Berth: "North Haven is closed: open it first", Close: "no berth is open at North Haven"}}
	return []view.PortView{gate, north}
}

func cardTexts(c Card) []string {
	out := []string{}
	for _, l := range c.Lines {
		out = append(out, l.Text)
	}
	return out
}

func TestHarborCardShowsThePortLevelAndBerths(t *testing.T) {
	f := harborFrame(t)
	c := HarborCard(f)
	want := []string{"Level 2: caravels", "NEXT: GALLEONS", "1/5 berths: ships here x1.25", "Next: +25% ships at Gate Harbor for 100 MiB (x1.25 now, x1.56 after)"}
	if !slices.Equal(cardTexts(c), want) || c.Title != GateHarborTitle || !strings.HasPrefix(c.Portrait, SpritePortrait+"ship/caravel/") {
		t.Fatalf("card = %q %q %v", c.Title, c.Portrait, cardTexts(c))
	}
	if bar := c.Lines[1]; !bar.Bar || bar.Suffix != "200K" || bar.Value != 1500.0/200000 {
		t.Fatalf("level bar = %+v", bar)
	}
	f.State.Ports[0].Next = nil
	if got := HarborCard(f).Lines[1]; got.Bar || got.Text != "The top of the harbor ladder" {
		t.Fatalf("top level = %+v", got)
	}
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(r string) bool { return r == BerthsReveal })
	if got := cardTexts(HarborCard(f)); len(got) != 2 {
		t.Fatalf("berths shown before they are offered: %v", got)
	}
	h := New(nil, (&fakeRunner{}).backend())
	if b, ok := HitID(h.CommandButtons(f), PortButtonID(PortVerbBerth, "haven-0")); ok {
		t.Fatalf("a berth cell before the berths are offered = %+v", b)
	}
}

func portCell(t *testing.T, f Frame, verb, id string) Button {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	b, ok := HitID(h.CommandButtons(f), PortButtonID(verb, id))
	if !ok {
		t.Fatalf("no %s cell for %s in %v", verb, id, gridIDs(h, f))
	}
	return b
}

func TestAClosedPortOffersToOpenIt(t *testing.T) {
	f := harborFrame(t)
	f.Selection.ID = "haven-1"
	if got := cardTexts(HarborCard(f)); got[0] != "A bare shore: open it for 150 gold" || len(got) != 2 {
		t.Fatalf("closed card = %v", got)
	}
	if b := portCell(t, f, PortVerbOpen, "haven-1"); b.Label != "OPEN PORT" || b.Cost != "150" || b.Disabled || !strings.HasPrefix(b.Tip, "Its own cogs") {
		t.Fatalf("closed port cell = %+v", b)
	}
	f.State.Ports[1].Gates.Open = "opening North Haven needs 150 MiB (you have 10)"
	if b := portCell(t, f, PortVerbOpen, "haven-1"); !b.Disabled || !strings.HasPrefix(b.Tip, "Opening North Haven needs 150 MiB") {
		t.Fatalf("poor = %+v", b)
	}
}

func TestTheUpgradeCellNamesTheNextShipsOrListener(t *testing.T) {
	f := harborFrame(t)
	if up := portCell(t, f, PortVerbRaise, "haven-0"); up.Label != "UPGRADE: GALLEONS" || !up.Disabled || !strings.HasPrefix(up.Tip, "Upgrading Gate Harbor needs") {
		t.Fatalf("upgrade = %+v", up)
	}
	ui.SetNamer(names.Kubernetes(64))
	defer ui.SetNamer(nil)
	if up := portCell(t, f, PortVerbRaise, "haven-0"); up.Label != "ADD HTTPS LISTENER" {
		t.Fatalf("NAMES upgrade to galleons = %q", up.Label)
	}
	f.State.Ports[0].Level, f.State.Ports[0].Listeners = 3, []string{"HTTP", "TCP", "HTTPS"}
	f.State.Ports[0].Next = &view.PortNextView{Level: 4, Ships: "carracks", Listeners: []string{"HTTP", "TCP", "HTTPS"}}
	if up := portCell(t, f, PortVerbRaise, "haven-0"); up.Label != "ADMIT HUGE BURSTS" {
		t.Fatalf("NAMES upgrade to carracks = %q", up.Label)
	}
}

func TestHarborCardNamesTheHarborClicked(t *testing.T) {
	f := harborFrame(t)
	if len(f.Site.Havens) < 2 {
		t.Skip("this seed plans a single harbor")
	}
	f.Selection.ID = f.Site.Havens[1].ID
	if got := HarborTitle(f); got != strings.ToUpper(f.Site.Havens[1].Name) {
		t.Fatalf("title = %q", got)
	}
	f.Selection.ID = "nowhere"
	if got := HarborTitle(f); got != GateHarborTitle {
		t.Fatalf("an unknown harbor is titled %q", got)
	}
}

func TestOpenBerthButtonsSayWhyTheyWait(t *testing.T) {
	f := harborFrame(t)
	f.State.Ports[0].Gates.Berth = "a berth opens only while your crews hold the Nine: 86.4% delivered fresh, 90% needed"
	f.State.Berths.Gate = f.State.Ports[0].Gates.Berth
	if b := portCell(t, f, PortVerbBerth, "haven-0"); !b.Disabled || !strings.HasPrefix(b.Tip, "A berth opens only") {
		t.Fatalf("gated = %+v", b)
	}
	if got := cardTexts(HarborCard(f)); !strings.HasPrefix(got[len(got)-1], "A berth opens only while your crews hold the Nine") {
		t.Fatalf("gated card = %v", got)
	}
	f.State.Ports[0].Berths = 5
	if got := cardTexts(HarborCard(f)); got[len(got)-1] != "Its quay is full: upgrade it for more berths." {
		t.Fatalf("full card = %v", got)
	}
}

func TestPressingThePortButtonsActs(t *testing.T) {
	for id, want := range map[string]view.Action{
		"harbor:berth:haven-0": {Action: view.ActOpenBerth, ID: "haven-0"}, "harbor:close:haven-0": {Action: view.ActCloseBerth, ID: "haven-0"},
		"harbor:upgrade:haven-2": {Action: view.ActUpgradePort, ID: "haven-2"}, "harbor:open:haven-1": {Action: view.ActOpenPort, ID: "haven-1"},
	} {
		if a, ok := PortAction(id); !ok || a != want {
			t.Errorf("%s = %+v", id, a)
		}
		if got := PressPort(nil, strings.TrimPrefix(id, "harbor:"), Frame{}); len(got) != 1 || got[0].Action != want {
			t.Errorf("pressing %s = %+v", id, got)
		}
	}
	for _, id := range []string{"harbor:berth", "harbor:sail:haven-0", "harbor:open:"} {
		if _, ok := PortAction(id); ok {
			t.Errorf("%s reads as a port button", id)
		}
	}
	h := New(nil, Backend{})
	f := harborFrame(t)
	f.State.Gold = 1e6
	if it, ok := hasIntent(clickOn(t, h, f, "harbor:berth:haven-0"), IntentAction); !ok || it.Action.ID != "haven-0" {
		t.Errorf("the card's berth button = %+v", it)
	}
}

func TestTheGuideWalksToTheHarborThenToOpenBerth(t *testing.T) {
	if HarborStep(BerthButton, Selection{}) != GateHaven || HarborStep(SurfaceBerths, Selection{Kind: "structure"}) != GateHaven {
		t.Fatal("the berth is not reached through the Gate Harbor")
	}
	if HarborStep(BerthButton, Selection{Kind: SelHaven}) != BerthButton || HarborStep("cmd:build", Selection{}) != "cmd:build" {
		t.Fatal("HarborStep rewrote a step it does not own")
	}
	if HarborStep("harbor:upgrade:haven-2", Selection{Kind: SelHaven, ID: "haven-0"}) != "haven:haven-2" || HarborStep("harbor:upgrade:haven-2", Selection{Kind: SelHaven, ID: "haven-2"}) != "harbor:upgrade:haven-2" {
		t.Fatal("a port's step does not walk to that port first")
	}
	h := New(nil, Backend{})
	f := harborFrame(t)
	f.Selection = Selection{}
	f.Guide = Guide{Target: BerthButton}
	f.Beacons = []ui.Rect{{X: 40, Y: 60, W: 20, H: 20}}
	s, ok := h.Step(f)
	if !ok || !s.Map || s.ID != GateHaven || s.Rect != f.Beacons[0] {
		t.Fatalf("step before the harbor is open = %+v", s)
	}
	f.Selection = Selection{Kind: SelHaven, ID: "haven-0"}
	f.State.Ports = nil
	s, ok = h.Step(f)
	if !ok || s.Map || s.ID != BerthButton || s.Rect != CellRect(BerthCell) {
		t.Fatalf("step on the harbor card = %+v", s)
	}
}

func TestTheNextBarAndTheAdviceOpenTheHarbor(t *testing.T) {
	h := New(nil, Backend{})
	f := harborFrame(t)
	f.Selection = Selection{}
	f.State.Next = &view.NextView{Kind: view.NextBerth, Label: "Harbor berth", Target: BerthButton, Ready: true}
	if _, ok := hasIntent(h.PressNext(f), IntentHarbor); !ok {
		t.Fatal("the NEXT bar does not lead to the harbor")
	}
	f.State.Advice = "The quay overflows: close a berth"
	if _, ok := hasIntent(h.FollowNext(f), IntentHarbor); !ok {
		t.Fatal("advice to close a berth does not lead to the harbor")
	}
}

func TestTheHarborHasItsOwnDetails(t *testing.T) {
	h := New(nil, Backend{})
	sel := Selection{Kind: SelHaven, ID: "haven-0"}
	if SelectionTitle(sel) != "The harbor" || DetailCommands(sel, DetailDescribe) != nil {
		t.Fatalf("title %q", SelectionTitle(sel))
	}
	if lines := h.LoreLines(sel); !strings.Contains(strings.Join(lines, " "), "not a Kubernetes object") {
		t.Fatalf("lore = %v", lines)
	}
}

func TestTheHarborCardDraws(t *testing.T) {
	h := New(nil, Backend{})
	f := harborFrame(t)
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	if got := h.CardFor(f); got.Title != GateHarborTitle {
		t.Fatalf("card = %+v", got)
	}
	if CardBottom(h.CardFor(f), f) != CardBox.Y+CardBox.H-4 {
		t.Fatal("with its berths in the grid the card's lines use the whole card")
	}
}
