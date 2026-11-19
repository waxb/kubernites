package art

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

func TestColumnCountGrowsWithTheRadius(t *testing.T) {
	cases := map[float64]int{0.15: 1, 0.26: 2, 0.34: 2, 0.42: 3, 0.5: 3, 0.6: 4, 2: 4}
	for r, want := range cases {
		if got := columnCount(r); got != want {
			t.Errorf("radius %.2f gives %d columns, want %d", r, got, want)
		}
	}
}

func TestRoundColumnsSpreadEvenlyAcrossTheVisibleFace(t *testing.T) {
	for n := 1; n <= 4; n++ {
		for _, turn := range columnTurn {
			a := columnAngles(shapeRound, n, turn)
			if len(a) != n {
				t.Fatalf("%d columns asked, %d given", n, len(a))
			}
			sum := 0.0
			for i, deg := range a {
				if _, _, f := (towerSpec{}).facing(deg); f < 0.3 {
					t.Errorf("n %d turn %.0f: column %.1f barely faces the viewer (%.2f)", n, turn, deg, f)
				}
				if i > 1 && math.Abs((deg-a[i-1])-(a[1]-a[0])) > 1e-9 {
					t.Errorf("n %d: columns %v are not evenly spaced", n, a)
				}
				sum += deg
			}
			if math.Abs(sum/float64(n)-45-turn) > 1e-9 {
				t.Errorf("n %d turn %.0f: columns %v are not centred on the viewer", n, turn, a)
			}
		}
	}
}

func TestOctagonColumnsSitOnEveryVisibleFacetCentre(t *testing.T) {
	s := testFrame(ageEmpire, 1)
	tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.4}
	if s.shapeOf(tw) != shapeOct {
		t.Fatal("variant 1 of the Empire age is octagonal")
	}
	p := s.windowPlan(tw, 0, 50)
	visible := []float64{}
	for k := -4; k < 4; k++ {
		if _, _, f := tw.facing(float64(k) * 45); f > 0.01 {
			visible = append(visible, float64(k)*45)
		}
	}
	if len(p.angles) != len(visible) {
		t.Fatalf("octagon columns %v, want one per visible facet %v", p.angles, visible)
	}
	for i, deg := range p.angles {
		if deg != visible[i] {
			t.Errorf("column %d sits at %.2f, not on the facet centre %.0f", i, deg, visible[i])
		}
	}
}

func TestSmallOctagonsLeaveABlankFacetBetweenWindows(t *testing.T) {
	s := testFrame(ageEmpire, 1)
	for _, r := range []float64{0.26, 0.33} {
		p := s.windowPlan(towerSpec{cx: 1.5, cy: 1.5, r: r}, 0, 50)
		for i, deg := range p.angles {
			if math.Mod(deg, 45) != 0 {
				t.Errorf("radius %.2f: column %.1f is off a facet centre", r, deg)
			}
			if i > 0 && deg-p.angles[i-1] <= 45 {
				t.Errorf("radius %.2f: windows on neighbouring facets %v crowd each other", r, p.angles)
			}
		}
		if len(p.angles) != 2 {
			t.Errorf("radius %.2f: %d octagon columns, want the two side facets", r, len(p.angles))
		}
	}
}

func TestSquareColumnsCentreOnEachFace(t *testing.T) {
	if got := columnAngles(shapeSquare, 2, 6); len(got) != 2 || got[0] != 0 || got[1] != 90 {
		t.Errorf("a square tower puts one window in the middle of each face, got %v", got)
	}
	if got := columnAngles(shapeSquare, 1, 6); len(got) != 1 || got[0] != 90 {
		t.Errorf("a lone window on a square tower faces the path side, got %v", got)
	}
}

