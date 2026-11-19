package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/waxb/kubernites/pkg/client/hud"
)

const settingsFile = "settings.json"

type Settings struct {
	ReduceEffects bool                    `json:"reduceEffects"`
	TrueNames     bool                    `json:"trueNames"`
	Fullscreen    bool                    `json:"fullscreen"`
	CoinFlights   string                  `json:"coinFlights,omitempty"`
	Panels        map[string]hud.PanelPin `json:"panels,omitempty"`
	DecreeFolded  bool                    `json:"decreeFolded,omitempty"`
	MasterVolume  *float64                `json:"masterVolume,omitempty"`
	MusicVolume   *float64                `json:"musicVolume,omitempty"`
	SoundVolume   *float64                `json:"soundVolume,omitempty"`
	Taught        []string                `json:"taught,omitempty"`
	PauseFor      string                  `json:"pauseFor,omitempty"`
	Stone         string                  `json:"stone,omitempty"`
	LiveHints     bool                    `json:"liveHints,omitempty"`
	LiveNames     *bool                   `json:"liveNames,omitempty"`
}

func defaultSettings() Settings {
	return Settings{CoinFlights: hud.CoinFlightsFew}
}

func SettingsPath(savePath string) string {
	if savePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(savePath), settingsFile)
}

func (o Options) SettingsFile() string {
	if o.SettingsDir != "" {
		return filepath.Join(o.SettingsDir, settingsFile)
	}
	return SettingsPath(o.SavePath)
}

func LoadSettings(path string) (Settings, error) {
	if path == "" {
		return defaultSettings(), nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultSettings(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse settings: %w", err)
	}
	s.CoinFlights = hud.CoinFlightMode(s.CoinFlights)
	s.Panels = hud.KnownPanels(s.Panels)
	return s, nil
}

func SaveSettings(path string, s Settings) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return SaveAtomic(path, data)
}

func (g *Game) cycleCoinFlights() {
	g.Settings.CoinFlights = hud.NextCoinFlights(g.Settings.CoinFlights)
	if g.Sess != nil {
		g.Sess.Hud.SetCoinFlights(g.Settings.CoinFlights)
	}
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func (g *Game) savePanels() {
	s := g.realmSession()
	if s == nil {
		return
	}
	g.Settings.Panels = s.Hud.Panels()
	g.Settings.DecreeFolded = s.Hud.TrackerFolded()
	g.Settings.Taught = s.Hud.Taught()
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}
