package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func roadFixture() view.RoadView {
	return view.RoadView{
		Current: "feudal",
		Ages: []view.AgeStep{
			{ID: "dark", Name: "First Panel", Goal: "Hold 99% ready: 90% of requests served, and answer an incident report.", Unlocks: []string{"Shifts (ReplicaSets)", "Augury (kubectl top)", "Pledges (requests)"}, Status: "done"},
			{ID: "feudal", Name: "Second Panel", Goal: "Hold 99.9% ready on two Keeps with honest pledges and a War Horn sounding over the Shift.", Unlocks: []string{"Stone Keeps (Nodes)", "War Horns (HorizontalPodAutoscaler)", "The Archive: a second crew that depends on the first", "Traffic spikes (load tests)", "Burst limits (rate limits)", "Rollout restarts (kubectl rollout)"}, Status: "current"},
			{ID: "castle", Name: "Third Panel", Goal: "Hold 99.99% ready through a traffic spike across two islands.", Unlocks: []string{"Great and Citadel Keeps (instance types)", "Island zones (topology)"}, Status: "locked"},
			{ID: "imperial", Name: "Fourth Panel", Goal: "Hold 99.999% ready while the realm upgrades itself.", Unlocks: []string{"Banners (Deployments)"}, Status: "future"},
			{ID: "ascension", Name: "Ascension", Goal: "Eight burst tests pass and open the gate.", Unlocks: nil, Status: "future"},
		},
		ReforgeKeeps: []string{"Manual pages", "Registry classes", "Spare threads", "The Tapestry of notes"},
		ReforgeLoses: []string{"CPU, Keeps and islands", "berths and renown", "hand-cast huts"},
	}
}

func roadFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Road = roadFixture()
	return f
}

func TestRoadColumnsFitTheOverlay(t *testing.T) {
	right := roadRight()
	for n := 1; n <= 7; n++ {
		cols := RoadColumns(n)
		if len(cols) != n {
			t.Fatalf("%d columns, want %d", len(cols), n)
		}
		for i, c := range cols {
			if c.X < RoadBox.X+8 || c.X+c.W > right.X-4 || c.W < 80 {
				t.Errorf("n=%d column %d = %+v leaves its region", n, i, c)
			}
			if i > 0 && cols[i-1].X+cols[i-1].W > c.X {
				t.Errorf("n=%d columns %d and %d overlap", n, i-1, i)
			}
		}
	}
	if RoadColumns(0) != nil {
		t.Error("no ages, no columns")
	}
	if right.X+right.W > RoadBox.X+RoadBox.W-8 || RoadBox.Y+RoadBox.H > ConsoleY {
		t.Errorf("road box %+v or ledger %+v leave the screen", RoadBox, right)
	}
	pane := roadPane()
	if pane.Y+pane.H > roadLoomButton().Rect.Y+roadLoomButton().Rect.H || pane.Y < RoadBox.Y+roadColTop+roadColH {
		t.Errorf("side pane %+v overlaps", pane)
	}
}

func TestRoadStatusNormalises(t *testing.T) {
	cases := map[string]string{"done": RoadDone, "past": RoadDone, "completed": RoadDone, "current": RoadHere, "now": RoadHere, "locked": RoadAhead, "": RoadAhead}
	for in, want := range cases {
		if got := RoadStatus(view.AgeStep{Status: in}, ""); got != want {
			t.Errorf("status %q = %q, want %q", in, got, want)
		}
	}
	if RoadStatus(view.AgeStep{ID: "castle", Status: "locked"}, "castle") != RoadHere {
		t.Error("the current tier is where the player stands, whatever its status says")
	}
	for _, s := range []string{RoadDone, RoadHere, RoadAhead} {
		tag, _ := roadTag(s)
		if tag == "" {
			t.Errorf("status %s has no tag: never colour alone", s)
		}
	}
}

func TestRoadTextFitsItsColumn(t *testing.T) {
	road := roadFixture()
	road.Ages[1].Name = "The Very Long Second And Also Rather Wordy"
	road.Ages[1].Goal = strings.Repeat("Hold 99.9% ready on two Keeps with honest pledges. ", 4)
	cols := RoadColumns(len(road.Ages))
	for i, a := range road.Ages {
		txt := roadText(a, RoadStatus(a, road.Current), cols[i].W-8)
		for _, l := range append([]string{txt.name, txt.tag}, txt.goal...) {
			if w, _ := (ui.Font{}).Measure(l, 1); w > cols[i].W-8 {
				t.Errorf("age %s line %q is %dpx in a %dpx column", a.ID, l, w, cols[i].W-8)
			}
		}
		if len(txt.goal) > roadGoalLines {
			t.Errorf("age %s goal has %d lines", a.ID, len(txt.goal))
		}
	}
	if _, clr := roadTag(RoadHere); clr != ui.Palette.GoldBright {
		t.Error("you are here in the bright color")
	}
}

func TestRoadPaneColumnsFitAndCount(t *testing.T) {
	items := []string{}
	for i := range 40 {
		items = append(items, fmt.Sprintf("Page %d of the manual: a ReplicaSet or some other object", i))
	}
	colW := (roadPane().W - roadGap) / roadPaneCols
	cols := RoadPaneColumns(items, colW, roadPaneRows, roadPaneCols)
	if len(cols) != roadPaneCols {
		t.Fatalf("columns = %d", len(cols))
	}
	for i, c := range cols {
		if len(c) > roadPaneRows {
			t.Errorf("column %d has %d rows", i, len(c))
		}
		for _, l := range c {
			if w, _ := (ui.Font{}).Measure(l, 1); w > colW {
				t.Errorf("%q is %dpx in %dpx", l, w, colW)
			}
		}
	}
	last := cols[len(cols)-1]
	if !strings.HasPrefix(last[len(last)-1], "+") {
		t.Errorf("hidden unlocks are counted: %q", last[len(last)-1])
	}
	few := RoadPaneColumns([]string{"a", "b"}, colW, roadPaneRows, roadPaneCols)
	if len(few[0]) != 2 || len(few[1]) != 0 {
		t.Errorf("few = %v", few)
	}
	exact := RoadPaneColumns([]string{"a", "b", "c"}, colW, 1, 3)
	if exact[2][0] != "- c" {
		t.Errorf("the last item fits without a count: %v", exact)
	}
}

