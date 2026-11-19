package soulstone

import (
	"strings"
	"testing"
)

func TestServedLinesNameTheBoxesEachTradeCarries(t *testing.T) {
	cases := map[string]string{
		"porter:1.0":     "carried 3 parcels in the last 30s",
		"clerk:1.0":      "carried 3 ledgers in the last 30s",
		"hauler:1.0":     "carried 3 crates in the last 30s",
		"courier:1.0":    "carried 3 dispatches in the last 30s",
		"sealwright:1.0": "carried 3 sealed boxes in the last 30s",
	}
	for image, want := range cases {
		p, ok := Lookup(image)
		if !ok {
			t.Fatalf("%s missing", image)
		}
		if got := p.ServedLine(3, 30); got != want {
			t.Errorf("%s: %q, want %q", image, got, want)
		}
	}
	if got := (Profile{}).ServedLine(1, 30); got != "carried 1 box in the last 30s" {
		t.Errorf("no kinds: %q", got)
	}
}

func TestNoProfileSpeaksOfPilgrims(t *testing.T) {
	for _, p := range catalog {
		for _, text := range append([]string{p.Description, p.CrashLine, p.DutyLine}, p.StartLines...) {
			if strings.Contains(strings.ToLower(text), "pilgrim") {
				t.Errorf("%s still says pilgrims: %q", p.Image, text)
			}
		}
	}
}
