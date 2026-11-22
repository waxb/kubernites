package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func clickTitle(t *testing.T, ti *Title, id string) []Intent {
	t.Helper()
	b, ok := HitID(ti.Buttons(), id)
	if !ok {
		t.Fatalf("no %s among %v", id, ti.Buttons())
	}
	return ti.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true})
}

func TestTheLiveOnlyTitlesSettingsShowEveryMenuSettingAndEachRowAsksWhatTheMenuRowAsks(t *testing.T) {
	ti := &Title{LiveOnly: true, Settings: TitleSettings{Muted: true, Reduce: true, TrueNames: true, Edge: true, Fullscreen: true, Hints: true, StoneKubectl: true, Volumes: Volumes{Master: 0.5, Music: 0.25, Sounds: 1}}}
	if got := clickTitle(t, ti, TitleSettingsID); len(got) != 1 || got[0].Kind != IntentSound || !ti.SettingsOpen() {
		t.Fatalf("SETTINGS gives %v, open %v", got, ti.SettingsOpen())
	}
	labels := map[string]string{}
	last := 0
	for _, b := range ti.Buttons() {
		labels[b.ID] = b.Label
		if b.Rect.X < 0 || b.Rect.X+b.Rect.W > Width || b.Rect.Y < titleSettingsHead {
			t.Errorf("%s at %v leaves the settings panel", b.ID, b.Rect)
		}
		last = max(last, b.Rect.Y+b.Rect.H)
	}
	if last+12 >= 484 {
		t.Errorf("the settings panel ends at %d, over the title's note line", last+12)
	}
	want := map[string]string{
		"menu:mute": "SOUND: OFF", volumePrefix + "master": "MASTER 50%", volumePrefix + "music": "MUSIC 25%", volumePrefix + "sounds": "SOUNDS 100%",
		"menu:reduce": "EFFECTS: REDUCED", "menu:fullscreen": "FULLSCREEN: ON", "menu:edge": "EDGE SCROLL: ON", "menu:names": "NAMES: KUBERNETES",
		StoneMenuID: stoneMenuRow(true)[1], menuHints: "HINTS: ON", titleSettingsBack: "BACK",
	}
	for id, label := range want {
		if labels[id] != label {
			t.Errorf("%s reads %q, want %q", id, labels[id], label)
		}
	}
	asks := map[string]string{
		"menu:mute": IntentMute, "menu:reduce": IntentReduce, "menu:fullscreen": IntentFullscreen, "menu:edge": IntentEdge,
		"menu:names": IntentNames, StoneMenuID: IntentStone, menuHints: IntentHints,
	}
	for id, kind := range asks {
		if got := clickTitle(t, ti, id); len(got) != 1 || got[0].Kind != kind {
			t.Errorf("%s gives %v, want %s as the MENU's row", id, got, kind)
		}
	}
	if got := clickTitle(t, ti, volumePrefix+"music:-"); len(got) != 1 || got[0].Kind != IntentVolume || got[0].Command != "music:-" {
		t.Errorf("MUSIC - gives %v", got)
	}
	if got := ti.Update(Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.Key3}}); len(got) != 0 {
		t.Errorf("3 behind the settings leads to %v", got)
	}
	if clickTitle(t, ti, titleSettingsBack); ti.SettingsOpen() {
		t.Fatal("BACK leaves the settings open")
	}
	clickTitle(t, ti, TitleSettingsID)
	if ti.Update(Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyEscape}}); ti.SettingsOpen() {
		t.Fatal("Esc leaves the settings open")
	}
	if _, ok := HitID(ti.Buttons(), "title:live"); !ok {
		t.Fatal("after the settings the title offers LIVE CLUSTER again")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	clickTitle(t, ti, TitleSettingsID)
	ti.Draw(dst)
}
