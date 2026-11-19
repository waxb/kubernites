package sound

import (
	"encoding/binary"
	"math"
)

const (
	softKnee     = 0.75
	masterHP     = 35.0
	masterPrime  = 0.5
	reverbPrime  = 2.0
	echoRepeats  = 10
	echoDamping  = 3200.0
	reverbDamp   = 0.35
	reverbFB     = 0.78
	reverbSpread = 23
)

var (
	combTuning    = [4]int{1116, 1277, 1422, 1557}
	allpassTuning = [2]int{556, 341}
)

type mixBus struct {
	l, r, room, echo []float32
}

func newMixBus(n int) mixBus {
	return mixBus{l: make([]float32, n), r: make([]float32, n), room: make([]float32, n), echo: make([]float32, n)}
}

func (b mixBus) place(at int, src []float32, amp, pan, room, echo float64) {
	n := len(b.l)
	if n == 0 {
		return
	}
	gl, gr := panGains(pan)
	start := at % n
	for len(src) > 0 {
		span := min(len(src), n-start)
		chunk := src[:span]
		mixInto(b.l[start:start+span], chunk, float32(amp*gl))
		mixInto(b.r[start:start+span], chunk, float32(amp*gr))
		mixInto(b.room[start:start+span], chunk, float32(amp*room))
		mixInto(b.echo[start:start+span], chunk, float32(amp*echo))
		src, start = src[span:], 0
	}
}

func mixInto(dst, src []float32, gain float32) {
	if gain == 0 {
		return
	}
	dst = dst[:len(src)]
	for i, v := range src {
		dst[i] += v * gain
	}
}

type comb struct {
	buf         []float32
	idx         int
	store       float32
	fb, damping float32
}

func (c *comb) tick(x float32) float32 {
	y := c.buf[c.idx]
	c.store = y*(1-c.damping) + c.store*c.damping
	c.buf[c.idx] = x + c.store*c.fb
	c.idx++
	if c.idx == len(c.buf) {
		c.idx = 0
	}
	return y
}

type allpass struct {
	buf []float32
	idx int
}

func (a *allpass) tick(x float32) float32 {
	b := a.buf[a.idx]
	a.buf[a.idx] = x + b*0.5
	a.idx++
	if a.idx == len(a.buf) {
		a.idx = 0
	}
	return b - x
}

type reverb struct {
	combs  [4]comb
	passes [2]allpass
}

func newReverb(rate, spread int) *reverb {
	scale := float64(rate) / 44100
	v := &reverb{}
	for i, n := range combTuning {
		size := max(int(float64(n+spread)*scale), 1)
		v.combs[i] = comb{buf: make([]float32, size), fb: reverbFB, damping: reverbDamp}
	}
	for i, n := range allpassTuning {
		size := max(int(float64(n+spread)*scale), 1)
		v.passes[i] = allpass{buf: make([]float32, size)}
	}
	return v
}

func (v *reverb) tick(x float32) float32 {
	var sum float32
	for i := range v.combs {
		sum += v.combs[i].tick(x)
	}
	out := sum * 0.25
	for i := range v.passes {
		out = v.passes[i].tick(out)
	}
	return out
}

func addReverb(send, l, r []float32, gain float64, rate int, circular bool) {
	n := len(send)
	left, right := newReverb(rate, 0), newReverb(rate, reverbSpread)
	if circular {
		for i := max(0, n-frames(reverbPrime, rate)); i < n; i++ {
			left.tick(send[i])
			right.tick(send[i])
		}
	}
	g := float32(gain)
	for i, x := range send {
		l[i] += left.tick(x) * g
		r[i] += right.tick(x) * g
	}
}

type pingPong struct {
	left, right []float32
	idx         int
	fb          float32
	damp        onePole
}

func (p *pingPong) tick(x float32) (float32, float32) {
	outL, outR := p.left[p.idx], p.right[p.idx]
	p.left[p.idx] = x + p.fb*float32(p.damp.low(float64(outR)))
	p.right[p.idx] = p.fb * outL
	p.idx++
	if p.idx == len(p.left) {
		p.idx = 0
	}
	return outL, outR
}

func addEcho(send, l, r []float32, delay int, fb float64, rate int) {
	n := len(send)
	d := max(delay, 1)
	p := &pingPong{left: make([]float32, d), right: make([]float32, d), fb: float32(fb), damp: newLowpass(echoDamping, rate)}
	for i := max(0, n-echoRepeats*d); i < n; i++ {
		p.tick(send[i])
	}
	for i, x := range send {
		el, er := p.tick(x)
		l[i] += el
		r[i] += er
	}
}

func highpass(buf []float32, cutoff float64, rate, prime int) {
	f := newLowpass(cutoff, rate)
	for i := max(0, len(buf)-prime); i < len(buf); i++ {
		f.low(float64(buf[i]))
	}
	for i, x := range buf {
		buf[i] = float32(f.high(float64(x)))
	}
}

func peakOf(l, r []float32) float64 {
	var peak float32
	for i := range l {
		peak = max(peak, l[i], -l[i], r[i], -r[i])
	}
	return float64(peak)
}

func softClip(x float64) float64 {
	a := math.Abs(x)
	if a <= softKnee {
		return x
	}
	y := softKnee + (1-softKnee)*math.Tanh((a-softKnee)/(1-softKnee))
	return math.Copysign(y, x)
}

func putSample(b []byte, x float64) {
	if math.IsNaN(x) {
		x = 0
	}
	v := min(max(x, -1), 1) * 32767
	if v >= 0 {
		v += 0.5
	} else {
		v -= 0.5
	}
	binary.LittleEndian.PutUint16(b, uint16(int16(v)))
}

func masterPCM(l, r []float32, level float64) []byte {
	peak := peakOf(l, r)
	gain := 0.0
	if peak > 0 {
		gain = level / peak
	}
	out := make([]byte, len(l)*4)
	for i := range l {
		putSample(out[4*i:], softClip(float64(l[i])*gain))
		putSample(out[4*i+2:], softClip(float64(r[i])*gain))
	}
	return out
}
