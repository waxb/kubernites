package world

import (
	"math"
)

const (
	WalkSpeed = 1.4
	WorkPause = 2.0
	gatePause = .4
	BankPause = .6
	HomePause = 1.5
)

type Pose struct {
	Pos    Vec
	Action string
	Dir    string
	Frame  int
	Carry  bool
}

type leg struct {
	from, to Vec
	pause    float64
	action   string
	carry    bool
}

func Dist(a, b Vec) float64 { return math.Hypot(b.X-a.X, b.Y-a.Y) }

func Lerp(a, b Vec, t float64) Vec {
	t = min(max(t, 0), 1)
	return Vec{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
}

func Facing(from, to Vec) string {
	if (to.X-from.X)-(to.Y-from.Y) >= 0 {
		return "se"
	}
	return "sw"
}

func Frame(t float64, fps float64) int {
	return int(math.Floor(t*fps)) % 4
}

func seedOffset(seed uint32) float64 {
	return float64(seed%1000) / 37
}

func loopLegs(home, work, bank Vec) ([3]leg, float64) {
	legs := [3]leg{
		{from: home, to: work, pause: WorkPause, action: "work"},
		{from: work, to: bank, pause: BankPause, action: "idle", carry: true},
		{from: bank, to: home, pause: HomePause, action: "idle"},
	}
	total := 0.0
	for _, l := range legs {
		total += Dist(l.from, l.to)/WalkSpeed + l.pause
	}
	return legs, max(total, .001)
}

func LoopLength(home, work, bank Vec) float64 {
	_, total := loopLegs(home, work, bank)
	return total
}

func DeliveryAge(t float64, seed uint32, home, work, bank Vec) float64 {
	return DeliveryAgeAt(t+seedOffset(seed), home, work, bank)
}

func DeliveryAgeAt(clock float64, home, work, bank Vec) float64 {
	legs, total := loopLegs(home, work, bank)
	arrival := Dist(legs[0].from, legs[0].to)/WalkSpeed + legs[0].pause + Dist(legs[1].from, legs[1].to)/WalkSpeed
	return wrap(wrap(clock, total)-arrival, total)
}

func wrap(v, span float64) float64 {
	return math.Mod(math.Mod(v, span)+span, span)
}

func WalkerPose(t float64, seed uint32, home, work, bank Vec, serving bool) Pose {
	if !serving {
		return IdlePose(t, seed, home)
	}
	return CyclePose(t+seedOffset(seed), home, work, bank)
}

func CyclePose(clock float64, home, work, bank Vec) Pose {
	legs, total := loopLegs(home, work, bank)
	phase := wrap(clock, total)
	for _, l := range legs[:len(legs)-1] {
		if pose, ok := legPose(l, phase, clock); ok {
			return pose
		}
		phase -= Dist(l.from, l.to)/WalkSpeed + l.pause
	}
	pose, _ := legPose(legs[len(legs)-1], phase, clock)
	return pose
}

func legPose(l leg, phase, clock float64) (Pose, bool) {
	travel := Dist(l.from, l.to) / WalkSpeed
	if phase < travel {
		return Pose{Pos: Lerp(l.from, l.to, phase/max(travel, .001)), Action: carryAction(l.carry), Dir: Facing(l.from, l.to), Frame: Frame(clock, 6), Carry: l.carry}, true
	}
	return Pose{Pos: l.to, Action: l.action, Dir: Facing(l.from, l.to), Frame: Frame(clock, 4)}, phase < travel+l.pause
}

func carryAction(carry bool) string {
	if carry {
		return "carry"
	}
	return "walk"
}

func IdlePose(t float64, seed uint32, home Vec) Pose {
	local := t + seedOffset(seed)
	if int(local/3)%3 != 0 {
		return Pose{Pos: Vec{home.X + .15, home.Y + .35}, Action: "idle", Dir: "se", Frame: Frame(local, 2)}
	}
	angle := local * .9
	pos := Vec{home.X + .15 + .35*math.Cos(angle), home.Y + .35 + .35*math.Sin(angle)}
	ahead := Vec{home.X + .35*math.Cos(angle+.1), home.Y + .35*math.Sin(angle+.1)}
	return Pose{Pos: pos, Action: "walk", Dir: Facing(pos, ahead), Frame: Frame(local, 5)}
}

type Gateway struct {
	Door   Vec
	Inward Vec
}

func (g Gateway) Side() Vec {
	return Vec{-g.Inward.Y, g.Inward.X}
}
