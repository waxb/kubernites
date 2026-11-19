package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var (
	raiseClerks = view.CommandOffer{ID: "raise-crew:clerks", Label: "Raise a crew of Clerks (Deployment clerks)",
		Command: "kubectl create deployment clerks --image=clerk:1.0 --replicas=2 --port=8080",
		Action:  view.Action{Action: "raiseCrew", Name: "clerks", Image: "clerk:1.0", Replicas: 2}}
	openParcels = view.CommandOffer{ID: "open-depot:parcel", Label: "Open the parcels depot (Service parcels)",
		Command: "kubectl expose deployment porters --name=parcels --port=80 --target-port=8080",
		Action:  view.Action{Action: "openDepot", Kind: "parcel"}}
)

func offerFrame(t *testing.T, offers ...view.CommandOffer) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = []string{"serve", "gold", "build", "stone", "tidings", "depot", "crew"}
	f.State.Commands = offers
	f.State.Objectives = []view.ObjectiveView{{ID: "o1", Text: "Wait", Hint: "Wait"}}
	f.State.Freight.Kinds = []view.KindView{{Kind: "parcel", Open: true, Depot: "parcels"}, {Kind: "ledger", Depot: "ledgers"}}
	return f
}

func TestLineForMatchesTheOfferThatDoesTheSameThing(t *testing.T) {
	offers := []view.CommandOffer{raiseClerks, openParcels}
	l := LineFor("kubectl create deployment clerks --image=clerk:1.0 --replicas=2", offers)
	if l.Action == nil || l.Action.Action != "raiseCrew" || l.Action.Name != "clerks" || l.Command != "kubectl create deployment clerks --image=clerk:1.0 --replicas=2" {
		t.Errorf("crew line = %+v", l)
	}
	door := LineFor("kubectl expose pod porter-lone-tower --name=parcels --port=80 --target-port=8080", nil)
	if door.Action == nil || door.Action.Action != "openDepot" || door.Action.Kind != "parcel" {
		t.Errorf("the door's expose still opens the parcels depot: %+v", door)
	}
	spaced := LineFor("expose deployment porters --name parcels", offers)
	if spaced.Action == nil || spaced.Action.Kind != "parcel" || spaced.Label != openParcels.Label {
		t.Errorf("--name with a space: %+v", spaced)
	}
	if l := LineFor("kubectl get pods", offers); l.Action != nil || l.Command != "kubectl get pods" {
		t.Errorf("a reading verb has no button twin: %+v", l)
	}
	if l := LineFor("kubectl create deployment haulers --image=hauler:1.0", offers); l.Action != nil {
		t.Errorf("no offer, no twin: %+v", l)
	}
}

func TestDepotAndCrewLinesPreferTheObjectiveText(t *testing.T) {
	f := offerFrame(t, raiseClerks, openParcels)
	if l, ok := DepotLine(f); !ok || l.Command != openParcels.Command || l.Action.Kind != "parcel" {
		t.Errorf("depot line = %+v", l)
	}
	if l, ok := CrewLine(f); !ok || l.Command != raiseClerks.Command {
		t.Errorf("crew line = %+v", l)
	}
	f.State.Objectives[0].Command = "kubectl create deployment clerks --image=clerk:1.0 --replicas=2"
	if l, _ := CrewLine(f); l.Command != f.State.Objectives[0].Command || l.Action == nil {
		t.Errorf("the task's words win: %+v", l)
	}
	quiet := offerFrame(t)
	if _, ok := CrewLine(quiet); ok {
		t.Error("no offer, no crew line")
	}
	if _, ok := DepotLine(quiet); ok {
		t.Error("no offer and no depot task, no depot line")
	}
	quiet.State.Objectives[0].Target = CmdDepot
	if l, ok := DepotLine(quiet); !ok || l.Command != "" || l.Action == nil || l.Action.Kind != "parcel" {
		t.Errorf("a DEPOT task without words still has its button: %+v", l)
	}
}

func TestObjectiveLineOffersTheDepotAndCrewChips(t *testing.T) {
	f := offerFrame(t, raiseClerks, openParcels)
	f.State.Objectives[0].Target = CmdCrew
	if l, ok := f.ObjectiveLine(); !ok || l.Command != raiseClerks.Command || l.Runnable() {
		t.Errorf("a CREW task's chip is the Stone alternative, the button does it: %+v", l)
	}
	if f.SpeakCommand() != raiseClerks.Command {
		t.Errorf("speak command = %q", f.SpeakCommand())
	}
	f.State.Objectives[0].Target = CmdDepot
	if l, _ := f.ObjectiveLine(); l.Command != openParcels.Command {
		t.Errorf("depot task chip = %+v", l)
	}
	f.State.Objectives[0].Target = "cmd:build"
	if _, ok := f.ObjectiveLine(); ok {
		t.Error("other targets carry no chip")
	}
	f.State.Objectives = nil
	if _, ok := f.ObjectiveLine(); ok {
		t.Error("no task, no chip")
	}
}

