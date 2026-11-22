package ui

import (
	"strings"
	"testing"
)

func TestNamerRenamesDrawnText(t *testing.T) {
	SetNamer(func(s string) string { return strings.ReplaceAll(s, "Hearth", "Pod") })
	defer SetNamer(nil)
	var f Font
	if got := Name("a Hearth"); got != "a Pod" {
		t.Fatalf("Name = %q", got)
	}
	if w, _ := f.Measure("Hearth", 1); w != 3*Advance-1 {
		t.Errorf("Measure used the story name: %d", w)
	}
	if lines := f.Wrap("Hearth Hearth", 7*Advance, 1); len(lines) != 1 || lines[0] != "Pod Pod" {
		t.Errorf("Wrap = %q", lines)
	}
	if got := Clip("Hearth Hearth Hearth", 7); got != "Pod ..." {
		t.Errorf("Clip = %q", got)
	}
	literal := Font{Literal: true}
	if w, _ := literal.Measure("Hearth", 1); w != 6*Advance-1 {
		t.Errorf("a literal font must not rename: %d", w)
	}
	if lines := literal.Wrap("Hearth", 20*Advance, 1); lines[0] != "Hearth" {
		t.Errorf("literal Wrap = %q", lines)
	}
}

func TestNoNamerLeavesTextAlone(t *testing.T) {
	SetNamer(nil)
	if got := Name("a Hearth"); got != "a Hearth" {
		t.Fatalf("Name = %q", got)
	}
}

func TestAStoryNamerRenamesButIsNotKubernetesMode(t *testing.T) {
	defer SetNamer(nil)
	SetStoryNamer(func(s string) string { return strings.ReplaceAll(s, "Pod", "hut") })
	if Naming() {
		t.Fatal("the story namer claims Kubernetes mode")
	}
	if got := Name("a Pod"); got != "a hut" {
		t.Fatalf("story namer gave %q", got)
	}
	if got := Bridge("the hut", "the Pod"); got != "the hut" {
		t.Fatalf("a bridge in story mode gave %q", got)
	}
}

func TestVerbatimTextIsNeverRenamedInEitherModeOrAfterTheMemoryFills(t *testing.T) {
	defer SetNamer(nil)
	said := Verbatim(`deployments.apps "web" is forbidden in the namespace "market"`)
	rename := func(s string) string {
		return strings.NewReplacer("deployments", "crews", "namespace", "fief").Replace(s)
	}
	var f Font
	for _, set := range []func(func(string) string){SetStoryNamer, SetNamer} {
		set(rename)
		for i := range namedMemory + 1 {
			Name("filler " + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + string(rune(i)))
		}
		if got := Name(said); got != said {
			t.Errorf("Name renamed the cluster's words: %q", got)
		}
		if lines := f.Wrap(said, 20*Advance, 1); strings.Join(lines, " ") != said {
			t.Errorf("Wrap renamed the cluster's words: %q", lines)
		}
	}
	if got := Name("the deployments"); got != "the crews" {
		t.Errorf("other text is no longer renamed: %q", got)
	}
}
