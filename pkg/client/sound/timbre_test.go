package sound

import (
	"math"
	"math/cmplx"
	"testing"
)

const (
	timbreFrame = 4096
	timbreSplit = 3000.0
)

type timbre struct {
	centroid float64
	high     float64
	rms      float64
	weighted float64
	loudness float64
}

func luteSolo(t *testing.T, s song, rate int, lute instrument) []float64 {
	t.Helper()
	step := stepFrames(s.bpm, rate)
	out := make([]float64, step*s.steps)
	r := newRNG(s.seed)
	for _, p := range s.parts {
		if p.inst != lute {
			continue
		}
		events, err := noteEvents(p.line, s.steps, p.shift)
		if err != nil {
			t.Fatal(err)
		}
		var scratch []float32
		for _, e := range events {
			hold := max(int(float64(e.length*step)*p.gate), 1)
			scratch = lute.render(scratch, midiFrequency(e.note), hold, rate, r)
			for i, v := range scratch {
				out[(e.step*step+i)%len(out)] += float64(v) * p.amp
			}
		}
	}
	return out
}

func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		w := cmplx.Exp(complex(0, -2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			tw := complex(1, 0)
			for k := 0; k < size/2; k++ {
				a, b := x[start+k], x[start+k+size/2]*tw
				x[start+k], x[start+k+size/2] = a+b, a-b
				tw *= w
			}
		}
	}
}

func powerSpectrum(signal []float64) []float64 {
	power := make([]float64, timbreFrame/2)
	frame := make([]complex128, timbreFrame)
	for start := 0; start+timbreFrame <= len(signal); start += timbreFrame / 2 {
		for i := range frame {
			w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(timbreFrame-1))
			frame[i] = complex(signal[start+i]*w, 0)
		}
		fft(frame)
		for k := range power {
			power[k] += real(frame[k])*real(frame[k]) + imag(frame[k])*imag(frame[k])
		}
	}
	return power
}

func aWeight(f float64) float64 {
	f2 := f * f
	num := 12194.0 * 12194.0 * f2 * f2
	den := (f2 + 20.6*20.6) * math.Sqrt((f2+107.7*107.7)*(f2+737.9*737.9)) * (f2 + 12194.0*12194.0)
	return num / den * math.Pow(10, 2.0/20)
}

func kWeight(f float64) float64 {
	shelf := math.Pow(10, 4.0/10)
	x := (f / 1500) * (f / 1500)
	hp := math.Pow(f/38, 4)
	return (1 + x*shelf) / (1 + x) * hp / (1 + hp)
}

func measureTimbre(signal []float64, rate int) timbre {
	power := powerSpectrum(signal)
	bin := float64(rate) / timbreFrame
	var total, moment, high, weighted, loud float64
	for k, p := range power {
		f := float64(k) * bin
		total += p
		moment += f * p
		weighted += p * aWeight(f) * aWeight(f)
		loud += p * kWeight(f)
		if f > timbreSplit {
			high += p
		}
	}
	sum := 0.0
	for _, v := range signal {
		sum += v * v
	}
	rms := math.Sqrt(sum / float64(len(signal)))
	level := 20 * math.Log10(rms)
	return timbre{centroid: moment / total, high: high / total, rms: level,
		weighted: 10*math.Log10(weighted/total) + level, loudness: 10*math.Log10(loud/total) + level}
}

func luteSongs() map[string]song {
	return map[string]song{"dark": darkSong(), "feudal": feudalSong(), "title": titleSong(), "crisis": crisisSong()}
}

var (
	legacyLute     = instrument{kind: kindPluck, release: 0.08, decay: 1.5, tone: 0.5}
	legacyLuteAmps = map[float64]float64{0.89: 0.85, 0.34: 0.32, 0.84: 0.8, 0.24: 0.22}
	legacyLevels   = map[string]float64{"title": 0.82}
)

