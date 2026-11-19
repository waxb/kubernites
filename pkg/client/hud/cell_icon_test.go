package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestACellWithItsGateAboveATwoRowLabelDrawsNoIconOverIt(t *testing.T) {
	r := ui.Rect{X: 0, Y: 0, W: CellW, H: CellH}
	gated := Button{Label: "PARCELS DEPOT", Why: "CHANCERY", Disabled: true, Icon: "icon/depot", Rect: r}
	if cellShowsIcon(gated) {
		t.Error("slot 7's gate line, its two label rows and its icon share one cell (founding-node-menu: \"CHANCERY\" over \"PARCELS DEPOT\" with the icon on \"PARCELS\")")
	}
	open := Button{Label: "PARCELS DEPOT", Icon: "icon/depot", Rect: r}
	if !cellShowsIcon(open) {
		t.Error("an open two-row cell lost its icon")
	}
}
