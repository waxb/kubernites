package ui

import (
	"testing"
)

func TestALongWordBreaksAtTheLastBreakThatFitsTheLine(t *testing.T) {
	lines := wrapParagraph("pvc-f18ef7c2-79dd-496e-9f0f-27ce7d84bb09, StorageClass standard", 34)
	if lines[0] != "pvc-f18ef7c2-79dd-496e-9f0f-" {
		t.Fatalf("the UUID broke early: %q", lines)
	}
}
