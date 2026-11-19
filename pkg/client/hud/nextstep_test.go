package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func nextStepOn(h *HUD, f Frame) (Button, bool) {
	for _, b := range h.ObjectiveButtons(f) {
		if b.ID == NextStepID {
			return b, true
		}
	}
	return Button{}, false
}

func TestTheDonePanelOffersNextStepOnlyWhileTheNextStepIsHeld(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "serve-hand", Text: "Carry 5 parcels from the quay by hand", Why: "so that you serve more requests", Done: true, Lesson: "You carried every box yourself."}}
	if _, ok := nextStepOn(h, f); ok {
		t.Fatal("NEXT STEP shows while no step is held")
	}
	f.State.Enjoying = true
	b, ok := nextStepOn(h, f)
	if !ok || b.Label != NextStepLabel {
		t.Fatalf("the done step's panel has no NEXT STEP while the next step is held: %+v", h.ObjectiveButtons(f))
	}
	card := h.ObjectiveRectOf(f)
	if b.Rect.X < card.X || b.Rect.Y < card.Y || b.Rect.X+b.Rect.W > card.X+card.W || b.Rect.Y+b.Rect.H > card.Y+card.H {
		t.Fatalf("NEXT STEP %+v sits outside its panel %+v", b.Rect, card)
	}
	for _, other := range h.ObjectiveButtons(f) {
		if other.ID != NextStepID && Overlaps(other.Rect, b.Rect) {
			t.Fatalf("NEXT STEP %+v covers %s %+v", b.Rect, other.ID, other.Rect)
		}
	}
	intents := h.Press(NextStepID, f)
	pressed := false
	for _, it := range intents {
		pressed = pressed || it.Kind == IntentAction && it.Action.Action == view.ActNextStep
	}
	if !pressed {
		t.Fatalf("NEXT STEP does not ask the realm for the next step: %+v", intents)
	}
	f.State.Objectives = append(f.State.Objectives, view.ObjectiveView{ID: "raise-keep", Text: "Raise a Keep"})
	f.State.Enjoying = false
	if _, ok := nextStepOn(h, f); ok {
		t.Fatal("NEXT STEP stays once the next step is out")
	}
}

func handDoneHUD(t *testing.T) (*HUD, Frame, view.ObjectiveView) {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	hand := view.ObjectiveView{ID: "serve-hand", Text: "Carry 5 parcels from the quay by hand", Purpose: "Each box you carry adds memory; your first Keep needs 10 of it", Progress: "2 of 5 carried"}
	f.State.Objectives = []view.ObjectiveView{hand}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 20)
	done := hand
	done.Done, done.Progress, done.Lesson = true, "", "You carried every box yourself: each added its memory."
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{done}, true
	return h, f, done
}

func TestNextStepBringsTheNextStepOnTheFrameItIsPressed(t *testing.T) {
	h, f, done := handDoneHUD(t)
	waitUntil(t, h, &f, paceLimit, "the hand step's done box", func(Frame) bool { return h.Director.Stage.DoneOn == "serve-hand" })
	waitUntil(t, h, &f, paceLimit, "NEXT STEP", func(s Frame) bool { _, ok := nextStepOn(h, s); return ok })
	clickOn(t, h, f, NextStepID)
	keep := view.ObjectiveView{ID: "raise-keep", Text: "Raise a Keep", Purpose: "Huts need a Keep to stand on", Progress: "ready: press BUILD"}
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{done, keep}, false
	tick(h, &f, 1.0/60)
	if !shownStep("raise-keep", false)(h.Stage(f)) {
		t.Fatalf("NEXT STEP pressed, and the next step is not on screen on the next frame: %s", h.StageDebug())
	}
	tick(h, &f, 5)
	noAttentionFaults(t, h)
	if faults := CheckBeats(h.Beats()); len(faults) > 0 {
		t.Fatalf("beat faults: %v", faults)
	}
}

