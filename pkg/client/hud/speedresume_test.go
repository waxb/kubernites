package hud

import "testing"

func TestPressingASpeedWhilePausedResumesAtThatSpeed(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.Paused = true
	out := clickOn(t, h, f, "speed:5")
	if it, _ := hasIntent(out, IntentAction); it.Action.Speed != 5 {
		t.Errorf("speed = %+v", it)
	}
	if _, ok := hasIntent(out, IntentPause); !ok {
		t.Error("a speed pressed while paused must resume the realm")
	}
	f.Paused = false
	if _, ok := hasIntent(clickOn(t, h, f, "speed:5"), IntentPause); ok {
		t.Error("a speed pressed while running must not toggle the pause")
	}
}
