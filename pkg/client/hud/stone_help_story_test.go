package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestStoryHelpKeepsEveryVerbTheGuideSpeaks(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	help := []string{
		"  kubectl rollout status|history|undo deployment/<name>",
		"  kubectl autoscale deployment|rs <name> --max=N [--min=N]",
		"  kubectl set image deployment/<name> <container>=<image>",
		"  kubectl label <type> <name> k=v [k-] [--overwrite]",
	}
	if out := StoryConsole(help); len(out) != len(help) {
		t.Fatalf("story help keeps %d of %d command lines: %q", len(out), len(help), out)
	}
	if CommandSyntax("the rollout stalled on an image") || !CommandSyntax("  "+lexicon.StoryCommand+" rollout undo crew/porters") {
		t.Fatal("only a line that starts with the Stone's own command is read as syntax")
	}
}
