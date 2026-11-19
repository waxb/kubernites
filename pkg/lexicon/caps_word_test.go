package lexicon

import "testing"

func TestAnAllCapsLabelStaysAllCapsInTheStory(t *testing.T) {
	for k8s, want := range map[string]string{"YAML": "SCRIPT", "LOGS": "LOGS"} {
		if got := StoryText(k8s); got != want {
			t.Errorf("StoryText(%q) = %q, want %q", k8s, got, want)
		}
	}
	if got := StoryText("Open the YAML of the pod"); got == "Open the SCRIPT of the hut" {
		t.Errorf("a word in a sentence turned to capitals: %q", got)
	}
}
