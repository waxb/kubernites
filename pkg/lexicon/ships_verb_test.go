package lexicon

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/soulstone"
)

func TestShipsAsAVerbStaysAVerbInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"A familiar: ships the logs":                 "A sidecar: ships the logs",
		"The raven ships every chronicle line south": "ships every",
	} {
		if got := TrueNames(story); !strings.Contains(got, want) || strings.Contains(got, "request burst") {
			t.Errorf("TrueNames(%q) = %q, want it to keep %q", story, got, want)
		}
	}
	if got := TrueNames("More berths bring more ships: more traffic"); !strings.Contains(got, "request bursts") {
		t.Errorf("the noun ships is still a request burst: %q", got)
	}
}

func TestTheTradesTaglinesReadInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"Writes ledgers, needs room":              "Writes database requests, needs room",
		"Hauls crates and chests":                 "Hauls uploads and large uploads",
		"Keeps out of food: no room to feed more": "Nodes out of CPU: no room to feed more",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) = %q, want %q", story, got, want)
		}
	}
}

func TestATradesTaglineKeepsItsVerbInNamesMode(t *testing.T) {
	kinds := map[string]bool{"Nodes": true, "Node": true, "Pods": true, "Pod": true, "Services": true, "Deployments": true, "ReplicaSets": true}
	for _, p := range soulstone.Playable() {
		line := soulstone.Tagline(p.Image)
		if first, _, _ := strings.Cut(TrueNames(line), " "); kinds[first] {
			t.Errorf("the %s tagline %q reads %q in NAMES mode: its verb became a kind", p.Repository, line, TrueNames(line))
		}
	}
}
