package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestStoryModeNamesEveryDrawnStringInGameWords(t *testing.T) {
	defer ApplyNames(false)
	ApplyNames(false)
	if ui.Naming() {
		t.Fatal("story mode claims Kubernetes names")
	}
	if got := ui.Name("3 Pods are Pending on the Node"); got == "3 Pods are Pending on the Node" {
		t.Fatalf("story mode left Kubernetes words: %q", got)
	}
	ApplyNames(true)
	if !ui.Naming() || ui.Name("a hut") == "a hut" {
		t.Fatal("Kubernetes mode lost its names")
	}
}
