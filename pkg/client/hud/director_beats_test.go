package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func beatKeys(beats []Beat) []string {
	out := []string{}
	for _, b := range beats {
		if b.Kind != BeatGone && b.Kind != BeatShut && b.Kind != BeatReveal && b.Kind != BeatToast {
			out = append(out, b.Key)
		}
	}
	return out
}

func runFor(h *HUD, f *Frame, seconds float64) {
	for t := 0.0; t < seconds; t += .25 {
		tick(h, f, .25)
	}
}

func TestAStepDoneSinksInThenTheDecreeThenTheStepThenTheLessonOneAtATime(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	runFor(h, &f, 1)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true},
		{ID: "build-codex", Text: "Check the control plane", Progress: "ready: press BUILD", Build: "codex", Quest: "borg-founding-dev"}}
	f.Guide.Target = "build:codex"
	f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Try: "Raise the four Deployments.", Via: ViaAck}
	f.State.Quests = []view.QuestView{{ID: "borg-founding-dev", Title: "Operator brief", Giver: "Borg", Parts: []view.QuestPartView{{Title: "The Codex", Objective: "build-codex"}}}}
	f.State.Reveals = append(slices.Clone(f.State.Reveals), view.BorgIntro)
	f.Guide.Staged = true
	h.LessonCard = LessonCard{Title: "MEMORY, USE AND LIMIT", Text: "Your memory.", Shown: true, Teach: true}
	waitUntil(t, h, &f, paceLimit, "the done step", shownStep("door-hearth", true))
	waitUntil(t, h, &f, paceLimit, "the brief", func(s Frame) bool { return h.QuestCardShown(s) })
	if h.LessonShown(h.Stage(f)) {
		t.Fatal("the memory hint came with the brief")
	}
	f.Guide.Intro = nil
	f.Guide.Introduced = IntroTargets([]string{view.BorgIntro})
	waitUntil(t, h, &f, paceLimit, "the step", shownStep("build-codex", false))
	if h.LessonShown(h.Stage(f)) {
		t.Fatal("the hint came with the step")
	}
	waitUntil(t, h, &f, paceLimit, "the hint", func(s Frame) bool { return h.LessonShown(s) })
	noAttentionFaults(t, h)
	want := []string{"done:door-hearth", "quest:borg-founding-dev", "step:build-codex", "lesson:MEMORY, USE AND LIMIT"}
	if got := boxKeys(h.Attention()); !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}

func TestTheFirstShipsTreasureChestReachesTheScreenAtAnySpeed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.Result("A GILDED SHIP DOCKED", "Claim its boxes.")
	runFor(h, &f, .5)
	h.CelebrateChest(f, "The first ship", 0, 6800)
	runFor(h, &f, 5)
	if _, ok := h.BannerRect(h.Stage(f)); ok {
		t.Fatal("the chest's banner landed on top of a card that waits for GOT IT")
	}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	clickOn(t, h, h.Stage(f), LessonAck)
	waitUntil(t, h, &f, paceLimit, "the chest's banner", func(s Frame) bool {
		_, ok := h.BannerRect(s)
		return ok && strings.Contains(h.Juice.Cause, "6.8K")
	})
	shown := h.Now
	for range 8 {
		f.Clock += .25
		f.EventClock += 5
		h.Update(Input{MX: -10, MY: -10}, f)
	}
	if _, ok := h.BannerRect(h.Stage(f)); !ok {
		t.Fatalf("at x20 the chest's banner was gone after %.1f s", h.Now-shown)
	}
}

func TestTheVerdictComesFirstThenTheQuietThenTheEnding(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.FeatBanner("SURGE HANDLED", "99% delivered fresh.")
	f.State.Kube.Ending = &view.EndingView{At: 1, Goal: "the full release"}
	runFor(h, &f, 1)
	if h.Overlay() == OverlayEnding {
		t.Fatal("the ending opened over the verdict")
	}
	for waited := 0.0; waited < paceLimit && h.Overlay() != OverlayEnding; waited += .25 {
		f.Clock += .25
		if h.Update(Input{MX: -10, MY: -10}, f); h.Overlay() == OverlayEnding {
			break
		}
		if h.BannerActive() && h.Overlay() == OverlayEnding {
			t.Fatal("the ending opened over the verdict")
		}
	}
	if h.Overlay() != OverlayEnding {
		t.Fatal("the ending never followed the verdict")
	}
}

