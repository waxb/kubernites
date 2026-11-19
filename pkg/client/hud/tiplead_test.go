package hud

import (
	"strings"
	"testing"
)

func TestTooltipsOpenWithAPlainSentenceThenTheDetail(t *testing.T) {
	buttons := WithLeads([]Button{
		{ID: "cmd:door", Tip: "Hut at the door (static Pod): runs from a file"},
		{ID: "summon:porter:1.0", Tip: "Summon a porter. It eats a Keep's food."},
		{ID: "top:pause", Tip: "Pause the realm [Space]"},
		{ID: "cmd:summon"},
	}, Frame{})
	for _, b := range buttons[:2] {
		lead, _, ok := strings.Cut(b.Tip, "\n")
		if !ok || !strings.HasSuffix(lead, ".") || strings.Contains(lead, "(") {
			t.Errorf("%s does not open with one plain sentence: %q", b.ID, b.Tip)
		}
	}
	if buttons[2].Tip != "Pause the realm [Space]" || buttons[3].Tip != "" {
		t.Errorf("a lead was added where none belongs: %+v", buttons[2:])
	}
	if again := WithLeads(buttons, Frame{}); again[0].Tip != buttons[0].Tip {
		t.Errorf("the lead was added twice: %q", again[0].Tip)
	}
}
