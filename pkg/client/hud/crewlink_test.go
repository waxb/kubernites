package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestAHutCardLinksToItsCrewsBoard(t *testing.T) {
	defer ui.SetNamer(nil)
	f := crewSelFrame(Selection{Kind: "hearth", ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")})
	b, text, ok := crewLink(f)
	if !ok || b.ID != ChainButton || text != "crew porters: open the board" {
		t.Fatalf("story link = %+v %q %v", b, text, ok)
	}
	ui.SetNamer(names.Kubernetes(64))
	if _, text, _ := crewLink(f); text != "Deployment porters: open the board" {
		t.Fatalf("NAMES link = %q", text)
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.Update(Input{}, f)
	if !CardBox.Contains(b.Rect.X, b.Rect.Y) || !CardBox.Contains(b.Rect.X+b.Rect.W-1, b.Rect.Y+b.Rect.H-1) {
		t.Fatalf("the link sits outside the card: %+v", b.Rect)
	}
	if _, ok := HitID(h.SelectionButtons(f), ChainButton); !ok {
		t.Fatal("the hut card carries no clickable crew line")
	}
	h.Press(ChainButton, f)
	if !h.ChainShown(f) {
		t.Fatal("pressing the crew line does not open the board")
	}
}

func TestOnlyAHutOfACrewHasTheLink(t *testing.T) {
	f := crewSelFrame(crewSel())
	if _, _, ok := crewLink(f); ok {
		t.Error("a crew selection needs no link")
	}
	f.Selection = Selection{Kind: "hearth", ID: "default/bare", Pod: crewPod("bare")}
	f.Crews = nil
	if _, _, ok := crewLink(f); ok {
		t.Error("a bare hut has no crew to link")
	}
}

func TestAHutCardsCrewLinkReadsWholeInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	f := crewSelFrame(Selection{Kind: "hearth", ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")})
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		b, text, _ := crewLink(f)
		rows := crewLinkRows(text, b.Rect.W)
		if strings.Join(rows, " ") != ui.Name(text) {
			t.Errorf("%s: the link %q reads %q", mode, ui.Name(text), rows)
		}
		if b.Rect.H != len(rows)*ui.LineHeight || b.Rect.Y+b.Rect.H > CardBox.Y+CardBox.H {
			t.Errorf("%s: the link's %d rows sit in %+v", mode, len(rows), b.Rect)
		}
		ui.SetNamer(nil)
	}
}
