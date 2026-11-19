package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAJoiningKeepsCardSaysItIsJoining(t *testing.T) {
	keep := view.Structure{Kind: view.StructKeep, Label: "Stone Keep 6", State: view.StateJoining, Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-6"}}
	c := StructureCard(keep, fullFrame(t))
	if !strings.HasPrefix(c.Lines[0].Text, "JOINING") || strings.Contains(c.Lines[0].Text, "DARK") {
		t.Fatalf("a joining Keep's card reads %q", c.Lines[0].Text)
	}
}
