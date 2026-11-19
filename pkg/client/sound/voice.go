package sound

import (
	"math"
	"unicode"
)

const (
	VoiceLevel   = 0.3
	VoiceCPS     = 30.0
	TimbreChirp  = "chirp"
	TimbreHum    = "hum"
	beepShortest = 0.030
	beepSpread   = 31
	beepFade     = 0.1
	voiceTail    = 0.05
	chirpRise    = 1.3
	humDrop      = 0.5
	humOvertone  = 0.3
)

var voiceNotes = [...]float64{800, 900, 1000, 1100, 1200}

type Voice struct {
	Pitch  float64
	CPS    float64
	Seed   uint32
	Timbre string
}

type Beep struct {
	At   float64
	Freq float64
	Dur  float64
}

func Voiced(r rune) bool {
	return !unicode.IsSpace(r) && !unicode.IsPunct(r)
}

func (v Voice) cps() float64 {
	if v.CPS <= 0 {
		return VoiceCPS
	}
	return v.CPS
}

func (v Voice) pitch() float64 {
	if v.Pitch <= 0 {
		return 1
	}
	return v.Pitch
}

func voiceHash(seed uint32, i int, r rune) uint32 {
	x := seed ^ uint32(i)*0x9e3779b9 ^ uint32(r)*0x85ebca6b
	x ^= x >> 16
	x *= 0x7feb352d
	x ^= x >> 15
	x *= 0x846ca68b
	x ^= x >> 16
	return x
}

func (v Voice) Beeps(text string) []Beep {
	out := []Beep{}
	for i, r := range []rune(text) {
		if !Voiced(r) {
			continue
		}
		h := voiceHash(v.Seed, i, r)
		dur := beepShortest + float64(h>>8%beepSpread)/1000
		out = append(out, Beep{At: float64(i) / v.cps(), Freq: voiceNotes[h%uint32(len(voiceNotes))] * v.pitch(), Dur: dur})
	}
	return out
}

func (v Voice) Length(text string) float64 {
	end := 0.0
	for _, b := range v.Beeps(text) {
		end = math.Max(end, b.At+b.Dur)
	}
	return end + voiceTail
}

func RenderVoice(v Voice, text string, rate int) ([]byte, error) {
	if err := checkRate(rate); err != nil {
		return nil, err
	}
	n := frames(v.Length(text), rate)
	l := make([]float32, n)
	for _, b := range v.Beeps(text) {
		start := frames(b.At, rate)
		for i, s := range v.beep(b, rate) {
			if start+i < n {
				l[start+i] += s
			}
		}
	}
	r := append([]float32(nil), l...)
	return masterPCM(l, r, VoiceLevel), nil
}

func (v Voice) beep(b Beep, rate int) []float32 {
	n := frames(b.Dur, rate)
	out := make([]float32, n)
	fade := max(int(float64(n)*beepFade), 1)
	phase, over := 0.0, 0.0
	for i := range out {
		t := float64(i) / float64(max(n-1, 1))
		freq := b.Freq
		x := 0.0
		switch v.Timbre {
		case TimbreChirp:
			freq *= 1 + (chirpRise-1)*t
			phase = advance(phase, freq/float64(rate))
			x = sine(phase)
		case TimbreHum:
			freq *= humDrop
			phase = advance(phase, freq/float64(rate))
			over = advance(over, 2*freq/float64(rate))
			x = (sine(phase) + humOvertone*sine(over)) / (1 + humOvertone)
		default:
			phase = advance(phase, freq/float64(rate))
			x = sine(phase)
		}
		out[i] = float32(x * linearFade(i, n, fade))
	}
	return out
}

func linearFade(i, n, fade int) float64 {
	switch {
	case i < fade:
		return float64(i) / float64(fade)
	case i >= n-fade:
		return float64(n-1-i) / float64(fade)
	}
	return 1
}

type VoiceSample struct {
	Name  string
	Voice Voice
	Line  string
}

func VoiceSamples() []VoiceSample {
	return []VoiceSample{
		{Name: "deep", Voice: Voice{Pitch: 0.55, CPS: 26, Seed: 1}, Line: "Box in, box out. That's the whole song."},
		{Name: "middle", Voice: Voice{Pitch: 1, CPS: 30, Seed: 2}, Line: "Is this right? I think this is right."},
		{Name: "chirpy", Voice: Voice{Pitch: 1.55, CPS: 34, Seed: 3}, Line: "Burning bright! That's the good kind of burning."},
		{Name: "raven", Voice: Voice{Pitch: 1.5, CPS: 22, Seed: 4, Timbre: TimbreChirp}, Line: "Kraa! Kraa-kraa."},
		{Name: "hound", Voice: Voice{Pitch: 0.6, CPS: 18, Seed: 5, Timbre: TimbreHum}, Line: "Hrrmm. Hrrmmm."},
	}
}
