package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheDescribePanelPrintsKubectlOutputAsItIsInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	h := New(nil, Backend{})
	line := "  cpu:                2"
	if got := h.detailFont().Clip(line, 80); got != line {
		t.Fatalf("the describe panel prints %q as %q", line, got)
	}
}
