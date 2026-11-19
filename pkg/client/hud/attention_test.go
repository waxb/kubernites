package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func attnRules(faults []AttnFault) []string {
	out := []string{}
	for _, f := range faults {
		if !slices.Contains(out, f.Rule) {
			out = append(out, f.Rule)
		}
	}
	return out
}

func wantRule(t *testing.T, faults []AttnFault, rule string) {
	t.Helper()
	if !slices.Contains(attnRules(faults), rule) {
		t.Fatalf("want the fault %q, got %v", rule, faults)
	}
}

func wantClean(t *testing.T, faults []AttnFault) {
	t.Helper()
	if len(faults) > 0 {
		t.Fatalf("want no fault, got %v", faults)
	}
}

func attnCard(key string, from, to float64) AttnBox {
	return AttnBox{Kind: BoxCard, Key: key, Text: "Carry boxes and keep up", From: from, To: to, Realm: from, RealmTo: to, Waits: true}
}

func attnToast(key string, from float64) AttnBox {
	return AttnBox{Kind: BoxToast, Key: key, Text: "The Codex stands", From: from, To: from + ReadTime("The Codex stands"), Realm: from, RealmTo: from + ReadTime("The Codex stands")}
}

func TestAReadingTimeIsOneSecondAndFourTenthsAWordAndNeverLessThanFour(t *testing.T) {
	cases := map[string]float64{"": BoxReadMin, "The Codex stands": BoxReadMin, strings.Repeat("word ", 20): 1 + 0.4*20}
	for text, want := range cases {
		if got := ReadTime(text); got != want {
			t.Errorf("ReadTime(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestTheCheckerPassesBoxesThatTakeTheirTurns(t *testing.T) {
	log := AttnLog{Boxes: []AttnBox{attnToast("a", 0), attnCard("intro:serve", 7, 20), attnToast("b", 23)}}
	wantClean(t, CheckAttention(log, AttnRules{}))
}

func TestTheCheckerFailsTwoBoxesAtOnce(t *testing.T) {
	log := AttnLog{Boxes: []AttnBox{attnCard("intro:serve", 0, 20), attnToast("a", 10)}}
	wantRule(t, CheckAttention(log, AttnRules{}), RuleTwoAtOnce)
}

func TestTheCheckerFailsABoxBeforeTheLastHadItsReadingTime(t *testing.T) {
	first := attnToast("a", 0)
	first.To = 2
	log := AttnLog{Boxes: []AttnBox{first, attnToast("b", 3)}}
	wantRule(t, CheckAttention(log, AttnRules{}), RuleUnread)
}

func TestTheCheckerFailsTwoBoxesLessThanThreeSecondsApart(t *testing.T) {
	log := AttnLog{Boxes: []AttnBox{attnToast("a", 0), attnToast("b", BoxReadMin+1)}}
	wantRule(t, CheckAttention(log, AttnRules{}), RuleFloor)
}

func TestTheCheckerFailsAShownBoxWhoseTextChanges(t *testing.T) {
	b := attnToast("a", 0)
	b.Texts = []AttnText{{At: 1, Text: "The Chancery stands"}}
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}), RuleTextChanged)
}

func TestTheCheckerLetsACounterTickOnceASecondWithoutResizing(t *testing.T) {
	b := AttnBox{Kind: BoxStep, Key: "step:raise-keep", Text: "Cordon Keep node-a CPU 3/10", From: 0, To: 30, Rect: ui.Rect{W: 100, H: 40}, Face: true}
	b.Texts = []AttnText{{At: 5, Text: "Cordon Keep node-a CPU 4/10", Rect: b.Rect}, {At: 6, Text: "Cordon Keep node-a CPU 5/10", Rect: b.Rect}}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}))
	b.Texts = append(b.Texts, AttnText{At: 6.5, Text: "Cordon Keep node-a CPU 6/10", Rect: b.Rect})
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}), RuleCounterFast)
	b.Texts = []AttnText{{At: 5, Text: "Cordon Keep node-a CPU 4/10", Rect: ui.Rect{W: 100, H: 52}}}
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}), RuleResized)
}

