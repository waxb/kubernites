package sound

import (
	"fmt"
	"math"
)

const stepsPerBeat = 4

type part struct {
	inst  instrument
	line  string
	amp   float64
	pan   float64
	gate  float64
	room  float64
	echo  float64
	shift int
}

type beat struct {
	line string
	amp  float64
	pan  float64
	room float64
}

type song struct {
	bpm      int
	steps    int
	level    float64
	room     float64
	echo     int
	feedback float64
	seed     uint32
	parts    []part
	beats    []beat
}

func stepFrames(bpm, rate int) int {
	return int(math.Round(float64(rate) * 60 / float64(bpm*stepsPerBeat)))
}

func renderSong(s song, rate int) ([]byte, error) {
	if s.bpm <= 0 || s.steps <= 0 {
		return nil, fmt.Errorf("song needs a positive tempo and length")
	}
	step := stepFrames(s.bpm, rate)
	bus := newMixBus(step * s.steps)
	r := newRNG(s.seed)
	for i, p := range s.parts {
		if err := renderPart(bus, p, s.steps, step, rate, r); err != nil {
			return nil, fmt.Errorf("part %d: %w", i, err)
		}
	}
	for i, b := range s.beats {
		if err := renderBeat(bus, b, s.steps, step, rate, r); err != nil {
			return nil, fmt.Errorf("beat %d: %w", i, err)
		}
	}
	l, rr := s.finish(bus, step, rate)
	return masterPCM(l, rr, s.level), nil
}

func renderPart(bus mixBus, p part, steps, step, rate int, r *rng) error {
	events, err := noteEvents(p.line, steps, p.shift)
	if err != nil {
		return err
	}
	var scratch []float32
	for _, e := range events {
		hold := max(int(float64(e.length*step)*p.gate), 1)
		scratch = p.inst.render(scratch, midiFrequency(e.note), hold, rate, r)
		bus.place(e.step*step, scratch, p.amp, p.pan, p.room, p.echo)
	}
	return nil
}

func renderBeat(bus mixBus, b beat, steps, step, rate int, r *rng) error {
	hits, err := hitEvents(b.line, steps)
	if err != nil {
		return err
	}
	for _, h := range hits {
		bus.place(h.step*step, renderHit(h.kind, rate, r), b.amp, b.pan, b.room, 0)
	}
	return nil
}

func (s song) finish(bus mixBus, step, rate int) ([]float32, []float32) {
	l, r := bus.l, bus.r
	if s.room > 0 {
		addReverb(bus.room, l, r, s.room, rate, true)
	}
	if s.echo > 0 {
		addEcho(bus.echo, l, r, s.echo*step, s.feedback, rate)
	}
	prime := frames(masterPrime, rate)
	highpass(l, masterHP, rate, prime)
	highpass(r, masterHP, rate, prime)
	return l, r
}
