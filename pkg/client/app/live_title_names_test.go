package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestALiveOnlyTitleAndPickerSpeakTheLiveNamesBeforeAndAfterAConnection(t *testing.T) {
	defer ApplyNames(false)
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: t.TempDir()}, nil, nil)
	if !ui.Naming() {
		t.Fatal("a live-only title opens in story words, though the live NAMES are Kubernetes by default")
	}
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	rig.g.RealmIntent(hud.Intent{Kind: hud.IntentTitle})
	if rig.g.Scene != SceneTitle || !ui.Naming() {
		t.Fatalf("after leaving a live cluster the live-only title speaks story words (scene %v)", rig.g.Scene)
	}
	_ = g
}
