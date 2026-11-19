package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheLedgerFlagsANewIdeaInsideTheSoak(t *testing.T) {
	burst := []Beat{
		{At: 10, Kind: BeatCard, Key: "quest:borg-founding-dev", Waits: true, Idea: true},
		{At: 14, Kind: BeatGone, Key: "quest:borg-founding-dev"},
		{At: 16.5, Kind: BeatStep, Key: "step:build-codex"},
		{At: 19, Kind: BeatCard, Key: "lesson:CPU, RATE AND OVERHEAD", Waits: true, Idea: true},
	}
	faults := CheckBeats(burst)
	if len(faults) != 1 || faults[0].Rule != RuleBeatSoak {
		t.Fatalf("faults %v, want the CPU hint inside the brief's soak", faults)
	}
	calm := slices.Clone(burst)
	calm[3].At = 14 + SoakGap + .1
	if faults := CheckBeats(calm); len(faults) != 0 {
		t.Fatalf("a step inside the soak and an idea after it are calm: %v", faults)
	}
	urgent := slices.Clone(burst)
	urgent[3] = Beat{At: 19, Kind: BeatCard, Key: "trouble:spoiling:parcel", Waits: true, Idea: true, Urgent: true}
	if faults := CheckBeats(urgent); len(faults) != 0 {
		t.Fatalf("urgent trouble comes at once, whatever soaks: %v", faults)
	}
	after := []Beat{
		{At: 10, Kind: BeatCard, Key: "trouble:spoiling:parcel", Waits: true, Idea: true, Urgent: true},
		{At: 14, Kind: BeatGone, Key: "trouble:spoiling:parcel"},
		{At: 16.5, Kind: BeatCard, Key: "intro:rail", Waits: true, Idea: true},
	}
	if faults := CheckBeats(after); len(faults) != 0 {
		t.Fatalf("urgent trouble is no idea to soak: the card it set aside comes back after the beat: %v", faults)
	}
	aside := []Beat{
		{At: 10, Kind: BeatCard, Key: "intro:rail", Waits: true, Idea: true},
		{At: 20, Kind: BeatGone, Key: "intro:rail"},
		{At: 20, Kind: BeatCard, Key: "trouble:spoiling:parcel", Waits: true, Idea: true, Urgent: true},
		{At: 25, Kind: BeatGone, Key: "trouble:spoiling:parcel"},
		{At: 27.5, Kind: BeatCard, Key: "intro:rail", Waits: true, Idea: true},
		{At: 31, Kind: BeatGone, Key: "intro:rail"},
		{At: 40, Kind: BeatCard, Key: "lesson:BADGE", Waits: true, Idea: true},
	}
	faults = CheckBeats(aside)
	if len(faults) != 1 || faults[0].Rule != RuleBeatSoak || faults[0].Keys[0] != "lesson:BADGE" {
		t.Fatalf("a card set aside for urgent trouble comes back as no new idea, and the next new one soaks after it: %v", faults)
	}
	quiet := slices.Clone(burst)
	quiet[3] = Beat{At: 19, Kind: BeatCard, Key: "trouble:unhoused:porters", Waits: true, Idea: true}
	if faults := CheckBeats(quiet); len(faults) != 1 || faults[0].Rule != RuleBeatSoak {
		t.Fatalf("trouble that loses nothing waits out the soak: %v", faults)
	}
	page := []Beat{
		{At: 10, Kind: BeatNotice, Key: "notice:decision:cmd:grimoire", Idea: true},
		{At: 13, Kind: BeatCard, Key: "intro:pledges", Waits: true, Idea: true},
	}
	if faults := CheckBeats(page); len(faults) != 1 || faults[0].Rule != RuleBeatSoak {
		t.Fatalf("a manual page on the rail soaks in too: %v", faults)
	}
}

func twoIdeaCards(h *HUD, f Frame) bool {
	n := 0
	if h.IntroShown(f) {
		n++
	}
	if h.LessonShown(f) {
		n++
	}
	if _, ok := h.QuestCard(f); ok {
		n++
	}
	if v, ok := h.ShownCalamity(f); ok && v.Mended == "" {
		n++
	}
	return n > 1
}