func TestNextStepDrawnWhileOnlyTheRhythmHoldsTheStepBringsItOnTheFrame(t *testing.T) {
	h, f, done := handDoneHUD(t)
	waitUntil(t, h, &f, paceLimit, "the hand step's done box", func(Frame) bool { return h.Director.Stage.DoneOn == "serve-hand" })
	keep := view.ObjectiveView{ID: "raise-keep", Text: "Raise a Keep", Purpose: "Huts need a Keep to stand on", Progress: "ready: press BUILD"}
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{done, keep}, false
	tick(h, &f, .5)
	if shownStep("raise-keep", false)(h.Stage(f)) {
		t.Fatal("the next step came inside its done box's enjoyment time: the test proves nothing")
	}
	b, ok := nextStepOn(h, h.Stage(f))
	if !ok {
		t.Fatal("the step is held by the rhythm alone and NEXT STEP is not drawn")
	}
	h.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}, f)
	tick(h, &f, 1.0/60)
	if !shownStep("raise-keep", false)(h.Stage(f)) {
		t.Fatalf("NEXT STEP was drawn, pressed once, and the next step did not come on the next frame: %s", h.StageDebug())
	}
	tick(h, &f, 5)
	noAttentionFaults(t, h)
}

func TestNextStepAsksTheRealmWhileAPayoffHoldsTheStep(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "enroll", Text: "Put your porters on the depot's roll", Done: true}}
	f.State.Held = true
	pressed := false
	for _, it := range h.Press(NextStepID, f) {
		pressed = pressed || it.Kind == IntentAction && it.Action.Action == view.ActNextStep
	}
	if !pressed {
		t.Fatal("NEXT STEP, drawn while the realm holds the step for its result, does not ask the realm for it")
	}
}

func TestNextStepOnTheDoorsDoneBringsBorgsDecreeOnThePress(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := pendingDecreeFrame(t)
	codex := f.State.Objectives[1]
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the door step", shownStep("door-hearth", false))
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true, Lesson: "A static Pod."}, codex}
	f.State.Enjoying = true
	waitUntil(t, h, &f, paceLimit, "the door's done box", shownStep("door-hearth", true))
	b, ok := nextStepOn(h, h.Stage(f))
	if !ok {
		t.Fatal("the door's done box offers no NEXT STEP")
	}
	h.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}, f)
	f.State.Enjoying = false
	f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Purpose: "The operator briefs.", Try: "Read the brief.", Target: SurfaceDecree, Via: ViaAck}
	tick(h, &f, .25)
	if !h.QuestCardShown(h.Stage(f)) {
		t.Fatalf("NEXT STEP pressed, and Operator brief, the next step's own card, is not on screen a quarter second later: %s", h.StageDebug())
	}
	tick(h, &f, 5)
	noAttentionFaults(t, h)
}

func TestAStepBroughtOnItsPredecessorsDoneFrameIsAnsweredWhenItIsMet(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "deployment", Text: "Raise a crew", Done: true}, {ID: "crier", Text: "Open a berth", Done: true}}
	h.Now = 314.28
	s := &h.Director.Stage
	s.Primed, s.StepOn, s.StepAt, s.StepDoneAt, s.DoneOn = true, "crier", 314.28, 314.28, "deployment"
	h.Now = 317.8
	h.NoteDone(f)
	answered := false
	for _, e := range h.Attention().Events {
		answered = answered || e.Kind == EventStepDone && e.Key == ActStep+"crier"
	}
	if !answered || s.StepDoneAt != 317.8 {
		t.Fatalf("the crier step, brought on the frame the crew step was done, was met and never answered (done at %.2f)", s.StepDoneAt)
	}
}

func TestAStepMetUnderACardNeverShowsItsCountFromBeforeTheDeed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	s := &h.Director.Stage
	s.Primed, s.StepOn, s.DoneOn = true, "grow-crew", ""
	s.View = view.ObjectiveView{ID: "grow-crew", Text: "Scale the porters to 4", Progress: "3/4 porters declared"}
	st := view.GameState{}
	st.Objectives = []view.ObjectiveView{{ID: "grow-crew", Text: "Scale the porters to 4", Done: true}}
	last, ok := h.LastStep(st)
	if !ok || strings.Contains(last.Progress, "3/4") || last.Progress != StepMetLine {
		t.Fatalf("the step is met while its DONE waits behind another card, and its card reads %q", last.Progress)
	}
}

