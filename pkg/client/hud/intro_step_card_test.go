package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

var topBarIntros = []string{MeterGold, MeterWisdom, MeterRenown, MeterNines, MeterUpkeep, "meter:hands", "meter:lore", "meter:codex-bloat", SurfaceSpeed, SurfaceOverlays}

func hireStepFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{
		ID: view.SummonObjective, Text: "Hire porters by hand: a big ship's boxes spoil at the door",
		Purpose:  "A big ship's boxes time out at the door, each one a request nobody keeps",
		Progress: "ready: press HIRE (hired 0/3)", Target: "cmd:summon",
	}}
	f.Guide.Target = "cmd:summon"
	return f
}

func TestNoTopBarIntroCardCoversTheStepCard(t *testing.T) {
	for _, target := range topBarIntros {
		h := New(nil, (&fakeRunner{}).backend())
		h.SetStaging(false)
		f := hireStepFrame(t)
		f.Guide.Intro = &Intro{ID: "top", Title: "Knowledge", Purpose: "New kubectl words that read or change the cluster list results, history and steps too. It opens pages.",
			Try: "Point at the owl number: it lists what each word returned.", Target: target, Via: ViaAck}
		tick(h, &f, 1)
		s := h.Stage(f)
		if !h.IntroShown(s) || !h.ObjectiveShown(s) {
			t.Fatalf("%s: the fixture shows intro %v, step card %v: it proves nothing", target, h.IntroShown(s), h.ObjectiveShown(s))
		}
		if _, ok := h.TargetRect(target, s); !ok {
			continue
		}
		card, step := h.IntroCard(s), h.ObjectiveRectOf(s)
		if got := overlapArea(card, step); got != 0 {
			t.Errorf("%s: the intro card %+v covers %d px of the step card %+v", target, card, got, step)
		}
	}
}

const midStepWatch = 60.0

func TestACardThePlayerDidNotCauseWaitsUntilThePlayerIsNotMidStep(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := hireStepFrame(t)
	tick(h, &f, 1)
	f.Guide.Intro = &Intro{ID: "wisdom", Title: "Knowledge", Purpose: "New kubectl words list details.", Try: "Point at the owl number.", Target: MeterWisdom, Via: ViaAck}
	for waited := 0.0; waited < midStepWatch; waited += .5 {
		tick(h, &f, .5)
		if h.IntroShown(h.Stage(f)) {
			t.Fatalf("at %.1f s the Knowledge card came while the player is hiring", waited)
		}
	}
	f.State.Objectives = []view.ObjectiveView{
		{ID: view.SummonObjective, Text: "Hire porters by hand", Done: true},
		{ID: "build-stewards", Text: "Raise the Stewards' Hall", Progress: "being built", Build: "stewards-hall"},
	}
	f.Guide.Target = ""
	for waited := 0.0; waited < midStepWatch; waited += .5 {
		tick(h, &f, .5)
		if h.IntroShown(h.Stage(f)) {
			return
		}
	}
	t.Fatal("the Knowledge card never came while the player only waits for the Hall to rise")
}

func TestACardTheRealmSetAsideNeverFlashesBackWithinTheQuiet(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := hireStepFrame(t)
	f.State.Objectives[0].Progress = "being built"
	tick(h, &f, 1)
	speed := &Intro{ID: "speed", Title: "Hasten time", Purpose: "Realm time may be hastened.", Try: "Press 2X.", Target: SurfaceSpeed, Via: ViaAck}
	f.Guide.Intro = speed
	shown := false
	for waited := 0.0; waited < midStepWatch && !shown; waited += .5 {
		tick(h, &f, .5)
		shown = h.IntroShown(h.Stage(f))
	}
	if !shown {
		t.Fatal("the speed card never came while the player waits")
	}
	f.Guide.Intro = nil
	tick(h, &f, .5)
	f.Guide.Intro = speed
	for waited := .5; waited < BeatQuiet+BeatGap; waited += .5 {
		tick(h, &f, .5)
		if h.IntroShown(h.Stage(f)) {
			t.Fatalf("the card set aside came back %.1f s later, inside the quiet", waited+.5)
		}
	}
	if faults := CheckBeats(h.Beats()); len(faults) > 0 {
		t.Fatalf("beat faults: %v", faults)
	}
}
