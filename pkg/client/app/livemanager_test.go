package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestALiveRealmOpensInNamesModeAndTheSavedSettingStays(t *testing.T) {
	defer ApplyNames(false)
	ApplyNames(false)
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	g := rig.g
	if ui.StoryNaming() || !rig.s.TrueNames {
		t.Fatal("a live realm did not open in NAMES mode")
	}
	if g.Settings.TrueNames {
		t.Fatal("the saved NAMES setting changed")
	}
	if err := g.RealmIntent(hud.Intent{Kind: hud.IntentNames}); err != nil {
		t.Fatal(err)
	}
	if !ui.StoryNaming() || g.Settings.LiveNames == nil || *g.Settings.LiveNames || g.Settings.TrueNames {
		t.Fatalf("toggling NAMES in a live realm: story %v live %v saved %v", ui.StoryNaming(), g.Settings.LiveNames, g.Settings.TrueNames)
	}
	data, err := os.ReadFile(filepath.Join(rig.dir, "settings.json"))
	if err != nil || !strings.Contains(string(data), `"liveNames": false`) || !strings.Contains(string(data), `"trueNames": false`) {
		t.Fatalf("settings.json %s %v", data, err)
	}
	g.leaveLive()
	if !ui.StoryNaming() {
		t.Fatal("leaving the live realm did not restore the saved names")
	}
}

func TestHintsAreOffByDefaultAndRemembered(t *testing.T) {
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	g := rig.g
	if g.Settings.LiveHints || rig.s.Hud.LiveCard().Hints {
		t.Fatal("HINTS is on by default")
	}
	g.toggleHints()
	if !g.Settings.LiveHints || !rig.s.Hud.LiveCard().Hints {
		t.Fatal("HINTS did not turn on")
	}
	data, err := os.ReadFile(filepath.Join(rig.dir, "settings.json"))
	if err != nil || !strings.Contains(string(data), `"liveHints": true`) {
		t.Fatalf("settings.json %s %v", data, err)
	}
	labels := ""
	for _, b := range rig.s.Hud.MenuButtons(rig.s.Frame(true)) {
		labels += b.Label + "|"
	}
	if !strings.Contains(labels, "HINTS: ON") || !strings.Contains(labels, "RESET LIVE CHOICES") {
		t.Fatalf("the live MENU: %s", labels)
	}
}
