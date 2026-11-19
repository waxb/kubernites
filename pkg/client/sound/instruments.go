package sound

import "math"

const pluckAttack = 0.0015

type instKind uint8

const (
	kindPluck instKind = iota
	kindRecorder
	kindPad
	kindBass
	kindPulse
	kindHorn
	kindBell
)

type instrument struct {
	kind    instKind
	attack  float64
	release float64
	decay   float64
	tone    float64
	duty    float64
	detune  float64
	vibrato float64
	damp    float64
	body    float64
}

type partial struct {
	ratio, amp, decay float64
}

var (
	instLute     = instrument{kind: kindPluck, attack: 0.003, release: 0.08, decay: 1.5, tone: 0.25, damp: 0.6, body: 0.08}
	instHarp     = instrument{kind: kindPluck, release: 0.3, decay: 3.2, tone: 0.6}
	instRecorder = instrument{kind: kindRecorder, attack: 0.035, release: 0.09, vibrato: 0.0045}
	instPad      = instrument{kind: kindPad, attack: 0.35, release: 0.6, tone: 3.2, detune: 7}
	instDrone    = instrument{kind: kindPad, attack: 0.7, release: 0.9, tone: 2.4, detune: 4}
	instBass     = instrument{kind: kindBass, attack: 0.004, release: 0.06, decay: 1.4, tone: 1400}
	instChip     = instrument{kind: kindPulse, attack: 0.003, release: 0.035, decay: 1.8, duty: 0.5, tone: 3400}
	instReed     = instrument{kind: kindPulse, attack: 0.006, release: 0.05, decay: 2.5, duty: 0.25, tone: 2200, vibrato: 0.003}
	instHorn     = instrument{kind: kindHorn, attack: 0.07, release: 0.14, tone: 4, vibrato: 0.0035}
	instBrass    = instrument{kind: kindHorn, attack: 0.012, release: 0.06, tone: 3.5}
	instChime    = instrument{kind: kindBell, release: 0.05, decay: 1.6}
	instToll     = instrument{kind: kindBell, release: 0.1, decay: 2.4}
)

var chimePartials = []partial{
	{1, 1, 1}, {2.0, 0.38, 0.55}, {3.01, 0.16, 0.35}, {4.17, 0.1, 0.22}, {5.43, 0.05, 0.14},
}

func (in instrument) render(buf []float32, freq float64, hold, rate int, r *rng) []float32 {
	switch in.kind {
	case kindPluck:
		return renderPluck(in, buf, freq, hold, rate, r)
	case kindRecorder:
		return renderRecorder(in, buf, freq, hold, rate, r)
	case kindPad:
		return renderPad(in, buf, freq, hold, rate)
	case kindBass:
		return renderBass(in, buf, freq, hold, rate)
	case kindPulse:
		return renderPulse(in, buf, freq, hold, rate)
	case kindHorn:
		return renderHorn(in, buf, freq, hold, rate)
	default:
		return renderPartials(buf, freq, chimePartials, in.decay, frames(in.decay, rate), rate)
	}
}

func sized(buf []float32, n int) []float32 {
	if cap(buf) < n {
		return make([]float32, n)
	}
	out := buf[:n]
	clear(out)
	return out
}

func renderPluck(in instrument, buf []float32, freq float64, hold, rate int, r *rng) []float32 {
	attack, release := frames(max(in.attack, pluckAttack), rate), frames(in.release, rate)
	out := sized(buf, hold+release)
	period := float64(rate) / freq
	n := max(int(period+0.4), 2)
	frac := period + 0.5 + 0.5*in.damp - float64(n)
	coef := (1 - frac) / (1 + frac)
	line := exciteString(n, in.tone, rate, r)
	loss := math.Pow(0.001, 1/(in.decay*freq))
	far := 0.25 * in.damp
	var apIn, apOut float64
	idx := 0
	for i := range out {
		next := idx + 1
		if next == n {
			next = 0
		}
		cur := line[idx]
		avg := (0.5*(cur+line[next]) + far*(line[(next+1)%n]-cur)) * loss
		y := coef*avg + apIn - coef*apOut
		apIn, apOut = avg, y
		line[idx] = y
		out[i] = float32(cur * envelope(i, attack, hold, release))
		idx = next
	}
	return addBody(out, in, freq, attack, hold, release, rate)
}

func addBody(out []float32, in instrument, freq float64, attack, hold, release, rate int) []float32 {
	if in.body <= 0 {
		return out
	}
	inc, fade, level := freq/float64(rate), decayPerSample(in.decay, rate), in.body
	ph := 0.0
	for i := range out {
		out[i] += float32(level * sine(ph) * envelope(i, attack, hold, release))
		ph = advance(ph, inc)
		level *= fade
	}
	return out
}

func exciteString(n int, tone float64, rate int, r *rng) []float64 {
	line := make([]float64, n)
	lp := newLowpass(600+tone*7000, rate)
	sum := 0.0
	for i := range line {
		line[i] = lp.low(r.signed())
		sum += line[i]
	}
	mean := sum / float64(n)
	peak := 0.0
	for i := range line {
		line[i] -= mean
		peak = math.Max(peak, math.Abs(line[i]))
	}
	if peak == 0 {
		return line
	}
	for i := range line {
		line[i] /= peak
	}
	return line
}