func TestFloorWindowsTraceTheRing(t *testing.T) {
	for v := 0; v < BuildingVariants; v++ {
		s := testFrame(ageCastle, v)
		tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.45}
		p := s.windowPlan(tw, 10, 50)
		if len(p.floors) < 2 || len(p.angles) != 3 {
			t.Fatalf("variant %d: a 50 px tower of radius 0.45 gets %d floors of %d columns", v, len(p.floors), len(p.angles))
		}
		for _, z := range p.floors {
			for _, deg := range p.angles {
				sp, _ := s.roundPane(tw, deg, z)
				a := deg * math.Pi / 180
				ring := s.p(tw.cx+0.97*tw.r*math.Cos(a), tw.cy+0.97*tw.r*math.Sin(a), z)
				if sp.y != int(math.Round(ring.y)) {
					t.Errorf("variant %d floor %.0f column %.1f: sill at y %d, the ring passes y %.2f", v, z, deg, sp.y, ring.y)
				}
			}
		}
	}
	s := testFrame(ageCastle, 0)
	tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.45}
	p := s.windowPlan(tw, 10, 50)
	z := p.floors[0]
	l, m, r := s.opening(tw, p.angles[0], z), s.opening(tw, p.angles[1], z), s.opening(tw, p.angles[2], z)
	if m.y <= l.y || math.Abs(l.y-r.y) > 1e-9 {
		t.Errorf("a floor sags in the middle like the ring: left %.2f middle %.2f right %.2f", l.y, m.y, r.y)
	}
}

func TestColumnsLineUpAcrossFloors(t *testing.T) {
	for _, age := range []int{ageCastle, ageEmpire} {
		for v := 0; v < BuildingVariants; v++ {
			s := testFrame(age, v)
			tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.6}
			p := s.windowPlan(tw, 0, 70)
			if len(p.floors) < 2 {
				t.Fatalf("age %d: a 70 px tower gets %d floors", age, len(p.floors))
			}
			for _, deg := range p.angles {
				top, tw0 := s.roundPane(tw, deg, p.floors[0])
				for _, z := range p.floors[1:] {
					sp, w := s.roundPane(tw, deg, z)
					if sp.x != top.x || w != tw0 || sp.slope != top.slope {
						t.Errorf("age %d column %.1f drifts between floors: x %d/%d width %d/%d", age, deg, top.x, sp.x, tw0, w)
					}
					if sp.y+int(z) != top.y+int(p.floors[0]) {
						t.Errorf("age %d column %.1f: floors do not repeat the ring", age, deg)
					}
				}
			}
		}
	}
}

func TestFloorCountFollowsTheTowerHeight(t *testing.T) {
	for age := 0; age < AgeTiers; age++ {
		s := testFrame(age, 0)
		tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.34}
		last := 0
		for top := 10.0; top <= 90; top += 4 {
			p := s.windowPlan(tw, 8, top)
			if len(p.floors) < last || len(p.floors) > mostFloors {
				t.Errorf("age %d top %.0f: %d floors after %d", age, top, len(p.floors), last)
			}
			last = len(p.floors)
			for i, z := range p.floors {
				if z < 8 || z+float64(windowHeight[age]) > top {
					t.Errorf("age %d top %.0f: floor at %.0f leaves the wall", age, top, z)
				}
				if i > 0 && z >= p.floors[i-1] {
					t.Errorf("age %d: floors %v are not listed from the top down", age, p.floors)
				}
			}
		}
		if last < 2 {
			t.Errorf("age %d: a 90 px tower gets only %d floors", age, last)
		}
	}
}

func TestLowerFloorsThinOutButKeepTheirColumns(t *testing.T) {
	patterns := map[string]bool{}
	for seed := 0; seed < 40; seed++ {
		for _, r := range []float64{0.34, 0.45, 0.6} {
			s := testFrame(ageCastle, seed%BuildingVariants)
			s.lk.seed += seed * 131
			p := s.windowPlan(towerSpec{cx: 1.5, cy: 1.5, r: r}, 10, 60)
			n := len(p.angles)
			kept := make([][]bool, len(p.floors))
			for i := range kept {
				kept[i] = make([]bool, n)
			}
			for _, q := range p.panes() {
				if q.deg != p.angles[q.col] || q.z != p.floors[q.floor] {
					t.Fatalf("pane %+v leaves the grid %v x %v", q, p.floors, p.angles)
				}
				kept[q.floor][q.col] = true
			}
			checkThinning(t, kept)
			if len(kept) > 1 {
				patterns[fmt.Sprint(kept[1])] = true
			}
		}
	}
	if len(patterns) < 4 {
		t.Errorf("seeds give only %d lower floor patterns", len(patterns))
	}
}

