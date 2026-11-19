package sound

import (
	"encoding/binary"
	"math"
	"testing"
	"unicode"
)

func TestVoiceBeepsOncePerTypedLetter(t *testing.T) {
	v := Voice{Pitch: 1, CPS: 30, Seed: 7}
	text := "Box in, box out."
	beeps := v.Beeps(text)
	voiced := 0
	for _, r := range text {
		if !unicode.IsSpace(r) && !unicode.IsPunct(r) {
			voiced++
		}
	}
	if len(beeps) != voiced {
		t.Fatalf("%d beeps for %d letters", len(beeps), voiced)
	}
	if beeps[0].At != 0 || math.Abs(beeps[1].At-1.0/30) > 1e-9 {
		t.Errorf("letters type at 30 a second: %+v", beeps[:2])
	}
	for _, b := range beeps {
		if b.Dur < 0.030 || b.Dur > 0.060 {
			t.Errorf("beep lasts %v", b.Dur)
		}
		ok := false
		for _, n := range voiceNotes {
			ok = ok || math.Abs(b.Freq-n) < 1e-9
		}
		if !ok {
			t.Errorf("beep at %v Hz is off the band", b.Freq)
		}
	}
}

func TestSameWorkerSameVoice(t *testing.T) {
	a := Voice{Pitch: 0.7, CPS: 27, Seed: 11}
	b := Voice{Pitch: 1.4, CPS: 33, Seed: 12}
	x, _ := RenderVoice(a, "Trip, fall, relight.", SampleRate)
	y, _ := RenderVoice(a, "Trip, fall, relight.", SampleRate)
	if string(x) != string(y) {
		t.Error("a worker sounds the same every time")
	}
	z, _ := RenderVoice(b, "Trip, fall, relight.", SampleRate)
	if string(x) == string(z) {
		t.Error("two workers sound different")
	}
	for _, beep := range a.Beeps("Trip, fall, relight.") {
		if beep.Freq > 1200*0.7+1e-9 {
			t.Errorf("a 0.7 voice beeps at %v Hz", beep.Freq)
		}
	}
}

func TestBeepsFadeSoTheyNeverClick(t *testing.T) {
	for _, timbre := range []string{"", TimbreChirp, TimbreHum} {
		v := Voice{Pitch: 1, Seed: 3, Timbre: timbre}
		s := v.beep(Beep{Freq: 1000, Dur: 0.04}, SampleRate)
		if math.Abs(float64(s[0])) > 1e-6 || math.Abs(float64(s[len(s)-1])) > 1e-6 {
			t.Errorf("%q beep starts %v ends %v", timbre, s[0], s[len(s)-1])
		}
		fade := int(float64(len(s)) * beepFade)
		for i := 1; i < fade; i++ {
			if math.Abs(float64(s[i])) > float64(i)/float64(fade)+1e-6 {
				t.Fatalf("%q beep rises faster than a linear 10%% fade at %d", timbre, i)
			}
		}
	}
}

func TestVoicePeaksAtDialogueLevel(t *testing.T) {
	pcm, err := RenderVoice(Voice{Pitch: 1, Seed: 5}, "Hello there", SampleRate)
	if err != nil {
		t.Fatal(err)
	}
	peak := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(pcm[i:])))
		peak = max(peak, v, -v)
	}
	if got := float64(peak) / 32767; math.Abs(got-VoiceLevel) > 0.02 {
		t.Errorf("voice peaks at %v", got)
	}
}

func TestSpeakCutsTheLineBeforeAndHonoursMute(t *testing.T) {
	m, out, _ := fakeManager()
	m.Speak(Voice{Seed: 1}, "First line")
	m.Speak(Voice{Seed: 2}, "Second line")
	if _, shots := out.counts(); shots != 2 {
		t.Fatalf("shots = %d", shots)
	}
	if _, closed, _ := out.shotAt(0).state(); !closed {
		t.Error("a new line cuts the line before")
	}
	if vol, closed, plays := out.shotAt(1).state(); closed || plays != 1 || math.Abs(vol-defaultSFXVolume) > 1e-9 {
		t.Errorf("the line plays at the SFX volume: vol %v closed %v plays %d", vol, closed, plays)
	}
	m.SetSFXVolume(0.25)
	if vol, _, _ := out.shotAt(1).state(); math.Abs(vol-0.25) > 1e-9 {
		t.Errorf("speech follows the SFX volume: %v", vol)
	}
	m.SetMuted(true)
	if vol, _, _ := out.shotAt(1).state(); vol != 0 {
		t.Errorf("muted speech is silent: %v", vol)
	}
	m.Speak(Voice{Seed: 3}, "Third line")
	if _, shots := out.counts(); shots != 2 {
		t.Error("nothing speaks while muted")
	}
	if _, closed, _ := out.shotAt(1).state(); !closed {
		t.Error("speaking while muted ends the line before")
	}
}

func TestSilentManagerNeverSpeaks(t *testing.T) {
	m := NewSilent()
	m.Speak(Voice{Seed: 1}, "Hello")
	if m.Speaking() {
		t.Error("a silent manager (captures) speaks nothing")
	}
}
