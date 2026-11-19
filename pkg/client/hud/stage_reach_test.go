package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func reachFrame(t *testing.T) Frame {
	t.Helper()
	f := directorFrame(t)
	f.Guide.Staged = true
	f.Guide.Introduced = nil
	f.State.Reveals = []string{"serve", "build"}
	f.State.Gold, f.State.Wisdom = 0, 0
	return f
}

func waitingCard(h *HUD, f *Frame) {
	h.LessonCard = LessonCard{Title: "BURNING", Text: "Up and ready for boxes.", Shown: true, Teach: true}
	*f = h.SettleStage(Input{MX: -10, MY: -10}, *f, settleLimit)
	if !h.LessonShown(h.Stage(*f)) {
		panic("the waiting card never came")
	}
}

func statMeters(h *HUD, f Frame) []string {
	out := []string{}
	for _, st := range h.TopStats(f) {
		out = append(out, st.Meter)
	}
	return out
}

func nextBeat(h *HUD, f *Frame) {
	f.State.Objectives = append(slices.Clone(f.State.Objectives), view.ObjectiveView{ID: "later", Text: "A later step"})
	f.State.Objectives[0].Done = true
	runFor(h, f, BoxFloor+1)
}

func TestGoldWisdomAndTheSpeedsComeWithTheNextBeatNotBehindEveryCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := reachFrame(t)
	runFor(h, &f, 1)
	waitingCard(h, &f)
	f.State.Gold, f.State.Wisdom = 2.4, 2
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "gold", "wisdom", "speed")
	nextBeat(h, &f)
	s := h.Stage(f)
	if m := statMeters(h, s); !slices.Contains(m, MeterGold) || !slices.Contains(m, MeterWisdom) {
		t.Fatalf("the CPU and points meters stay hidden while a card waits, though the realm holds both: %v", m)
	}
	if ids := buttonIDs(h.TopButtons(s)); !ids["speed:5"] {
		t.Fatalf("the speeds stay hidden behind the Hasten card: %v", ids)
	}

}

func TestAControlThePlayerAlreadyMetComesBackWithTheNextBeat(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := reachFrame(t)
	runFor(h, &f, 1)
	waitingCard(h, &f)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "grimoire")
	f.Guide.Introduced = IntroTargets([]string{"grimoire"})
	nextBeat(h, &f)
	if _, ok := h.MainCells(h.Stage(f))["cmd:grimoire"]; ok {
		t.Fatal("the manual's tile rode in on the step's own frame, beside a card that waits")
	}
	h.Press("lesson:ack", h.Stage(f))
	waitUntil(t, h, &f, paceLimit, "the manual's tile once the card is answered", func(s Frame) bool {
		_, ok := h.MainCells(s)["cmd:grimoire"]
		return ok
	})
}

func TestTextThatNamesTheGrimoireOrHiringBringsItsTile(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := reachFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "grimoire", "summon")
	runFor(h, &f, 1)
	waitingCard(h, &f)
	h.Stone.Output = append(slices.Clone(h.Stone.Output), StoneLine{Text: "error: speakstctl create crew needs Crews from the Grimoire: learn it first (60 gold, 10 wisdom)"})
	f.State.Quests = []view.QuestView{{ID: "borg", Title: "Operator brief", Fulfilled: true, Parts: []view.QuestPartView{
		{Title: "The Stewards", Objective: "build-stewards", Done: true, Gained: "What you ask for is now kept true by itself. Now you can hire porters."}}}}
	nextBeat(h, &f)
	cells := h.MainCells(h.Stage(f))
	if _, ok := cells["cmd:grimoire"]; !ok {
		t.Fatal("a Stone error names the manual while no MANUAL tile is reachable")
	}
	if _, ok := cells[CmdSummonID]; !ok {
		t.Fatal("the brief says 'Now you can hire porters.' while HIRE is not on the grid")
	}
}

func TestAMeterItsCardShowedStaysWhenAnAchievementSetsTheCardAside(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := reachFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "slo")
	f.Guide.Intro = &Intro{ID: "nines", Title: "The Nines", Try: "Watch the Nines.", Target: MeterNines, Via: ViaAck}
	runFor(h, &f, 2*BeatGap+StepRead+1)
	if !slices.Contains(statMeters(h, h.Stage(f)), MeterNines) {
		t.Fatal("the Nines card came without its meter")
	}
	nextBeat(h, &f)
	if !slices.Contains(statMeters(h, h.Stage(f)), MeterNines) {
		t.Fatal("NINES left the top bar when a step done set its card aside")
	}
}
