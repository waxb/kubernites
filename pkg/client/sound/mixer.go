package sound

import (
	"math"
	"time"
)

const (
	crossfade           = 1500 * time.Millisecond
	sfxThrottle         = 60 * time.Millisecond
	maxVoices           = 8
	defaultMasterVolume = 1.0
	defaultMusicVolume  = 0.6
	defaultSFXVolume    = 0.8
)

type fade struct {
	from, to float64
	start    time.Time
	dur      time.Duration
}

func newFade(from, to float64, start time.Time) fade {
	span := math.Abs(to - from)
	return fade{from: from, to: to, start: start, dur: time.Duration(span * float64(crossfade))}
}

func (f fade) at(now time.Time) float64 {
	return fadeGain(f.from, f.to, now.Sub(f.start), f.dur)
}

func (f fade) finished(now time.Time) bool {
	return now.Sub(f.start) >= f.dur
}

func fadeGain(from, to float64, elapsed, dur time.Duration) float64 {
	if dur <= 0 || elapsed >= dur {
		return to
	}
	if elapsed <= 0 {
		return from
	}
	x := float64(elapsed) / float64(dur)
	return from + (to-from)*fadeCurve(x, to > from)
}

func fadeCurve(x float64, rising bool) float64 {
	if rising {
		return math.Sin(x * math.Pi / 2)
	}
	return 1 - math.Cos(x*math.Pi/2)
}

func clampVolume(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	return math.Min(v, 1)
}

func throttled(last time.Time, seen bool, now time.Time) bool {
	return seen && now.Sub(last) < sfxThrottle
}

type slot struct {
	priority int
	started  time.Time
}

func pickVictim(slots []slot, incoming int) int {
	victim := -1
	for i, s := range slots {
		if s.priority > incoming {
			continue
		}
		if victim < 0 || s.priority < slots[victim].priority ||
			(s.priority == slots[victim].priority && s.started.Before(slots[victim].started)) {
			victim = i
		}
	}
	return victim
}