func legacyTake(t *testing.T, s song) song {
	t.Helper()
	parts := make([]part, len(s.parts))
	for i, p := range s.parts {
		parts[i] = p
		if p.inst != instLute {
			continue
		}
		amp, ok := legacyLuteAmps[p.amp]
		if !ok {
			t.Fatalf("lute part amp %.2f has no legacy level", p.amp)
		}
		parts[i].inst, parts[i].amp = legacyLute, amp
	}
	s.parts = parts
	return s
}

func TestLuteIsWarmerAtTheSameLoudness(t *testing.T) {
	for name, s := range luteSongs() {
		before := measureTimbre(luteSolo(t, legacyTake(t, s), SampleRate, legacyLute), SampleRate)
		after := measureTimbre(luteSolo(t, s, SampleRate, instLute), SampleRate)
		t.Logf("%-7s centroid %7.1f -> %7.1f Hz  above 3k %6.2f%% -> %6.2f%%  A %6.2f -> %6.2f dB  K %6.2f -> %6.2f dB",
			name, before.centroid, after.centroid, 100*before.high, 100*after.high, before.weighted, after.weighted, before.loudness, after.loudness)
		if after.centroid > 0.75*before.centroid {
			t.Errorf("%s: lute centroid %.0f Hz is not warmer than %.0f Hz", name, after.centroid, before.centroid)
		}
		if after.high > 0.6*before.high {
			t.Errorf("%s: lute keeps %.1f%% of its energy above 3 kHz (was %.1f%%)", name, 100*after.high, 100*before.high)
		}
		if math.Abs(after.weighted-before.weighted) > 0.75 || math.Abs(after.loudness-before.loudness) > 0.75 {
			t.Errorf("%s: lute loudness moved: A %.2f -> %.2f dB, K %.2f -> %.2f dB", name, before.weighted, after.weighted, before.loudness, after.loudness)
		}
	}
}

func TestPluckDampingAndBodyDarken(t *testing.T) {
	base := instrument{kind: kindPluck, release: 0.3, decay: 3.2, tone: 0.6}
	plain := renderPluck(base, nil, 330, SampleRate/4, SampleRate, newRNG(3))
	warm := base
	warm.damp, warm.body, warm.attack = 0.6, 0.08, 0.003
	soft := renderPluck(warm, nil, 330, SampleRate/4, SampleRate, newRNG(3))
	if len(plain) != len(soft) {
		t.Fatalf("damping changed the note length: %d vs %d", len(plain), len(soft))
	}
	a := measureTimbre(widen(plain), SampleRate)
	b := measureTimbre(widen(soft), SampleRate)
	if b.centroid >= a.centroid || b.high >= a.high {
		t.Errorf("damping and body must darken a pluck: %.0f/%.3f -> %.0f/%.3f", a.centroid, a.high, b.centroid, b.high)
	}
}

func TestLuteSongsKeepTheirLoudness(t *testing.T) {
	for name, s := range luteSongs() {
		old := legacyTake(t, s)
		if level, ok := legacyLevels[name]; ok {
			old.level = level
		}
		before, err := renderSong(old, SampleRate)
		if err != nil {
			t.Fatal(err)
		}
		after, err := renderSong(s, SampleRate)
		if err != nil {
			t.Fatal(err)
		}
		a, b := pcmLevel(before), pcmLevel(after)
		t.Logf("%-7s mix rms %6.2f -> %6.2f dBFS", name, a, b)
		if math.Abs(b-a) > 0.5 {
			t.Errorf("%s: the mix moved from %.2f to %.2f dBFS", name, a, b)
		}
	}
}

func pcmLevel(pcm []byte) float64 {
	samples := append(channel(pcm, 0), channel(pcm, 1)...)
	sum := 0.0
	for _, v := range samples {
		f := float64(v) / 32768
		sum += f * f
	}
	return 10 * math.Log10(sum/float64(len(samples)))
}

func widen(in []float32) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(v)
	}
	return out
}