func TestTheCheckerFailsACardShownTwice(t *testing.T) {
	log := AttnLog{Boxes: []AttnBox{attnCard("intro:minimap", 0, 10), attnCard("intro:minimap", 100, 110)}}
	wantRule(t, CheckAttention(log, AttnRules{}), RuleShownTwice)
}

func TestTheCheckerLetsACardComeBackFromItsTray(t *testing.T) {
	back := attnCard("intro:minimap", 40, 60)
	back.Tray = true
	trouble := AttnBox{Kind: BoxTrouble, Key: "trouble:spoiling", Text: "Boxes spoil", From: 14, To: 30, Urgent: true, Taught: true, Waits: true}
	log := AttnLog{Boxes: []AttnBox{attnCard("intro:minimap", 0, 10), trouble, back}}
	wantClean(t, CheckAttention(log, AttnRules{}))
}

func TestTheCheckerFailsIdeasCloserThanTheirAgeAllows(t *testing.T) {
	a, b := attnCard("intro:serve", 0, 10), attnCard("intro:gold", 60, 70)
	a.Idea, b.Idea = true, true
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{a, b}}, AttnRules{}), RuleIdeaGap)
	a.Age, b.Age = "feudal", "feudal"
	b.From, b.To = 61, 70
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{a, b}}, AttnRules{}))
}

func TestTheCheckerFailsMoreThanFourBoxesAMinuteEarlyOn(t *testing.T) {
	boxes := []AttnBox{}
	for i := range 5 {
		boxes = append(boxes, attnToast(string(rune('a'+i)), float64(i)*11))
	}
	wantRule(t, CheckAttention(AttnLog{Boxes: boxes}, AttnRules{}), RuleBusyMinute)
	late := []AttnBox{}
	for i := range 5 {
		late = append(late, attnToast(string(rune('a'+i)), FirstMinutes+float64(i)*11))
	}
	wantClean(t, CheckAttention(AttnLog{Boxes: late}, AttnRules{}))
}

func TestTheCheckerFailsAStepBeforeItsEnjoymentUnlessNextStepWasPressed(t *testing.T) {
	done := AttnBox{Kind: BoxDone, Key: "done:serve-hand", Text: "Done: Carry 5 parcels", From: 0, To: 6, Realm: 0, RealmTo: 6, Face: true}
	step := AttnBox{Kind: BoxStep, Key: "step:raise-keep", Text: "Raise a Keep", From: 10, To: 16, Realm: 10, RealmTo: 16, Face: true}
	log := AttnLog{Boxes: []AttnBox{done, step}}
	wantRule(t, CheckAttention(log, AttnRules{}), RuleEnjoy)
	log.Events = []AttnEvent{{At: 9, Realm: 9, Kind: EventNextStep}}
	wantClean(t, CheckAttention(log, AttnRules{}))
}

func TestTheCheckerFailsAnUntaughtTroubleWhileALessonIsOpen(t *testing.T) {
	lesson := attnCard("intro:codex", 0, 30)
	trouble := AttnBox{Kind: BoxTrouble, Key: "trouble:unhoused", Text: "A hut waits", From: 10, To: 40, Waits: true}
	faults := CheckAttention(AttnLog{Boxes: []AttnBox{lesson, trouble}}, AttnRules{})
	wantRule(t, faults, RuleUntaught)
}

func TestTheCheckerLetsATaughtUrgentTroubleSetALessonAside(t *testing.T) {
	lesson := attnCard("intro:codex", 0, 10)
	lesson.Aside = true
	trouble := AttnBox{Kind: BoxTrouble, Key: "trouble:spoiling", Text: "Boxes spoil", From: 10, To: 30, Waits: true, Urgent: true, Taught: true}
	back := attnCard("intro:codex", 33, 50)
	back.Tray = true
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{lesson, trouble, back}}, AttnRules{}))
}

