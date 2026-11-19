package world

import (
	"math"
	"sort"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	YardCols    = 3
	YardRows    = 4
	YardColumns = YardCols * YardRows
	YardStep    = .34
	YardWide    = YardCols * YardStep
	YardDeep    = YardRows * YardStep
	yardShore   = -.5
	YardLeft    = -.4
	YardRight   = .6
)

type cohortView = view.CohortView

var FreightKinds = []string{"parcel", "ledger", "crate", "dispatch"}

func KindName(kind int) string {
	if kind < 0 || kind >= len(FreightKinds) {
		return FreightKinds[0]
	}
	return FreightKinds[kind]
}

type QuayBox struct {
	Lane   Lane
	Kind   int
	Haven  int
	Index  int
	Column int
	Level  int
	Lift   int
	Reveal int
	At     float64
	Look   gfx.BoxLook
}

func (l Load) freshness() float64 {
	if l.TTL <= 0 {
		return 1
	}
	return min(max(l.Remaining/l.TTL, 0), 1)
}

func (l Load) size(i int) int {
	if i < 0 || i >= len(l.Sizes) {
		return 0
	}
	return int(l.Sizes[i])
}

func (l Load) Boxes() []QuayBox {
	out := make([]QuayBox, 0, l.Count)
	seed := l.key.hash()
	for _, st := range l.Stacks {
		bottom := st.first + st.N - 1
		for i := max(st.first, l.Total-l.Count); i <= bottom; i++ {
			lift := 0
			for b := i + 1; b <= bottom; b++ {
				lift += gfx.BoxLift(l.size(b))
			}
			look := gfx.BoxLook{Size: l.size(i), Kind: KindName(l.Lane.Kind), Variant: int(mix32(seed+uint32(i)*0x165667b1) % gfx.BoxVariants),
				Sealed: l.Lane.Sealed, Priority: l.Lane.Crowned, Fresh: l.freshness()}
			out = append(out, QuayBox{Lane: l.Lane, Kind: l.Lane.Kind, Haven: l.Haven, Index: i, Column: st.Spot,
				Level: bottom - i, Lift: lift, Reveal: l.Total - 1 - i, At: l.At, Look: look})
		}
	}
	return out
}

func YardOrigin(kind int) (float64, float64) {
	across, depth := YardLeft, yardShore
	if kind%2 == 1 {
		across = YardRight
	}
	if kind >= 2 {
		depth += YardDeep
	}
	return across, depth
}

func YardSpot(kind, column int) (float64, float64) {
	c := ((column % YardColumns) + YardColumns) % YardColumns
	a0, d0 := YardOrigin(kind)
	return a0 + (float64(c%YardCols)+.5)*YardStep, d0 + (float64(c/YardCols)+.5)*YardStep
}

func PileSpots(p HavenPlan) [][]Vec {
	out := make([][]Vec, len(FreightKinds))
	for kind := range out {
		spots := make([]Vec, 0, YardColumns)
		for c := 0; c < YardColumns; c++ {
			across, depth := YardSpot(kind, c)
			spots = append(spots, QuayPoint(p, across, depth))
		}
		sort.SliceStable(spots, func(i, j int) bool {
			if math.Abs(spots[i].X-spots[j].X) > 1e-6 {
				return spots[i].X > spots[j].X
			}
			return spots[i].Y > spots[j].Y
		})
		out[kind] = spots
	}
	return out
}

func StackSpot(p HavenPlan, kind, column int) Vec {
	if kind < 0 || kind >= len(p.Piles) || len(p.Piles[kind]) == 0 {
		across, depth := YardSpot(kind, column)
		return QuayPoint(p, across, depth)
	}
	spots := p.Piles[kind]
	return spots[((column%len(spots))+len(spots))%len(spots)]
}

func YardCentre(kind int) (float64, float64) {
	a0, d0 := YardOrigin(kind)
	return a0 + YardWide/2, d0 + YardDeep/2
}

func QuayPoint(p HavenPlan, across, depth float64) Vec {
	return offset(offset(p.Quay, p.Side, across), p.Out, -depth)
}

type Yard struct {
	Haven    int
	Kind     int
	Boxes    []QuayBox
	Overflow int
}

func Yards(loads []*Load) []Yard {
	byKey := map[yardKey]*Yard{}
	order := []yardKey{}
	for _, l := range loads {
		yk := l.yard()
		y, ok := byKey[yk]
		if !ok {
			y = &Yard{Haven: l.Haven, Kind: l.Lane.Kind}
			byKey[yk] = y
			order = append(order, yk)
		}
		for _, b := range l.Boxes() {
			if b.Column == offYard {
				y.Overflow++
				continue
			}
			y.Boxes = append(y.Boxes, b)
		}
	}
	out := make([]Yard, 0, len(order))
	for _, yk := range order {
		out = append(out, *byKey[yk])
	}
	return out
}

func HeapPoint(p HavenPlan, kind int) Vec {
	across, depth := YardCentre(kind)
	return QuayPoint(p, across, depth+heapBack)
}
