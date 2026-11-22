package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestAtSeamlessEveryCardOrderAsksOnceExceptKillAsK9sDoes(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	asked, killed := map[string]bool{}, false
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			sp, _ := rig.s.spec(o.Key)
			if o.Disabled || o.Key == orderMore || o.Key == "drain" || sp.stone != "" || sp.read {
				continue
			}
			before := len(changes(rig.rec.Full()))
			rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: o.Key})
			sent := changes(rig.rec.Full())[before:]
			form, open := rig.s.Hud.OrderOpen()
			if sp.quick {
				killed = true
				if len(sent) != 1 {
					t.Errorf("%s (Ctrl-%s) is k9s's kill and should go from one chord: sent %v", o.Key, o.Chord, sent)
				}
				continue
			}
			if len(sent) != 0 {
				t.Errorf("%s (Ctrl-%s) sent %v before any answer", o.Key, o.Chord, sent)
			}
			if !open {
				t.Errorf("%s (Ctrl-%s) asked nothing", o.Key, o.Chord)
				continue
			}
			if len(sp.inputs) == 0 && form.Go != sp.label+" [Y]" {
				t.Errorf("%s asks with %q, want %q", o.Key, form.Go, sp.label+" [Y]")
			}
			asked[o.Key] = true
			rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: rig.s.Hud.OrderValues()})
			if got := changes(rig.rec.Full())[before:]; len(got) != 1 {
				t.Errorf("%s: its one answer sent %v", o.Key, got)
			}
		}
	}
	t.Logf("asked: %v", asked)
	for _, key := range []string{"restart", "cordon", "delete", "scale"} {
		if !asked[key] {
			t.Errorf("no %s was asked", key)
		}
	}
	if !killed {
		t.Error("no kill was sent")
	}
}

const oneChord = "KILL, SHELL and ATTACH go from one chord, as in k9s."

func TestAtSeamlessForwardAsksOnceWhileKillShellAndAttachGoFromOneChordAsInK9s(t *testing.T) {
	if warning := strings.Join(hud.SeamlessWarningLines(), " "); !strings.Contains(warning, oneChord) {
		t.Errorf("the SEAMLESS warning does not say %q:\n%s", oneChord, warning)
	}
	readme, err := os.ReadFile(filepath.Join("..", "..", "..", "README.md"))
	if err != nil || !strings.Contains(string(readme), strings.TrimSuffix(oneChord, ".")) {
		t.Errorf("README does not say %q (%v)", oneChord, err)
	}
	for _, level := range []string{hud.LevelSeamless, hud.LevelWrites} {
		addr, _, _ := shellFront(t)
		rig := connectStoneTo(t, level, addr)
		s := rig.s
		sel, orders := hutOf(t, rig)
		for _, key := range []string{"shell", "attach", "port-forward"} {
			if orders[key].Disabled {
				t.Fatalf("%s at %s: %+v", key, level, orders[key])
			}
			rig.select_(sel)
			rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: key})
			form, open := s.Hud.OrderOpen()
			asks := key == "port-forward" && level == hud.LevelSeamless
			if open != asks {
				t.Fatalf("%s at %s: an order scroll open=%v %+v, want %v", key, level, open, form, asks)
			}
			if asks {
				if form.Go != "FORWARD [Y]" || s.Hud.StoneRunning() {
					t.Fatalf("FORWARD at %s asks %+v, the Stone running %v", level, form, s.Hud.StoneRunning())
				}
				rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: s.Hud.OrderValues()})
				if _, still := s.Hud.OrderOpen(); still {
					t.Fatal("FORWARD's Y left its scroll open")
				}
			}
			if !s.Hud.StoneRunning() && !strings.Contains(string(s.Hud.Stone.Input), key) {
				t.Fatalf("%s at %s did not reach the Stone: input %q", key, level, string(s.Hud.Stone.Input))
			}
			for range 3 {
				if s.Hud.StoneRunning() {
					stoneFrame(s, ebiten.KeyEscape)
				}
			}
			stoneEnds(t, s)
			closeStone(t, rig)
		}
	}
}
