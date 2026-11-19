package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const controlsFile = "controls.json"

type Controls struct {
	EdgeScroll bool `json:"edgeScroll"`
}

func ControlsPath(savePath string) string {
	if savePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(savePath), controlsFile)
}

func LoadControls(path string) (Controls, error) {
	if path == "" {
		return Controls{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Controls{}, nil
	}
	if err != nil {
		return Controls{}, fmt.Errorf("read controls: %w", err)
	}
	var c Controls
	if err := json.Unmarshal(data, &c); err != nil {
		return Controls{}, fmt.Errorf("parse controls: %w", err)
	}
	return c, nil
}

func SaveControls(path string, c Controls) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode controls: %w", err)
	}
	return SaveAtomic(path, data)
}

func (g *Game) ToggleEdge() {
	g.Controls.EdgeScroll = !g.Controls.EdgeScroll
	g.Pointer.Edge = g.Controls.EdgeScroll
	if err := SaveControls(g.ControlsPath(), g.Controls); err != nil {
		g.Log.Warn("could not save controls", "err", err)
	}
}

func (g *Game) ControlsPath() string {
	if g.Capture != nil {
		return ""
	}
	return ControlsPath(g.Opts.SavePath)
}