func TestShowMeOnTheStepGoesOnlyWhereTheStepItselfPoints(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "harbor-two", Text: "Upgrade a port for caravels", Progress: "memory 2134/3000 toward Gate Harbor's caravels"}}
	f.Guide.Target = ""
	f.GoalAt = &[2]float64{40, 20}
	f.GoalOfNext = true
	runFor(h, &f, 1)
	if _, ok := HitID(h.Buttons(f), "obj:goal"); ok {
		t.Fatal("the step's SHOW ME glides to the harbor board's NEXT, a different place than the step names")
	}
	f.GoalOfNext = false
	if _, ok := HitID(h.Buttons(f), "obj:goal"); !ok {
		t.Fatal("a step with its own place lost SHOW ME")
	}
}

func TestTheDawnsLinesComeWithTheBannerAndNothingElseUntilItIsRead(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.Open(OverlayAge)
	f.State.Age = view.AgeFeudal
	f.State.AgeMoment = &view.AgeMomentView{Age: view.AgeFeudal, Lines: []string{"Now you can read 3 pages in the manual: Pledges, Masonry and 1 more", "Your first step: Find the Ward on the first Keep.", "Your Keeps now cost 0.4 requests a second in overhead, taken from what the cluster serves, never from the pool."},
		Opened: []string{"Stone Keeps cost more now: the next 420 MiB"}, Changed: []string{"The Nines now ask 99.9% ready"}}
	h.FeatBanner("KEEP NODE-A CORDONED", "Hold what you built.")
	runFor(h, &f, 1)
	if _, ok := h.BannerRect(h.Stage(f)); !ok {
		t.Fatal("the banner does not show over the panel")
	}
	if !h.DawnLinesShown(h.Stage(f)) {
		t.Fatal("the dawn's lines wait behind the banner's quiet: a player who closes the panel at once never reads what was built or the first step")
	}
	runFor(h, &f, BeatQuiet)
	if _, ok := HitID(h.Buttons(f), DawnMore); !ok {
		t.Fatal("no MORE for the rest")
	}
	f.Guide.Intro = &Intro{ID: "lore", Title: "Housing", Try: "Compare.", Target: MeterLore, Via: ViaAck}
	h.Result("A GILDED SHIP DOCKED", "Claim its boxes.")
	runFor(h, &f, 10)
	if s := h.Stage(f); h.IntroShown(s) || h.LessonShown(s) {
		t.Fatal("a card came over the open panel")
	}
	h.Open(OverlayNone)
	waitUntil(t, h, &f, paceLimit, "one card after the panel closed", func(s Frame) bool { return h.IntroShown(s) || h.LessonShown(s) })
	s := h.Stage(f)
	if h.IntroShown(s) && h.LessonShown(s) {
		t.Fatal("after the panel closed, two cards came at once")
	}
}

func TestTheGildedShipsCardCarriesAShowButtonToItsQuay(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	at := [2]float64{40, 22}
	h.ResultAt("A GILDED SHIP DOCKED", "Claim its boxes.", &at)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	it, ok := hasIntent(clickOn(t, h, h.Stage(f), LessonShow), IntentFocus)
	if !ok || it.X != 40 || it.Y != 22 {
		t.Fatalf("SHOW did not glide to the quay: %+v %v", it, ok)
	}
}

func TestACardAlreadyUpStaysUpThroughAnAchievement(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	f.Guide.Staged = true
	h.SetAttention(true)
	h.LessonCard = LessonCard{Title: "MEMORY, USE AND LIMIT", Text: "Your memory.", Shown: true, Teach: true}
	waitUntil(t, h, &f, paceLimit, "the hint card", func(s Frame) bool { return h.LessonShown(s) })
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true},
		{ID: "build-codex", Text: "Check the control plane", Progress: "ready: press BUILD", Build: "codex"}}
	f.Guide.Target = "build:codex"
	for range 60 {
		tick(h, &f, .25)
		if !h.LessonShown(h.Stage(f)) {
			t.Fatal("the finished step set the hint card aside: it flashes and comes back")
		}
	}
	noAttentionFaults(t, h)
}

