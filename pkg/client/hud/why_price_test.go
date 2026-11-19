package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func lockedGalleons() Button {
	up := Button{ID: "port:raise:gate", Label: ui.Bridge("UPGRADE: GALLEONS", "ADD HTTPS LISTENER"), Icon: "icon/rank-high", Cost: "200K", Rect: CellRect(5)}
	return disabledBy(up, "galleons sail only to a realm that holds two Keeps")
}

func TestALockedTwoRowTilesReasonNeverSitsUnderItsPrice(t *testing.T) {
	h := New(nil, Backend{})
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		b := lockedGalleons()
		if price, ok := h.priceRect(b); ok && Overlaps(price, WhyStrip(b)) {
			t.Errorf("%s: the figure %q is drawn over the lock reason %q, so the tile reads STAGI200K", mode, b.Cost, ui.Name(b.Why))
		}
		ui.SetNamer(nil)
	}
}

func TestALockedTwoRowTilesReasonIsNotCut(t *testing.T) {
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		b := lockedGalleons()
		if shown, room := ui.Name(cellWhy(b)), ui.MaxChars(WhyStrip(b).W-2, 1); len([]rune(shown)) > room {
			t.Errorf("%s: the lock reason %q is cut to %d characters", mode, shown, room)
		}
		ui.SetNamer(nil)
	}
}

func TestEveryLockedReasonFitsItsCellInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	room := ui.MaxChars(CellW-4-2, 1)
	reasons := []string{"Needs The Chancery.", "Needs the Codex.", "Needs a Keep", "Needs The Quartermaster", "Needs The Stewards' Hall", "Needs 300 gold"}
	for _, r := range whyRules {
		reasons = append(reasons, r.has)
	}
	for _, reason := range reasons {
		if shown := ui.Name(WhyOf(reason)); len([]rune(shown)) > room {
			t.Errorf("%q reads %q in NAMES mode, %d characters, a cell holds %d", reason, shown, len([]rune(shown)), room)
		}
	}
}

func TestTheHarborsLockReasonsUseTheHarborsOwnWordsInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	for reason, want := range map[string]string{
		"Gate Harbor's quay holds 5 berths at level 2: upgrade it for more": "UPGRADE",
		"no berth is open at Gate Harbor":                                   "NO RAMP",
	} {
		if got := ui.Name(WhyOf(reason)); got != want {
			t.Errorf("%q reads %q in NAMES mode, want %q", reason, got, want)
		}
	}
}
