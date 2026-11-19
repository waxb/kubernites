package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func init() {
	overlayBounds[OverlayRaise] = fixed(RaiseRect())
}

func chooserFrame(t *testing.T) Frame {
	t.Helper()
	f := offerFrame(t, raiseClerks)
	f.State.Freight = fixtureFreight()
	f.State.Crews = []view.CrewView{{Name: "porters", Image: "porter:1.0", Replicas: 3, Ready: 3}}
	f.State.Soulstones = []view.SoulstoneEntry{
		{Image: "porter:1.0", Owned: true}, {Image: "porter:1.1", Owned: true}, {Image: "porter:2.1", Owned: true},
		{Image: "porter:2.0"}, {Image: "clerk:1.0", Owned: true}, {Image: "hauler:1.0"},
	}
	return f
}

func openChooser(t *testing.T, h *HUD, f Frame, crew string) {
	t.Helper()
	clickOn(t, h, f, CmdCrew)
	if h.Overlay() != OverlayRaise {
		clickOn(t, h, f, CrewsKey+":raise:"+crew)
	}
	if h.Overlay() != OverlayRaise {
		t.Fatalf("the chooser did not open for %s", crew)
	}
}

func actionIntent(t *testing.T, intents []Intent) view.Action {
	t.Helper()
	it, ok := hasIntent(intents, IntentAction)
	if !ok {
		t.Fatalf("no action in %+v", intents)
	}
	return it.Action
}

func TestCrewOpensTheRaiseChooserInsteadOfActing(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	intents := clickOn(t, h, f, CmdCrew)
	if _, acted := hasIntent(intents, IntentAction); acted || h.Overlay() != OverlayNone || !h.CrewsList.Open {
		t.Fatalf("with a crew standing CREW opens the crews list and does nothing yet: overlay %q, %+v", h.Overlay(), intents)
	}
	clickOn(t, h, f, CrewsKey+":raise:clerks")
	if h.Overlay() != OverlayRaise || h.CrewsList.Open {
		t.Fatalf("RAISE on a crewless trade opens the chooser: overlay %q", h.Overlay())
	}
	rows := RaiseRows(f)
	if len(rows) != 3 || rows[0].Crew != "porters" || !rows[0].Standing || rows[1].Crew != "clerks" || rows[1].Standing {
		t.Fatalf("rows = %+v", rows)
	}
	if h.Ops.Raise.Pick != "clerks" {
		t.Fatalf("the offered trade is picked first, got %q", h.Ops.Raise.Pick)
	}
	confirm, ok := HitID(h.Buttons(f), RaiseConfirm)
	if !ok || confirm.Disabled || !strings.Contains(confirm.Tip, "kubectl create deployment clerks --image=clerk:1.0 --replicas=2 --port=8080") {
		t.Fatalf("confirm = %+v", confirm)
	}
	if !RaiseRect().Contains(confirm.Rect.X, confirm.Rect.Y) || Overlaps(RaiseRect(), Command) {
		t.Fatal("the chooser sits above the command card and never covers it")
	}
	clickOn(t, h, f, CmdCrew)
	if h.Overlay() != OverlayNone {
		t.Fatal("CREW again closes the chooser")
	}
}

func TestConfirmRaisesTheChosenTradeWithItsClassAndWorkers(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	openChooser(t, h, f, "clerks")
	clickOn(t, h, f, "raise:workers:up")
	a := actionIntent(t, clickOn(t, h, f, RaiseConfirm))
	if a.Action != "raiseCrew" || a.Name != "clerks" || a.Image != "clerk:1.0" || a.Replicas != 3 || h.Overlay() != OverlayNone {
		t.Fatalf("confirm = %+v, overlay %q", a, h.Overlay())
	}
	it, _ := hasIntent(clickOnAfterOpen(t, h, f), IntentAction)
	if it.From != FromChooser {
		t.Fatalf("the echo names the chooser: %q", it.From)
	}
}

func clickOnAfterOpen(t *testing.T, h *HUD, f Frame) []Intent {
	t.Helper()
	openChooser(t, h, f, "clerks")
	return clickOn(t, h, f, RaiseConfirm)
}

func TestTheChooserListsOnlyOwnedClassesAndDefaultsToAKnownGoodOne(t *testing.T) {
	f := chooserFrame(t)
	got := OwnedClasses(f, "porter:1.0")
	if strings.Join(got, ",") != "porter:2.1,porter:1.1,porter:1.0" {
		t.Fatalf("owned porter classes = %v", got)
	}
	if DefaultClass(got) != "porter:1.1" {
		t.Fatalf("the default is the newest class that runs, got %q", DefaultClass(got))
	}
	f.State.Soulstones[2].Cursed = true
	if got := OwnedClasses(f, "porter:1.0"); strings.Join(got, ",") != "porter:1.1,porter:1.0" {
		t.Fatalf("once the class that lies is known, it is never offered again: %v", got)
	}
	if NewerClasses(f, "porter:1.0")[0] != "porter:1.1" || len(NewerClasses(f, "porter:1.0")) != 1 {
		t.Fatalf("UPGRADE drops the known-bad class too: %v", NewerClasses(f, "porter:1.0"))
	}
	for _, rows := range RaiseRows(f) {
		for _, image := range rows.Images {
			if image == "porter:2.0" || image == "hauler:1.0" {
				t.Fatalf("an unbought class is listed: %+v", rows)
			}
		}
	}
}

