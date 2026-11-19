package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestStoryModeOffersNoRawPatchToSpeak(t *testing.T) {
	patch := `kubectl patch deployment porters -p '{"spec":{"template":{"spec":{"containers":[{"name":"porter","livenessProbe":{"httpGet":{"path":"/healthz","port":8080}}}]}}}}'`
	r := view.RemedyView{Label: "Set the Watchman's Bell", Command: patch}
	ui.SetStoryNamer(func(s string) string { return s })
	story := RemedyLine(r)
	tip := SpeakTip(patch)
	ui.SetNamer(nil)
	if story.Command != "" || tip != "" {
		t.Fatalf("story mode shows a raw patch to speak: line %q, tip %q", story.Command, tip)
	}
	if RemedyLine(r).Command != patch || SpeakTip("kubectl scale deployment porters --replicas=3") == "" {
		t.Fatal("outside story mode the Stone's words are hidden")
	}
}
