package world

import (
	"math"
)

const (
	laborCycle   = 36.0
	laborSpeed   = 1.0
	laborStand   = .42
	laborWorkFPS = 5.0
	laborWalkFPS = 7.0
	RoleCutter   = "woodcutter"
	RoleMason    = "mason"
	ScarNone     = 0
	ScarFresh    = 1
	ScarMending  = 2
)

type trade struct {
	role    string
	work    float64
	fresh   float64
	mending float64
	salt    uint32
}

var (
	cutterTrade = trade{role: RoleCutter, work: 6, fresh: 150, mending: 90, salt: 0x5bd1e995}
	masonTrade  = trade{role: RoleMason, work: 8, fresh: 200, mending: 160, salt: 0x27d4eb2f}
)

type job struct {
	start  float64
	target int
}

type scar struct {
	tile  int
	at    float64
	stage int
}

func laborHash(a, b, c uint32) uint32 {
	return mix32(a*0x9e3779b1 ^ mix32(b+0x85ebca6b) ^ mix32(c*0xc2b2ae35+1))
}

func (tr trade) phase(seed uint32, slot int) float64 {
	return float64(laborHash(seed, tr.salt, uint32(slot))%1000) / 1000 * laborCycle
}

func (tr trade) jobIndex(t float64, seed uint32, slot int) int {
	return int(math.Floor((t - tr.phase(seed, slot)) / laborCycle))
}

func (tr trade) jobAt(seed uint32, slot, j, n int) job {
	start := float64(j)*laborCycle + tr.phase(seed, slot)
	if n <= 0 {
		return job{start: start, target: -1}
	}
	return job{start: start, target: int(laborHash(seed^tr.salt, uint32(slot), uint32(j)) % uint32(n))}
}

func (tr trade) walk(door, at Vec) float64 {
	return Dist(door, standBy(door, at)) / laborSpeed
}

func (tr trade) doneAt(jb job, door, at Vec) float64 {
	return jb.start + tr.walk(door, at) + tr.work
}

func standBy(door, at Vec) Vec {
	d := max(Dist(door, at), 1e-9)
	return Vec{at.X + (door.X-at.X)/d*laborStand, at.Y + (door.Y-at.Y)/d*laborStand}
}

func (tr trade) pose(t float64, jb job, door, at Vec) (Pose, bool) {
	e := t - jb.start
	w := tr.walk(door, at)
	spot := standBy(door, at)
	switch {
	case e < 0 || e >= 2*w+tr.work:
		return Pose{}, false
	case e < w:
		return Pose{Pos: Lerp(door, spot, e/max(w, 1e-9)), Action: "walk", Dir: Facing(door, spot), Frame: Frame(t, laborWalkFPS)}, true
	case e < w+tr.work:
		return Pose{Pos: spot, Action: "work", Dir: Facing(spot, at), Frame: Frame(t, laborWorkFPS)}, true
	}
	back := (e - w - tr.work) / max(w, 1e-9)
	return Pose{Pos: Lerp(spot, door, back), Action: "carry", Dir: Facing(spot, door), Frame: Frame(t, laborWalkFPS)}, true
}

func (tr trade) stage(t, at float64) int {
	age := t - at
	switch {
	case age < 0:
		return ScarNone
	case age < tr.fresh:
		return ScarFresh
	case age < tr.fresh+tr.mending:
		return ScarMending
	}
	return ScarNone
}

func (tr trade) window() float64 {
	return tr.fresh + tr.mending + 2*laborCycle
}