func waitForLesson(t *testing.T, h *HUD, f *Frame, title string, limit float64) float64 {
	limit = max(limit, paceLimit)
	t.Helper()
	for waited := 0.0; waited < limit; waited += .25 {
		tick(h, f, .25)
		s := h.Stage(*f)
		if twoIdeaCards(h, s) {
			t.Fatalf("two cards stacked at %.2f: %s", h.Now, h.StageDebug())
		}
		if h.LessonShown(s) && h.LessonTitle() == title {
			return h.Now
		}
	}
	t.Fatalf("the card %q never came within %.0f s: %s", title, limit, h.StageDebug())
	return 0
}

func TestTheFeudalDawnsCardsComeOneAtATimeEachAnIdeaGapApart(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.FeatBanner("KEEP NODE-A CORDONED", "Pods move away.")
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	h.Result("WHILE YOU WERE AWAY (12:57)", "Your porters carried 900 boxes.")
	h.Result("THE OMENS ARE READ", "TOP PODS lists what each hut really uses.")
	last := -1.0
	for _, title := range []string{"A TRAFFIC SPIKE ARRIVED", "WHILE YOU WERE AWAY (12:57)", "THE OMENS ARE READ"} {
		at := waitForLesson(t, h, &f, title, paceLimit)
		if last >= 0 && at-last < ideaGapOf(f.State.Age) {
			t.Fatalf("%q came %.2f s after the last card, inside the %.0f s between ideas", title, at-last, ideaGapOf(f.State.Age))
		}
		runFor(h, &f, 2)
		clickOn(t, h, h.Stage(f), LessonAck)
		last = at
	}
	noAttentionFaults(t, h)
}

func TestTheDarkAgeBurstLetsTheDecreeSinkInBeforeTheGoldLesson(t *testing.T) {
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
	f.State.Quests = []view.QuestView{{ID: "borg-founding-dev", Title: "Operator brief", Giver: "Operator", Parts: []view.QuestPartView{{Title: "The Codex", Objective: "build-codex"}}}}
	f.State.Reveals = append(slices.Clone(f.State.Reveals), view.BorgIntro)
	f.Guide.Staged = true
	h.LessonCard = LessonCard{Title: "CPU, RATE AND OVERHEAD", Text: "Your CPU.", Shown: true, Teach: true}
	decree := waitUntil(t, h, &f, paceLimit, "the brief", func(s Frame) bool { return h.QuestCardShown(s) })
	runFor(h, &f, 3)
	f.Guide.Intro = nil
	f.Guide.Introduced = IntroTargets([]string{view.BorgIntro})
	stepAt := -1.0
	for h.Now < decree+IdeaGapDark-.25 {
		tick(h, &f, .25)
		s := h.Stage(f)
		if twoIdeaCards(h, s) {
			t.Fatalf("two cards stacked at %.2f", h.Now)
		}
		if o, ok := s.ShownObjective(); ok && o.ID == "build-codex" && !o.Done && stepAt < 0 {
			stepAt = h.Now
		}
		if h.LessonShown(s) {
			t.Fatalf("the CPU hint came %.2f s after the brief, inside the %.0f s between ideas (an early burst: brief, CPU hint, speeds and the Chancery's card in 26 s)", h.Now-decree, IdeaGapDark)
		}
	}
	if stepAt < 0 {
		t.Fatal("the step is no idea and should follow the brief before the next idea")
	}
	waitUntil(t, h, &f, paceLimit, "the CPU hint", func(s Frame) bool { return h.LessonShown(s) })
	noAttentionFaults(t, h)
}

func troubleAfterACard(t *testing.T, kind string) (float64, float64) {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	card := waitForLesson(t, h, &f, "A TRAFFIC SPIKE ARRIVED", paceLimit)
	runFor(h, &f, 1)
	clickOn(t, h, h.Stage(f), LessonAck)
	answered := h.Now
	f.State.Calamities = []view.CalamityView{{ID: kind + ":parcel", Kind: kind, First: true, Cause: "The parcels wait."}}
	for waited := 0.0; waited < paceLimit; waited += .25 {
		tick(h, &f, .25)
		s := h.Stage(f)
		if twoIdeaCards(h, s) {
			t.Fatalf("two cards stacked at %.2f", h.Now)
		}
		if v, ok := h.ShownCalamity(s); ok && v.Kind == kind {
			return answered - card, h.Now - card
		}
	}
	t.Fatalf("the %s card never came: %s", kind, h.StageDebug())
	return 0, 0
}