func TestAnEmptyHarborLeavesTheChooserWithNothingToRaise(t *testing.T) {
	if got := RaiseHeadline(nil); got != "No kind of box has landed yet." {
		t.Fatalf("headline = %q", got)
	}
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	f.State.Freight.Kinds = nil
	clickOn(t, h, f, CmdCrew)
	if _, ok := HitID(h.Buttons(f), RaiseConfirm); ok {
		t.Fatal("with no trade open there is nothing to confirm")
	}
}

func TestWhenEveryTradeHasACrewTheListScalesAndOpensTheirBoards(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	f.State.Freight.Kinds = f.State.Freight.Kinds[:1]
	clickOn(t, h, f, CmdCrew)
	rows := CrewsRows(f)
	if len(rows) != 1 || !rows[0].Standing || rows[0].Chain.Name != "porters" {
		t.Fatalf("rows = %+v", rows)
	}
	a, from := actionOf(clickOn(t, h, f, CrewsKey+":scale:default/porters:up"))
	if a.Action != view.ActScaleCrew || a.Name != "porters" || a.Replicas != f.Crews[0].Desired+1 || from != FromCrewCard {
		t.Fatalf("scale = %+v from %q", a, from)
	}
	if got := CommandOf(a); !strings.HasPrefix(got, "kubectl scale deployment porters --replicas=") {
		t.Fatalf("echo = %q", got)
	}
	open := clickOn(t, h, f, CrewsKey+":open:default/porters")
	if it, ok := hasIntent(open, IntentChoose); !ok || it.Target != SelectCrew || it.Command != "default/porters" || h.CrewsList.Open {
		t.Fatalf("a row opens its crew's board: %+v", open)
	}
}

func TestStallsOpenDepotsFromTheBuildPage(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	f.State.Commands = nil
	h.PageID = PageBuild
	if _, ok := HitID(h.CommandButtons(f), StallTarget("ledger")); ok {
		t.Fatal("no clerks, no ledgers stall yet")
	}
	f.State.Crews = append(f.State.Crews, view.CrewView{Name: "clerks", Replicas: 2})
	stall, ok := HitID(h.CommandButtons(f), StallTarget("ledger"))
	if !ok || !strings.Contains(stall.Tip, "$ kubectl expose deployment clerks --name=ledgers") {
		t.Fatalf("the ledgers stall = %+v", stall)
	}
	a := actionIntent(t, clickOn(t, h, f, StallTarget("ledger")))
	if a.Action != "openDepot" || a.Kind != "ledger" {
		t.Fatalf("open = %+v", a)
	}
	if _, ok := HitID(h.CommandButtons(f), StallTarget("parcel")); ok {
		t.Fatal("an open depot has no stall to open")
	}
}

func TestTheGuideMovesToConfirmOnceTheChooserIsOpen(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "ledgers", Text: "Raise the clerks", Target: CmdCrew, Command: raiseClerks.Command}}
	f.Guide.Target = CmdCrew
	if s, ok := h.Step(f); !ok || s.ID != CmdCrew {
		t.Fatalf("the task points at CREW first: %+v %v", s, ok)
	}
	clickOn(t, h, f, CmdCrew)
	if s, ok := h.Step(f); !ok || s.ID != CrewsKey+":raise:clerks" {
		t.Fatalf("with the list open the guide points at RAISE on the crewless trade: %+v %v", s, ok)
	}
	clickOn(t, h, f, CrewsKey+":raise:clerks")
	s, ok := h.Step(f)
	if !ok || s.ID != RaiseConfirm || !RaiseRect().Contains(s.Rect.X, s.Rect.Y) {
		t.Fatalf("with the chooser open the guide points at its confirm: %+v %v", s, ok)
	}
	if w := h.WalkGuide(f); !w.OK() {
		t.Fatalf("the walk resolves: %+v", w)
	}
}

func TestMenuButtonsSayTheyOpenAMenu(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	f.State.Reveals = append(f.State.Reveals, "door")
	for _, b := range h.CommandButtons(f) {
		menu := b.ID != "cmd:serve" && b.ID != "cmd:door"
		switch {
		case b.Menu != menu:
			t.Errorf("%s menu mark = %v", b.ID, b.Menu)
		case menu && !strings.HasSuffix(b.Tip, MenuNote):
			t.Errorf("%s tip does not say it opens a menu: %q", b.ID, b.Tip)
		case !menu && !strings.HasSuffix(b.Tip, ActsNote):
			t.Errorf("%s tip does not say it acts at once: %q", b.ID, b.Tip)
		}
	}
}
