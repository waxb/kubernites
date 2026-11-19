package hud

import (
	"strings"
	"testing"
)

func TestAHeldStepNeverShowsACountdown(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := Frame{}
	f.State.Pause = 44.2
	f.State.Reveals = []string{"speed"}
	for _, l := range h.ObjectiveLines(f) {
		if strings.Contains(l.Text, " s") || strings.Contains(l.Text, "x5") {
			t.Fatalf("a held step shows a countdown: %+v", h.ObjectiveLines(f))
		}
	}
}
