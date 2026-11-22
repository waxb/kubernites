package livescenes

import (
	"strings"
	"testing"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live"
)

func TestALiveOnlyTitleCaptureSpeaksTheNamesItsSettingsShow(t *testing.T) {
	defer app.ApplyNames(false)
	for _, scene := range []string{"live-title-caption-open", "live-title-settings"} {
		for _, kube := range []bool{false, true} {
			app.ApplyNames(kube)
			g := app.NewGame(nil, app.Options{}, nil, nil)
			g.Settings.TrueNames = kube
			app.CaptureScenes[scene](g)
			shown := g.Settings.LiveNames == nil || *g.Settings.LiveNames
			if ui.Naming() != shown {
				t.Errorf("%s (kubernetes %v): the title speaks kubernetes names %v while its NAMES setting reads kubernetes %v", scene, kube, ui.Naming(), shown)
			}
			if ui.Naming() != kube {
				t.Errorf("%s (kubernetes %v): the capture speaks kubernetes names %v", scene, kube, ui.Naming())
			}
		}
	}
}

func TestTheFailedPickerCaptureShowsTheHintAPlayerGetsForAMissingName(t *testing.T) {
	err := live.NoAnswer(labWestLookup())
	for _, want := range []string{"no DNS record for lab-west.example.internal", "fetch its kubeconfig again", `Get "https://lab-west.example.internal:6443/version": dial tcp: lookup lab-west.example.internal: no such host`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the failed capture's message lacks %q: %v", want, err)
		}
	}
}
