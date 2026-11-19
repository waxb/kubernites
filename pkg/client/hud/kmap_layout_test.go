package hud

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func kp(id, branch, era, state string, requires ...string) view.GrimoirePage {
	return view.GrimoirePage{ID: id, Name: id, K8s: id, Branch: branch, Era: era, State: state, Requires: requires, MaxRank: 1}
}

func checkLayout(t *testing.T, l KMapLayout) {
	t.Helper()
	for i, a := range l.Plaques {
		ra := a.Rect()
		if ra.X < 0 || ra.Y < 0 || ra.X+ra.W > l.Width*KMapTile || a.Row+SlotTiles > l.Height {
			t.Errorf("%s at %+v leaves the %dx%d map", a.Page.ID, ra, l.Width, l.Height)
		}
		for _, b := range l.Plaques[:i] {
			if Overlaps(ra, b.Rect()) {
				t.Errorf("%s %+v overlaps %s %+v", a.Page.ID, ra, b.Page.ID, b.Rect())
			}
		}
		for _, req := range a.Page.Requires {
			q, ok := l.Find(req)
			if ok && q.Col+PlaqueTiles+2 >= a.Col {
				t.Errorf("prerequisite %s (col %d) is not left of %s (col %d)", req, q.Col, a.Page.ID, a.Col)
			}
		}
		for _, lane := range l.Lanes {
			if lane >= a.Row && lane < a.Row+SlotTiles {
				t.Errorf("%s sits on lane %d", a.Page.ID, lane)
			}
		}
	}
}

func TestUnknownPagesGoToASensibleDefaultSlot(t *testing.T) {
	pages := []view.GrimoirePage{
		kp("a", view.BranchNetworking, view.AgeFeudal, "unlocked"),
		kp("b", "", "", "available", "a"),
		kp("c", "", "", "locked"),
		kp("d", "magic", "space", "locked", "c"),
		kp("e", view.BranchScheduling, view.AgeDark, "locked", "a"),
	}
	l := LayoutKnowledge(pages, KmapRowsTiles)
	want := map[string][2]int{"a": {0, 1}, "b": {0, 1}, "c": {3, 0}, "d": {3, 0}, "e": {1, 1}}
	for id, at := range want {
		p, ok := l.Find(id)
		if !ok || p.Branch != at[0] || p.Era != at[1] {
			t.Errorf("%s at branch %d era %d, want %v", id, p.Branch, p.Era, at)
		}
	}
	checkLayout(t, l)
}

func TestOnlyRevealedPagesAndTheirFoggedPrerequisitesShow(t *testing.T) {
	pages := []view.GrimoirePage{
		kp("seen", view.BranchUpkeep, view.AgeDark, "available", "veiled", "gone"),
		kp("veiled", view.BranchUpkeep, view.AgeDark, "hidden", "deeper"),
		kp("deeper", view.BranchUpkeep, view.AgeDark, "hidden"),
		kp("far", view.BranchNetworking, view.AgeFeudal, "hidden"),
	}
	l := LayoutKnowledge(pages, KmapRowsTiles)
	ids := []string{}
	for _, p := range l.Plaques {
		ids = append(ids, p.Page.ID)
		if p.Fog != (p.Page.ID == "veiled") {
			t.Errorf("%s fog = %v", p.Page.ID, p.Fog)
		}
	}
	if !reflect.DeepEqual(ids, []string{"seen", "veiled"}) {
		t.Fatalf("shown = %v", ids)
	}
	if v, _ := l.Find("veiled"); v.Col >= mustFind(t, l, "seen").Col {
		t.Error("a fogged prerequisite still sits left of what needs it")
	}
}

func TestEmptyErasStayOpenAsUnchartedLand(t *testing.T) {
	l := LayoutKnowledge([]view.GrimoirePage{kp("a", view.BranchUpkeep, view.AgeDark, "available")}, KmapRowsTiles)
	next := 0
	for i, e := range l.Eras {
		if e.Start != next || e.Size <= 0 {
			t.Fatalf("era %d = %+v, want start %d", i, e, next)
		}
		next = e.Start + e.Size
	}
	for _, e := range l.Eras[1:] {
		if e.Size != EmptyEraTiles {
			t.Errorf("empty era %+v", e)
		}
	}
	if !l.Empty(1) || l.Empty(0) {
		t.Error("Empty reports the wrong eras")
	}
}

