package sound

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	musicTestRate = minRate
	sfxTestRate   = 22050
)

var (
	renderOnce sync.Once
	rendered   map[string][]byte
	renderErr  error
	renderTook time.Duration
)

func renderedSet(t *testing.T) map[string][]byte {
	t.Helper()
	renderOnce.Do(func() {
		start := time.Now()
		rendered, renderErr = renderEverything(musicTestRate, sfxTestRate)
		renderTook = time.Since(start)
	})
	if renderErr != nil {
		t.Fatal(renderErr)
	}
	return rendered
}

func renderEverything(musicRate, sfxRate int) (map[string][]byte, error) {
	keys := make([]string, 0, len(AllMusic())+len(AllSFX()))
	jobs := make([]func() ([]byte, error), 0, cap(keys))
	for _, m := range AllMusic() {
		keys = append(keys, "music/"+string(m))
		jobs = append(jobs, func() ([]byte, error) { return RenderMusic(m, musicRate) })
	}
	for _, s := range AllSFX() {
		keys = append(keys, "sfx/"+string(s))
		jobs = append(jobs, func() ([]byte, error) { return RenderSFX(s, sfxRate) })
	}
	out := make(map[string][]byte, len(keys))
	for i, job := range jobs {
		pcm, err := job()
		if err != nil {
			return nil, err
		}
		out[keys[i]] = pcm
	}
	return out, nil
}

func channel(pcm []byte, ch int) []int {
	n := len(pcm) / 4
	out := make([]int, n)
	for i := range out {
		out[i] = int(int16(binary.LittleEndian.Uint16(pcm[4*i+2*ch:])))
	}
	return out
}

func peakAbs(samples []int) int {
	p := 0
	for _, v := range samples {
		p = max(p, v, -v)
	}
	return p
}

func TestRenderingIsDeterministic(t *testing.T) {
	set := renderedSet(t)
	t.Logf("rendered %d clips (music at %d Hz, effects at %d Hz) in %v", len(set), musicTestRate, sfxTestRate, renderTook)
	for _, s := range AllSFX() {
		again, err := RenderSFX(s, sfxTestRate)
		if err != nil || !bytes.Equal(again, set["sfx/"+string(s)]) {
			t.Errorf("effect %s differs between renders (%v)", s, err)
		}
	}
	again, err := RenderMusic(MusicTitle, musicTestRate)
	if err != nil || !bytes.Equal(again, set["music/"+string(MusicTitle)]) {
		t.Errorf("title theme differs between renders (%v)", err)
	}
}

func TestEveryClipIsAudibleStereoAndUnclipped(t *testing.T) {
	for name, pcm := range renderedSet(t) {
		if len(pcm) == 0 || len(pcm)%4 != 0 {
			t.Fatalf("%s has %d bytes, want whole stereo frames", name, len(pcm))
		}
		left, right := channel(pcm, 0), channel(pcm, 1)
		for _, ch := range [][]int{left, right} {
			peak := peakAbs(ch)
			if peak < 3000 {
				t.Errorf("%s is nearly silent: peak %d", name, peak)
			}
			if peak > 31800 {
				t.Errorf("%s reaches the clip ceiling: peak %d", name, peak)
			}
		}
		if strings.HasPrefix(name, "music/") && equalInts(left, right) {
			t.Errorf("%s has no stereo spread", name)
		}
	}
}

