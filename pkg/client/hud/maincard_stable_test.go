package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

var revealOrder = []struct{ reveal, id string }{
	{"build", "cmd:build"}, {"door", "cmd:door"}, {"summon", CmdSummonID}, {"depot", StallTarget("parcel")},
	{"crew", CmdCrew}, {"chronicles", "cmd:chronicles"}, {"grimoire", "cmd:grimoire"}, {"soulvault", "cmd:soulvault"},
	{"ageup", "cmd:age"}, {"siege", "cmd:siege"},
}

func stableFrame(t *testing.T, reveals int, crewServes bool) Frame {
	t.Helper()
	f := fullFrame(t)
	f.Pods = nil
	f.State.Commands = nil
	f.State.Unlocks = map[string]bool{"war-horns": true}
	f.State.Reveals = nil
	f.Guide = Guide{Staged: true}
	for _, r := range revealOrder[:reveals] {
		f.State.Reveals = append(f.State.Reveals, r.reveal)
		f.Guide.Introduced = append(f.Guide.Introduced, r.id)
	}
	f.State.Objectives = []view.ObjectiveView{{ID: CrewServesObjective, Text: "The crew carries", Done: crewServes}}
	return f
}

func cellsOf(h *HUD, f Frame) map[string]int {
	return h.MainCells(f)
}

func TestEveryCommandKeepsItsHomeCellAsOthersArrive(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	seen := map[string]int{}
	for n := 1; n <= len(revealOrder); n++ {
		for _, served := range []bool{false, true} {
			for id, cell := range cellsOf(h, stableFrame(t, n, served)) {
				if cell == MoreCell {
					continue
				}
				if cell != MainHomes[id] {
					t.Errorf("%d reveals: %s sits in cell %d, its home is %d", n, id, cell, MainHomes[id])
				}
				if was, ok := seen[id]; ok && was != cell {
					t.Errorf("%s moved from cell %d to %d", id, was, cell)
				}
				seen[id] = cell
			}
		}
	}
}

func TestGuidanceNeverShufflesTheOtherButtons(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	for _, served := range []bool{false, true} {
		plain := cellsOf(h, stableFrame(t, len(revealOrder), served))
		for _, r := range revealOrder {
			f := stableFrame(t, len(revealOrder), served)
			f.Guide.Intro = &Intro{ID: "x", Title: "x", Try: "x", Target: r.id, Via: ViaOpen}
			guided := cellsOf(h, f)
			if guided[r.id] == MoreCell {
				t.Errorf("the guided %s went to MORE", r.id)
			}
			for id, cell := range guided {
				if id != r.id && cell != MoreCell && cell != plain[id] {
					t.Errorf("guiding %s moved %s from cell %d to %d", r.id, id, plain[id], cell)
				}
			}
		}
	}
}

func TestTheFeudalCardSendsTheOpeningToysToMoreInAFixedOrder(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := stableFrame(t, len(revealOrder), true)
	f.State.Reveals = slices.DeleteFunc(f.State.Reveals, func(r string) bool { return r == "door" })
	cells := cellsOf(h, f)
	if cells["cmd:siege"] != 7 || cells[CmdCrew] != 0 || cells[CmdSummonID] != MoreCell {
		t.Fatalf("CREW holds SERVE's old cell and the spike button its own once the crews carry: %v", cells)
	}
	_, more := h.MainCard(f)
	ids := []string{}
	for _, b := range more {
		ids = append(ids, b.ID)
	}
	if !slices.Equal(ids, []string{CmdSummonID}) {
		t.Fatalf("MORE holds the opening toys in a fixed order: %v", ids)
	}
	opening := stableFrame(t, len(revealOrder), false)
	if cells := cellsOf(h, opening); cells[CmdCrew] != 0 || cells["cmd:door"] != 2 || hasKey(cells, CmdDepot) {
		t.Fatalf("before the crews carry, the door keeps its cell and CREW holds SERVE's: %v", cells)
	}
}

func hasKey(cells map[string]int, id string) bool {
	_, ok := cells[id]
	return ok
}