func TestLedgerLinesFit(t *testing.T) {
	long := strings.Repeat("The Tapestry: every object you declared (last-applied-configuration) ", 1)
	keeps := []string{long, long, long, long, long, long, long, long, long, long}
	loses := []string{"Gold", "The Tier", "Keeps", "Harbor berths", "Huts cast by hand", "Seals", "Objectives"}
	k, l := LedgerLines(keeps, loses, roadRight().W, roadLedgerRows-2)
	if len(k)+len(l) > roadLedgerRows-1 || len(l) != len(loses) {
		t.Errorf("keeps %d + loses %d lines", len(k), len(l))
	}
	for _, line := range append(k, l...) {
		if w, _ := (ui.Font{}).Measure(line, 1); w > roadRight().W {
			t.Errorf("%q overflows", line)
		}
	}
}

func TestReforgeLedgerFallsBackAndFits(t *testing.T) {
	f := roadFrame(t)
	keeps, loses := ReforgeLedger(f)
	if len(keeps) != 4 || keeps[0] != "Manual pages" || len(loses) != 3 {
		t.Errorf("server lists = %v / %v", keeps, loses)
	}
	f.State.Road.ReforgeKeeps, f.State.Road.ReforgeLoses = nil, nil
	keeps, loses = ReforgeLedger(f)
	if len(keeps) == 0 || len(loses) == 0 {
		t.Error("the rebuild ledger falls back to the client's own lists")
	}
	if got := SealsText(view.SiegeView{Seals: 1, SealsGoal: 8}); got != "1 OF 8 SEALS" {
		t.Errorf("seals = %q", got)
	}
	if got := SealsText(view.SiegeView{}); got != "0 OF 8 SEALS" {
		t.Errorf("seals without a goal = %q", got)
	}
}

func TestRoadOpensFromTheAgeAndTheKey(t *testing.T) {
	h := New(nil, Backend{})
	f := roadFrame(t)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyR}}, f)
	if h.Overlay() != OverlayRoad {
		t.Fatal("R opens the road panel")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	if h.Overlay() != OverlayNone {
		t.Error("Esc closes the road opened by key")
	}
	h.Open(OverlayAge)
	clickOn(t, h, f, "age:road")
	if h.Overlay() != OverlayRoad {
		t.Fatal("the tier screen opens the road")
	}
	for _, b := range h.Buttons(f) {
		if strings.HasPrefix(b.ID, "road:age:") && b.Tip == "" {
			t.Errorf("%s has no tooltip", b.ID)
		}
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	if h.Overlay() != OverlayAge {
		t.Error("Esc backs out to the tier screen it came from")
	}
	h.Open(OverlayRoad)
	clickOn(t, h, f, "road:age:castle")
	if a, _ := h.selectedAge(f); a.ID != "castle" {
		t.Errorf("clicking an age selects it: %s", a.ID)
	}
	clickOn(t, h, f, "road:loom")
	if h.Overlay() != OverlayAge {
		t.Error("the last button opens the tier screen")
	}
	hidden := roadFrame(t)
	hidden.State.Reveals = []string{"serve", "gold"}
	h = New(nil, Backend{})
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyR}}, hidden)
	if h.Overlay() == OverlayRoad {
		t.Error("the road is gated behind the ageup reveal")
	}
	uncharted := fullFrame(t)
	h.Open(OverlayAge)
	for _, b := range h.Buttons(uncharted) {
		if b.ID == "age:road" {
			t.Error("no road button before the server charts the road")
		}
	}
}

func TestRoadDraws(t *testing.T) {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := roadFrame(t)
	for _, reduce := range []bool{false, true} {
		h := New(nil, Backend{})
		h.Reduce = reduce
		h.Open(OverlayRoad)
		h.Draw(dst, f)
	}
	empty := fullFrame(t)
	h := New(nil, Backend{})
	h.Open(OverlayRoad)
	h.Draw(dst, empty)
	if len(roadButtons(empty)) != 3 {
		t.Error("an uncharted road still offers its three buttons")
	}
}

func TestRoadUnlockLines(t *testing.T) {
	lines := RoadUnlockLines([]string{"a", "b", "c"}, 120, 5)
	if len(lines) != 3 || lines[0] != "- a" {
		t.Errorf("lines = %v", lines)
	}
	if got := RoadUnlockLines([]string{"one", "two", "three", "four"}, 120, 2); len(got) != 2 || got[1] != "+3 more" {
		t.Errorf("capped = %v", got)
	}
	wrapped := RoadUnlockLines([]string{"War Horns (HorizontalPodAutoscaler) sound on their own"}, 100, 5)
	if len(wrapped) < 2 || !strings.HasPrefix(wrapped[1], "  ") {
		t.Errorf("continuations are indented: %v", wrapped)
	}
	if unlockTitle(view.AgeStep{ID: "feudal", Name: "The Second Panel"}, "feudal") != "WHAT THE SECOND PANEL OFFERS YOU NOW" ||
		unlockTitle(view.AgeStep{ID: "castle", Name: "Third Panel"}, "feudal") != "WHAT THE THIRD PANEL UNLOCKS" {
		t.Error("side pane titles")
	}
}
