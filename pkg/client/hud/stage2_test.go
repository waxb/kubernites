package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheObjectiveCardStaysWhileAnIntroSpeaks(t *testing.T) {
	h := New(nil, Backend{})
	f := introFrame(t, Intro{ID: "build", Title: "Build", Purpose: "Raise buildings.", Try: "Press BUILD.", Target: "cmd:build"})
	f.State.Objectives = []view.ObjectiveView{{ID: "o1", Text: "Serve 5 requests", Hint: "Press serve", Target: "pile:parcel"}}
	f.GoalAt = &[2]float64{3, 4}
	if !h.ObjectiveShown(f) {
		t.Fatal("an intro card hid the task card")
	}
	if card, obj := h.IntroCard(f), h.ObjectiveRectOf(f); Overlaps(card, obj) {
		t.Fatalf("the intro card %+v covers the task card %+v", card, obj)
	}
	if _, ok := HitID(h.Buttons(f), "obj:goal"); !ok {
		t.Fatal("SHOW ME is not listed while an intro is up")
	}
}

func TestAStepsOwnControlShowsWithoutItsLesson(t *testing.T) {
	f := stagedFrame(t, MeterGold)
	if f.Shown(SurfaceBerths) {
		t.Fatal("berths show with no step calling them")
	}
	f.Guide.Target, f.Guide.Button = BerthButton, BerthButton
	if !f.Shown(SurfaceBerths) || !f.GoalCalls(SurfaceBerths) {
		t.Fatal("the berth step does not show its own control")
	}
	f.Guide.Target, f.Guide.Button = PortButtonID(PortVerbOpen, "haven-1"), ""
	if !f.Shown(SurfaceBerths) {
		t.Fatal("the open-port step does not show the port controls")
	}
}

func TestAGreyedTileNamesItsReasonShortEnoughForTheTile(t *testing.T) {
	for reason, want := range map[string]string{
		"Needs 100 gold": "NEEDS GOLD", "Stone Keep comes with the Feudal Age": "FEUDAL AGE",
		"Carry the shipment on the quay first: its boxes pay for the Codex":                   "CARRY FIRST",
		"Endow the Clerks' class (clerk:1.0) at the Academy first: an Archive with no Clerks": "ENDOW FIRST",
		"the caravel masters are on their way":                                                "SHIPS SOON", "Upgrading Gate Harbor costs 3000 gold (you have 20)": "NEEDS GOLD",
		"a berth opens only while your crews hold the Nine: 97.5% delivered fresh, 99% needed": "TWO NINES",
		"a berth opens only while your crews hold the Nine: 85% delivered fresh, 90% needed":   "ONE NINE",
		"Learn the Charters of the Isles in the Grimoire":                                      "CHARTERS",
	} {
		got := WhyOf(reason)
		if got != want || len(got) > 11 {
			t.Errorf("%q reads %q, want %q within one tile row", reason, got, want)
		}
	}
	if whyUnless(true, "Needs 100 gold") != "" {
		t.Error("an available tile carries a reason")
	}
}

func TestThePledgeStepsInLineAndUseSetsWhatWorkersReallyTake(t *testing.T) {
	steps := []int64{}
	for v := int64(0); v < 200; v = StepValue(CpuStep, CpuMost, v, true) {
		steps = append(steps, v)
	}
	if !slices.Equal(steps, []int64{0, 50, 100, 150}) {
		t.Fatalf("the food pledge steps %v", steps)
	}
	if StepValue(CpuStep, CpuMost, 120, false) != 100 || StepValue(memStep, memMost, 0, false) != 0 {
		t.Fatal("stepping down is not linear")
	}
	cpu, _ := FieldNamed("cpu-req")
	mem, _ := FieldNamed("mem-req")
	use := view.UsePeakView{CPU: 400, Memory: 326 * mebi, Window: 720, Noon: true}
	if got := cpu.measured(use); got != 400 {
		t.Errorf("food use reads %d, want the day peak of 400", got)
	}
	if got := mem.measured(use); got != 352 {
		t.Errorf("housing use rounds to %d, want 352", got)
	}
	if limit, _ := FieldNamed("cpu-lim"); limit.measured(use) != 0 || cpu.measured(view.UsePeakView{}) != 0 {
		t.Error("a limit the workers never reach or an unmeasured crew offers a use shortcut")
	}
}