func TestTheCrewButtonTakesAFreeCellAndDepotIsGone(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := offerFrame(t, raiseClerks, openParcels)
	buttons := h.CommandButtons(f)
	crew, okC := HitID(buttons, CmdCrew)
	if _, okD := HitID(buttons, CmdDepot); okD || !okC {
		t.Fatalf("depot %v crew %v", okD, okC)
	}
	for _, b := range buttons {
		if b.ID != crew.ID && Overlaps(b.Rect, crew.Rect) {
			t.Errorf("%s overlaps an offer button", b.ID)
		}
	}
	if crew.Tip == "" {
		t.Error("offer buttons explain themselves")
	}
	full := offerFrame(t, raiseClerks, openParcels)
	full.State.Reveals = append(append([]string(nil), allReveals...), "depot", "crew")
	crowded := h.CommandButtons(full)
	for i, a := range crowded {
		for _, b := range crowded[i+1:] {
			if Overlaps(a.Rect, b.Rect) {
				t.Errorf("a full card never stacks %s on %s", a.ID, b.ID)
			}
		}
	}
	none := offerFrame(t)
	none.Guide = Guide{Staged: true, Intro: &Intro{ID: "depot", Target: CmdDepot, Via: ViaOpen}}
	if _, ok := HitID(h.CommandButtons(none), CmdDepot); ok {
		t.Error("the DEPOT intro leads to BUILD, not to a DEPOT button")
	}
}

func TestTheDepotChipPrefillsTheStone(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := offerFrame(t, openParcels)
	h.UseLine(OfferLine(openParcels), FromDepotCard)
	if h.Overlay() != OverlayStone || string(h.Stone.Input) != "expose deployment porters --name=parcels --port=80 --target-port=8080" {
		t.Fatalf("stone %q %q", h.Overlay(), string(h.Stone.Input))
	}
	do, ok := HitID(h.Buttons(f), StoneDo)
	if !ok {
		t.Fatal("the Stone offers DO IT for a pre-filled offer")
	}
	for _, b := range h.StoneButtons() {
		if b.ID != StoneDo && Overlaps(b.Rect, do.Rect) {
			t.Errorf("DO IT overlaps %s", b.ID)
		}
	}
	intents := clickOn(t, h, f, StoneDo)
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "openDepot" || it.From != FromStoneButton {
		t.Fatalf("DO IT sends the action: %+v", intents)
	}
	if h.Overlay() != OverlayNone {
		t.Error("DO IT closes the Stone")
	}
	h.UseLine(OfferLine(openParcels), FromDepotCard)
	h.Update(Input{Chars: []rune("x")}, f)
	if _, ok := HitID(h.Buttons(f), StoneDo); ok {
		t.Error("an edited command has no button twin")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	noCrews := offerFrame(t)
	noCrews.Crews = nil
	if got := h.OpenRaise(noCrews); h.Overlay() != OverlayRaise || got == nil {
		t.Error("CREW without an offer still opens its chooser")
	}
}

func TestDepotIntroAcksWhenItsStallIsPressed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := offerFrame(t, openParcels)
	f.State.Crews = []view.CrewView{{Name: "porters", Replicas: 2}}
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "depot", Target: CmdDepot, Via: ViaOpen}, Introduced: []string{"cmd:serve", "cmd:build"}}
	if w := h.WalkGuide(f); !w.OK() || len(w.Steps) != 2 || w.Steps[0] != "cmd:build" || w.Steps[1] != StallTarget("parcel") {
		t.Fatalf("the DEPOT intro walks BUILD, then the stall: %+v", w)
	}
	h.PageID = PageBuild
	intents := clickOn(t, h, f, StallTarget("parcel"))
	acked := false
	for _, it := range intents {
		acked = acked || it.Kind == IntentAction && it.Action.Action == "ack" && it.Action.ID == "depot"
	}
	if !acked {
		t.Errorf("pressing DEPOT completes its intro: %+v", intents)
	}
}

func TestObjectiveChipGainsDoIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := offerFrame(t, raiseClerks)
	f.State.Objectives[0].Command = "kubectl create deployment clerks --image=clerk:1.0 --replicas=2"
	card := h.ObjectiveRectOf(f)
	chip, okC := HitID(h.ObjectiveButtons(f), "obj:speak")
	do, okD := HitID(h.ObjectiveButtons(f), "obj:doit")
	if !okC || !okD || Overlaps(chip.Rect, do.Rect) || !contains(card, chip.Rect) || !contains(card, do.Rect) {
		t.Fatalf("chip %+v do %+v card %+v", chip.Rect, do.Rect, card)
	}
	intents := clickOn(t, h, f, "obj:doit")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "raiseCrew" {
		t.Errorf("DO IT raises the crew: %+v", intents)
	}
	clickOn(t, h, f, "obj:speak")
	if _, ok := HitID(h.Buttons(f), StoneDo); !ok {
		t.Error("the chip's Stone offers DO IT too")
	}
}