func TestUrgentTroubleComesAfterTheCalmWhileTroubleThatLosesNothingWaitsTheIdeaGap(t *testing.T) {
	for _, kind := range []string{view.CalamitySpoiling, view.CalamityNoCrew, view.CalamityNines, view.TagCrashLoop} {
		answered, at := troubleAfterACard(t, kind)
		if at-answered > CalmGap+CalamitySettle+1 {
			t.Errorf("urgent %s waited %.2f s after the card was answered, past the calm", kind, at-answered)
		}
	}
	for _, kind := range []string{view.TagUnhoused, view.TagThrottled, view.CalamityFrozen} {
		_, at := troubleAfterACard(t, kind)
		if at < IdeaGapDark {
			t.Errorf("%s loses no boxes, yet came %.2f s after the last card, inside the %.0f s between ideas", kind, at, IdeaGapDark)
		}
	}
}

func TestACardSetAsideForUrgentTroubleComesBackAtOnceAndTheNextNewIdeaSoaksAfterIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := calamityFrame(t, noDepot)
	f.State.Calamities = nil
	f.Guide = Guide{Intro: &Intro{ID: "rail", Title: "Matters at hand", Try: "Glance at the rail.", Target: SurfaceRail, Via: ViaAck}}
	f = settled(h, f)
	f.State.Calamities = []view.CalamityView{noDepot}
	waitUntil(t, h, &f, TroubleWait+CalamitySettle+2, "the taught spoiling trouble setting the card aside", func(s Frame) bool {
		_, up := h.ShownCalamity(s)
		return up
	})
	h.NoteAct("openDepot", f.State.RealmTime)
	f.State.Calamities = nil
	back := waitUntil(t, h, &f, CalmGap+CalamitySettle+2, "the card set aside, back without an idea gap", func(s Frame) bool { return h.IntroShown(s) })
	f.Guide.Intro = nil
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	at := waitForLesson(t, h, &f, "A TRAFFIC SPIKE ARRIVED", paceLimit)
	if at-back < CalmGap {
		t.Fatalf("the next new idea came %.2f s after the card came back", at-back)
	}
	noAttentionFaults(t, h)
}

func TestAFinishedStepNeverShowsAgainAsOpenOnceTheRealmDropsIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "pledges", Text: "Pledge food for your porters", Progress: "ready: press it", Target: "cmd:door", Lesson: "A pledge reserves food."}}
	f.Guide.Target = "cmd:door"
	runFor(h, &f, 1)
	if o, ok := h.Stage(f).ShownObjective(); !ok || o.ID != "pledges" {
		t.Fatalf("the pledge step never reached the stage: %s", h.StageDebug())
	}
	f.State.Objectives = []view.ObjectiveView{{ID: "pledges", Text: "Pledge food for your porters", Done: true},
		{ID: "see-ward", Text: "See the Royal Ward", Progress: "ready: press it", Target: "cmd:door"}}
	runFor(h, &f, .5)
	f.State.Objectives = []view.ObjectiveView{{ID: "see-ward", Text: "See the Royal Ward", Progress: "ready: press it", Target: "cmd:door"}}
	for waited := 0.0; waited < paceLimit; waited += .25 {
		tick(h, &f, .25)
		o, ok := h.Stage(f).ShownObjective()
		if ok && o.ID == "pledges" && !o.Done && waited > 1 {
			t.Fatalf("at %.2f the finished pledge step shows as open again (step:pledges after done:pledges)", h.Now)
		}
		if ok && o.ID == "see-ward" {
			break
		}
	}
	if o, _ := h.Stage(f).ShownObjective(); o.ID != "see-ward" {
		t.Fatalf("the next step never came: %+v", o)
	}
	if o := h.Director.Stage.View; o.ID == "pledges" && o.Lesson == "" {
		t.Fatal("the finished step lost its hint")
	}
}

func TestALessonThePlayerOpensSoaksInLikeAnyOther(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.Lesson("THE CODEX", "The Codex keeps every order the realm was given.")
	opened := h.Now
	runFor(h, &f, 2)
	clickOn(t, h, h.Stage(f), LessonAck)
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	at := waitForLesson(t, h, &f, "A TRAFFIC SPIKE ARRIVED", paceLimit)
	if at-opened < ideaGapOf(f.State.Age) {
		t.Fatalf("the next idea came %.2f s after the hint the player opened, inside the %.0f s between ideas", at-opened, ideaGapOf(f.State.Age))
	}
}

