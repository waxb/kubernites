package app

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

var describedNode = regexp.MustCompile(`(?i)describe (?:node|keep) (\S+)`)

func TestTheLiveStoneGreetsWithANodeTheClusterHas(t *testing.T) {
	defer ApplyNames(false)
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	var nodes []string
	for _, n := range rig.snap.Nodes {
		nodes = append(nodes, n.Name)
	}
	if len(nodes) == 0 {
		t.Fatal("the recorded cluster has no node")
	}
	first := slices.Min(nodes)
	for _, names := range []bool{false, true} {
		ApplyNames(names)
		var greeting []string
		for _, l := range rig.s.Hud.Stone.Output {
			greeting = append(greeting, ui.Name(l.Text))
		}
		text := strings.Join(greeting, "\n")
		named := describedNode.FindAllStringSubmatch(text, -1)
		for _, m := range named {
			if !slices.Contains(nodes, m[1]) {
				t.Errorf("[names %v] the live Stone suggests %q, but the cluster's nodes are %v:\n%s", names, m[0], nodes, text)
			}
		}
		if len(named) != 1 || named[0][1] != first {
			t.Errorf("[names %v] the live Stone does not suggest describing its first node %s:\n%s", names, first, text)
		}
	}
	if got := strings.Join(hud.LiveGreeting(nil), "\n"); describedNode.MatchString(got) {
		t.Errorf("a cluster with no node is greeted with a node to describe:\n%s", got)
	}
}
