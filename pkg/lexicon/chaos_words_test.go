package lexicon

import (
	"strings"
	"testing"
)

func TestTheChaosEventReadsInEachModesWords(t *testing.T) {
	msg := ChaosEventMessage
	if strings.Contains(msg, "Simian") || strings.Contains(msg, "hut") || !strings.Contains(msg, "chaos monkey") {
		t.Errorf("the Event in the API reads %q: an Event holds Kubernetes words, and story mode translates", msg)
	}
	if story := StoryText(StoryNames(msg)); !strings.Contains(story, "Simian Imp") || strings.Contains(story, "chaos monkey") || strings.Contains(story, "Pod") {
		t.Errorf("story mode reads the chaos Event as %q", story)
	}
}
