package sound

import "math"

const sineSize = 4096

var sineTable = buildSineTable()

func buildSineTable() []float64 {
	table := make([]float64, sineSize+1)
	for i := range table {
		table[i] = math.Sin(2 * math.Pi * float64(i) / sineSize)
	}
	return table
}

func sine(phase float64) float64 {
	p := phase - math.Floor(phase)
	x := p * sineSize
	i := int(x)
	f := x - float64(i)
	if i >= sineSize {
		i, f = 0, 0
	}
	return sineTable[i] + (sineTable[i+1]-sineTable[i])*f
}

type rng struct {
	state uint32
}

func newRNG(seed uint32) *rng {
	if seed == 0 {
		seed = 0x9e3779b9
	}
	return &rng{state: seed}
}

func (r *rng) next() uint32 {
	x := r.state
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	r.state = x
	return x
}

func (r *rng) signed() float64 {
	return float64(int32(r.next())) / math.MaxInt32
}

func (r *rng) unit() float64 {
	return float64(r.next()>>8) / (1 << 24)
}

type onePole struct {
	a, y float64
}

func newLowpass(cutoff float64, rate int) onePole {
	c := math.Min(cutoff, 0.45*float64(rate))
	return onePole{a: 1 - math.Exp(-2*math.Pi*c/float64(rate))}
}

func (f *onePole) low(x float64) float64 {
	f.y += f.a * (x - f.y)
	return f.y
}

func (f *onePole) high(x float64) float64 {
	return x - f.low(x)
}

func (f *onePole) retune(cutoff float64, rate int) {
	c := math.Min(cutoff, 0.45*float64(rate))
	f.a = 1 - math.Exp(-2*math.Pi*c/float64(rate))
}

type bandpass struct {
	hp, lp1, lp2 onePole
}

func newBandpass(lo, hi float64, rate int) bandpass {
	return bandpass{hp: newLowpass(lo, rate), lp1: newLowpass(hi, rate), lp2: newLowpass(hi, rate)}
}

func (b *bandpass) run(x float64) float64 {
	return b.lp2.low(b.lp1.low(b.hp.high(x)))
}

func polyBLEP(t, dt float64) float64 {
	if t < dt {
		t /= dt
		return t + t - t*t - 1
	}
	if t > 1-dt {
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

func sawBLEP(phase, dt float64) float64 {
	return 2*phase - 1 - polyBLEP(phase, dt)
}

func pulseBLEP(phase, dt, duty float64) float64 {
	v := -1.0
	if phase < duty {
		v = 1
	}
	shifted := phase + 1 - duty
	if shifted >= 1 {
		shifted--
	}
	return v + polyBLEP(phase, dt) - polyBLEP(shifted, dt)
}

func triangle(phase float64) float64 {
	return 1 - 4*math.Abs(phase-0.5)
}

func advance(phase, inc float64) float64 {
	phase += inc
	if phase >= 1 {
		phase -= math.Floor(phase)
	}
	return phase
}

func frames(seconds float64, rate int) int {
	if seconds <= 0 {
		return 0
	}
	return int(math.Round(seconds * float64(rate)))
}

func decayPerSample(t60 float64, rate int) float64 {
	if t60 <= 0 {
		return 0
	}
	return math.Pow(0.001, 1/(t60*float64(rate)))
}

func envelope(i, attack, hold, release int) float64 {
	gain := 1.0
	if i < attack {
		gain = float64(i) / float64(attack)
	}
	if i >= hold {
		if i >= hold+release {
			return 0
		}
		gain *= float64(hold+release-i) / float64(release)
	}
	return gain
}

func midiFrequency(note int) float64 {
	return 440 * math.Pow(2, float64(note-69)/12)
}

func centsRatio(cents float64) float64 {
	return math.Pow(2, cents/1200)
}

func panGains(pan float64) (float64, float64) {
	p := math.Max(-1, math.Min(1, pan))
	angle := (p + 1) * math.Pi / 4
	return math.Cos(angle), math.Sin(angle)
}
