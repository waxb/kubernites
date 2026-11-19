package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func keepSurvey(ok bool) SurveyPanel {
	sv := view.Survey{Kind: "keep", Radius: 4, Wood: 18, Stone: 8, CPU: 6, MemoryGiB: 8, Class: "balanced", Instance: "general-xlarge", OK: ok}
	if !ok {
		sv = view.Survey{Kind: "keep", Radius: 4, Wood: 2, Stone: 1, CPU: 0, MemoryGiB: 1, Class: "memory", Instance: "memory-xlarge",
			Reason: "Too little forest and stone here: a crew needs 1 Hand and 1 GiB of Lore free after the Reeve's share (6 trees and 2 stones)"}
	}
	return SurveyPanel{Survey: sv, Name: "Stone Keep", Keep: true, At: [2]int{400, 200}}
}

func TestSurveyRowsSpeakHandsLoreAndClass(t *testing.T) {
	rows := SurveyRows(keepSurvey(true))
	want := []string{"6,000 rations from 18 trees", "8,192 rooms from 8 stones", "balanced (general-xlarge)"}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v", rows)
	}
	for i, w := range want {
		if rows[i].Text != w {
			t.Errorf("row %d = %q, want %q", i, rows[i].Text, w)
		}
	}
	if rows[0].Icon != IconWood || rows[1].Icon != IconStone || rows[2].Icon != "" || rows[2].Clr != ui.Palette.Green {
		t.Errorf("icons and class colour: %+v", rows)
	}
	poor := SurveyRows(keepSurvey(false))
	if last := poor[len(poor)-1]; !strings.HasPrefix(last.Text, "Too little forest") || last.Clr != ui.Palette.RedText {
		t.Errorf("a refused spot says why: %+v", poor)
	}
	if poor[0].Text != "0 rations from 2 trees" || poor[1].Text != "1,024 rooms from 1 stone" {
		t.Errorf("singulars: %+v", poor)
	}
	if ClassLine("rich", "") != "rich" {
		t.Error("class without an instance")
	}
}

func TestSurveyRowsForOtherBuildingsNameTheLand(t *testing.T) {
	on := SurveyRows(SurveyPanel{Survey: view.Survey{OK: true, Keep: "keep-3"}, Name: "The Archive", Land: "Stone Keep 3"})
	if len(on) != 1 || on[0].Text != "On Stone Keep 3's land" {
		t.Errorf("rows = %+v", on)
	}
	off := SurveyRows(SurveyPanel{Survey: view.Survey{Reason: "the Archive must stand on a Keep's land: build within a Keep's border stones"}, Name: "The Archive"})
	if len(off) != 1 || off[0].Clr != ui.Palette.RedText || !strings.Contains(off[0].Text, "border stones") {
		t.Errorf("rows = %+v", off)
	}
	free := SurveyRows(SurveyPanel{Survey: view.Survey{OK: true}, Name: "The Gate"})
	if free[0].Text != "Free to build here" {
		t.Errorf("rows = %+v", free)
	}
}

func TestSurveyPanelStaysOnTheMapAndClearOfTheRail(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.Notices = []Notice{{ID: "a", Kind: NoticeDanger, Icon: "icon/alert"}, {ID: "b", Kind: NoticeNews, Icon: "icon/chronicle"}}
	for _, at := range [][2]int{{400, 200}, {930, 60}, {10, 400}, {880, 380}, {500, 30}} {
		p := keepSurvey(false)
		p.At = at
		f.Survey = &p
		r, ok := h.SurveyRect(f)
		if !ok {
			t.Fatal("no panel")
		}
		if r.X < 0 || r.Y < TopH || r.X+r.W > Width || r.Y+r.H > ConsoleY {
			t.Errorf("panel %+v leaves the map at %v", r, at)
		}
		if rail := h.RailRect(f); rail.W > 0 && Overlaps(r, rail) {
			t.Errorf("panel %+v covers the rail %+v", r, rail)
		}
		if h.ObjectiveShown(f) && Overlaps(r, h.ObjectiveRectOf(f)) {
			t.Errorf("panel %+v covers the task", r)
		}
	}
	h.Open(OverlayMenu)
	if _, ok := h.SurveyRect(f); ok {
		t.Error("no panel under a window")
	}
}

func TestSurveyPanelDrawsAndFlashesWhenRefused(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	p := keepSurvey(false)
	p.RefusedAt = f.Clock - .1
	f.Survey = &p
	if SurveyBorder(p, f.Clock, true) != ui.Palette.Red {
		t.Error("a fresh refusal flashes red")
	}
	if SurveyBorder(keepSurvey(true), f.Clock, false) != ui.Palette.Gold {
		t.Error("a good spot has the highlight edge")
	}
	if SurveyBorder(keepSurvey(false), f.Clock, false) != ui.Palette.RedText {
		t.Error("a poor spot has a red edge")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	_, w, height := h.SurveyBlocks(p)
	if w > SurveyMaxW || w < SurveyMinW || height < 4*ui.LineHeight {
		t.Errorf("panel %dx%d", w, height)
	}
}

func TestSurveyPanelClearsTheWouldBeLand(t *testing.T) {
	near := SurveySpots([2]int{400, 200}, 0, 150, 60)
	far := SurveySpots([2]int{400, 200}, 92, 150, 60)
	if far[0].X-400 < 92 || far[1].X+far[1].W > 400-92 {
		t.Errorf("the panel steps past the borders: %+v", far[:2])
	}
	if near[0].X-400 != SurveyGap {
		t.Errorf("small reach keeps the panel near: %+v", near[0])
	}
}

func TestRefusalToastsStepBelowTheObjective(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	h.Toast("Too little forest here: 0.9 Hands of Labor free after the Reeve's share, a crew needs 1 (6 trees)", true)
	obj := h.ObjectiveRectOf(f)
	for _, r := range h.ToastRects(f) {
		if Overlaps(r, obj) {
			t.Errorf("toast %+v covers the task %+v", r, obj)
		}
	}
}

func TestThePlacementCardIsAPanelTheWorldsLabelsAvoid(t *testing.T) {
	h, f := calamityCase(t)
	sp := keepSurvey(true)
	f.Survey = &sp
	card, ok := h.SurveyRect(f)
	if !ok {
		t.Fatal("no placement card")
	}
	if rects := h.PanelRects(f); !slices.ContainsFunc(rects, func(r ui.Rect) bool { return containsRect(r, card) }) {
		t.Errorf("the placement card %+v is missing from %+v", card, rects)
	}
}

func TestAKeepsSurveyShowsItsWholeForecastWithTheUpkeepFloor(t *testing.T) {
	h := New(nil, Backend{})
	p := keepSurvey(true)
	p.Payoff = "Room for 19 clerks (the 2 waiting move in first), overhead up to 28.4 MiB/s.\nWith it: -0.1 requests a second now, -0.1 at night (2 more clerks at work, overhead 0.05 MiB a second, its rent for food and housing, until its huts earn more at night, up to 28.4)"
	blocks, _, _ := h.SurveyBlocks(p)
	for _, b := range blocks {
		if last := b.Lines[len(b.Lines)-1]; strings.HasSuffix(last, "...") {
			t.Errorf("the survey cuts %q at %q (capture survey-good: '(2 more clerks at work,...')", b.Row.Text, last)
		}
	}
}
