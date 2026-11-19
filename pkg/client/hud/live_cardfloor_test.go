package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestAFullLiveCardKeepsItsRowsAboveTheCardFloor(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		h, _ := liveHUD(t, false)
		h.SetLive(LiveCard{Context: "kind-a", Server: "https://127.0.0.1:6443", Who: "admin", Keeps: 3, Isles: 1, Huts: 20, Fiefs: 4, Shrines: 2,
			Stale: "14:09", Draining: "DRAINING node-a", Trouble: 2, Level: LevelSeamless, Refused: []string{"pods"},
			Quiet: []QuietTrouble{{ID: "q", Kind: "q", Cause: "x"}}, Idle: []string{"a", "b"}})
		var runs []ui.TextRun
		stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
		h.drawLiveCard(ebiten.NewImage(Width, Height))
		stop()
		if len(runs) < 3 {
			t.Fatalf("names %v: %d rows drawn", ui.Naming(), len(runs))
		}
		for _, r := range runs {
			if r.Rect.Y+ui.LineHeight > cardFloor() {
				t.Errorf("names %v: %q at y %d ends at %d, below the card floor %d", ui.Naming(), r.Text, r.Rect.Y, r.Rect.Y+ui.LineHeight, cardFloor())
			}
		}
		for _, l := range h.live.shown() {
			if l.id == liveIdleID {
				t.Errorf("names %v: a full card keeps %q and drops another row first", ui.Naming(), l.text)
			}
		}
	}
}