func TestTheCheckerFailsTwoSurfacesThatOverlap(t *testing.T) {
	frame := AttnFrame{At: 3, Surfaces: []Surface{{Slot: SlotStep, Key: "step", Rect: ui.Rect{X: 0, Y: 20, W: 100, H: 50}}, {Slot: SlotBox, Key: "toast:a", Rect: ui.Rect{X: 90, Y: 60, W: 80, H: 20}, Box: true}}}
	wantRule(t, CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{}), RuleOverlap)
	frame.Surfaces[1].Rect.X = 100
	wantClean(t, CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{}))
}

func TestADimmingModalMayCoverWhatIsBehindIt(t *testing.T) {
	frame := AttnFrame{At: 3, Surfaces: []Surface{{Slot: SlotStep, Key: "step", Rect: ui.Rect{X: 0, Y: 20, W: 100, H: 50}}, {Slot: SlotPanel, Key: "panel:welcome", Rect: ui.Rect{X: 0, Y: 0, W: Width, H: Height}, Modal: true}}}
	wantClean(t, CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{}))
}

func TestAPointerMayTouchOnlyItsOwnBoxAndTarget(t *testing.T) {
	card := Surface{Slot: SlotBox, Key: "intro:build", Rect: ui.Rect{X: 600, Y: 300, W: 200, H: 60}, Box: true}
	target := Surface{Slot: SlotConsole, Key: "console", Rect: ui.Rect{X: 0, Y: ConsoleY, W: Width, H: ConsoleH}}
	pointer := Surface{Slot: SlotPointer, Key: "pointer:intro:build", Owner: "intro:build", Target: "console", Line: [4]int{700, 360, 800, 420}, Rect: ui.Rect{X: 700, Y: 360, W: 101, H: 61}}
	frame := AttnFrame{Surfaces: []Surface{card, target, pointer}}
	wantClean(t, CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{}))
	step := Surface{Slot: SlotStep, Key: "step", Rect: ui.Rect{X: 740, Y: 380, W: 40, H: 10}}
	frame.Surfaces = append(frame.Surfaces, step)
	wantRule(t, CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{}), RuleOverlap)
}

func TestAStepDoneEndsItsBoxAndItsDoneBoxFollowsAfterTheFloor(t *testing.T) {
	step := AttnBox{Kind: BoxStep, Key: "step:serve-hand", Text: strings.Repeat("word ", 30), From: 0, To: 10, Face: true, Answered: 7}
	done := AttnBox{Kind: BoxDone, Key: "done:serve-hand", Text: "Done: Carry 5 parcels", From: 10, To: 40, Realm: 10, Face: true}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{step, done}}, AttnRules{}))
	done.From = 8
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{step, done}}, AttnRules{}), RuleFloor)
	step.Answered = 0
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{step, done}}, AttnRules{}), RuleTwoAtOnce)
}

func TestACardThePlayerClosesEarlyIsReadWhenClosedButOneTakenAwayIsNot(t *testing.T) {
	long := AttnBox{Kind: BoxLesson, Key: "lesson:topbar", Text: strings.Repeat("word ", 40), From: 10, To: 13, Realm: 10, RealmTo: 13, Waits: true}
	next := attnToast("toast:the codex stands", 13+BoxFloor)
	closed := long
	closed.ByPlayer = true
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{closed, next}}, AttnRules{}))
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{long, next}}, AttnRules{}), RuleUnread)
}

func TestTheCheckerLetsAStatusLineChangeOnceASecondButNotTheTextItIsReadWith(t *testing.T) {
	b := AttnBox{Kind: BoxStep, Key: "step:serve-hand", Text: "Carry 5 parcels from the quay by hand", Status: "a ship is sailing in", From: 0, To: 30, Rect: ui.Rect{W: 100, H: 40}, Face: true}
	b.Texts = []AttnText{{At: 2, Text: b.Text, Status: "0/5 carried", Rect: b.Rect}, {At: 3, Text: b.Text, Status: "1/5 carried", Rect: b.Rect}}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}))
	fast := b
	fast.Texts = append(slices.Clone(b.Texts), AttnText{At: 3.5, Text: b.Text, Status: "2/5 carried", Rect: b.Rect})
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{fast}}, AttnRules{}), RuleCounterFast)
	grown := b
	grown.Texts = []AttnText{{At: 2, Text: b.Text, Status: "0/5 carried", Rect: ui.Rect{W: 100, H: 30}}}
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{grown}}, AttnRules{}), RuleResized)
	read := b
	read.Texts = []AttnText{{At: 2, Text: b.Text + " Click a waiting parcel", Status: "0/5 carried", Rect: b.Rect}}
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{read}}, AttnRules{}), RuleTextChanged)
}

