package app

import (
	"os"
	"path/filepath"
	"runtime"
)

const alsaPlaybackPCMs = "/dev/snd/pcmC*D*p"

type audioProbe struct {
	goos   string
	getenv func(string) string
	exists func(string) bool
	glob   func(string) ([]string, error)
}

func hostAudioProbe() audioProbe {
	return audioProbe{
		goos:   runtime.GOOS,
		getenv: os.Getenv,
		exists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		glob: filepath.Glob,
	}
}

func (p audioProbe) hasDevice() bool {
	if p.goos != "linux" {
		return true
	}
	if p.getenv("PULSE_SERVER") != "" {
		return true
	}
	if dir := p.getenv("XDG_RUNTIME_DIR"); dir != "" && p.exists(filepath.Join(dir, "pulse", "native")) {
		return true
	}
	pcms, err := p.glob(alsaPlaybackPCMs)
	return err == nil && len(pcms) > 0
}
