package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestGuidanceWaitsWhileThePlayerReadsAWindow(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	tick(h, &f, 1)
	if h.Voice(f) != VoiceIntro || !h.IntroShown(f) {
		t.Fatal("the intro does not speak with no window open")
	}
	h.OverlayID = OverlayChronicles
	tick(h, &f, 1)
	if h.Voice(f) != "" || h.IntroShown(f) {
		t.Fatalf("while a window is read the intro speaks: voice %q", h.Voice(f))
	}
	if _, ok := h.Step(f); ok {
		t.Fatal("the guide points out of the window being read")
	}
	f.Guide.Intro = nil
	if h.Voice(f) != "" {
		t.Fatalf("while a window is read the task speaks: %q", h.Voice(f))
	}
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	tick(h, &f, readGrace)
	if h.Voice(f) != VoiceIntro {
		t.Fatalf("after a long read the guide still waits: %q", h.Voice(f))
	}
	h.Update(Input{MX: -10, MY: -10, Wheel: 1}, f)
	if h.Voice(f) != "" {
		t.Fatal("scrolling the window does not restart the reading pause")
	}
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	h.OverlayID = OverlayNone
	tick(h, &f, .1)
	if h.Voice(f) != VoiceIntro {
		t.Fatalf("once the window closes the intro is quiet: %q", h.Voice(f))
	}
}

func TestAnOffScreenTargetGetsOnlyTheArrow(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := navIntroFrame(t, "pod:default/porter-a")
	f.Beacons = []ui.Rect{{X: 300, Y: -200, W: 20, H: 30}}
	if h.holeStroked(f) {
		t.Fatal("an off-screen target draws a square at the edge")
	}
	f.Beacons = []ui.Rect{{X: 300, Y: 200, W: 20, H: 30}}
	if !h.holeStroked(f) {
		t.Fatal("an on-screen target loses its outline")
	}
}

func TestTheQoSLessonPointsAtAHutUntilOneIsSelected(t *testing.T) {
	f := fullFrame(t)
	f.Beacons = []ui.Rect{{X: 300, Y: 200, W: 20, H: 30}}
	if got := qosRect(nil, f); got != f.Beacons[0] {
		t.Fatalf("with nothing selected the QoS hint points at %+v", got)
	}
	f.Selection = Selection{Kind: SelectHearth, ID: "default/porters-a"}
	if got := qosRect(nil, f); got != CardBox {
		t.Fatalf("with a hut selected the QoS hint points at %+v", got)
	}
}