func TestCardLinesOfferTheDepotsCrew(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := offerFrame(t, raiseClerks, openParcels)
	f.State.Reveals = append(f.State.Reveals, "stone")
	f.Selection = Selection{Kind: "structure", ID: "d1", Structure: view.Structure{ID: "d1", Kind: "depot-ledger", Label: "The ledgers depot",
		Object: view.ObjectRef{Kind: "Service", Name: "ledgers"}, State: view.StateReady}}
	lines := CardLines(f)
	if len(lines) != 1 || lines[0].Command != raiseClerks.Command {
		t.Fatalf("card lines = %+v", lines)
	}
	do, ok := HitID(h.CommandButtons(f), "line:card:0:do")
	if !ok || do.Rect != CellRect(2) || !strings.Contains(do.Tip, "$ kubectl create deployment clerks") {
		t.Fatalf("the depot's offer is no grid cell: %+v", do)
	}
	intents := clickOn(t, h, f, "line:card:0:do")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Name != "clerks" {
		t.Errorf("card DO IT: %+v", intents)
	}
	f.Selection.Structure = view.Structure{ID: "c", Kind: view.StructChancery, State: view.StateReady}
	if len(CardLines(f)) != 1 {
		t.Error("the Chancery's two rows of commands leave room for one offer")
	}
	f.Selection.Structure = view.Structure{ID: "k", Kind: view.StructKeep, State: view.StateReady}
	if len(CardLines(f)) != 0 {
		t.Error("a Keep has no depot offers")
	}
	if h.PressLine("nope:speak", f) != nil || h.PressLine("card:9:do", f) != nil {
		t.Error("unknown lines do nothing")
	}
}

func TestOfferPathsDrawAndFallBack(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := offerFrame(t, raiseClerks)
	f.State.Objectives[0].Command = raiseClerks.Command
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	f.State.Freight.Kinds = append(f.State.Freight.Kinds, view.KindView{Kind: "ledger", Name: "ledgers", Open: true, Depot: "ledgers", CrewName: "clerks"})
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "clerk:1.0", Owned: true})
	openChooser(t, h, f, "clerks")
	intents := clickOn(t, h, f, RaiseConfirm)
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "raiseCrew" || it.Action.Name != "clerks" || it.From != FromChooser || h.Overlay() == OverlayStone {
		t.Fatalf("CREW raises the offered crew through its chooser: %+v", intents)
	}
	clickOn(t, h, f, ObjSpeak)
	if string(h.Stone.Input) != BareCommandOf(raiseClerks.Command) {
		t.Fatalf("the task chip pre-fills: %q", string(h.Stone.Input))
	}
	h.Draw(dst, f)
	if intents := h.PressPrefill(); len(intents) != 1 {
		t.Errorf("the Stone's DO IT acts: %+v", intents)
	}
	if h.PressPrefill() != nil {
		t.Error("after DO IT the Stone has nothing pre-filled")
	}
	empty := offerFrame(t)
	empty.Guide = Guide{Staged: true, Intro: &Intro{ID: "crew", Target: CmdCrew, Via: ViaOpen}}
	h.OverlayID = OverlayNone
	empty.Crews = nil
	clickOn(t, h, empty, CmdCrew)
	if h.Overlay() != OverlayRaise {
		t.Error("a CREW intro with no offer opens the chooser")
	}
	if l := (CommandLine{}); len(LineButtons("x", ui.Rect{W: 100, H: 12}, l)) != 0 {
		t.Error("an empty line has no buttons")
	}
	if FlagValue([]string{"expose", "--name"}, "--name") != "" {
		t.Error("a dangling flag has no value")
	}
}

func TestATryWithItsOwnButtonAndNoCommandOffersNothingToSpeak(t *testing.T) {
	f := offerFrame(t, raiseClerks, openParcels)
	f.State.Objectives[0].ID, f.State.Objectives[0].Target, f.State.Objectives[0].Button = "try:vault", CmdCrew, "kind:verb:dig"
	f.State.Objectives[0].Command = ""
	if l, ok := f.ObjectiveLine(); ok {
		t.Fatalf("the vault try borrows another step's command: %+v", l)
	}
	f.State.Objectives[0].Command = "kubectl scale deployment archive --replicas=2"
	if l, ok := f.ObjectiveLine(); !ok || l.Command != f.State.Objectives[0].Command {
		t.Fatalf("a try with its own command lost it: %+v", l)
	}
}