func TestAStepsOwnCardNeverWaitsMoreThanAMinutePastTheEnjoymentForAPlayerWhoNeverPresses(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := pendingDecreeFrame(t)
	codex := f.State.Objectives[1]
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the door step", shownStep("door-hearth", false))
	h.Director.Stage.Idea = h.Now
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true, Lesson: "A static Pod."}, codex}
	f.State.Enjoying = true
	doneAt := f.State.RealmTime
	waitUntil(t, h, &f, paceLimit, "the door's done box", shownStep("door-hearth", true))
	for f.State.RealmTime < doneAt+view.EnjoyTime+StepDueMost+5 && !h.QuestCardShown(h.Stage(f)) {
		if f.State.RealmTime >= doneAt+view.EnjoyTime {
			f.State.Enjoying = false
			f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Purpose: "The operator briefs.", Try: "Read the brief.", Target: SurfaceDecree, Via: ViaAck}
		}
		tick(h, &f, .25)
	}
	if late := f.State.RealmTime - doneAt - view.EnjoyTime; !h.QuestCardShown(h.Stage(f)) || late > StepDueMost {
		t.Fatalf("the codex step's own card came %.1f s past the door's enjoyment, more than %.0f s, for a player who never presses", late, StepDueMost)
	}
}

func doneUnderPanel(t *testing.T) (*HUD, Frame) {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	chronicle := view.ObjectiveView{ID: "chronicle", Text: "Answer an incident report", Purpose: "Explain a calamity and earn points for the manual", Progress: "ready: open INCIDENT LOG"}
	f.State.Objectives = []view.ObjectiveView{chronicle}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the chronicle step", shownStep("chronicle", false))
	tick(h, &f, 30)
	h.Open(OverlayChronicles)
	tick(h, &f, 1)
	chronicle.Done, chronicle.Progress = true, ""
	chronicle.Why = "so that you learn why a calamity struck, and earn points"
	chronicle.Lesson = "Every calamity here strikes real realms too: Unhoused, Guttering, Voidtaken, Banished. Knowing why is the skill."
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{chronicle, {ID: "augury", Text: "Read the Pod usage"}}, true
	tick(h, &f, .25)
	if h.Director.Stage.DoneOn != "chronicle" || h.ObjectiveDrawn(h.Stage(f)) {
		t.Fatalf("the incident log's STEP DONE did not come under the incident log panel: the test proves nothing: %s", h.StageDebug())
	}
	return h, f
}

func ticks(h *HUD, f *Frame, seconds float64) {
	for waited := 0.0; waited < seconds; waited += .25 {
		tick(h, f, .25)
	}
}

func TestANewsBoxWaitsForTheReadingTimeOfAStepDoneMetUnderAPanelThatClosesAtOnce(t *testing.T) {
	h, f := doneUnderPanel(t)
	h.Open(OverlayNone)
	h.News("An incident is answered: the manual keeps why the calamity struck, and adds its points.", false)
	ticks(h, &f, 40)
	noAttentionFaults(t, h)
}

func TestAStepDoneMetUnderAPanelIsReadFromWhenThePanelCloses(t *testing.T) {
	h, f := doneUnderPanel(t)
	ticks(h, &f, 25)
	h.Open(OverlayNone)
	ticks(h, &f, 40)
	noAttentionFaults(t, h)
}

func TestAStepsOwnCardThatComesDueInsideTheIdeaGapBreaksNoRuleOfTheRhythm(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := pendingDecreeFrame(t)
	codex := f.State.Objectives[1]
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the door step", shownStep("door-hearth", false))
	h.Result("MEMORY, RATE AND OVERHEAD", "The number is your memory. The +/S beside it is the rate: the memory your boxes add each second.")
	lessonAt := waitUntil(t, h, &f, paceLimit, "the memory hint", func(s Frame) bool { return h.LessonShown(s) })
	tick(h, &f, 3)
	clickOn(t, h, f, LessonAck)
	tick(h, &f, 1)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true, Lesson: "A static Pod."}, codex}
	f.State.Enjoying = true
	doneAt := f.State.RealmTime
	waitUntil(t, h, &f, paceLimit, "the door's done box", shownStep("door-hearth", true))
	for f.State.RealmTime < doneAt+view.EnjoyTime+StepDueMost+5 && !h.QuestCardShown(h.Stage(f)) {
		if f.State.RealmTime >= doneAt+view.EnjoyTime {
			f.State.Enjoying = false
			f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Purpose: "The operator briefs.", Try: "Read the brief.", Target: SurfaceDecree, Via: ViaAck}
		}
		tick(h, &f, .25)
	}
	if !h.QuestCardShown(h.Stage(f)) || h.Now-lessonAt >= IdeaGapDark {
		t.Fatalf("the codex step's own card came %.1f s after the memory hint (shown %v): it did not come due inside the idea gap, the test proves nothing", h.Now-lessonAt, h.QuestCardShown(h.Stage(f)))
	}
	tick(h, &f, 5)
	noAttentionFaults(t, h)
}

