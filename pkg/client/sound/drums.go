package sound

type hitShape struct {
	length   float64
	gain     float64
	base     float64
	sweep    float64
	sweepT   float64
	bodyAmp  float64
	bodyT    float64
	noiseAmp float64
	noiseT   float64
	lo, hi   float64
	attack   float64
	jingle   float64
}

var hitShapes = map[byte]hitShape{
	'K': {length: 0.42, gain: 0.9, base: 64, sweep: 80, sweepT: 0.09, bodyAmp: 1, bodyT: 0.42, noiseAmp: 0.5, noiseT: 0.03, lo: 40, hi: 900, attack: 0.002},
	'T': {length: 0.12, gain: 0.55, base: 330, sweep: 60, sweepT: 0.02, bodyAmp: 0.6, bodyT: 0.05, noiseAmp: 1, noiseT: 0.025, lo: 1500, hi: 6000, attack: 0.001},
	'S': {length: 0.26, gain: 0.8, base: 185, sweep: 40, sweepT: 0.03, bodyAmp: 0.45, bodyT: 0.09, noiseAmp: 1.6, noiseT: 0.18, lo: 900, hi: 6000, attack: 0.0015},
	'H': {length: 0.1, gain: 0.4, noiseAmp: 1.4, noiseT: 0.07, lo: 4000, hi: 12000, attack: 0.008},
	'J': {length: 0.3, gain: 0.4, noiseAmp: 1.5, noiseT: 0.22, lo: 5000, hi: 14000, attack: 0.002, jingle: 31},
	'B': {length: 0.6, gain: 0.8, base: 62, sweep: 60, sweepT: 0.08, bodyAmp: 1, bodyT: 0.6, noiseAmp: 0.4, noiseT: 0.04, lo: 30, hi: 500, attack: 0.003},
}

func renderHit(kind byte, rate int, r *rng) []float32 {
	shape := hitShapes[kind]
	n := frames(shape.length, rate)
	out := make([]float32, n)
	band := newBandpass(shape.lo, shape.hi, rate)
	body, bodyDecay := shape.bodyAmp, decayPerSample(shape.bodyT, rate)
	sweep, sweepDecay := 1.0, decayPerSample(shape.sweepT, rate)
	noise, noiseDecay := shape.noiseAmp, decayPerSample(shape.noiseT, rate)
	attack := frames(shape.attack, rate)
	var ph, jph float64
	for i := range out {
		v := band.run(r.signed()) * noise
		if shape.jingle > 0 {
			v *= 0.65 + 0.35*sine(jph)
			jph = advance(jph, shape.jingle/float64(rate))
		}
		if shape.bodyAmp > 0 {
			ph = advance(ph, (shape.base+shape.sweep*sweep)/float64(rate))
			v += sine(ph) * body
		}
		out[i] = float32(v * shape.gain * envelope(i, attack, n, 1))
		body, sweep, noise = body*bodyDecay, sweep*sweepDecay, noise*noiseDecay
	}
	return edgeFade(out, 0, frames(0.01, rate))
}
