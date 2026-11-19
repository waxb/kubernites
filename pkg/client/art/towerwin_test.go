package art

import (
	"math"
	"testing"
)

func TestOctagonFacetsGiveIsoWallSlopes(t *testing.T) {
	cases := map[float64]float64{0: -0.5, math.Pi / 2: 0.5, math.Pi / 4: 0}
	for a, want := range cases {
		if got := wallSlope(a); math.Abs(got-want) > 1e-9 {
			t.Errorf("wallSlope(%.2f) = %.3f, want %.3f", a, got, want)
		}
	}
}

func TestRoundWallSlopeFollowsTheRing(t *testing.T) {
	s := iso{ox: 64, oy: 64}
	tw := towerSpec{cx: 0, cy: 0, r: 1, round: true}
	for _, deg := range []float64{-20, 10, 45, 80, 110} {
		o := s.opening(tw, deg, 0)
		step := 3.0
		a := (deg + step) * math.Pi / 180
		b := (deg - step) * math.Pi / 180
		pa := s.p(0.97*math.Cos(a), 0.97*math.Sin(a), 0)
		pb := s.p(0.97*math.Cos(b), 0.97*math.Sin(b), 0)
		chord := (pa.y - pb.y) / (pa.x - pb.x)
		if math.Abs(chord-o.slope) > 0.05 {
			t.Errorf("deg %.0f: slope %.3f does not follow the ring chord %.3f", deg, o.slope, chord)
		}
	}
}

func TestSpanShearsColumnsAlongTheWall(t *testing.T) {
	sp := span{x: 10, y: 20, slope: -0.5, cx: 12}
	if sp.drop(10) <= sp.drop(13) {
		t.Errorf("a rising wall lifts the right column: drops %d and %d", sp.drop(10), sp.drop(13))
	}
	if upright(3, 4).drop(99) != 0 {
		t.Error("an upright span never shears")
	}
}

func TestOctagonOpeningsSnapToAFacet(t *testing.T) {
	s := iso{ox: 64, oy: 64, lk: look{age: ageEmpire, variant: 1}}
	tw := towerSpec{cx: 0, cy: 0, r: 1}
	if s.shapeOf(tw) != shapeOct {
		t.Skip("variant 1 of the Empire age is octagonal")
	}
	a, b := s.opening(tw, 10, 0), s.opening(tw, -10, 0)
	if a.x != b.x || a.y != b.y || a.slope != -0.5 {
		t.Errorf("openings near a facet sit on its centre with its slope: %+v %+v", a, b)
	}
}
