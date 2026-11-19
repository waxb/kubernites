package app

import (
	"math"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
)

const volumeStep = 0.1

func (g *Game) volumes() hud.Volumes {
	master, music, sounds := g.audio.Volumes()
	return hud.Volumes{
		Master: orVolume(g.Settings.MasterVolume, master),
		Music:  orVolume(g.Settings.MusicVolume, music),
		Sounds: orVolume(g.Settings.SoundVolume, sounds),
	}
}

func orVolume(saved *float64, fallback float64) float64 {
	if saved == nil {
		return fallback
	}
	return *saved
}

func (g *Game) applyVolumes() {
	v := g.volumes()
	g.audio.SetMasterVolume(v.Master)
	g.audio.SetMusicVolume(v.Music)
	g.audio.SetSFXVolume(v.Sounds)
	if g.Sess != nil {
		g.Sess.volumes = v
	}
}

func (g *Game) stepVolume(cmd string) {
	key, dir, ok := strings.Cut(cmd, ":")
	if !ok {
		return
	}
	delta := volumeStep
	if dir == "-" {
		delta = -volumeStep
	}
	v := g.volumes()
	switch key {
	case "master":
		g.Settings.MasterVolume = stepped(v.Master, delta)
	case "music":
		g.Settings.MusicVolume = stepped(v.Music, delta)
	case "sounds":
		g.Settings.SoundVolume = stepped(v.Sounds, delta)
	default:
		return
	}
	g.applyVolumes()
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func stepped(current, delta float64) *float64 {
	next := math.Round((current+delta)*10) / 10
	next = math.Max(0, math.Min(1, next))
	return &next
}
