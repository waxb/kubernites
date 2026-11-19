package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func boardIDs(h *HUD, f Frame) map[string]bool {
	out := map[string]bool{}
	for _, b := range h.Buttons(f) {
		out[b.ID] = true
	}
	return out
}

func TestTheHarborBoardStartsAsTierFlowAndBottleneck(t *testing.T) {
	f := nextFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	ids := boardIDs(h, f)
	for _, want := range []string{FreightTip + "tier", FreightTip + "rate", FlowbarNeck, boardMore} {
		if !ids[want] {
			t.Errorf("the folded board lacks %s", want)
		}
	}
	for _, hidden := range []string{FlowbarMoney, nextRow, DemandRowID, tideRow} {
		if ids[hidden] {
			t.Errorf("the folded board still answers %s", hidden)
		}
	}
	staged := h.Stage(f)
	if got, max := FreightRect(staged).H, 2*BoardPad+(boardFoldBody+1)*ui.LineHeight; got != max {
		t.Errorf("the folded board is %d tall, want %d", got, max)
	}
}

func TestMoreOpensEveryLineAndLessFoldsThemBack(t *testing.T) {
	f := nextFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	clickOn(t, h, f, boardMore)
	ids := boardIDs(h, f)
	if !ids[nextRow] || !ids[FlowbarMoney] {
		t.Fatalf("MORE did not open the board: %v", ids)
	}
	clickOn(t, h, f, boardMore)
	if boardIDs(h, f)[nextRow] {
		t.Fatal("LESS did not fold the board")
	}
}

func TestTheBoardOpensForAStepThatNamesAHiddenLine(t *testing.T) {
	cases := map[string]func(*Frame){
		"a step naming NEXT": func(f *Frame) {
			f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Wait", Progress: "CPU 5/9; meanwhile the harbor board's NEXT names a cheaper rung that raises your throughput"}}
		},
		"a guide target on NEXT": func(f *Frame) { f.Guide.Target = nextRow },
		"an intro on the losses": func(f *Frame) { f.Guide.Intro = &Intro{ID: "losses", Target: SurfaceLosses} },
	}
	for name, put := range cases {
		f := nextFrame(t)
		put(&f)
		h := New(nil, (&fakeRunner{}).backend())
		h.SetStaging(false)
		ids := boardIDs(h, f)
		if !ids[nextRow] && name != "an intro on the losses" {
			t.Errorf("%s: the board stayed folded", name)
		}
		if ids[boardMore] {
			t.Errorf("%s: a forced board still offers MORE", name)
		}
		if h.Stage(f).BoardCompact {
			t.Errorf("%s: the staged frame is compact", name)
		}
	}
}

func TestTheSpotlightDimsTheWorldBeforeAnyHudPanelIsDrawn(t *testing.T) {
	at := map[string]int{}
	for i, l := range HudLayers {
		at[l.Name] = i
	}
	for _, panel := range []string{"topbar", "windows", "console", overlayLayer, "intro", "spot"} {
		if at["dim"] >= at[panel] {
			t.Errorf("the dim layer (%d) is drawn after %s (%d): it would darken the HUD", at["dim"], panel, at[panel])
		}
	}
}

func TestAnOverlayHidesTheHarborBoardSoNothingIsCutMidWord(t *testing.T) {
	f := nextFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	dst := ebiten.NewImage(Width, Height)
	drawn := func() bool {
		for _, r := range recordRuns(func() { h.drawFreight(dst, h.Stage(f)) }) {
			if strings.HasPrefix(r.Text, "TIER") {
				return true
			}
		}
		return false
	}
	if !drawn() {
		t.Fatal("the board is not drawn with nothing open")
	}
	for _, overlay := range []string{OverlaySoulvault, OverlayChronicles, OverlayAge} {
		h.OverlayID = overlay
		if drawn() {
			t.Errorf("the board peeks out beside the %s overlay", overlay)
		}
	}
	h.OverlayID = OverlayStone
	if !drawn() {
		t.Error("the small Stone drawer hides the board")
	}
}
