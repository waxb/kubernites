package ui

import (
	"strings"
	"testing"
)

func TestAWrappedCommandKeepsAFlagWithItsValue(t *testing.T) {
	var f Font
	for _, text := range []string{
		"$ kubectl get pvc backup-kv-0 -n market",
		"$ kubectl logs storefront-5c4d3b72-akpxz --namespace market",
	} {
		lines := f.Wrap(text, 220, 1)
		if len(lines) < 2 {
			t.Fatalf("%q fits one line at this width: %q", text, lines)
		}
		for _, l := range lines[:len(lines)-1] {
			words := strings.Fields(l)
			if last := words[len(words)-1]; strings.HasPrefix(last, "-") {
				t.Errorf("%q leaves the flag %s at a line's end, away from its value: %q", text, last, lines)
			}
		}
		if strings.Join(lines, " ") != text {
			t.Errorf("the wrap changed the words: %q", lines)
		}
	}
}

func TestProseWithADashOrAnAttachedValueWrapsAsBefore(t *testing.T) {
	var f Font
	text := "The crew --trade=porter:1.0 and -5 coins and - a dash all wrap word by word as the width allows here"
	var want []string
	line := ""
	for _, w := range strings.Fields(text) {
		switch {
		case line == "":
			line = w
		case len(line)+1+len(w) <= MaxChars(220, 1):
			line += " " + w
		default:
			want, line = append(want, line), w
		}
	}
	want = append(want, line)
	if got := f.Wrap(text, 220, 1); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAFlagThatClosesItsCommandTakesNoValueFromTheNextWord(t *testing.T) {
	var f Font
	text := "Delete with gracePeriodSeconds 1, as k9s's kill (kubectl delete --now). [CTRL-K]"
	for width := 120; width <= 480; width++ {
		max := MaxChars(width, 1)
		var want []string
		line := ""
		for _, w := range strings.Fields(text) {
			switch {
			case line == "":
				line = w
			case len(line)+1+len(w) <= max:
				line += " " + w
			default:
				want, line = append(want, line), w
			}
		}
		want = append(want, line)
		if got := f.Wrap(text, width, 1); strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("width %d (%d chars): got %q, want %q", width, max, got, want)
		}
	}
}
