package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	MoodRest     = "rest"
	MoodSteady   = "steady"
	MoodBrisk    = "brisk"
	MoodStrained = "strained"
	MoodSick     = "sick"

	busyRate    = 0.8
	restRate    = 0.04
	wakeRate    = 0.08
	easyPace    = 0.8
	briskGain   = 0.6
	briskLoad   = 0.6
	strainPace  = 0.55
	strainTau   = 1.2
	strainOn    = 0.5
	maxStep     = 1.5
	strideStale = 5.0
	pruneEvery  = 2.0
)

type Pace struct {
	Mood  string
	Speed float64
}

func PaceOf(rate float64, working, strained bool) Pace {
	switch {
	case !working:
		return Pace{Mood: MoodRest, Speed: easyPace}
	case strained:
		return Pace{Mood: MoodStrained, Speed: strainPace}
	}
	load := min(max(rate/busyRate, 0), 1)
	mood := MoodSteady
	if load >= briskLoad {
		mood = MoodBrisk
	}
	return Pace{Mood: mood, Speed: easyPace + briskGain*load}
}

func Workload(sc Scene, p model.PodView, hasWork bool) (float64, bool, bool) {
	m, measured := sc.Metrics.Pods[p.Namespace+"/"+p.Name]
	switch {
	case !hasWork:
		return 0, false, false
	case !measured:
		return 0, sc.ArrivalRate > 0, false
	}
	return m.ServedRate, m.ServedRate >= restRate, m.Throttled
}

func Wakes(resting bool, rate float64, working bool) bool {
	if resting && rate > 0 && rate < wakeRate {
		return false
	}
	return working
}

type stride struct {
	clock   float64
	at      float64
	strain  float64
	resting bool
	ride    *boarding
}

func newStride(now float64, seed uint32, throttled, rest bool) *stride {
	s := &stride{clock: now + seedOffset(seed), at: now, resting: rest}
	if throttled {
		s.strain = 1
	}
	return s
}

func (s *stride) tick(now float64) float64 {
	dt := min(max(now-s.at, 0), maxStep)
	s.at = now
	return dt
}

func (s *stride) feel(dt float64, throttled bool) {
	target := 0.0
	if throttled {
		target = 1
	}
	s.strain += (target - s.strain) * min(dt/strainTau, 1)
}

func (s *stride) walk(dt, speed, cycle float64, rest bool) {
	if s.resting && rest {
		return
	}
	if s.resting {
		s.resting = false
		s.clock = loopStart(s.clock, cycle)
	}
	next := s.clock + dt*speed
	if rest && cycle > 0 && math.Floor(next/cycle) > math.Floor(s.clock/cycle) {
		next, s.resting = loopStart(next, cycle), true
	}
	s.clock = next
}

func loopStart(clock, cycle float64) float64 {
	if cycle <= 0 {
		return clock
	}
	return math.Floor(clock/cycle) * cycle
}

type town struct {
	strides map[string]*stride
	pruned  float64
	props   propSet
	names   map[NameKey][]string
	figs    []Figure
}

func (r *Renderer) townOf() *town {
	if r.town == nil {
		r.town = &town{strides: map[string]*stride{}}
	}
	return r.town
}

func (t *town) strideFor(key string, now float64, seed uint32, throttled, rest bool) *stride {
	s, ok := t.strides[key]
	if !ok {
		s = newStride(now, seed, throttled, rest)
		t.strides[key] = s
	}
	return s
}

func (t *town) prune(now float64) {
	if now >= t.pruned && now-t.pruned < pruneEvery {
		return
	}
	t.pruned = now
	for key, s := range t.strides {
		if math.Abs(now-s.at) > strideStale {
			delete(t.strides, key)
		}
	}
}
