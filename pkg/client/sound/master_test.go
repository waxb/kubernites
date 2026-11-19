package sound

import "testing"

func TestMasterVolumeScalesMusicAndEffects(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicDark)
	clk.advance(crossfade)
	m.Update()
	music := out.loopAt(0)
	m.PlaySFX(SFXCoin)
	shot := out.shotAt(0)
	m.SetMasterVolume(0.5)
	if vol, _, _ := music.state(); !near(vol, 0.5*defaultMusicVolume) {
		t.Errorf("music at master 0.5 plays at %v", vol)
	}
	if vol, _, _ := shot.state(); !near(vol, 0.5*defaultSFXVolume) {
		t.Errorf("an effect at master 0.5 plays at %v", vol)
	}
	m.SetMasterVolume(2)
	if master, music, sounds := m.Volumes(); master != 1 || !near(music, defaultMusicVolume) || !near(sounds, defaultSFXVolume) {
		t.Errorf("volumes = %v %v %v", master, music, sounds)
	}
}

func TestANilManagerReportsDefaultVolumes(t *testing.T) {
	var m *Manager
	m.SetMasterVolume(0.3)
	if master, music, sounds := m.Volumes(); master != 1 || music != defaultMusicVolume || sounds != defaultSFXVolume {
		t.Errorf("volumes = %v %v %v", master, music, sounds)
	}
}
