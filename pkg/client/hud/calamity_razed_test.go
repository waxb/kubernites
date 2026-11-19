package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestARazedHutsCardStopsWaitingSoItsPauseLifts(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	open := view.CalamityView{ID: "frozen:porters-1", Kind: view.CalamityFrozen, Object: "pod/default/porters-1", First: true,
		Cause:    "porters-1 reads Running (up and ready), yet nothing leaves it",
		Remedies: []view.RemedyView{{Label: "Replace it by hand: raze the hut", Command: "kubectl delete pod porters-1"}}}
	f.State.Calamities = []view.CalamityView{open}
	runFor(h, &f, CalamitySettle+CalamityGap+8)
	if v, _, _, ok := h.CurrentCalamity(h.Stage(f)); !ok || v.ID != open.ID || !h.TroubleOpen(open.ID) {
		t.Fatal("the frozen hut's card never opened in this fixture")
	}
	done := open
	done.Remedies, done.Cause, done.Why = []view.RemedyView{}, "", ""
	done.Mended = "Done: porters-1 is razed, and its crew raises a new hut in its place."
	f.State.Calamities = []view.CalamityView{done}
	runFor(h, &f, 1)
	if h.TroubleOpen(open.ID) || h.TroubleSetAside(open.ID) {
		t.Fatal("the card's hut is razed, yet its trouble still waits for an answer, so the pause holds")
	}
	for _, l := range h.calamityLines(h.Stage(f)) {
		if strings.Contains(l.Command, "delete") {
			t.Fatalf("the answered card still offers a raze: %+v", l)
		}
	}
}
