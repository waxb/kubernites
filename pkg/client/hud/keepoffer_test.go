package hud

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func keepFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.Site.Structures = append(f.Site.Structures, view.Structure{ID: "k", Kind: view.StructKeep, Variant: "tower", W: 3, H: 3})
	f.State.KeepShop = []view.KeepOffer{{KeepType: "stone-keep", Name: "Stone Keep", CPU: "4", Memory: "8Gi", Territory: 4, Class: "balanced", Instance: "general-xlarge",
		Available: true, Effect: "+4 Hands and 8Gi of Lore (balanced land); overhead 0.35 MiB/s"}}
	f.State.Buildables = append(f.State.Buildables, view.Buildable{Kind: "keep", Name: "Keep", K8s: "Node", Description: "A machine", Cost: 150, W: 3, H: 3, Available: true})
	f.State.Reveals = append(f.State.Reveals, "keeps")
	return f
}

func TestKeepOffersShowClassMachineAndBorders(t *testing.T) {
	f := keepFrame(t)
	o, ok := KeepOfferFor(f, "keep")
	if !ok || o.Instance != "general-xlarge" {
		t.Fatalf("offer = %+v", o)
	}
	line := KeepOfferLine(o)
	for _, want := range []string{"general-xlarge", "balanced land", "4,000 rations of food", "8,192 rooms of housing", "4 tiles"} {
		if !strings.Contains(line, want) {
			t.Errorf("%q misses %q", line, want)
		}
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.PageID = PageBuild
	b, found := HitID(h.CommandButtons(f), "build:keep")
	if !found || !strings.Contains(b.Tip, "general-xlarge") {
		t.Errorf("the Keep button tells the machine: %+v", b)
	}
	if _, ok := KeepOfferFor(f, "codex"); ok {
		t.Error("only Keeps have offers")
	}
	bare := KeepOfferLine(view.KeepOffer{K8s: "Node", CPU: "2", Memory: "4Gi"})
	if !strings.Contains(bare, "open land") || !strings.Contains(bare, "Node") {
		t.Errorf("bare offer = %q", bare)
	}
}

func TestKeepCardsNameTheirLand(t *testing.T) {
	s := view.Structure{ID: "k2", Kind: view.StructKeep, Variant: "stone", Label: "Stone Keep 2", Class: "compute", CPU: 8, MemoryGiB: 10, Instance: "compute-xlarge", Territory: 4, State: view.StateReady}
	lines := KeepLines(s)
	if len(lines) != 2 || lines[0].Text != "compute-xlarge: "+ui.IconFood+"8,000 "+ui.IconHousing+"10,240" || lines[1].Text != "compute land, reach 4 tiles" {
		t.Errorf("keep lines = %+v", lines)
	}
	card := StructureCard(s, fullFrame(t))
	joined := ""
	for _, l := range card.Lines {
		joined += l.Text + "|"
	}
	if !strings.Contains(joined, "compute-xlarge") {
		t.Errorf("the card carries the machine: %q", joined)
	}
	if KeepLines(view.Structure{Kind: view.StructKeep}) != nil {
		t.Error("an old Keep without a survey adds nothing")
	}
}

func TestALockedKeepCellSaysWhyAfterItsOwnWords(t *testing.T) {
	f := fullFrame(t)
	b := view.Buildable{Kind: "keep", Name: "Stone Keep", K8s: "Node", Description: "A long description of a Node.", Available: false, Locked: true, Reason: "Needs a second Keep: node-a holds two Keeps"}
	if got := KeepButton(b, f).Tip; !strings.Contains(got, "Needs a second Keep: node-a holds two Keeps.") || !strings.Contains(got, "long description") || !strings.Contains(got, "Costs") {
		t.Fatalf("locked keep tip = %q", got)
	}
}

func TestAKeepTileShowsItsUpkeepAndItsOfferWarnsWhenItWouldStandEmpty(t *testing.T) {
	f := fullFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, Name: "Stone Keep", Cost: 6400, UpkeepPerSec: 10.8,
		Idle: "Nothing waits for room now: a new Stone Keep would stand empty."}}
	b := KeepButton(view.Buildable{Kind: view.KeepTypeStone, Name: "Stone Keep", Cost: 6400, Available: true}, f)
	rows := CellLabel(b.Label, ui.MaxChars(CellW-4, 1))
	if len(rows) != 2 || rows[0] != "-10.8/S" || strings.Contains(rows[1], "...") {
		t.Fatalf("the Keep tile reads %q, want its name over its overhead", rows)
	}
	if !strings.Contains(b.Tip, "Nothing waits for room now") {
		t.Fatalf("the Keep's tip does not warn it would stand empty: %q", b.Tip)
	}
}

func TestALockedTileShowsItsGateNotAPrice(t *testing.T) {
	f := fullFrame(t)
	archive := view.Buildable{Kind: "archive", Name: "Archive", Cost: 300, Available: false, Locked: true, Reason: "Archive comes with the Feudal Age"}
	f.State.Buildables = []view.Buildable{archive}
	found := false
	for _, b := range (&HUD{}).BuildButtons(f) {
		if b.ID != "build:archive" {
			continue
		}
		found = true
		if b.Cost != "" || b.Why != "FEUDAL AGE" {
			t.Fatalf("the Archive tile reads %q over %q: a gate label shows a price", b.Why, b.Cost)
		}
	}
	if !found {
		t.Fatal("no Archive tile")
	}
	keep := view.Buildable{Kind: "keep", Name: "Stone Keep", Cost: 170, Available: false, Locked: true, Reason: "Needs a second Keep: node-a holds two Keeps"}
	if b := KeepButton(keep, f); b.Cost != "" || b.Why != "TWO KEEPS" {
		t.Fatalf("the third Keep's tile reads %q over %q with two Keeps standing", b.Why, b.Cost)
	}
	short := view.Buildable{Kind: "keep", Name: "Stone Keep", Cost: 170, Available: false, Reason: "Needs 170 MiB (you have 20)"}
	if b := KeepButton(short, f); b.Cost == "" {
		t.Fatal("a tile waiting only for memory hides its size")
	}
}
