package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestANeedsReasonKeepsWhatIsNeededWhereTheCellCutsIt(t *testing.T) {
	for reason, want := range map[string]string{
		"Needs The Chancery.":      "CHANCERY",
		"Needs the Codex.":         "NEEDS CODEX",
		"Needs a Keep":             "NEEDS KEEP",
		"Needs The Quartermaster":  "QMASTER",
		"Needs The Stewards' Hall": "STEWARDS",
		"Needs 300 gold":           "NEEDS GOLD",
	} {
		if got := WhyOf(reason); got != want {
			t.Errorf("%q reads %q, want %q", reason, got, want)
		}
	}
}

func TestEveryLockedReasonFitsItsCellWithoutAnEllipsis(t *testing.T) {
	room := ui.MaxChars(CellW-4-2, 1)
	reasons := []string{"HIRE FIRST", "PODS FIRST", "QMASTER", "STEWARDS"}
	for _, r := range whyRules {
		reasons = append(reasons, r.short)
	}
	for _, short := range reasons {
		if len([]rune(short)) > room {
			t.Errorf("%q is %d characters, a cell holds %d", short, len([]rune(short)), room)
		}
	}
	if WhyMaxChars > room {
		t.Errorf("whyMaxChars %d exceeds the cell's %d", WhyMaxChars, room)
	}
}

func TestAStrikeButtonNamesTheActionAndKeepsTheHutForItsTip(t *testing.T) {
	f := fullFrame(t)
	f.State.Roll.Hand, f.State.Roll.Entries = true, []view.RollEntry{{Hut: "porter-2v44t-extra-long", Address: "10.244.0.3", Live: false}}
	acts := withRollDepot(nil, parcelsService, f)
	for _, a := range acts {
		if a.Key == keyRollStrike {
			if a.Label != "STRIKE" || !strings.Contains(a.Tip, "porter-2v44t-extra-long") {
				t.Fatalf("strike label %q tip %q", a.Label, a.Tip)
			}
			return
		}
	}
	t.Fatal("no STRIKE action")
}

func TestTheDepotTileOpensOnTheThirdHire(t *testing.T) {
	f := offerFrame(t, openParcels)
	f.State.Freight.Kinds[0].Door = true
	f.Guide = Guide{Staged: true, Introduced: []string{"cmd:serve", "cmd:build"}}
	tile := func() (Button, bool) {
		for _, b := range stallButtons(f) {
			if b.ID == StallTarget("parcel") {
				return b, true
			}
		}
		return Button{}, false
	}
	f.State.Roll.Hired = 2
	if b, ok := tile(); !ok || !b.Disabled {
		t.Fatalf("with two porters hired the parcels depot tile is open: %+v", b)
	}
	f.State.Roll.Hired = 3
	if b, _ := tile(); b.Disabled || strings.Contains(b.Why, "HIRE FIRST") {
		t.Fatalf("three porters are hired, yet the depot tile reads %q (\"HIRE FIRST\", \"Opens once three porters are hired\")", b.Why)
	}
}
