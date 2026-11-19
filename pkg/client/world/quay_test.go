package world

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

func kindsWith(cohorts ...[]view.CohortView) []view.KindView {
	out := make([]view.KindView, len(FreightKinds))
	for i := range out {
		out[i] = view.KindView{Kind: FreightKinds[i], SizeMix: [4]float64{1}}
		if i < len(cohorts) {
			out[i].Cohorts = cohorts[i]
		}
	}
	return out
}

func alwaysHaven(int) HavenPicker { return func(float64, uint32) int { return 0 } }

func TestManifestSeatsEachLoadInWholeStacks(t *testing.T) {
	var m Manifest
	kinds := kindsWith([]view.CohortView{{Count: 12, Age: 20, Remaining: 8, TTL: 28}, {Count: 5, Age: 2, Remaining: 26, TTL: 28}})
	if !m.Update(kinds, 30, 1, alwaysHaven(0)) {
		t.Fatal("a fresh manifest reports no change")
	}
	loads := m.Sorted()
	if len(loads) != 2 || len(loads[0].Stacks) != 6 || len(loads[1].Stacks) != 3 || loads[1].Stacks[2].N != 1 {
		t.Fatalf("stacks %+v / %+v", loads[0].Stacks, loads[1].Stacks)
	}
	for i, st := range append(append([]Stack(nil), loads[0].Stacks...), loads[1].Stacks...) {
		if st.Spot != i {
			t.Errorf("stack %d stands on spot %d: the oldest load takes the front spots first", i, st.Spot)
		}
	}
	if loads[0].At != 10 || loads[0].Total != 12 || loads[1].Count != 5 {
		t.Errorf("loads %+v %+v", *loads[0], *loads[1])
	}
	if m.Update(kinds, 30, 1.1, alwaysHaven(0)) {
		t.Error("the same cohorts again report a change")
	}
}

func TestServedBoxesLeaveTheTopAndTheRestStayPut(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 7, Age: 1, Remaining: 27, TTL: 28}}), 10, 0, alwaysHaven(0))
	before := m.Sorted()[0].Boxes()
	m.Update(kindsWith([]view.CohortView{{Count: 6, Age: 2, Remaining: 26, TTL: 28}}), 11, 1, alwaysHaven(0))
	after := m.Sorted()[0].Boxes()
	if len(before) != 7 || len(after) != 6 {
		t.Fatalf("boxes %d then %d", len(before), len(after))
	}
	if gone := before[0]; gone.Index != 0 || gone.Level != StackHeight-1 {
		t.Errorf("the first box served is the top of the first stack: %+v", gone)
	}
	for i, b := range after {
		was := before[i+1]
		if b.Column != was.Column || b.Level != was.Level || b.Lift != was.Lift || b.Look.Variant != was.Look.Variant {
			t.Errorf("box %d moved: %+v -> %+v", b.Index, was, b)
		}
	}
}

func onSpots(l Load) Load {
	l.Stacks = NewStacks(0, l.Total)
	for i := range l.Stacks {
		l.Stacks[i].Spot = i
	}
	return l
}

func TestLoadsStackBottomUpWithTheirSizes(t *testing.T) {
	l := onSpots(Load{Total: 7, Count: 7, Sizes: []uint8{0, 1, 2, 0, 0, 3, 1}, TTL: 10, Remaining: 5})
	boxes := l.Boxes()
	if StackHeight != 2 {
		t.Fatalf("the expectations below assume stacks of 2, not %d", StackHeight)
	}
	want := map[int][2]int{0: {1, gfx.BoxLift(1)}, 1: {0, 0}, 2: {1, gfx.BoxLift(0)}, 4: {1, gfx.BoxLift(3)}, 6: {0, 0}}
	for _, b := range boxes {
		if w, ok := want[b.Index]; ok && (b.Level != w[0] || b.Lift != w[1]) {
			t.Errorf("box %d level %d lift %d, want %v", b.Index, b.Level, b.Lift, w)
		}
		if b.Reveal != 6-b.Index || b.Look.Fresh != .5 {
			t.Errorf("box %d reveal %d fresh %v", b.Index, b.Reveal, b.Look.Fresh)
		}
	}
	if boxes[6].Column != 3 || boxes[3].Column != 1 {
		t.Errorf("columns %d %d", boxes[6].Column, boxes[3].Column)
	}
}