func TestAnOpenTroubleCardReadsTheLiveRealmAndOffersAFix(t *testing.T) {
	live := view.CalamityView{ID: "spoil", Kind: view.TagOOMKilled, Cause: "first cause", Remedies: []view.RemedyView{{Label: "Scale the clerks to 22", Command: "kubectl scale"}}}
	var st CalamityState
	st = st.Step([]view.CalamityView{live}, 10, CalamityGate{})
	st = st.Step([]view.CalamityView{live}, 10+CalamitySettle+1, CalamityGate{})
	changed := live
	changed.Cause, changed.Remedies = "second cause", nil
	st = st.Step([]view.CalamityView{changed}, 20, CalamityGate{})
	ready := st.Ready([]view.CalamityView{changed}, 20)
	if len(ready) != 1 || ready[0].Cause != "second cause" || len(ready[0].Remedies) != 0 {
		t.Fatalf("the open card kept facts the realm no longer holds: %+v", ready)
	}
	a := view.Action{Action: view.ActScaleCrew, Name: "clerks", Replicas: 22}
	if got := fixWords(a, "Scale the clerks"); got != "scale clerks to 22" {
		t.Errorf("fix words = %q", got)
	}
	l, ok := fixLine([]CommandLine{{Label: "x", Command: "c"}, {Label: "y", Action: &a, Show: true}})
	if !ok || l.Label != "y" {
		t.Errorf("the primary fix is %+v (%v)", l, ok)
	}
}

func TestTheTopBarIsTaughtOnceWhenTheRateSettles(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Economy.Income, f.State.Economy.Upkeep, f.State.Economy.Profit = 2, .5, 1.5
	for range 20 {
		f.Clock++
		h.Update(Input{MX: -5, MY: -5}, f)
	}
	if h.LessonTitle() != TopBarTitle() || !h.TaughtChanged() || !slices.Equal(h.Taught(), []string{TeachTopBar}) {
		t.Fatalf("the rate moved and nothing taught it: %q %v", h.LessonTitle(), h.Taught())
	}
	h.LessonCard.Shown = false
	for range 20 {
		f.Clock++
		h.Update(Input{MX: -5, MY: -5}, f)
	}
	if h.LessonTitle() != "" || h.TaughtChanged() {
		t.Fatal("the top bar was taught twice")
	}
	stat := GoldStat(f)
	if stat.Net == "" || !strings.HasPrefix(stat.Net, "=+1.5") {
		t.Errorf("net rate reads %q", stat.Net)
	}
}

func TestAResultCardStaysUntilItIsRead(t *testing.T) {
	h := New(nil, Backend{})
	h.SetTaught([]string{TeachTopBar})
	f := fullFrame(t)
	h.Result("A POD WAS KILLED", "porter-1 was toppled")
	for range 120 {
		f.Clock += 5
		h.Update(Input{MX: -5, MY: -5}, f)
	}
	if h.LessonTitle() != "A POD WAS KILLED" {
		t.Fatalf("the result faded unread: %q", h.LessonTitle())
	}
	clickOn(t, h, f, lessonClose)
	if h.LessonTitle() != "" {
		t.Fatal("the result ignored its close button")
	}
}

func TestTheWelcomeLegendExplainsEachThingWithoutNamingItself(t *testing.T) {
	for _, row := range WelcomeLegend {
		if strings.Contains(strings.ToLower(row[2]), strings.ToLower(row[0])) || strings.TrimSpace(row[2]) == "" {
			t.Errorf("%s is explained by %q", row[0], row[2])
		}
		if ui.Bridge(row[2], row[1]) != row[2] {
			t.Errorf("story mode shows %q for %s", ui.Bridge(row[2], row[1]), row[0])
		}
	}
	spend := welcomeSpend()
	if strings.Contains(strings.Join(spend[:], " "), "Kubernetes") {
		t.Error("story mode names Kubernetes in the welcome")
	}
}

func TestTheStoneExplainsBurningOnce(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h := New(nil, Backend{})
	h.glossStone([]string{"porter   1/1   Burning   0   2min 9s"})
	h.glossStone([]string{"porter   1/1   Burning   0   2min 9s"})
	hints := 0
	for _, l := range h.Stone.Output {
		if l.Kind == LineHint && strings.Contains(l.Text, "Burning means") {
			hints++
		}
	}
	if hints != 1 {
		t.Fatalf("Burning was explained %d times", hints)
	}
}

func TestAPriceReadsAsACostAndAnAgeReadsWithItsUnits(t *testing.T) {
	const unit = "gold"
	met := costRow(250, 300, unit)
	short := costRow(250, 100, unit)
	if !met.Price || met.Text != "Costs 250 "+unit || !met.Met || short.Met || !strings.HasPrefix(short.Text, "Costs 250 "+unit+": you have 100") {
		t.Fatalf("costs read %+v and %+v", met, short)
	}
	for in, want := range map[string]string{"2m9s": "2min 9s", "33m": "33min", "1h5m": "1h 5min", "17s": "17s", "-": "-"} {
		if got := storyAge(in); got != want {
			t.Errorf("age %q reads %q, want %q", in, got, want)
		}
	}
}