func TestTheLedgerFlagsACardSetAsideAndShownAgainWithinTheQuiet(t *testing.T) {
	beats := []Beat{
		{At: 10, Kind: BeatCard, Key: "lesson:gold", Waits: true},
		{At: 20, Kind: BeatDone, Key: "done:door", Feat: true},
		{At: 20, Kind: BeatGone, Key: "lesson:gold"},
		{At: 24.5, Kind: BeatCard, Key: "lesson:gold", Waits: true},
	}
	faults := CheckBeats(beats)
	if len(faults) != 1 || faults[0].Rule != RuleBeatFlash {
		t.Fatalf("faults %v, want one flash", faults)
	}
	calm := []Beat{
		{At: 10, Kind: BeatCard, Key: "lesson:gold", Waits: true},
		{At: 20, Kind: BeatGone, Key: "lesson:gold"},
		{At: 40, Kind: BeatCard, Key: "lesson:gold", Waits: true},
	}
	if faults := CheckBeats(calm); len(faults) != 0 {
		t.Fatalf("a card shown again long after reads as a flash: %v", faults)
	}
	trouble := []Beat{
		{At: 10, Kind: BeatCard, Key: "trouble:unhoused:haulers", Waits: true},
		{At: 20, Kind: BeatGone, Key: "trouble:unhoused:haulers"},
		{At: 23, Kind: BeatCard, Key: "trouble:unhoused:haulers", Waits: true},
	}
	if faults := CheckBeats(trouble); len(faults) != 0 {
		t.Fatalf("a trouble that strikes again follows the realm, not the stage: %v", faults)
	}
}

func TestAStepDoneUnderAnUnreadCardComesAtOnceAndTheCardStepsAsideUntilAfter(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	f.Guide.Staged = true
	runFor(h, &f, 1)
	h.LessonCard = LessonCard{Title: "MEMORY, USE AND LIMIT", Text: "Your memory.", Shown: true, Teach: true}
	waitUntil(t, h, &f, paceLimit, "the hint card", func(s Frame) bool { return h.LessonShown(s) })
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true},
		{ID: "build-codex", Text: "Check the control plane", Progress: "ready: press BUILD", Build: "codex"}}
	f.State.Enjoying = true
	f.Guide.Target = "build:codex"
	tick(h, &f, .25)
	if !shownStep("door-hearth", true)(h.Stage(f)) {
		t.Fatalf("the step was met under an unread card and its STEP DONE is not on screen a quarter second later: %s", h.StageDebug())
	}
	if h.LessonShown(h.Stage(f)) {
		t.Fatal("the STEP DONE came beside the hint card: one box at a time")
	}
	b, ok := nextStepOn(h, h.Stage(f))
	if !ok {
		t.Fatal("the done step offers no NEXT STEP")
	}
	h.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}, f)
	f.State.Enjoying = false
	tick(h, &f, 1.0/60)
	if !shownStep("build-codex", false)(h.Stage(f)) {
		t.Fatalf("NEXT STEP pressed, and the next step is not on screen on the next frame: %s", h.StageDebug())
	}
	back := waitUntil(t, h, &f, paceLimit, "the hint card back", func(s Frame) bool { return h.LessonShown(s) })
	if back <= 0 {
		t.Fatal("the hint set aside never came back")
	}
	clickOn(t, h, h.Stage(f), LessonAck)
	tick(h, &f, 5)
	noAttentionFaults(t, h)
}

func TestAStepDoneUnderATroubleStillBeingReadWaitsForItsReadingTime(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	runFor(h, &f, 1)
	s := &h.Director.Stage
	s.On[ActTrouble+"spoiling"], s.Waits[ActTrouble+"spoiling"] = h.Now, true
	s.Slot = SlotBoxOf{Key: ActTrouble + "spoiling", From: h.Now, Read: 8, Waits: true}
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true},
		{ID: "build-codex", Text: "Check the control plane", Progress: "ready: press BUILD", Build: "codex"}}
	f.State.Enjoying = true
	h.AdmitAsked(f, nil)
	if s.DoneOn == "door-hearth" || !s.Has(ActTrouble+"spoiling") {
		t.Fatal("a STEP DONE set aside a trouble card the player had not had time to read")
	}
	h.Now += 8
	h.AdmitAsked(f, nil)
	if s.DoneOn != "door-hearth" || s.Has(ActTrouble+"spoiling") || !h.Aside(ActTrouble+"spoiling") {
		t.Fatalf("after the trouble's reading time the STEP DONE did not take the screen with the trouble aside (done on %q)", s.DoneOn)
	}
}