func TestABoxOnScreenWhenTheLogStartsStillKeepsItsText(t *testing.T) {
	b := AttnBox{Kind: BoxStep, Key: "step:serve-hand", Text: "Carry 5 parcels from the quay by hand", From: 0, To: 30, Rect: ui.Rect{W: 100, H: 40}, Face: true, Entry: true}
	b.Texts = []AttnText{{At: 2, Text: b.Text + " Click a waiting parcel", Rect: ui.Rect{W: 100, H: 60}}}
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}), RuleTextChanged)
}

func TestTheCheckerLetsThePlayerChangeABoxTheyAreReading(t *testing.T) {
	b := AttnBox{Kind: BoxStep, Key: "step:build-codex", Text: "Check the control plane", From: 0, To: 30, Rect: ui.Rect{W: 100, H: 40}, Face: true}
	b.Texts = []AttnText{{At: 2, Text: "Check the control plane Checklist 0/4", Rect: ui.Rect{W: 100, H: 80}, Pressed: true}}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}))
}

func TestATroubleThatComesBackIsNotACardShownTwice(t *testing.T) {
	first := AttnBox{Kind: BoxTrouble, Key: "trouble:spoiling:parcel", Text: "Parcels spoil", From: 0, To: 20, Waits: true}
	again := first
	again.From, again.To = 200, 220
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{first, again}}, AttnRules{}))
}

func TestATroubleCardsDiagnosisMayChangeOnceItIsRead(t *testing.T) {
	b := AttnBox{Kind: BoxTrouble, Key: "trouble:spoiling:parcel", Text: "Parcels spoil faster than the porters carry them", From: 0, To: 60, Rect: ui.Rect{W: 100, H: 40}, Waits: true}
	b.Texts = []AttnText{{At: 30, Text: "Parcels spoil faster than the porters carry them: the porters go hungry", Rect: b.Rect}}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}))
}

func TestATroubleThatEndsBeforeItsReadingTimeHasEndedItsTurn(t *testing.T) {
	gone := AttnBox{Kind: BoxTrouble, Key: "trouble:storm", Text: strings.Repeat("word ", 30), From: 0, To: 5, Waits: true}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{gone, attnToast("toast:the storm passes", 5+BoxFloor)}}, AttnRules{}))
}

func TestAPanelThePlayerClosesAtTheFirstInstantIsReadWhenClosed(t *testing.T) {
	welcome := AttnBox{Kind: BoxModal, Key: "panel:welcome", From: 0, To: 0, Waits: true, ByPlayer: true}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{welcome, attnToast("step:serve-hand", BoxFloor)}}, AttnRules{}))
}

func TestTheSameNewsTwiceIsNotACardShownTwice(t *testing.T) {
	first := AttnBox{Kind: BoxNews, Key: "lesson:A NOTICE (5:00)", Text: "Your crews delivered 40 boxes", From: 0, To: 10, Waits: true, ByPlayer: true}
	again := first
	again.From, again.To = 400, 410
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{first, again}}, AttnRules{}))
}

func TestAResultIsRegisteredAsNewsAndATeachingCardAsALesson(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	h.SetStaging(false)
	h.Result("A NOTICE (5:00)", "Your crews delivered 40 boxes.")
	h.Update(Input{MX: -10, MY: -10}, f)
	kinds := map[string]string{}
	for _, s := range h.Surfaces(f) {
		kinds[s.Key] = s.Kind
	}
	if kinds["lesson:A NOTICE (5:00)"] != BoxNews {
		t.Fatalf("the away summary is registered as %q", kinds["lesson:A NOTICE (5:00)"])
	}
}

