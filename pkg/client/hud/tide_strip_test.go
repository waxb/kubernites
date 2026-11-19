package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheRushTideStripSaysHowManyBoxesTheTideAdds(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Siege.Phase, f.State.Siege.TimeLeft, f.State.Siege.Extra = "active", 120, 7.6
	f.State.Siege.Served, f.State.Siege.Target, f.State.Siege.SealsGoal = 0.995, 0.99, 8
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	text := []string{}
	for _, r := range recordRuns(func() { h.DrawSiegeStrip(dst, f) }) {
		text = append(text, r.Text)
	}
	if !strings.Contains(strings.Join(text, " "), "+7.6/S") {
		t.Fatalf("the traffic spike strip never says what the spike adds: %q", text)
	}
}