func TestACohortThatOutlivesItsFreshnessSpoilsAsAGhost(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 4, Age: 27.9, Remaining: .1, TTL: 28}, {Count: 1, Age: 5, Remaining: 23, TTL: 28}}), 30, 1, alwaysHaven(0))
	m.Update(kindsWith(), 30.25, 1.25, alwaysHaven(0))
	if len(m.Ghosts) != 1 || len(m.Ghosts[0].Boxes) != 4 || !m.Ghosts[0].Boxes[0].Look.Spoiled {
		t.Fatalf("ghosts %+v", m.Ghosts)
	}
	m.Update(kindsWith(), 30.5, 1.25+SpoilLinger+.01, alwaysHaven(0))
	if len(m.Ghosts) != 0 {
		t.Error("a spoil ghost outlives its linger")
	}
	if Spoiled(Load{Count: 0, At: 0, TTL: 1}, 5) || !Spoiled(Load{Count: 2, At: 0, TTL: 1}, 1) || Spoiled(Load{Count: 2, At: 0, TTL: 10}, 5) {
		t.Error("only boxes left past their freshness spoil")
	}
}

func TestManifestStartsAfreshWhenTheRealmIsReloaded(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 3, Age: 1, Remaining: 20, TTL: 28}}), 100, 1, alwaysHaven(0))
	m.Update(kindsWith([]view.CohortView{{Count: 3, Age: 1, Remaining: 20, TTL: 28}}), 3, 2, alwaysHaven(0))
	if len(m.Loads) != 1 || m.Sorted()[0].At != 2 || m.Sorted()[0].Stacks[0].Spot != 0 || len(m.Cooling) != 0 {
		t.Errorf("after a reload: %+v %v", m.Sorted(), m.Cooling)
	}
}

func TestAGrowingCohortAddsNewStacksAndKeepsItsOld(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 3, Age: 1, Remaining: 20, TTL: 28}}), 10, 0, alwaysHaven(0))
	m.Update(kindsWith([]view.CohortView{{Count: 6, Age: 1.2, Remaining: 19.8, TTL: 28}}), 10.2, 0, alwaysHaven(0))
	l := m.Sorted()[0]
	want := []Stack{{0, 2, 0}, {2, 1, 1}, {3, 2, 2}, {5, 1, 3}}
	if l.Total != 6 || l.Count != 6 || len(l.Sizes) != 6 || len(l.Stacks) != len(want) {
		t.Fatalf("grown load %+v", *l)
	}
	for i, st := range l.Stacks {
		if st != want[i] {
			t.Errorf("stack %d = %+v, want %+v", i, st, want[i])
		}
	}
}

func TestSizeFromMixWalksTheShares(t *testing.T) {
	mix := [4]float64{.5, .25, .25, 0}
	cases := map[float64]int{0: 0, .49: 0, .5: 1, .74: 1, .76: 2, .999: 2}
	for u, want := range cases {
		if got := SizeFromMix(mix, u); got != want {
			t.Errorf("size(%v) = %d, want %d", u, got, want)
		}
	}
	if SizeFromMix([4]float64{}, .5) != 0 || SizeFromMix([4]float64{0, 0, 0, 1}, .99) != 3 || SizeFromMix(mix, 1.5) != 3 {
		t.Error("edge mixes")
	}
	sizes := LoadSizes(7, [4]float64{.5, .5}, 200)
	small := 0
	for _, s := range sizes {
		if s == 0 {
			small++
		}
	}
	if small < 70 || small > 130 {
		t.Errorf("an even mix gives %d small boxes of 200", small)
	}
}

func TestHandsServeCrownedParcelsFirstThenTheOldest(t *testing.T) {
	plain := &Load{Lane: Lane{Kind: 0}, At: 5, Count: 3}
	older := &Load{Lane: Lane{Kind: 0}, At: 2, Count: 1}
	crowned := &Load{Lane: Lane{Kind: 0, Crowned: true}, At: 9, Count: 2}
	ledger := &Load{Lane: Lane{Kind: 1}, At: 0, Count: 4}
	empty := &Load{Lane: Lane{Kind: 0, Crowned: true}, At: 1}
	got, ok := NextToServe([]*Load{plain, older, crowned, ledger, empty}, 0, -1)
	if !ok || got != crowned {
		t.Errorf("next = %+v", got)
	}
	got, _ = NextToServe([]*Load{plain, older, ledger}, 0, -1)
	if got != older {
		t.Errorf("next plain = %+v", got)
	}
	if _, ok := NextToServe([]*Load{ledger}, 0, -1); ok {
		t.Error("a ledger served as a parcel")
	}
	if _, ok := NextToServe([]*Load{{Lane: Lane{Kind: 0}, Count: 1, Haven: 1}}, 0, 0); ok {
		t.Error("a parcel of another harbor")
	}
}

