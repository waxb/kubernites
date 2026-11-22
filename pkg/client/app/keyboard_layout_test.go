package app

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
)

func TestTheLabelledCtrlPCtrlQLeavesTheTerminalOnDvorakAndAzertyAndNeverReachesTheContainer(t *testing.T) {
	layouts := map[string]map[ebiten.Key]string{
		"Dvorak": {ebiten.KeyR: "p", ebiten.KeyX: "q", ebiten.KeyP: "l", ebiten.KeyQ: "'"},
		"AZERTY": {ebiten.KeyP: "p", ebiten.KeyA: "q", ebiten.KeyQ: "a"},
	}
	physical := map[string][2]ebiten.Key{"Dvorak": {ebiten.KeyR, ebiten.KeyX}, "AZERTY": {ebiten.KeyP, ebiten.KeyA}}
	for name, table := range layouts {
		t.Run(name, func(t *testing.T) {
			layout := func(k ebiten.Key) string { return table[k] }
			addr, shell, _ := shellFront(t)
			rig := connectStoneTo(t, hud.LevelSeamless, addr)
			s := rig.s
			askedOnTheStone(t, s, "exec -it shell-1 -n parity -- sh")
			stoneFrame(s, ebiten.KeyY)
			frames(t, s, "the terminal is live", s.Hud.StoneTerminal)
			for _, k := range physical[name] {
				s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{k}, Layout: layout}, 0, &Pointer{}, true)
			}
			stoneEnds(t, s)
			if typed := typedIn(shell); strings.ContainsAny(typed, "\x10\x11\x12\x18\x01") {
				t.Errorf("%s: the labelled ctrl-p ctrl-q reached the container as %q", name, typed)
			}
			if out := strings.Join(s.Hud.StoneOutput(), "\n"); !strings.Contains(out, "ctrl-p ctrl-q") {
				t.Errorf("%s: the Stone does not say the session was left with ctrl-p ctrl-q:\n%s", name, out)
			}
		})
	}
}