func equalInts(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMusicLoopsAreSeamless(t *testing.T) {
	set := renderedSet(t)
	for _, m := range AllMusic() {
		pcm := set["music/"+string(m)]
		for ch := 0; ch < 2; ch++ {
			jump, near, kink, rough := seamStats(channel(pcm, ch), 256)
			if float64(jump) > 1.25*float64(near) {
				t.Errorf("%s channel %d steps %d across the loop point, at most %d nearby", m, ch, jump, near)
			}
			if kink > 8*rough {
				t.Errorf("%s channel %d clicks at the loop point: curvature %.0f vs local %.0f", m, ch, kink, rough)
			}
		}
	}
}

func seamStats(s []int, w int) (int, int, float64, float64) {
	n := len(s)
	at := func(i int) int { return s[((i%n)+n)%n] }
	curve := func(i int) float64 { return math.Abs(float64(at(i+1) - 2*at(i) + at(i-1))) }
	near, sum, count := 0, 0.0, 0
	for i := -w; i < w; i++ {
		if i == -1 || i == 0 {
			continue
		}
		near = max(near, abs(at(i+1)-at(i)))
		sum += curve(i) * curve(i)
		count++
	}
	return abs(at(0) - at(-1)), near, math.Max(curve(-1), curve(0)), math.Sqrt(sum / float64(count))
}

func periodicSignal(n int) []float32 {
	r := newRNG(99)
	x := make([]float32, n)
	for i := range x {
		if i%(n/5) < n/40 {
			x[i] = float32(r.signed())
		}
	}
	x[n-3] = 1
	return x
}

func steadyState(x []float32, periods int, process func([]float32) ([]float32, []float32)) ([]float32, []float32) {
	long := make([]float32, 0, len(x)*periods)
	for range periods {
		long = append(long, x...)
	}
	l, r := process(long)
	return l[len(l)-len(x):], r[len(r)-len(x):]
}

func maxDiff(a, b []float32) float64 {
	d := 0.0
	for i := range a {
		d = math.Max(d, math.Abs(float64(a[i]-b[i])))
	}
	return d
}

func TestCircularEffectsMatchTheirSteadyState(t *testing.T) {
	const rate = 8000
	x := periodicSignal(3 * rate)
	reverbOf := func(in []float32, circular bool) ([]float32, []float32) {
		l, r := make([]float32, len(in)), make([]float32, len(in))
		addReverb(in, l, r, 1, rate, circular)
		return l, r
	}
	wantL, wantR := steadyState(x, 4, func(in []float32) ([]float32, []float32) { return reverbOf(in, false) })
	gotL, gotR := reverbOf(x, true)
	if d := math.Max(maxDiff(gotL, wantL), maxDiff(gotR, wantR)); d > 1e-4 {
		t.Errorf("circular reverb deviates from steady state by %g", d)
	}
	if gotL[0] == 0 {
		t.Error("the reverb tail must wrap into the loop start")
	}
	echo := func(in []float32) ([]float32, []float32) {
		l, r := make([]float32, len(in)), make([]float32, len(in))
		addEcho(in, l, r, rate/3, 0.38, rate)
		return l, r
	}
	wantL, wantR = steadyState(x, 12, echo)
	gotL, gotR = echo(x)
	if d := math.Max(maxDiff(gotL, wantL), maxDiff(gotR, wantR)); d > 1e-3 {
		t.Errorf("circular echo deviates from steady state by %g", d)
	}
	hp := func(in []float32) ([]float32, []float32) {
		out := append([]float32(nil), in...)
		highpass(out, masterHP, rate, 0)
		return out, out
	}
	wantL, _ = steadyState(x, 3, hp)
	gotL = append([]float32(nil), x...)
	highpass(gotL, masterHP, rate, frames(masterPrime, rate))
	if d := maxDiff(gotL, wantL); d > 1e-4 {
		t.Errorf("circular high-pass deviates from steady state by %g", d)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestMusicLoopLengths(t *testing.T) {
	set := renderedSet(t)
	for _, m := range AllMusic() {
		s := songBook[m]()
		want := stepFrames(s.bpm, musicTestRate) * s.steps * 4
		got := len(set["music/"+string(m)])
		if got != want {
			t.Errorf("%s is %d bytes, want %d", m, got, want)
		}
		seconds := float64(got) / 4 / musicTestRate
		if seconds < 10 || seconds > 90 {
			t.Errorf("%s loops every %.1fs, want 10..90s", m, seconds)
		}
	}
}

func TestEffectsAreShortAndEndQuietly(t *testing.T) {
	set := renderedSet(t)
	for _, s := range AllSFX() {
		pcm := set["sfx/"+string(s)]
		seconds := float64(len(pcm)) / 4 / sfxTestRate
		if seconds >= 2.5 || seconds < 0.05 {
			t.Errorf("%s lasts %.2fs, want under 2.5s", s, seconds)
		}
		for ch := 0; ch < 2; ch++ {
			samples := channel(pcm, ch)
			if last := abs(samples[len(samples)-1]); last > 64 {
				t.Errorf("%s ends abruptly on channel %d: %d", s, ch, last)
			}
		}
	}
}

func TestRenderRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		run  func() ([]byte, error)
	}{
		{"unknown music", func() ([]byte, error) { return RenderMusic("polka", SampleRate) }},
		{"unknown effect", func() ([]byte, error) { return RenderSFX("kazoo", SampleRate) }},
		{"music rate too low", func() ([]byte, error) { return RenderMusic(MusicDark, 100) }},
		{"effect rate too high", func() ([]byte, error) { return RenderSFX(SFXCoin, 1_000_000) }},
		{"broken song", func() ([]byte, error) {
			return renderSong(song{bpm: 90, steps: 16, parts: []part{{line: "Q4"}}}, SampleRate)
		}},
		{"broken beat", func() ([]byte, error) {
			return renderSong(song{bpm: 90, steps: 16, beats: []beat{{line: "Z"}}}, SampleRate)
		}},
		{"empty song", func() ([]byte, error) { return renderSong(song{}, SampleRate) }},
	}
	for _, c := range cases {
		if pcm, err := c.run(); err == nil || pcm != nil {
			t.Errorf("%s: want an error, got %d bytes", c.name, len(pcm))
		}
	}
}

func TestTitleThemeAtTheShippingRate(t *testing.T) {
	pcm, err := RenderMusic(MusicTitle, SampleRate)
	if err != nil {
		t.Fatal(err)
	}
	s := songBook[MusicTitle]()
	if want := stepFrames(s.bpm, SampleRate) * s.steps * 4; len(pcm) != want {
		t.Fatalf("title is %d bytes at %d Hz, want %d", len(pcm), SampleRate, want)
	}
	for ch := 0; ch < 2; ch++ {
		samples := channel(pcm, ch)
		if peak := peakAbs(samples); peak < 3000 || peak > 31800 {
			t.Errorf("channel %d peak %d", ch, peak)
		}
		jump, near, kink, rough := seamStats(samples, 256)
		if float64(jump) > 1.25*float64(near) || kink > 8*rough {
			t.Errorf("channel %d seam: step %d vs %d, curvature %.0f vs %.0f", ch, jump, near, kink, rough)
		}
	}
}

func TestRenderFollowsTheContextRate(t *testing.T) {
	pcm, err := RenderSFX(SFXCoin, 48000)
	if err != nil {
		t.Fatal(err)
	}
	if want := frames(sfxSpecs[SFXCoin].length, 48000) * 4; len(pcm) != want {
		t.Errorf("coin at 48kHz is %d bytes, want %d", len(pcm), want)
	}
}

func TestSongsAreWellFormed(t *testing.T) {
	for _, m := range AllMusic() {
		s := songBook[m]()
		for i, p := range s.parts {
			events, err := noteEvents(p.line, s.steps, p.shift)
			if err != nil {
				t.Fatalf("%s part %d: %v", m, i, err)
			}
			if len(events) == 0 || p.gate <= 0 && p.inst.kind != kindBell || p.gate > 1 {
				t.Errorf("%s part %d: %d events, gate %.2f", m, i, len(events), p.gate)
			}
			for _, e := range events {
				if e.note < 24 || e.note > 100 || e.length <= 0 {
					t.Errorf("%s part %d: odd event %+v", m, i, e)
				}
			}
		}
		for i, b := range s.beats {
			if _, err := hitEvents(b.line, s.steps); err != nil {
				t.Fatalf("%s beat %d: %v", m, i, err)
			}
		}
	}
}

func BenchmarkRenderAll(b *testing.B) {
	for b.Loop() {
		for _, m := range AllMusic() {
			if _, err := RenderMusic(m, SampleRate); err != nil {
				b.Fatal(err)
			}
		}
		for _, s := range AllSFX() {
			if _, err := RenderSFX(s, SampleRate); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestSoftClipStaysBelowFullScale(t *testing.T) {
	prev := -2.0
	for x := -20.0; x <= 20; x += 0.01 {
		y := softClip(x)
		if math.Abs(y) > 1 || y < prev || math.Abs(x) <= 1 && math.Abs(y) >= 0.95 {
			t.Fatalf("softClip(%.2f) = %.4f", x, y)
		}
		prev = y
	}
	if softClip(0.5) != 0.5 {
		t.Error("quiet samples must pass through untouched")
	}
	buf := make([]byte, 2)
	putSample(buf, math.NaN())
	if binary.LittleEndian.Uint16(buf) != 0 {
		t.Error("NaN must render as silence")
	}
}

func TestThemeFor(t *testing.T) {
	cases := []struct {
		age    string
		phase  float64
		crisis bool
		want   Music
	}{
		{"dark", 0.5, false, MusicDark},
		{"feudal", 0.5, false, MusicFeudal},
		{"feudal", 0.1, false, MusicFeudalNight},
		{"feudal", 0.85, false, MusicFeudalNight},
		{"dark", 0.9, false, MusicNight},
		{"dark", 1.1, false, MusicNight},
		{"dark", -0.05, false, MusicNight},
		{"dark", 0.2, false, MusicDark},
		{"feudal", 0.1, true, MusicCrisis},
		{"", math.NaN(), false, MusicDark},
		{"dark", math.Inf(1), false, MusicDark},
	}
	for _, c := range cases {
		if got := ThemeFor(c.age, c.phase, c.crisis); got != c.want {
			t.Errorf("ThemeFor(%q, %v, %v) = %s, want %s", c.age, c.phase, c.crisis, got, c.want)
		}
	}
}
