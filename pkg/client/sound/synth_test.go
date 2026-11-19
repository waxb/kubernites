package sound

import (
	"math"
	"strings"
	"testing"
)

func TestParseNote(t *testing.T) {
	good := map[string]int{"C4": 60, "A4": 69, "C#4": 61, "Db4": 61, "Bb1": 34, "B1": 35, "F#2": 42, "C0": 12}
	for text, want := range good {
		got, err := parseNote(text)
		if err != nil || got != want {
			t.Errorf("parseNote(%q) = %d, %v; want %d", text, got, err, want)
		}
	}
	for _, bad := range []string{"", "C", "H4", "C#", "Cx4", "C10", "c4"} {
		if _, err := parseNote(bad); err == nil {
			t.Errorf("parseNote(%q) should fail", bad)
		}
	}
}

func TestNoteEventsHoldAndWrap(t *testing.T) {
	cases := []struct {
		line  string
		steps int
		shift int
		want  []noteEvent
	}{
		{"C4 -- .. --  E4 -- -- --", 8, 0, []noteEvent{{0, 2, 60}, {4, 4, 64}}},
		{"-- -- G4 --", 4, 0, []noteEvent{{2, 4, 67}}},
		{"-- C4 -- D4", 4, 12, []noteEvent{{1, 2, 72}, {3, 2, 74}}},
		{"A4 B4", 4, 0, []noteEvent{{0, 1, 69}, {1, 1, 71}, {2, 1, 69}, {3, 1, 71}}},
		{".. -- -- --", 4, 0, []noteEvent{}},
	}
	for _, c := range cases {
		got, err := noteEvents(c.line, c.steps, c.shift)
		if err != nil {
			t.Fatalf("%q: %v", c.line, err)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%q: got %+v, want %+v", c.line, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q event %d: got %+v, want %+v", c.line, i, got[i], c.want[i])
			}
		}
	}
}

func TestPatternErrors(t *testing.T) {
	for _, c := range []struct {
		line  string
		steps int
	}{{"", 4}, {"C4 D4 E4", 8}, {"C4 Q4", 4}, {"C4", 0}} {
		if _, err := noteEvents(c.line, c.steps, 0); err == nil {
			t.Errorf("noteEvents(%q, %d) should fail", c.line, c.steps)
		}
	}
	for _, line := range []string{"K X", "K C4", "KK --"} {
		if _, err := hitEvents(line, 4); err == nil {
			t.Errorf("hitEvents(%q) should fail", line)
		}
	}
	hits, err := hitEvents("K -- S H", 8)
	if err != nil || len(hits) != 6 || hits[1] != (hitEvent{2, 'S'}) || hits[5] != (hitEvent{7, 'H'}) {
		t.Errorf("hitEvents = %+v, %v", hits, err)
	}
}

func TestPatternHelpers(t *testing.T) {
	cases := map[string]int{
		hold("C4", 4): 4, hold("C4", 1): 1, rest(3): 3, times("A4 --", 3): 6,
		arp4("C4", "E4", "G4", "E4"): 16, halves("C2", "G2"): 16, seq("A4", "B4 --"): 3,
	}
	for line, want := range cases {
		if got := len(strings.Fields(line)); got != want {
			t.Errorf("%q has %d steps, want %d", line, got, want)
		}
	}
}

func estimatePitch(signal []float32, rate int) float64 {
	best, bestLag := -1.0, 0
	corr := func(lag int) float64 {
		sum := 0.0
		for i := 0; i+lag < len(signal); i++ {
			sum += float64(signal[i]) * float64(signal[i+lag])
		}
		return sum
	}
	for lag := rate / 2000; lag <= rate/60; lag++ {
		if c := corr(lag); c > best {
			best, bestLag = c, lag
		}
	}
	a, b, c := corr(bestLag-1), best, corr(bestLag+1)
	offset := 0.5 * (a - c) / (a - 2*b + c)
	return float64(rate) / (float64(bestLag) + offset)
}

func TestPluckedStringsAreInTune(t *testing.T) {
	for _, freq := range []float64{98, 440, 1174.66} {
		out := renderPluck(instLute, nil, freq, SampleRate/2, SampleRate, newRNG(7))
		window := out[SampleRate/20 : SampleRate/20+4096]
		got := estimatePitch(window, SampleRate)
		if cents := 1200 * math.Log2(got/freq); math.Abs(cents) > 6 {
			t.Errorf("pluck at %.2f Hz sounds at %.2f Hz (%.1f cents)", freq, got, cents)
		}
	}
}

func TestInstrumentsAreBoundedAndAudible(t *testing.T) {
	kits := []instrument{instLute, instHarp, instRecorder, instPad, instDrone, instBass, instChip, instReed, instHorn, instBrass, instChime, instToll}
	for i, in := range kits {
		out := in.render(nil, 220, SampleRate/4, SampleRate, newRNG(uint32(i)))
		checkBounded(t, "instrument", i, out)
	}
	for kind := range hitShapes {
		checkBounded(t, "drum "+string(kind), 0, renderHit(kind, SampleRate, newRNG(1)))
	}
}

func checkBounded(t *testing.T, what string, i int, out []float32) {
	t.Helper()
	peak := 0.0
	for _, v := range out {
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatalf("%s %d produced %v", what, i, f)
		}
		peak = math.Max(peak, math.Abs(f))
	}
	if peak < 0.01 || peak > 4 {
		t.Errorf("%s %d peak %.3f", what, i, peak)
	}
}

func TestDSPHelpers(t *testing.T) {
	if midiFrequency(69) != 440 {
		t.Error("A4 must be 440 Hz")
	}
	if math.Abs(sine(0.25)-1) > 1e-6 || math.Abs(sine(-0.25)+1) > 1e-6 || sine(1) != 0 {
		t.Error("sine table lookup is off")
	}
	if frames(-1, SampleRate) != 0 || decayPerSample(0, SampleRate) != 0 {
		t.Error("non-positive durations collapse to zero")
	}
	if envelope(5, 0, 4, 0) != 0 || envelope(0, 10, 100, 10) != 0 || envelope(50, 10, 100, 10) != 1 {
		t.Error("envelope shape")
	}
	if l, r := panGains(-5); math.Abs(l-1) > 1e-9 || math.Abs(r) > 1e-9 {
		t.Error("hard left pan")
	}
	if got := advance(0.9, 2.3); got < 0 || got >= 1 {
		t.Errorf("advance wraps into [0,1): %v", got)
	}
	if newRNG(0).next() == 0 {
		t.Error("zero seed must still produce noise")
	}
	buf := []float32{1, 1, 1, 1}
	edgeFade(buf, 2, 2)
	if buf[0] != 0 || buf[3] != 0 || buf[1] != 0.5 {
		t.Errorf("edgeFade = %v", buf)
	}
}

func TestMixBusWrapsAroundTheLoop(t *testing.T) {
	bus := newMixBus(4)
	bus.place(3, []float32{1, 1}, 1, 0, 0.5, 0.25)
	if bus.l[3] == 0 || bus.l[0] == 0 || bus.l[1] != 0 || bus.room[0] == 0 || bus.echo[3] == 0 {
		t.Errorf("wrapped placement: %+v", bus)
	}
	newMixBus(0).place(0, []float32{1}, 1, 0, 0, 0)
}
