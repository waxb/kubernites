package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/view"
)

func demoEndingFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Kube.Ending = &view.EndingView{At: 99, Built: []string{"4 Keeps on 1 isle", "3 crews keeping 41 huts, behind 3 depots", "1 War Horn sounding and 0 Familiars bound", "212K boxes delivered and 3.1M requests served", "99.4% of boxes fresh in the last window"},
		Goal: "Hold the rollout steady: 99.9% of boxes fresh over a full window (now 99.4%).",
		PlayOn: "Your cluster keeps running. The panels after this one (extra Keeps, more fiefs, Familiars, ships, dispatches, contracts, storms and seals) are experimental: an early alpha. " +
			"They work, but they are unfinished and not tuned yet, and they may change. Each one says so on its card. Use them as they are, or stop here with a cluster that stands.",
		Close: "The full release finishes them and adds more panels, workshops and custom-item tasks."}
	return f
}

func TestTheEndingSaysTheDemoEndsAndWhatComesAfterIsExperimental(t *testing.T) {
	h := New(nil, Backend{})
	f := demoEndingFrame(t)
	h.showEnding(f)
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	text := []string{}
	for _, r := range recordRuns(func() { h.drawEnding(dst, f) }) {
		text = append(text, r.Text)
	}
	all := strings.Join(text, " ")
	for _, want := range []string{view.EndingTitle, "PLAY ON, AS IT IS", "experimental", "SEE IT IN KUBERNETES"} {
		if !strings.Contains(all, want) {
			t.Errorf("the ending panel lacks %q: %q", want, text)
		}
	}
	for _, b := range h.endingButtons(f) {
		if strings.Contains(b.Tip, "Rush Tide") {
			t.Errorf("the ending's %s tip names the surge, which a calm realm never has: %q", b.ID, b.Tip)
		}
	}
	assertTextFits(t, h, f, textPanel{name: "demo ending", bounds: fixed(RoadBox), draw: (*HUD).drawEnding, buttons: (*HUD).endingButtons})
}
