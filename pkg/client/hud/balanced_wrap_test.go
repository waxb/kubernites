package hud

import (
	"strings"
	"testing"
)

func TestAnOrderLineWrapsAtTheEndOfAClauseWhenOneFits(t *testing.T) {
	for _, text := range []string{
		"PodDisruptionBudget market/storefront: minAvailable 2, 2 healthy of 2 desired, 0 disruptions allowed",
	} {
		lines := balancedWrap(liveFont, text, OrderBox.W-32, 3)
		if len(lines) != 2 {
			t.Fatalf("%q wraps into %d lines: %q", text, len(lines), lines)
		}
		if !strings.HasSuffix(lines[0], ",") && !strings.HasSuffix(lines[0], ":") {
			t.Errorf("%q breaks inside a clause: %q", text, lines)
		}
		if strings.Join(lines, " ") != text {
			t.Errorf("the wrap changed the words: %q", lines)
		}
	}
	if lines := balancedWrap(liveFont, "a b c d e f g h i j k l m n o p q r s t u v w x y z a b c d e f g h i j k l m n o p q r s t u v w x y z a b c d e f g h i j k l m n o p", OrderBox.W-32, 3); len(lines) != 2 || len(lines[0])-len(lines[1]) > 4 {
		t.Errorf("a line with no clause ends is no longer balanced: %q", lines)
	}
}

func TestAnOrderLineStartsANewSentenceOnItsOwnLineWhenTheLinesStayAsFew(t *testing.T) {
	text := "A guard of the Chancery has side effects, so it refuses a preview. The order can go without one: what changes is seen only after it is sent."
	lines := balancedWrap(liveFont, text, OrderBox.W-32, orderLineMax)
	if len(lines) != 2 || lines[0] != "A guard of the Chancery has side effects, so it refuses a preview." {
		t.Errorf("the second sentence does not open the second line: %q", lines)
	}
	if strings.Join(lines, " ") != text {
		t.Errorf("the wrap changed the words: %q", lines)
	}
}

func TestOfTheClauseEndsThatKeepTheLinesAsFewTheMostBalancedWins(t *testing.T) {
	text := "This reaches into a workload: exec, attach, port-forward, cp, debug or proxy run inside it with these credentials."
	lines := balancedWrap(liveFont, text, OrderBox.W-32, 3)
	if len(lines) != 2 || !atClauses(lines) {
		t.Fatalf("%q wraps as %q", text, lines)
	}
	if d := len(lines[0]) - len(lines[1]); d > 4 || d < -4 {
		t.Errorf("a wider clause end wins over a more balanced one: %q", lines)
	}
}
