package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	ferrySpeed  = .9
	ferryDwell  = 4.0
	FerryBoats  = 2
	minCross    = 3.0
	ferryPoleHz = 2.0
	sideHome    = 0
	sideAway    = 1
	sideAfloat  = -1
	passWidth   = .45
)

type FerryPose struct {
	Pos    Vec
	Dir    string
	Moving bool
	Side   int
	Frame  int
}

type Sailing struct {
	Boat     int
	FromHome bool
	Board    float64
	Depart   float64
	Land     float64
}

func (l FerryLane) Cross() float64 { return max(l.Length/ferrySpeed, minCross) }

func (l FerryLane) Period() float64 { return 2 * (l.Cross() + ferryDwell) }

func (l FerryLane) shift(k int) float64 { return float64(k) * l.Period() / FerryBoats }

func easeInOut(u float64) float64 {
	u = min(max(u, 0), 1)
	return u * u * (3 - 2*u)
}

func (l FerryLane) Boat(k int, t float64) FerryPose {
	cross := l.Cross()
	phase := wrap(t+l.shift(k), l.Period())
	out := Vec{l.Path[len(l.Path)-1].X - l.Path[0].X, l.Path[len(l.Path)-1].Y - l.Path[0].Y}
	switch {
	case phase < ferryDwell:
		return FerryPose{Pos: l.Home.Dock, Dir: ShipHeading(Vec{-out.X, -out.Y}), Side: sideHome}
	case phase < ferryDwell+cross:
		return FerryPose{Pos: l.afloat(l.Path, (phase-ferryDwell)/cross, 1), Dir: ShipHeading(out), Moving: true, Side: sideAfloat, Frame: Frame(t, ferryPoleHz) % art.FerryFrames}
	case phase < 2*ferryDwell+cross:
		return FerryPose{Pos: l.Away.Dock, Dir: ShipHeading(out), Side: sideAway}
	}
	pos := l.afloat(reversed(l.Path), (phase-2*ferryDwell-cross)/cross, -1)
	return FerryPose{Pos: pos, Dir: ShipHeading(Vec{-out.X, -out.Y}), Moving: true, Side: sideAfloat, Frame: Frame(t, ferryPoleHz) % art.FerryFrames}
}

func (l FerryLane) afloat(path []Vec, u, side float64) Vec {
	s := easeInOut(u)
	pos, _ := PathAt(path, s)
	d := Vec{l.Away.Dock.X - l.Home.Dock.X, l.Away.Dock.Y - l.Home.Dock.Y}
	n := max(math.Hypot(d.X, d.Y), 1e-9)
	return offset(pos, Vec{-d.Y / n, d.X / n}, side*passWidth*math.Sin(math.Pi*s))
}

func reversed(path []Vec) []Vec {
	out := make([]Vec, len(path))
	for i, v := range path {
		out[len(path)-1-i] = v
	}
	return out
}

func (l FerryLane) NextSailing(fromHome bool, t float64) Sailing {
	base := ferryDwell
	if !fromHome {
		base = 2*ferryDwell + l.Cross()
	}
	best := Sailing{Depart: math.Inf(1)}
	for k := 0; k < FerryBoats; k++ {
		first := base - l.shift(k)
		depart := first + math.Ceil((t-first)/l.Period()-1e-12)*l.Period()
		if depart < best.Depart {
			best = Sailing{Boat: k, FromHome: fromHome, Depart: depart}
		}
	}
	best.Board = max(t, best.Depart-ferryDwell)
	best.Land = best.Depart + l.Cross()
	return best
}