func TestAnAchievementsCelebrationWaitsUntilTheCardUpIsAnswered(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "build-stewards", Text: "Raise the Stewards' Hall", Progress: "ready: press BUILD", Build: "stewards-hall"}}
	f.Guide.Target = "build:stewards-hall"
	f.Guide.Staged = true
	h.LessonCard = LessonCard{Title: "MEMORY, USE AND LIMIT", Text: "Your memory.", Shown: true, Teach: true}
	runFor(h, &f, BeatGap+1)
	f.State.Objectives = []view.ObjectiveView{{ID: "build-stewards", Text: "Raise the Stewards' Hall", Done: true},
		{ID: "summon-porter", Text: "Hire porters by hand", Progress: "ready: press HIRE", Target: "cmd:summon"}}
	f.Guide.Target = "cmd:summon"
	h.FeatBanner("THE ROLLOUT IS COMPLETE", "The operator brief is complete.")
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the hint card back after the step's STEP DONE", func(s Frame) bool { return h.LessonShown(s) })
	for range 4 * 60 {
		tick(h, &f, .25)
		if h.BannerActive() {
			t.Fatalf("the celebration played over the unread hint card: banner %q", h.Juice.Banner)
		}
		if !h.LessonShown(h.Stage(f)) {
			t.Fatal("the hint card was set aside for the celebration")
		}
	}
	h.LessonCard = LessonCard{}
	waitUntil(t, h, &f, paceLimit, "the celebration after GOT IT", func(Frame) bool { return h.BannerActive() })
}

func TestADoneStepThatLeavesTheListNeverComesBackAsLive(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.Guide.Staged = true
	f.State.Objectives = []view.ObjectiveView{{ID: "pledges", Text: "Pledge food for your porters", Progress: "the Deployment rolls out pledged porters"}}
	runFor(h, &f, BeatGap+1)
	f.State.Objectives = []view.ObjectiveView{{ID: "pledges", Text: "Pledge food for your porters", Done: true},
		{ID: "two-nines", Text: "Set availability on course for 99.9%", Progress: "96.3% delivered fresh"}}
	runFor(h, &f, 1)
	f.State.Objectives = []view.ObjectiveView{{ID: "see-ward", Text: "Find the Ward", Progress: "ready: select the Keep"}}
	for t2 := 0.0; t2 < BeatQuiet; t2 += .25 {
		tick(h, &f, .25)
		if o, ok := h.Stage(f).ShownObjective(); ok && o.ID == "pledges" && !o.Done {
			t.Fatalf("the finished step came back as a live step after its achievement: stage %s", h.StageDebug())
		}
	}
	for _, b := range h.Beats() {
		if b.Key == "step:pledges" {
			t.Fatalf("beats %v", beatKeys(h.Beats()))
		}
	}
}

func TestTheNextStepReachesTheCardWhileTheShortStoneStaysOpen(t *testing.T) {
	for _, tall := range []bool{false, true} {
		h := New(nil, (&fakeRunner{}).backend())
		f := directorFrame(t)
		f.State.Objectives = []view.ObjectiveView{{ID: "ask-keeps", Text: "Ask the Chancery which Keeps stand", Progress: "ready: speak it", Target: "well:stone"}}
		f.Guide.Target = "well:stone"
		f.Guide.Staged = true
		runFor(h, &f, BeatGap+1)
		h.Open(OverlayStone)
		h.Stone.Tall = tall
		f.State.Objectives = []view.ObjectiveView{{ID: "ask-keeps", Text: "Ask the Chancery which Keeps stand", Done: true},
			{ID: "build-codex", Text: "Check the control plane", Progress: "ready: press BUILD", Build: "codex"}}
		f.Guide.Target = "build:codex"
		for waited := 0.0; waited < paceLimit && h.Director.Stage.StepOn != "build-codex"; waited += .25 {
			tick(h, &f, .25)
		}
		switch on := h.Director.Stage.StepOn; {
		case !tall && on != "build-codex":
			t.Fatalf("a player who keeps the short Stone open never sees the next step: stage %s", h.StageDebug())
		case tall && on == "build-codex":
			t.Fatal("the next step was put on a card the tall Stone covers")
		}
	}
}
