package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var timingOut = view.CalamityView{ID: "spoiling:parcel", Kind: view.CalamitySpoiling, Object: "kind/parcel",
	Cause: "Parcels time out faster than the porters carry them: 42 of 84 lost in 30 s",
	Why:   "The porters on lone-tower get too few Hands: the Keep's Hands run out, so each carries slower (throttled).",
	Remedies: []view.RemedyView{
		{Label: "Add a Keep for more porters", Action: &view.Action{Action: "buyKeep", For: "porters"}},
		{Label: "Close a berth: fewer ships until the crews cope", Action: &view.Action{Action: view.ActCloseBerth}},
	}}

func TestCalamityRemediesReadInFullInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range []string{"story", "names"} {
		ui.SetNamer(nil)
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		h := New(nil, Backend{})
		f := settled(h, calamityFrame(t, timingOut))
		card, ok := h.CalamityRect(f)
		if !ok {
			t.Fatalf("[%s] no calamity card", mode)
		}
		remedies := 0
		for _, b := range h.CalamityButtons(f) {
			if !strings.HasSuffix(b.ID, ":"+DoOp) {
				continue
			}
			remedies++
			lines := h.buttonLines(b)
			if joined := strings.Join(lines, " "); joined != ui.Name(b.Label) || strings.Contains(joined, "...") {
				t.Errorf("[%s] %q reads %q", mode, ui.Name(b.Label), lines)
			}
			for _, l := range lines {
				if w := h.Font.Width(l, 1); w > b.Rect.W-4 {
					t.Errorf("[%s] %q is %d wide in a %d button", mode, l, w, b.Rect.W)
				}
			}
			if b.Rect.H < len(lines)*ui.LineHeight || !contains(card, b.Rect) {
				t.Errorf("[%s] %s %+v holds %d lines inside %+v", mode, b.ID, b.Rect, len(lines), card)
			}
		}
		if remedies != 2 {
			t.Fatalf("[%s] %d remedy buttons", mode, remedies)
		}
		bounds := func(h *HUD) ui.Rect { r, _ := h.CalamityRect(f); return r }
		assertTextFits(t, h, f, textPanel{name: "calamity " + mode, bounds: bounds,
			draw: (*HUD).DrawCalamity, buttons: (*HUD).CalamityButtons})
	}
}

func TestShortRemediesStayOneLine(t *testing.T) {
	h := New(nil, Backend{})
	f := settled(h, calamityFrame(t, timingOut))
	for _, b := range h.CalamityButtons(f) {
		if strings.HasSuffix(b.ID, ":0:"+DoOp) && (b.Rect.H != ChipH || len(h.buttonLines(b)) != 1) {
			t.Errorf("a short remedy grew: %+v %q", b.Rect, h.buttonLines(b))
		}
	}
}

func TestAKeepFixWithItsUpkeepReadsInFull(t *testing.T) {
	staging := timingOut
	staging.Remedies = []view.RemedyView{
		{Label: "Add a Keep for more clerks (11,000 MiB, overhead 18 requests a second)", Action: &view.Action{Action: "buyKeep", For: "clerks"}},
		timingOut.Remedies[1],
	}
	h := New(nil, Backend{})
	f := settled(h, calamityFrame(t, staging))
	card, _ := h.CalamityRect(f)
	for _, b := range h.CalamityButtons(f) {
		if !strings.HasSuffix(b.ID, ":"+DoOp) {
			continue
		}
		lines := h.buttonLines(b)
		if joined := strings.Join(lines, " "); strings.Contains(joined, "...") || !strings.HasSuffix(joined, "SECOND)") && !strings.HasSuffix(joined, "COPE") || b.Rect.H < len(lines)*ui.LineHeight || !contains(card, b.Rect) {
			t.Errorf("%q reads %q in %+v", ui.Name(b.Label), lines, b.Rect)
		}
	}
}
