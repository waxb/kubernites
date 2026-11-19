package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestStoryDescribeCallsAHutsAskItsPledge(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(func(s string) string { return lexicon.StoryText(lexicon.StoryNames(s)) })
	out := strings.Join(StoryConsole([]string{
		"Name:         porter-x",
		"Containers:",
		"  porter:",
		"    Image:      porter:1.0",
		"    Limits:",
		"      cpu:     500m",
		"    Requests:",
		"      cpu:     100m",
	}), "\n")
	if strings.Contains(out, "Requests:") || strings.Contains(out, "Limits:") || !strings.Contains(out, "Pledges:") || !strings.Contains(out, "Bindings:") {
		t.Errorf("story describe says 'Boxes are requests' against 'Requests: Food: 100 rations': it reads\n%s", out)
	}
}
