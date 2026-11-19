package hud

import (
	"strings"
	"testing"
)

func TestTheDawnNeverAsksForNinesAlreadyHeldWithoutSayingForHowLong(t *testing.T) {
	h := New(nil, Backend{})
	f := Frame{}
	f.State.SLO.Nines = 3
	for _, text := range append(h.FoundationLines(f), foundationCaption, loomTeaser) {
		if strings.Contains(text, "Two Nines") && !strings.Contains(text, "minutes") {
			t.Fatalf("with 3 nines of availability held, %q reads as unmet: it must say the figure must hold for a while", text)
		}
	}
}
