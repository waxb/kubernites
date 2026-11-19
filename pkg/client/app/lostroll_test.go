package app

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheBadgesFirstActGlidesOnceToTheDepot(t *testing.T) {
	hand := view.RollView{Hand: true, Entries: []view.RollEntry{{Hut: "porter-a", Live: true}, {Hut: "porter-b"}}}
	looks := view.RollView{Hand: true, LooksFor: true, Listed: []string{"porter-a", "porter-c"}}
	if !BadgeFirstAct(hand, looks) {
		t.Fatal("the depot's first look for a badge does not glide to it")
	}
	if BadgeFirstAct(looks, looks) || BadgeFirstAct(view.RollView{}, looks) {
		t.Fatal("the glide repeats, or fires on a loaded realm")
	}
	if got := fallenOnRoll(hand); len(got) != 1 || got[0] != "porter-b" {
		t.Fatalf("fallen huts on the hand roll = %v", got)
	}
	if got := fallenOnRoll(looks); len(got) != 0 {
		t.Fatalf("a roll the Stewards keep has no fallen huts to puff: %v", got)
	}
}

func TestTheStoryNamerNamesAHutsStructuredTip(t *testing.T) {
	for _, why := range []string{"0/3 nodes are available: 1 Insufficient cpu, 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }.",
		"0/3 nodes are available: 1 Insufficient cpu, 1 node(s) had untolerated taint(s)."} {
		got := newStoryNamer()(hud.HutTip(model.PodView{Name: "porter-a"}, view.PodMetrics{}, why, ""))
		for _, word := range []string{"nodes", "cpu", "control-plane", "taint"} {
			if strings.Contains(got, word) {
				t.Fatalf("the story namer leaves %q in a hut's tip: %q", word, got)
			}
		}
	}
}
