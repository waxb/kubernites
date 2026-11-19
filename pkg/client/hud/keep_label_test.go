package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAKeepTileWithUpkeepKeepsItsNodeNameInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := fullFrame(t)
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, UpkeepPerSec: 0.29}, {KeepType: view.KeepTypeCitadel, UpkeepPerSec: 67}}
	for kind, want := range map[string]string{"keep": "XLARGE NODE -$0.29/S", "citadel-keep": "4XLARGE NODE -$67/S"} {
		b := KeepButton(view.Buildable{Kind: kind, Name: "Stone Keep", Available: true}, f)
		if got := ui.Name(b.Label); got != want {
			t.Errorf("%s tile reads %q in NAMES mode, want %q", kind, got, want)
		}
	}
}

func TestEveryKeepTileFitsWithItsUpkeepInBothModes(t *testing.T) {
	f := fullFrame(t)
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, UpkeepPerSec: 0.79}, {KeepType: view.KeepTypeGreat, UpkeepPerSec: 28.6}, {KeepType: view.KeepTypeCitadel, UpkeepPerSec: 203}}
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, kind := range []string{"keep", "great-keep", "citadel-keep"} {
			b := KeepButton(view.Buildable{Kind: kind, Name: "Stone Keep", Available: true}, f)
			for _, row := range CellLabel(b.Label, ui.MaxChars(CellW-4, 1)) {
				if strings.Contains(row, "...") {
					t.Errorf("%s: the %s tile is cut: %q", mode, kind, CellLabel(b.Label, ui.MaxChars(CellW-4, 1)))
				}
			}
		}
		ui.SetNamer(nil)
	}
}

func TestNamesModeLabelsLeaveNoStoryWord(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := fullFrame(t)
	f.State.KeepShop = nil
	for kind, want := range map[string]string{"keep": "XLARGE NODE", "great-keep": "2XLARGE NODE", "citadel-keep": "4XLARGE NODE"} {
		if got := ui.Name(KeepLabel(view.Buildable{Kind: kind, Name: "Stone Keep"}, f)); got != want {
			t.Errorf("the %s build tile reads %q in NAMES mode, want %q (reserve-keep: \"press MEMORY NODE\" over a STONE NODE button)", kind, got, want)
		}
	}
	if got := ui.Name(templateLabel(TemplateScroll)); got != "CONFIGMAP" {
		t.Errorf("the Scriptorium's scroll template reads %q in NAMES mode (scriptorium-beside: \"SCROLL\")", got)
	}
	for _, b := range stallButtons(f) {
		if got := ui.Name(b.Label); strings.Contains(got, "PARCELS") || strings.Contains(got, "LEDGERS") {
			t.Errorf("a depot build tile reads %q in NAMES mode (build: \"PARCELS SERVICE\")", got)
		}
	}
}
