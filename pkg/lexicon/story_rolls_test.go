package lexicon

import "testing"

func TestTheRollsCommandSpeaksTheDepotsNameInStoryAndBack(t *testing.T) {
	kube := "kubectl get endpointslices -l kubernetes.io/service-name=parcels"
	story := StoryText(kube)
	if story != "speakstctl get rolls -l depot-name=parcels" {
		t.Fatalf("story mode reads %q (the label key kubernetes.io/service-name must read depot-name)", story)
	}
	if back := CommandFromStory(story); back != kube {
		t.Fatalf("the Stone hears %q as %q", story, back)
	}
	if got := TrueNames(kube); got != kube {
		t.Fatalf("NAMES mode rewrote the real command to %q", got)
	}
}
