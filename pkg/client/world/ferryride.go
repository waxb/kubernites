package world

import (
	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	rideEps    = 1e-3
	queueStep  = .24
	queueSide  = .14
	queueSlots = 3
	petBehind  = .12
)

type boarding struct {
	Hop   ferryHop `json:"-"`
	Sail  Sailing  `json:"-"`
	clock float64
	wait  Vec
	head  Vec
	out   Vec
	face  string
	carry bool
}

func (b *boarding) besideWait() Vec {
	along := dot(Vec{b.wait.X - b.head.X, b.wait.Y - b.head.Y}, b.out)
	axis := offset(b.head, b.out, along)
	side := Vec{b.wait.X - axis.X, b.wait.Y - axis.Y}
	return offset(Vec{axis.X - side.X, axis.Y - side.Y}, b.out, -petBehind)
}

func (b *boarding) Aboard(t float64) bool { return b != nil && t >= b.Sail.Board }

type ferried interface {
	Crossing(from, to float64) (float64, routeLeg, bool)
}

func queueSpot(j Jetty, key string) Vec {
	k := int(model.Hash(key) % queueSlots)
	side := Vec{-j.Out.Y, j.Out.X}
	return offset(offset(j.Head, j.Out, -float64(k)*queueStep), side, queueSide*float64(k%2*2-1))
}

func (r *Renderer) boardFerry(sc Scene, key string, st *stride, c cycle, prev float64) bool {
	fc, ok := c.(ferried)
	if !ok {
		return false
	}
	at, leg, found := fc.Crossing(prev, st.clock)
	h := r.HarborOf()
	if !found || leg.Ferry.Lane >= len(h.Lanes) || h.Lanes[leg.Ferry.Lane].Isle != leg.Ferry.isle {
		return false
	}
	lane := h.Lanes[leg.Ferry.Lane]
	dep, _ := lane.ends(leg.Ferry.FromHome)
	wait := queueSpot(dep, key)
	st.clock = at
	st.ride = &boarding{Hop: *leg.Ferry, Sail: lane.NextSailing(leg.Ferry.FromHome, sc.Time), clock: at, wait: wait, head: dep.Head, out: dep.Out,
		face: Facing(wait, dep.Dock), carry: leg.carry}
	return true
}

func (r *Renderer) stillRiding(sc Scene, st *stride) bool {
	b, h := st.ride, r.HarborOf()
	if b.Hop.Lane < len(h.Lanes) && h.Lanes[b.Hop.Lane].Isle == b.Hop.isle && sc.Time < b.Sail.Land {
		return true
	}
	st.clock, st.ride = b.clock+rideEps, nil
	return false
}

func (r *Renderer) riderFigure(sc Scene, p model.PodView, st *stride) Figure {
	b := st.ride
	pose := Pose{Pos: b.wait, Action: "idle", Dir: b.face, Frame: Frame(sc.Time, 2), Carry: b.carry}
	if b.Aboard(sc.Time) {
		boat := r.HarborOf().Lanes[b.Hop.Lane].Boat(b.Sail.Boat, sc.Time)
		pose.Pos = boat.Pos
	}
	return Figure{Pod: p, role: p.Sprite, Mood: MoodSteady, clock: st.clock, Pose: pose, Ride: b}
}
