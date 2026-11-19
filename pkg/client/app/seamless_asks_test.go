package app

import (
	"testing"

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
