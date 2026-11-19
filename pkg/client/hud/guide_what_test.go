package hud

import "testing"

func TestTheSpotlightTitleKeepsAClassName(t *testing.T) {
	for in, want := range map[string]string{
		"Roll porter:1.1 out hut by hut: the shift keeps the old class until the new one stands": "Roll porter:1.1 out hut by hut",
		"Raise the Quartermaster: someone must place every new hut":                              "Raise the Quartermaster",
		"Hold 99% ready": "Hold 99% ready",
	} {
		if got := guideWhat(in); got != want {
			t.Errorf("guideWhat(%q) = %q, want %q", in, got, want)
		}
	}
}