func renderRecorder(in instrument, buf []float32, freq float64, hold, rate int, r *rng) []float32 {
	attack, release := frames(in.attack, rate), frames(in.release, rate)
	out := sized(buf, hold+release)
	breath := newBandpass(900, 3200, rate)
	chiff, chiffDecay := 1.0, decayPerSample(0.03, rate)
	vibDelay := float64(max(frames(0.25, rate), 1))
	inc, vinc := freq/float64(rate), 5.1/float64(rate)
	var ph, vph float64
	for i := range out {
		depth := in.vibrato * math.Min(1, float64(i)/vibDelay)
		ph = advance(ph, inc*(1+depth*sine(vph)))
		vph = advance(vph, vinc)
		tone := sine(ph) + 0.16*sine(2*ph) + 0.05*sine(3*ph)
		air := breath.run(r.signed()) * (0.08 + 0.5*chiff)
		chiff *= chiffDecay
		out[i] = float32((tone + air) * envelope(i, attack, hold, release))
	}
	return out
}

func renderPad(in instrument, buf []float32, freq float64, hold, rate int) []float32 {
	attack, release := frames(in.attack, rate), frames(in.release, rate)
	out := sized(buf, hold+release)
	dt1 := freq * centsRatio(in.detune) / float64(rate)
	dt2 := freq * centsRatio(-in.detune) / float64(rate)
	cutoff := math.Max(700, freq*in.tone)
	lp1, lp2 := newLowpass(cutoff, rate), newLowpass(cutoff, rate)
	p1, p2 := 0.0, 0.37
	for i := range out {
		p1, p2 = advance(p1, dt1), advance(p2, dt2)
		v := 0.5 * (sawBLEP(p1, dt1) + sawBLEP(p2, dt2))
		out[i] = float32(lp2.low(lp1.low(v)) * envelope(i, attack, hold, release))
	}
	return out
}

func renderBass(in instrument, buf []float32, freq float64, hold, rate int) []float32 {
	attack, release := frames(in.attack, rate), frames(in.release, rate)
	out := sized(buf, hold+release)
	lp := newLowpass(in.tone, rate)
	decay, level := decayPerSample(in.decay, rate), 1.0
	inc := freq / float64(rate)
	ph := 0.0
	for i := range out {
		ph = advance(ph, inc)
		v := lp.low(0.8*triangle(ph) + 0.3*sine(ph) + 0.18*sine(2*ph))
		out[i] = float32(v * envelope(i, attack, hold, release) * (0.7 + 0.3*level))
		level *= decay
	}
	return out
}

func renderPulse(in instrument, buf []float32, freq float64, hold, rate int) []float32 {
	attack, release := frames(in.attack, rate), frames(in.release, rate)
	out := sized(buf, hold+release)
	lp := newLowpass(in.tone, rate)
	decay, level := decayPerSample(in.decay, rate), 1.0
	vibDelay := float64(max(frames(0.2, rate), 1))
	inc, vinc := freq/float64(rate), 5.5/float64(rate)
	offset := 1 - 2*in.duty
	var ph, vph float64
	for i := range out {
		depth := in.vibrato * math.Min(1, float64(i)/vibDelay)
		ph = advance(ph, inc*(1+depth*sine(vph)))
		vph = advance(vph, vinc)
		v := lp.low(pulseBLEP(ph, inc, in.duty)+offset) * 0.6
		out[i] = float32(v * envelope(i, attack, hold, release) * (0.75 + 0.25*level))
		level *= decay
	}
	return out
}

func renderHorn(in instrument, buf []float32, freq float64, hold, rate int) []float32 {
	attack, release := frames(in.attack, rate), frames(in.release, rate)
	out := sized(buf, hold+release)
	lp1, lp2 := newLowpass(freq, rate), newLowpass(freq, rate)
	vibDelay := float64(max(frames(0.25, rate), 1))
	inc, vinc := freq/float64(rate), 4.6/float64(rate)
	var ph, vph float64
	for i := range out {
		env := envelope(i, attack, hold, release)
		if i&63 == 0 {
			cutoff := freq * (1.2 + in.tone*env)
			lp1.retune(cutoff, rate)
			lp2.retune(cutoff, rate)
		}
		depth := in.vibrato * math.Min(1, float64(i)/vibDelay)
		ph = advance(ph, inc*(1+depth*sine(vph)))
		vph = advance(vph, vinc)
		out[i] = float32(lp2.low(lp1.low(sawBLEP(ph, inc))) * env)
	}
	return out
}

func renderPartials(buf []float32, freq float64, parts []partial, t60 float64, n, rate int) []float32 {
	out := sized(buf, n)
	nyquist := 0.45 * float64(rate)
	for _, p := range parts {
		f := freq * p.ratio
		if f >= nyquist {
			continue
		}
		inc, d := f/float64(rate), decayPerSample(p.decay*t60, rate)
		a, ph := p.amp, 0.0
		for i := range out {
			out[i] += float32(sine(ph) * a)
			ph = advance(ph, inc)
			a *= d
		}
	}
	return edgeFade(out, frames(0.002, rate), n/12)
}

func edgeFade(buf []float32, in, out int) []float32 {
	n := len(buf)
	for i := 0; i < in && i < n; i++ {
		buf[i] *= float32(i) / float32(in)
	}
	for i := 0; i < out && i < n; i++ {
		buf[n-1-i] *= float32(i) / float32(out)
	}
	return buf
}
