package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheRaiseLineCountsPodsOfOneImageInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if got, want := ui.Name(CrewChooserLine(RaiseRow{Crew: "clerks"}, 2)), "A Deployment keeps 2 clerk Pods running: its ReplicaSet replaces any that fail."; got != want {
		t.Fatalf("the raise line reads %q, want %q", got, want)
	}
}
