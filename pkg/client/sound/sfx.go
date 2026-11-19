package sound

import (
	"fmt"
	"math"
)

const sfxTailFade = 0.03

type sfxSpec struct {
	length   float64
	level    float64
	room     float64
	priority int
	seed     uint32
	draw     func(c *canvas)
}

type canvas struct {
	rate int
	l, r []float32
	rng  *rng
}

type glide struct {
	from, to, dur, t60 float64
}

type hiss struct {
	dur, lo, hi, attack, t60 float64
}

type strike struct {
	freq, t60, dur float64
}

func renderSFX(id SFX, rate int) ([]byte, error) {
	spec, ok := sfxSpecs[id]
	if !ok {
		return nil, fmt.Errorf("sound: unknown effect %q", id)
	}
	n := frames(spec.length, rate)
	c := &canvas{rate: rate, l: make([]float32, n), r: make([]float32, n), rng: newRNG(spec.seed)}
	spec.draw(c)
	if spec.room > 0 {
		c.reverberate(spec.room)
	}
	highpass(c.l, masterHP, rate, 0)
	highpass(c.r, masterHP, rate, 0)
	fade := frames(sfxTailFade, rate)
	edgeFade(c.l, 0, fade)
	edgeFade(c.r, 0, fade)
	return masterPCM(c.l, c.r, spec.level), nil
}

func (c *canvas) at(t float64) int {
	return frames(t, c.rate)
}

func (c *canvas) put(t float64, src []float32, amp, pan float64) {
	gl, gr := panGains(pan)
	fl, fr := float32(amp*gl), float32(amp*gr)
	start := c.at(t)
	for i, v := range src {
		j := start + i
		if j >= len(c.l) {
			return
		}
		c.l[j] += v * fl
		c.r[j] += v * fr
	}
}

func (c *canvas) note(in instrument, t, dur, freq, amp, pan float64) {
	c.put(t, in.render(nil, freq, c.at(dur), c.rate, c.rng), amp, pan)
}

func (c *canvas) hit(kind byte, t, amp, pan float64) {
	c.put(t, renderHit(kind, c.rate, c.rng), amp, pan)
}

func (c *canvas) bell(t float64, s strike, amp, pan float64, parts []partial) {
	c.put(t, renderPartials(nil, s.freq, parts, s.t60, c.at(s.dur), c.rate), amp, pan)
}

func (c *canvas) tone(t float64, g glide, amp, pan float64) {
	n := c.at(g.dur)
	out := make([]float32, n)
	ratio := math.Pow(g.to/g.from, 1/float64(max(n, 1)))
	decay := decayPerSample(g.t60, c.rate)
	attack := c.at(0.002)
	f, a, ph := g.from, 1.0, 0.0
	for i := range out {
		ph = advance(ph, f/float64(c.rate))
		out[i] = float32(sine(ph) * a * envelope(i, attack, n, 1))
		f, a = f*ratio, a*decay
	}
	c.put(t, edgeFade(out, 0, c.at(0.004)), amp, pan)
}

func (c *canvas) noise(t float64, h hiss, amp, pan float64) {
	c.put(t, c.noiseBurst(h, 0), amp, pan)
}

func (c *canvas) scratch(t float64, h hiss, amp, pan float64) {
	c.put(t, c.noiseBurst(h, c.at(0.003)), amp, pan)
}

func (c *canvas) noiseBurst(h hiss, grain int) []float32 {
	n := c.at(h.dur)
	out := make([]float32, n)
	band := newBandpass(h.lo, h.hi, c.rate)
	decay := decayPerSample(h.t60, c.rate)
	attack := c.at(h.attack)
	a, gate := 1.0, 1.0
	for i := range out {
		if grain > 0 && i%grain == 0 {
			gate = 0.25 + 0.75*c.rng.unit()
		}
		out[i] = float32(band.run(c.rng.signed()) * a * gate * envelope(i, attack, n, 1))
		if i >= attack {
			a *= decay
		}
	}
	return edgeFade(out, 0, c.at(0.004))
}

func (c *canvas) reverberate(amount float64) {
	mono := make([]float32, len(c.l))
	for i := range mono {
		mono[i] = 0.5 * (c.l[i] + c.r[i])
	}
	addReverb(mono, c.l, c.r, amount, c.rate, false)
}

func priorityOf(id SFX) int {
	return sfxSpecs[id].priority
}