func TestADoneThatHadItsDoneBeatGivesWayToTheNextStepsDoneAtOnce(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	hand := view.ObjectiveView{ID: "serve-hand", Text: "Carry 5 parcels from the quay by hand", Done: true,
		Lesson: "You carried every box yourself, and each added its memory the moment it reached the depot: the realm adds only from what is carried, so the more hands carry, the more it adds each second."}
	keep := view.ObjectiveView{ID: "raise-keep", Text: "Raise a Keep", Progress: "ready: press BUILD", Build: "keep"}
	f.State.Objectives = []view.ObjectiveView{{ID: "serve-hand", Text: hand.Text, Progress: "4/5 carried"}}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{hand}, true
	waitUntil(t, h, &f, paceLimit, "the hand step's done box", func(Frame) bool { return h.Director.Stage.DoneOn == "serve-hand" })
	tick(h, &f, view.ObjectiveSoak+1)
	keep.Done, keep.Progress = true, ""
	f.State.Objectives = []view.ObjectiveView{hand, keep, {ID: "door-hearth", Text: "Set a porter at the Lone Tower's door"}}
	tick(h, &f, .25)
	if !shownStep("raise-keep", true)(h.Stage(f)) {
		t.Fatalf("the Keep step was credited after the hand step's done beat and its STEP DONE is not on screen a quarter second later: %s", h.StageDebug())
	}
	tick(h, &f, 5)
	noAttentionFaults(t, h)
}

func TestAStepDoneJustAfterACelebrationStartsComesAtOnceAndTheCelebrationReplaysAfter(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	class := view.ObjectiveView{ID: "porter-class", Text: "Train the porters to class 2.0", Progress: "6/7 huts on porter:2.0"}
	f.State.Objectives = []view.ObjectiveView{class}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the class step", shownStep("porter-class", false))
	tick(h, &f, 30)
	h.FeatBanner("PORTERS: CLASS 2.0", "7 huts rebuilt one by one.")
	waitUntil(t, h, &f, paceLimit, "the celebration", func(Frame) bool { return h.BannerActive() })
	tick(h, &f, .5)
	class.Done, class.Progress = true, ""
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{class, {ID: "harbor-two", Text: "Upgrade a port for caravels"}}, true
	tick(h, &f, .25)
	if !shownStep("porter-class", true)(h.Stage(f)) || h.BannerActive() {
		t.Fatalf("the class step was met half a second into its celebration and its STEP DONE is not alone on screen a quarter second later (banner %v): %s", h.BannerActive(), h.StageDebug())
	}
	waitUntil(t, h, &f, paceLimit, "the celebration again, after the STEP DONE", func(Frame) bool { return h.BannerActive() })
	tick(h, &f, 10)
	noAttentionFaults(t, h)
}

func TestANewsBoxWaitsForTheReadingTimeOfTheStepDoneAsShownNotAsAdmitted(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	augury := view.ObjectiveView{ID: "augury", Text: "Read the Pod usage: TOP HUTS", Progress: "ready: use the Stone"}
	f.State.Objectives = []view.ObjectiveView{augury}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the augury step", shownStep("augury", false))
	tick(h, &f, 30)
	augury.Done, augury.Progress = true, ""
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{augury, {ID: "draw-crowd", Text: "Draw a crowd"}}, true
	tick(h, &f, .25)
	augury.Lesson = "The usage page reads what each hut really uses right now: the food and housing its workers eat and fill while boxes wait, so you can pledge what they need at noon and no more."
	f.State.Objectives = []view.ObjectiveView{augury, {ID: "draw-crowd", Text: "Draw a crowd"}}
	h.Toast("The usage page is read: TOP PODS lists the food and housing each of your 7 huts really uses right now.", false)
	tick(h, &f, 20)
	noAttentionFaults(t, h)
}

