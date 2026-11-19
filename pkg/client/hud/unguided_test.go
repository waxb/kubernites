package hud

import (
	"strings"
	"testing"
)

func TestWithTheGuideOffALessonArrivesAsANote(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := directorFrame(t)
	f.Unguided = true
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Purpose: "Use memory on buildings.", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	intents := h.Update(Input{MX: -10, MY: -10}, f)
	if len(intents) != 1 || intents[0].Kind != IntentAction || intents[0].Action.Action != AckAction || intents[0].Action.ID != "build" {
		t.Fatalf("the hint was not taken as read: %+v", intents)
	}
	if h.IntroShown(f) || h.Voice(f) != "" {
		t.Fatalf("the guide still speaks: card %v, voice %q", h.IntroShown(f), h.Voice(f))
	}
	toasts := h.Toasts()
	if len(toasts) != 1 || !strings.Contains(toasts[0].Text, "Build") || !strings.Contains(toasts[0].Text, "Use memory") {
		t.Fatalf("the hint left no note: %+v", toasts)
	}
	if again := h.Update(Input{MX: -10, MY: -10}, f); len(again) != 0 {
		t.Fatalf("the same hint was taken twice: %+v", again)
	}
	f.Guide.Intro = nil
	if _, ok := f.current(); ok && h.Voice(f) != "" {
		t.Fatalf("the task points with the guide off: %q", h.Voice(f))
	}
}

func TestANewNoteNeverArrivesAfterItsThingWasUsed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.Unguided = true
	runFor(h, &f, BeatGap)
	f.Guide.Intro = &Intro{ID: "summon", Title: "Hire a porter", Purpose: "One hut with one porter.", Try: "Press HIRE.", Target: CmdSummonID, Via: ViaAct}
	h.Director.Stage.LastNew = h.Now
	h.Update(Input{MX: -10, MY: -10}, f)
	f.Guide.Intro = nil
	for _, toast := range h.Toasts() {
		if strings.Contains(toast.Text, "New: Hire a porter") {
			t.Skip("the note showed at once, before HIRE could be pressed")
		}
	}
	h.Press(CmdSummonID, f)
	for range int((BeatQuiet + BeatGap + 2) * 4) {
		tick(h, &f, .25)
		for _, toast := range h.Toasts() {
			if strings.Contains(toast.Text, "New: Hire a porter") {
				t.Fatalf("the New: note came after HIRE was pressed: %q", toast.Text)
			}
		}
	}
}
