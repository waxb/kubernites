package sound

import "time"

const (
	DuckDepth   = 0.45
	DuckAttack  = 80 * time.Millisecond
	DuckRelease = 900 * time.Millisecond
)

type duck struct {
	start, until time.Time
}

func cueLength(id SFX) time.Duration {
	return time.Duration(sfxSpecs[id].length * float64(time.Second))
}

func (d duck) gain(now time.Time) float64 {
	switch {
	case d.until.IsZero() || now.Before(d.start):
		return 1
	case now.Before(d.start.Add(DuckAttack)):
		return 1 - (1-DuckDepth)*float64(now.Sub(d.start))/float64(DuckAttack)
	case now.Before(d.until):
		return DuckDepth
	case now.Before(d.until.Add(DuckRelease)):
		return DuckDepth + (1-DuckDepth)*float64(now.Sub(d.until))/float64(DuckRelease)
	}
	return 1
}

func (d duck) under(id SFX, now time.Time) duck {
	until := now.Add(cueLength(id))
	if d.gain(now) < 1 {
		return duck{start: now.Add(-DuckAttack), until: maxTime(d.until, until)}
	}
	return duck{start: now, until: until}
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (m *Manager) duckUnder(id SFX, now time.Time) {
	if Ducks(id) {
		m.duck = m.duck.under(id, now)
	}
}
