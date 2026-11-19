package lexicon

import (
	"strings"
	"testing"
)

func TestStoryModeNamesTheCitadelsWorkloadsByTheirStoryNames(t *testing.T) {
	for kube, want := range map[string]string{
		"Deployment/metrics-server": "augur",
		"Deployment/coredns":        "herald",
		"DaemonSet/kube-proxy":      "wayfinder",
	} {
		got := StoryText(StoryNames(kube))
		if !strings.Contains(got, "/"+want) || strings.Contains(got, strings.SplitN(kube, "/", 2)[1]) {
			t.Errorf("story mode reads %q for %q", got, kube)
		}
	}
}
