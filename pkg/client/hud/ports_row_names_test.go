package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheHarborsPortsRowNamesGatewaysAndShowsEveryPortInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := harborFrame(t)
	f.State.Ports = append(fixturePorts(), view.PortView{ID: "haven-2", Name: "Morrow Quay", MaxLevel: 4}, view.PortView{ID: "haven-3", Name: "Sunward Reach", MaxLevel: 4})
	f.State.Ports[1].Name = "Frostwater Quay"
	if got := ui.Name(portsLabel); got != "GATEWAYS" {
		t.Errorf("the row is headed %q in NAMES mode, want GATEWAYS", got)
	}
	if got := ui.Name("click a port on the PORTS row to go there, then upgrade it."); got != "click a Gateway on the GATEWAYS row to go there, then upgrade it." {
		t.Errorf("the hint reads %q", got)
	}
	h := New(nil, Backend{})
	if n := len(h.portChips(f, FreightRect(f))); n != len(f.State.Ports) {
		t.Errorf("the row shows %d of %d ports", n, len(f.State.Ports))
	}
}