func TestAStepDoneComesAtOnceWhenTheBoxBeforeItHasLeftTheScreenWithinItsReadingTime(t *testing.T) {
	for _, gone := range []struct {
		name string
		show func(h *HUD)
		up   func(h *HUD) bool
	}{
		{"banner", func(h *HUD) { h.Banner("NEW CARGO: LEDGERS") }, func(h *HUD) bool { return h.BannerActive() }},
		{"toast", func(h *HUD) {
			h.News("The porters carry the parcels now. Your hands still count: click the pile any time.", false)
		}, func(h *HUD) bool { return len(h.ToastList) > 0 }},
	} {
		t.Run(gone.name, func(t *testing.T) {
			h := New(nil, (&fakeRunner{}).backend())
			h.SetTaught([]string{TeachTopBar})
			h.SetAttention(true)
			f := directorFrame(t)
			bell := view.ObjectiveView{ID: "hang-bell", Text: "Hang the watchman's bell on the porters", Progress: "ready: set the probes"}
			f.State.Objectives = []view.ObjectiveView{bell}
			f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
			waitUntil(t, h, &f, paceLimit, "the bell step", shownStep("hang-bell", false))
			tick(h, &f, 30)
			gone.show(h)
			waitUntil(t, h, &f, paceLimit, "the "+gone.name, func(Frame) bool { return gone.up(h) })
			waitUntil(t, h, &f, paceLimit, "the "+gone.name+" leaving the screen", func(Frame) bool { return !gone.up(h) })
			if b := h.Director.Stage.Slot; b.Key == "" || h.Now >= b.ReadEnd() {
				t.Fatalf("the %s left the screen after its reading time (slot %+v at %.1f): the test proves nothing", gone.name, b, h.Now)
			}
			bell.Done, bell.Progress = true, ""
			f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{bell, {ID: "ledgers", Text: "Raise clerks"}}, true
			tick(h, &f, .25)
			if !shownStep("hang-bell", true)(h.Stage(f)) {
				t.Fatalf("the bell step was met after the %s left the screen, and its STEP DONE waits for the %s's reading time (the hasty player: 3.4 s behind NEW CARGO: LEDGERS at 10:01, 1.5 s behind a toast at 03:48 in NAMES): %s", gone.name, gone.name, h.StageDebug())
			}
			tick(h, &f, 5)
			noAttentionFaults(t, h)
		})
	}
}

func TestAStepDoneShowsOnceEvenWhenAnEarlierStepComesBackAndIsMet(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	horn := view.ObjectiveView{ID: "war-horn", Text: "Sound a War Horn", Progress: "ready: learn War Horns"}
	f.State.Objectives = []view.ObjectiveView{horn}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the War Horn step", shownStep("war-horn", false))
	horn.Done, horn.Progress = true, ""
	clerks := view.ObjectiveView{ID: "clerk-classes", Text: "Roll class 1.1 out to the clerks", Progress: "ready: pull clerk:1.1"}
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{horn, clerks}, true
	waitUntil(t, h, &f, paceLimit, "the War Horn's STEP DONE", shownStep("war-horn", true))
	f.State.Enjoying = false
	waitUntil(t, h, &f, paceLimit, "the clerks' step", shownStep("clerk-classes", false))
	ward := view.ObjectiveView{ID: "see-ward", Text: "Find the Royal Ward on the Lone Tower", Progress: "ready: select the Lone Tower"}
	f.State.Objectives = []view.ObjectiveView{ward}
	waitUntil(t, h, &f, paceLimit, "the Ward's step, felt now", shownStep("see-ward", false))
	ward.Done, ward.Progress = true, ""
	f.State.Objectives, f.State.Enjoying = []view.ObjectiveView{ward, horn, clerks}, true
	tick(h, &f, 30)
	f.State.Enjoying = false
	tick(h, &f, 30)
	shown := 0
	for _, b := range h.Attention().Boxes {
		if b.Key == "done:war-horn" {
			shown++
		}
	}
	if shown != 1 {
		t.Fatalf("the War Horn's STEP DONE showed %d times: the Ward's step, met when it came back, brought the old DONE behind it again (the hasty player at 15:04: done:war-horn beside done:see-ward)", shown)
	}
	noAttentionFaults(t, h)
}
