package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestADeltaNeverOverlapsAnotherDeltaThePurseOrAPanel(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.Guide.Staged = false
	h.Update(Input{MX: -10, MY: -10}, f)
	h.Coins.Landed = 4
	for i := range 4 {
		h.Delta(MeterGold, 2)
		h.Delta(MeterWisdom, 1)
		f.Clock += .3 * float64(i+1)
		h.Update(Input{MX: -10, MY: -10}, f)
		frame := AttnFrame{Surfaces: h.Surfaces(f)}
		if faults := CheckAttention(AttnLog{Frames: []AttnFrame{frame}}, AttnRules{OverlapOnly: true}); len(faults) > 0 {
			lines := []string{}
			for _, x := range faults {
				lines = append(lines, x.String())
			}
			t.Fatalf("after %d gains:\n  %s", i+1, strings.Join(lines, "\n  "))
		}
	}
}

func TestAHoverTipNeverCoversTheSurveyBesideIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	p := keepSurvey(false)
	p.At = [2]int{760, 360}
	f.Survey = &p
	survey, ok := h.SurveyRect(f)
	if !ok {
		t.Fatal("no survey panel")
	}
	h.Hovering = true
	h.Hover = Button{ID: "build:keep", Rect: ui.Rect{X: 740, Y: 420, W: 72, H: 36}, Tip: "Stone Keep: a Node where your workers live and carry boxes. Holds 10 huts."}
	tip := h.TooltipRect(h.Font.Wrap(h.Hover.Tip, TipWidth, 1), f)
	if Overlaps(tip, survey) {
		t.Fatalf("the hover tip %+v covers the survey %+v", tip, survey)
	}
}

func TestAToastKeepsClearOfTheStepPanelsWholeRegisteredRect(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	h.SetStaging(false)
	h.Update(Input{MX: -10, MY: -10}, f)
	step := Surface{}
	for _, s := range h.Surfaces(f) {
		if s.Slot == SlotStep {
			step = s
		}
	}
	if step.Key == "" {
		t.Fatal("no step panel: the test proves nothing")
	}
	for _, c := range h.toastClear(h.Stage(f)) {
		if c.X <= step.Rect.X && c.Y <= step.Rect.Y && c.X+c.W >= step.Rect.X+step.Rect.W && c.Y+c.H >= step.Rect.Y+step.Rect.H {
			return
		}
	}
	t.Fatalf("toasts keep clear of %v but the step panel is registered at %+v (its shadow): a toast may sit on its edge", h.toastClear(h.Stage(f)), step.Rect)
}

func TestAToastWhoseKeptSpotLandsOnAPanelMovesClearOfIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	h.SetStaging(false)
	h.Update(Input{MX: -10, MY: -10}, f)
	h.Toast("Nothing answers.", true)
	h.ToastList[0].Spot = ui.Rect{X: Width/2 - 40, Y: TopH + 8, W: 80, H: 17}
	h.ToastList[0].Spot.X = h.ObjectiveRectOf(h.Stage(f)).X + h.ObjectiveRectOf(h.Stage(f)).W - 1
	h.Update(Input{MX: -10, MY: -10}, f)
	h.settleToasts(h.Stage(f))
	panel := h.ObjectiveRectOf(h.Stage(f)).Inset(-TipClear)
	for _, r := range h.ToastRects(h.Stage(f)) {
		if Overlaps(r, panel) {
			t.Fatalf("the toast %+v stays on the step panel %+v that came back under it", r, panel)
		}
	}
}

func TestTheGainFloatStaysClearOfTheStepPanel(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	h.SetStaging(false)
	h.Update(Input{MX: -10, MY: -10}, f)
	for age := 0.0; age < GainLife; age += .1 {
		h.Coins.Gain = gainWatch{shown: true, Float: gainFloat{text: "+2", x: 140, start: h.events - age, Good: true, life: GainLife}}
		if r, ok := h.gainRect(f); ok && Overlaps(r, h.ObjectiveRectOf(h.Stage(f)).Inset(-TipClear)) {
			t.Fatalf("at %.1f s the gain float %+v sits on the step panel %+v", age, r, h.ObjectiveRectOf(h.Stage(f)))
		}
	}
}

const bannerEdge = 2

func TestABannerKeepsClearOfTheStepPanelsWholeRegisteredRect(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	h.SetStaging(false)
	h.Update(Input{MX: -10, MY: -10}, f)
	step := Surface{}
	for _, s := range h.Surfaces(f) {
		if s.Slot == SlotStep {
			step = s
		}
	}
	need := step.Rect.Inset(-bannerEdge)
	for _, c := range h.BannerBlocked(h.Stage(f)) {
		if c.X <= need.X && c.Y <= need.Y && c.X+c.W >= need.X+need.W && c.Y+c.H >= need.Y+need.H {
			return
		}
	}
	t.Fatalf("banners keep clear of %v but the step panel is registered at %+v", h.BannerBlocked(h.Stage(f)), step.Rect)
}

func TestAToastKeepsClearOfTheWorldsLabels(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	h.SetStaging(false)
	h.Update(Input{MX: -10, MY: -10}, f)
	h.Toast("pod porter-x deleted from default namespace", false)
	first := h.ToastRects(h.Stage(f))[0]
	f.WorldBoxes = []ui.Rect{first.Inset(-1)}
	h.ToastList[0].Spot = ui.Rect{}
	if r := h.ToastRects(h.Stage(f))[0]; Overlaps(r, f.WorldBoxes[0]) {
		t.Fatalf("the toast %+v covers a world label at %+v", r, f.WorldBoxes[0])
	}
}

func TestAPinnedTipInAWindowStillKeepsClearOfTheSurveyOverThatWindow(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := workerFrame(t)
	h.SetStaging(false)
	h.Update(Input{MX: -10, MY: -10}, f)
	w, ok := WindowSpecs[PanelWorker].Rect(h, h.Stage(f))
	if !ok {
		t.Fatal("the worker window is not open")
	}
	p := keepSurvey(false)
	p.At = [2]int{w.X + w.W/4, w.Y + w.H/2}
	f.Survey = &p
	survey, ok := h.SurveyRect(f)
	if !ok || !w.Contains(survey.X+survey.W/2, survey.Y+survey.H/2) {
		t.Skipf("the survey %+v does not sit over the worker window %+v", survey, w)
	}
	b := Button{ID: "sel:close", Rect: ui.Rect{X: w.X + w.W - 20, Y: w.Y + 4, W: 14, H: 14}, Tip: "Close this window"}
	h.Hovering, h.Hover = true, b
	for _, r := range h.pinBlockers(nil, b, f) {
		if r == survey.Inset(-TipClear) {
			return
		}
	}
	t.Fatalf("a tip pinned in the worker window %+v does not keep clear of the survey %+v drawn over it", w, survey)
}