func checkThinning(t *testing.T, kept [][]bool) {
	t.Helper()
	n := len(kept[0])
	for j := range kept[0] {
		if !kept[0][j] {
			t.Errorf("the top floor misses column %d of %d", j, n)
		}
	}
	for i, row := range kept[1:] {
		count := 0
		for j, k := range row {
			if k {
				count++
			}
			if n > 2 && k != row[n-1-j] {
				t.Errorf("floor %d keeps %v, not mirrored about the viewer", i+1, row)
			}
		}
		if count == 0 || count == n {
			t.Errorf("floor %d keeps %d of %d panes, want a thinner floor below the top one", i+1, count, n)
		}
	}
}

func TestDoorsSitOnTheGroundUnderAColumn(t *testing.T) {
	for _, age := range []int{ageCastle, ageEmpire} {
		for v := 0; v < BuildingVariants; v++ {
			s := testFrame(age, v)
			tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.45}
			p := s.doorPlan(tw, 50, 100, 12)
			if p.door < 0 {
				t.Fatal("a door plan has a door")
			}
			deg := p.angles[p.door]
			for _, a := range p.angles {
				if math.Abs(a-100) < math.Abs(deg-100) {
					t.Errorf("door at %.1f while column %.1f is nearer the path side", deg, a)
				}
			}
			for _, q := range p.panes() {
				if q.z < p.doorTop {
					t.Errorf("age %d: pane at %.0f sits in the door band below %.0f", age, q.z, p.doorTop)
				}
			}
			s.towerOpenings(tw, towerPlan{angles: p.angles, door: p.door, doorH: p.doorH})
			ground := s.p(tw.cx+0.97*tw.r*math.Cos(deg*math.Pi/180), tw.cy+0.97*tw.r*math.Sin(deg*math.Pi/180), 0)
			if low := lowestInk(s.c); math.Abs(float64(low)-ground.y) > 3 {
				t.Errorf("age %d variant %d: the door ends at y %d, the ground under its column is %.1f", age, v, low, ground.y)
			}
		}
	}
}

func TestADoorOnARingDropsOnlyItsOwnPane(t *testing.T) {
	s := testFrame(ageCastle, 0)
	tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.5, round: true}
	p := s.withDoor(s.ringPlan(tw, 10, 3), 100, 12)
	got := p.panes()
	if len(got) != 2 {
		t.Fatalf("a ring of three around a door shows %d panes", len(got))
	}
	for _, q := range got {
		if q.col == p.door {
			t.Errorf("a pane sits over the door: %+v", q)
		}
	}
}

func TestShortWallsGetNoFloor(t *testing.T) {
	s := testFrame(ageCastle, 0)
	p := s.doorPlan(towerSpec{cx: 0.5, cy: 0.44, r: 0.34}, 14, 80, 10)
	if len(p.floors) != 0 || p.door < 0 {
		t.Errorf("a 14 px drum keeps its door and no window: floors %v door %d", p.floors, p.door)
	}
}

func TestTowerOpeningsAreDeterministic(t *testing.T) {
	draw := func() []byte {
		s := testFrame(ageEmpire, 2)
		tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.45, h: 50, roofH: 16}
		top := s.tower(tw)
		s.towerOpenings(tw, s.doorPlan(tw, top, 100, 12))
		return s.c.img.Pix
	}
	if !bytes.Equal(draw(), draw()) {
		t.Error("tower openings are not deterministic")
	}
}

func lowestInk(c *canvas) int {
	low := -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.at(x, y).A > 0 {
				low = y
			}
		}
	}
	return low
}

func TestBuildingsPlaceTowerOpeningsThroughThePlanner(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "towerwin.go" || name == "towerplan.go" {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(src, []byte(".towerWindow(")) || bytes.Contains(src, []byte(".towerDoor(")) {
			t.Errorf("%s places a tower opening by hand instead of through a plan", name)
		}
	}
}