func TestACardNeverDrawnIsStillANewIdeaWhenItComesBack(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	f.Placing = "keep"
	f.Guide.Intro = &Intro{ID: "keep", Title: "A Keep", Try: "Press BUILD, then the Keep.", Target: "build:keep", Via: ViaAct}
	waitUntil(t, h, &f, paceLimit, "the Keep's card on the stage, hidden while its Keep is placed", func(s Frame) bool { return h.Director.Stage.Has(actIntro + "keep") })
	if h.IntroShown(h.Stage(f)) {
		t.Fatalf("the fixture wants the Keep's own card on the stage but hidden while its Keep is placed: %s", h.StageDebug())
	}
	f.Placing, f.Guide.Intro = "", nil
	runFor(h, &f, 1)
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	gilded := waitForLesson(t, h, &f, "A TRAFFIC SPIKE ARRIVED", paceLimit)
	runFor(h, &f, 1)
	clickOn(t, h, h.Stage(f), LessonAck)
	f.Guide.Intro = &Intro{ID: "keep", Title: "A Keep", Try: "Press BUILD, then the Keep.", Target: "build:keep", Via: ViaAct}
	for h.Now < gilded+ideaGapOf(f.State.Age)-.25 {
		tick(h, &f, .25)
		if h.IntroShown(h.Stage(f)) {
			t.Fatalf("the Keep's card, never drawn before, came %.2f s after the last card (the Chancery's card inside the CPU hint's soak)", h.Now-gilded)
		}
	}
}

func TestTheGuideWalkLeavesTheStageAsItFoundIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	gilded := waitForLesson(t, h, &f, "A TRAFFIC SPIKE ARRIVED", paceLimit)
	runFor(h, &f, 1)
	clickOn(t, h, h.Stage(f), LessonAck)
	if w := h.WalkGuide(h.Stage(f)); len(w.Steps) < 2 {
		t.Fatalf("the walk pressed nothing, so it proves nothing: %+v", w)
	}
	h.Result("WHILE YOU WERE AWAY (12:57)", "Your porters carried 900 boxes.")
	at := waitForLesson(t, h, &f, "WHILE YOU WERE AWAY (12:57)", paceLimit)
	if at-gilded < ideaGapOf(f.State.Age) {
		t.Fatalf("after the guide's walk the next card came %.2f s after the last: the walk's presses changed the stage it restored", at-gilded)
	}
}

func TestACardTheRealmHandedBeforeTheStepWasReadyIsNotHeldForTheStepAfterItsSoak(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "build-codex", Text: "Check the control plane", Progress: "CPU 6/10: carry the boxes on the quay", Target: view.ParcelPile, Build: "codex"}}
	f.Guide.Target = view.ParcelPile
	runFor(h, &f, 1)
	h.Result("A TRAFFIC SPIKE ARRIVED", "Answer its requests.")
	gilded := waitForLesson(t, h, &f, "A TRAFFIC SPIKE ARRIVED", paceLimit)
	runFor(h, &f, 1)
	clickOn(t, h, h.Stage(f), LessonAck)
	f.Guide.Intro = &Intro{ID: "minimap", Title: "The minimap", Purpose: "The whole realm at a glance.", Try: "Click it.", Target: SurfaceMinimap, Via: ViaAck}
	runFor(h, &f, 2)
	f.State.Objectives[0].Progress, f.State.Objectives[0].Target = "ready: press BUILD", ""
	f.Guide.Target = "build:codex"
	for h.Now < gilded+paceLimit {
		tick(h, &f, .25)
		if h.IntroShown(h.Stage(f)) {
			if h.Now-gilded < ideaGapOf(f.State.Age) {
				t.Fatalf("the minimap card came %.2f s after the last card, inside the gap between ideas", h.Now-gilded)
			}
			return
		}
	}
	t.Fatalf("the realm handed the minimap card while the player still gathered CPU; held for the step that became ready during its soak, it holds the step's own card back in the realm (quick-codex-card: the Codex card never came): %s", h.StageDebug())
}
