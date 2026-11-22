package app

import (
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
)

func TestTheLiveOnlyTitlesSettingsChangeAndSaveWhatTheMenuChanges(t *testing.T) {
	dir := t.TempDir()
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: dir}, nil, nil)
	press := func(id string) {
		t.Helper()
		b, ok := hud.HitID(g.Title.Buttons(), id)
		if !ok {
			t.Fatalf("no %s on the title", id)
		}
		if err := g.UpdateTitle(hud.Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}); err != nil {
			t.Fatal(err)
		}
	}
	label := func(id string) string {
		b, _ := hud.HitID(g.Title.Buttons(), id)
		return b.Label
	}
	press(hud.TitleSettingsID)
	if !g.Title.SettingsOpen() {
		t.Fatal("SETTINGS did not open")
	}
	if label("menu:names") != "NAMES: KUBERNETES" {
		t.Fatalf("a live-only title shows the live NAMES, Kubernetes by default: %q", label("menu:names"))
	}
	press("menu:names")
	if g.Settings.LiveNames == nil || *g.Settings.LiveNames || g.Settings.TrueNames || label("menu:names") != "NAMES: STORY" {
		t.Errorf("NAMES on a live-only title changes the live names: live %v, realm %v, label %q", g.Settings.LiveNames, g.Settings.TrueNames, label("menu:names"))
	}
	press(hud.StoneMenuID)
	press("menu:hints")
	press("menu:reduce")
	press("menu:vol:master:-")
	if g.Settings.Stone != hud.StoneModeKube || !g.Settings.LiveHints || !g.Settings.ReduceEffects || g.Settings.MasterVolume == nil {
		t.Errorf("the settings after STONE, HINTS, EFFECTS and MASTER -: %+v", g.Settings)
	}
	if label("menu:hints") != "HINTS: ON" || label("menu:reduce") != "EFFECTS: REDUCED" {
		t.Errorf("the rows read %q and %q after the presses", label("menu:hints"), label("menu:reduce"))
	}
	saved, err := LoadSettings(filepath.Join(dir, settingsFile))
	if err != nil {
		t.Fatal(err)
	}
	if saved.LiveNames == nil || *saved.LiveNames || saved.Stone != hud.StoneModeKube || !saved.LiveHints || !saved.ReduceEffects || saved.MasterVolume == nil || *saved.MasterVolume != *g.Settings.MasterVolume {
		t.Errorf("settings.json holds %+v", saved)
	}
	press("title:settings:back")
	if g.Title.SettingsOpen() {
		t.Error("BACK leaves the settings open")
	}
}