func TestTheCheckerLetsThePlayersOwnClicksChangeAStatusQuickly(t *testing.T) {
	b := AttnBox{Kind: BoxStep, Key: "step:raise-keep", Text: "Raise a Keep", Status: "ready: press BUILD", From: 0, To: 30, Rect: ui.Rect{W: 100, H: 40}, Face: true}
	b.Texts = []AttnText{{At: 20, Text: b.Text, Status: "ready: press X on this panel, then BUILD", Rect: b.Rect, Pressed: true}, {At: 20.4, Text: b.Text, Status: "ready: press BUILD", Rect: b.Rect, Pressed: true}}
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}))
}

func TestTheCheckerLetsAStepPanelSayItIsDoneOnceItsStepIsDoneButNotBefore(t *testing.T) {
	b := AttnBox{Kind: BoxStep, Key: "teacher:ch1:3:1", Text: "Web requests come to node-a's port 8080. Watch the static Pod serve five.", From: 0, To: 40, Rect: ui.Rect{W: 100, H: 40}, Face: true, Answered: 6}
	done := []AttnText{{At: 6, Text: "Good. Five requests served by one container.", Rect: ui.Rect{W: 100, H: 30}}}
	b.Texts = done
	wantClean(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}))
	b.Texts = []AttnText{{At: 5, Text: done[0].Text, Rect: done[0].Rect}}
	wantRule(t, CheckAttention(AttnLog{Boxes: []AttnBox{b}}, AttnRules{}), RuleTextChanged)
}

func TestABannerThatEndsUnderAWindowLeavesTheLogWhenItEnds(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	h.FeatBanner("CHAPTER 1 COMPLETE", "")
	for waited := 0.0; !h.BannerActive() && waited < 60; waited += .25 {
		tick(h, &f, .25)
	}
	if !h.BannerActive() {
		t.Fatal("the banner never came")
	}
	tick(h, &f, 1)
	h.OpenDebrief()
	for range 40 {
		tick(h, &f, .25)
	}
	h.OverlayID = OverlayNone
	tick(h, &f, 1)
	for _, b := range h.Attention().Boxes {
		if b.Key == "banner:CHAPTER 1 COMPLETE" && b.To > h.Juice.BannerAt+bannerIn+bannerHold+bannerOut+.5 {
			t.Fatalf("the banner left the log at %.1f s, %.1f s after it ended", b.To, b.To-(h.Juice.BannerAt+bannerIn+bannerHold+bannerOut))
		}
	}
}

func TestThePointerMayTouchItsTargetAndWhatHoldsItButNothingElse(t *testing.T) {
	console := Surface{Slot: SlotConsole, Key: "console", Rect: ui.Rect{X: 0, Y: 408, W: 962, H: 135}}
	lens := Surface{Slot: SlotConsole, Key: "lens", Owner: "console", Rect: ui.Rect{X: 10, Y: 420, W: 62, H: 17}}
	ring := Surface{Slot: SlotPointer, Kind: "ring", Key: "ring:intro:heraldry", Owner: "intro:heraldry", Rect: ui.Rect{X: 6, Y: 416, W: 68, H: 22}, Ring: true, Target: "lens"}
	frame := AttnFrame{Surfaces: []Surface{console, lens, ring}}
	if faults := CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{OverlapOnly: true}); len(faults) > 0 {
		t.Fatalf("a ring around a console button clashes with the console that holds it: %v", faults)
	}
	tab := Surface{Slot: SlotConsole, Key: "tab", Owner: "console", Rect: ui.Rect{X: 70, Y: 410, W: 40, H: 12}}
	frame.Surfaces = append(frame.Surfaces, tab)
	if faults := CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{OverlapOnly: true}); len(faults) == 0 {
		t.Fatal("a ring crossing a neighbouring button is not a fault")
	}
}
