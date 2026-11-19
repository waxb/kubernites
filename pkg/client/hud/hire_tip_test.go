package hud

import (
	"strings"
	"testing"
)

func hireTip(t *testing.T, h *HUD, f Frame) string {
	t.Helper()
	for _, b := range h.Buttons(f) {
		if b.ID == "cmd:summon" {
			return b.Tip
		}
	}
	t.Fatal("no HIRE button")
	return ""
}

func TestTheHireTipLeadsWithWhatTheClickDoesAndCosts(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Almanac = append(f.State.Almanac, "hut")
	tip := hireTip(t, h, f)
	first := strings.SplitN(tip, "\n", 2)[0]
	if !strings.HasPrefix(first, "Hire a porter by hand") || !strings.Contains(tip, "costs no") {
		t.Errorf("the HIRE tip opens with %q", first)
	}
	if strings.Contains(tip, "stands alone") {
		t.Errorf("with the Hut page met, the rule goes behind WHY? (its page says a bare hut razed is gone for good), yet the tip reads:\n%s", tip)
	}
	f.State.Almanac = nil
	tip = hireTip(t, h, f)
	if !strings.Contains(tip, "stands alone") || strings.HasPrefix(tip, "A hut placed by hand") {
		t.Errorf("with no page to open, the rule stays, after what the click does:\n%s", tip)
	}
	if lines := strings.Split(tip, "\n"); strings.Contains(lines[len(lines)-1], "stands alone") && strings.Contains(tip, "Click:") {
		t.Errorf("the rule comes before the click note:\n%s", tip)
	}
}