func TestYardSpotsFillTheirOwnQuadrant(t *testing.T) {
	for kind := range FreightKinds {
		seen := map[[2]float64]bool{}
		for c := 0; c < YardColumns; c++ {
			across, depth := YardSpot(kind, c+YardColumns*3)
			lo, d0 := YardOrigin(kind)
			if across <= lo || across >= lo+YardWide || depth <= d0 || depth >= d0+YardDeep || across < YardLeft || across > YardRight+YardWide {
				t.Errorf("kind %d column %d at %v,%v outside its yard", kind, c, across, depth)
			}
			seen[[2]float64{across, depth}] = true
		}
		if len(seen) != YardColumns {
			t.Errorf("kind %d: %d distinct spots", kind, len(seen))
		}
		ca, cd := YardCentre(kind)
		corners := YardCorners(kind)
		if math.Abs(ca-(corners[0][0]+corners[2][0])/2) > 1e-9 || math.Abs(cd-(corners[0][1]+corners[2][1])/2) > 1e-9 {
			t.Errorf("kind %d centre %v,%v corners %v", kind, ca, cd, corners)
		}
	}
	a, _ := YardSpot(0, 0)
	b, _ := YardSpot(0, 1)
	if math.Abs(math.Abs(a-b)-YardStep) > 1e-9 {
		t.Errorf("neighbouring columns %v %v", a, b)
	}
}

func TestPilesFillFromTheFrontSoTheirSealsShow(t *testing.T) {
	for _, out := range []view.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}} {
		h := view.Haven{Quay: view.Point{X: 20, Y: 20}, Out: out, Open: true}
		p := HavenPlan{Quay: VecOf(h.Quay), Out: VecOf(h.Out), Side: VecOf(h.Along())}
		p.Piles = PileSpots(p)
		for kind, spots := range p.Piles {
			if len(spots) != YardColumns {
				t.Fatalf("kind %d: %d spots", kind, len(spots))
			}
			for i := 1; i < len(spots); i++ {
				a, b := spots[i-1], spots[i]
				if a.X < b.X-1e-6 || (math.Abs(a.X-b.X) < 1e-6 && a.Y < b.Y) {
					t.Errorf("out %v kind %d: spot %d %v fills before %v", out, kind, i, b, a)
				}
			}
			if StackSpot(p, kind, YardColumns+2) != spots[2] || StackSpot(p, kind, -1) != spots[YardColumns-1] {
				t.Errorf("stack spots wrap around the yard")
			}
		}
		if got := StackSpot(HavenPlan{Quay: Vec{1, 1}, Out: Vec{-1, 0}, Side: Vec{0, 1}}, 9, 0); got == (Vec{}) {
			t.Error("a plan without piles still places boxes")
		}
	}
}

func TestYardsOverflowIntoAHeap(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 30, Age: 2, Remaining: 20, TTL: 28}, {Count: 6, Age: 1, Remaining: 20, TTL: 28}}, nil,
		[]view.CohortView{{Count: 4, Age: 1, Remaining: 20, TTL: 28}}), 10, 0, func(_ float64, key uint32) int { return 0 })
	loads := m.Sorted()
	loads[2].Haven = 1
	yards := Yards(loads)
	if len(yards) != 2 || yards[0].Kind != 0 || yards[1].Haven != 1 {
		t.Fatalf("yards %+v", yards)
	}
	if n := len(yards[0].Boxes); n != YardColumns*StackHeight || yards[0].Overflow != 36-n {
		t.Errorf("shown %d overflow %d", n, yards[0].Overflow)
	}
	if yards[1].Overflow != 0 || len(yards[1].Boxes) != 4 {
		t.Errorf("small yard %+v", yards[1])
	}
	if KindName(-1) != "parcel" || KindName(9) != "parcel" || KindName(3) != "dispatch" {
		t.Error("kind names")
	}
}

func TestFreightClockRunsSmoothlyBetweenPolls(t *testing.T) {
	var c FreightClock
	if got := c.Step(10, 0); got != 10 {
		t.Fatalf("first %v", got)
	}
	c.Step(10.25, .25)
	prev := 0.0
	for clock := .25; clock < .5; clock += 1.0 / 60 {
		got := c.Step(10.25, clock)
		if got < prev || got > 10.25+ClockAhead+1e-9 {
			t.Fatalf("at %v: %v after %v", clock, got, prev)
		}
		prev = got
	}
	if got := c.Step(10.5, .5); math.Abs(got-prev) > .1 {
		t.Errorf("a poll jumps the clock: %v -> %v", prev, got)
	}
	if got := c.Step(2, .6); got != 2 {
		t.Errorf("a reload resets: %v", got)
	}
	paused := c.Step(2, 5)
	if again := c.Step(2, 9); again != paused {
		t.Errorf("a paused realm drifts: %v -> %v", paused, again)
	}
}
