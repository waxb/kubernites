package lexicon

import (
	"strings"
	"testing"
)

func TestNamesModeSaysTaintedWhereTheStorySaysWarded(t *testing.T) {
	for story, want := range map[string]string{
		"The Chancery warded lone-tower with node.kubernetes.io/not-ready as it joined: its Reeve is not ready": "The kube-apiserver tainted lone-tower with node.kubernetes.io/not-ready as it joined",
		"The Stewards warded stone-keep-1 with node.kubernetes.io/memory-pressure: it runs short of housing":    "tainted stone-keep-1 with node.kubernetes.io/memory-pressure",
		"Warded Keeps turn away huts without a Charm.":                                                          "Tainted Nodes",
	} {
		got := TrueNames(story)
		if !strings.Contains(got, want) || strings.Contains(strings.ToLower(got), "warded") {
			t.Errorf("NAMES mode reads %q for %q, want %q and no 'warded'", got, story, want)
		}
	}
}

func TestNamesModeSaysReadyWhereTheStoryLightsTheCandle(t *testing.T) {
	for story, want := range map[string]string{
		"The Reeve of stone-keep-1 lit the candle: the Keep is Ready.": "The kubelet of stone-keep-1 reported the Node Ready",
		"until its Reeve lights its candle":                            "until its kubelet reports the Node Ready",
		"It counts only once its Reeve lights the candle (Ready).":     "once its kubelet reports the Node Ready",
	} {
		got := TrueNames(story)
		if !strings.Contains(got, want) || strings.Contains(got, "candle") {
			t.Errorf("NAMES reads %q for %q, want %q", got, story, want)
		}
	}
}