func TestCrowdedCellsSpillIntoNewColumns(t *testing.T) {
	pages := []view.GrimoirePage{}
	for i := range 5 {
		pages = append(pages, kp(fmt.Sprint("p", i), view.BranchScheduling, view.AgeDark, "available"))
	}
	l := LayoutKnowledge(pages, KmapRowsTiles)
	checkLayout(t, l)
	cols := map[int]int{}
	for _, p := range l.Plaques {
		cols[p.Col]++
	}
	for col, n := range cols {
		if n > 2 {
			t.Errorf("column %d stacks %d plaques", col, n)
		}
	}
	total := 0
	for _, r := range l.Rows {
		total += r.Size
	}
	if len(cols) != 3 || total != l.Height || l.Rows[1].Size < 2*SlotTiles+RowPadTiles {
		t.Errorf("cols %v, rows %+v", cols, l.Rows)
	}
}

func TestShortMapsFallBackToSingleStacks(t *testing.T) {
	pages := []view.GrimoirePage{}
	for _, b := range view.KnowledgeBranches {
		for i := range 2 {
			pages = append(pages, kp(fmt.Sprint(b, i), b, view.AgeDark, "available"))
		}
	}
	l := LayoutKnowledge(pages, 20)
	checkLayout(t, l)
	if n := sharedColumns(l); n != 0 {
		t.Errorf("a short map still stacks %d plaques", n)
	}
	if tall := LayoutKnowledge(pages, 28); sharedColumns(tall) != 4 {
		t.Errorf("a tall map stacks two in each branch: %d", sharedColumns(tall))
	}
}

func sharedColumns(l KMapLayout) int {
	seen, shared := map[[2]int]bool{}, 0
	for _, p := range l.Plaques {
		key := [2]int{p.Col, p.Branch}
		if seen[key] {
			shared++
		}
		seen[key] = true
	}
	return shared
}

func TestCyclesAndMissingPrerequisitesDoNotBreakTheLayout(t *testing.T) {
	pages := []view.GrimoirePage{
		kp("x", view.BranchUpkeep, view.AgeDark, "available", "y", "nowhere"),
		kp("y", view.BranchUpkeep, view.AgeDark, "available", "x"),
		kp("self", view.BranchUpkeep, view.AgeDark, "available", "self"),
	}
	l := LayoutKnowledge(pages, KmapRowsTiles)
	if len(l.Plaques) != 3 {
		t.Fatalf("plaques = %d", len(l.Plaques))
	}
	for i, a := range l.Plaques {
		for _, b := range l.Plaques[:i] {
			if Overlaps(a.Rect(), b.Rect()) {
				t.Errorf("%s overlaps %s", a.Page.ID, b.Page.ID)
			}
		}
	}
	for _, tile := range KnowledgeRoads(l) {
		if tile.X < 0 || tile.Y < 0 {
			t.Errorf("road tile %+v", tile)
		}
	}
}

func randomPages(rng *rand.Rand, n int) []view.GrimoirePage {
	states := []string{"hidden", "locked", "available", "unlocked"}
	out := make([]view.GrimoirePage, 0, n)
	for i := range n {
		reqs := []string{}
		for j := range i {
			if rng.IntN(6) == 0 {
				reqs = append(reqs, fmt.Sprint("r", j))
			}
		}
		branch := view.KnowledgeBranches[rng.IntN(4)]
		era := view.KnowledgeEras[rng.IntN(4)]
		if rng.IntN(8) == 0 {
			branch, era = "", ""
		}
		out = append(out, kp(fmt.Sprint("r", i), branch, era, states[rng.IntN(4)], reqs...))
	}
	return out
}

func TestRandomGrimoiresKeepEveryLayoutRule(t *testing.T) {
	rng := rand.New(rand.NewPCG(20, 26))
	for round := range 60 {
		pages := randomPages(rng, 4+rng.IntN(30))
		l := LayoutKnowledge(pages, KmapRowsTiles)
		checkLayout(t, l)
		checkRoadsAvoidPlaques(t, l)
		if t.Failed() {
			t.Fatalf("round %d broke the rules", round)
		}
	}
}

func checkRoadsAvoidPlaques(t *testing.T, l KMapLayout) {
	t.Helper()
	for _, tile := range KnowledgeRoads(l) {
		cell := ui.Rect{X: tile.X * KMapTile, Y: tile.Y * KMapTile, W: KMapTile, H: KMapTile}
		band := ui.Rect{X: cell.X + 5, Y: cell.Y + 5, W: 6, H: 6}
		for _, p := range l.Plaques {
			if Overlaps(band, p.Rect()) {
				t.Errorf("road tile %+v runs under %s", tile, p.Page.ID)
			}
		}
		if tile.X >= l.Width || tile.Y >= l.Height || !slices.Contains([]string{"locked", "open", "learned"}, tile.State) {
			t.Errorf("road tile %+v off the map", tile)
		}
	}
}

func mustFind(t *testing.T, l KMapLayout, id string) KMapPlaque {
	t.Helper()
	p, ok := l.Find(id)
	if !ok {
		t.Fatalf("no plaque %s", id)
	}
	return p
}
