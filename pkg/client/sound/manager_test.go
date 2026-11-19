package sound

import (
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

func near(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestCrossfadeBetweenTracks(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicDark)
	a := out.loopAt(0)
	if vol, _, plays := a.state(); vol != 0 || plays != 1 {
		t.Fatalf("new theme starts silent and playing: vol %v plays %d", vol, plays)
	}
	clk.advance(750 * time.Millisecond)
	m.Update()
	if vol, _, _ := a.state(); !near(vol, defaultMusicVolume*math.Sin(math.Pi/4)) {
		t.Errorf("half-way fade-in volume %v", vol)
	}
	clk.advance(750 * time.Millisecond)
	m.Update()
	m.PlayMusic(MusicNight)
	b := out.loopAt(1)
	clk.advance(750 * time.Millisecond)
	m.Update()
	va, _, _ := a.state()
	vb, _, _ := b.state()
	if !near(va, defaultMusicVolume*math.Cos(math.Pi/4)) || !near(vb, defaultMusicVolume*math.Sin(math.Pi/4)) {
		t.Errorf("equal-power midpoint: out %v in %v", va, vb)
	}
	clk.advance(800 * time.Millisecond)
	m.Update()
	_, aClosed, _ := a.state()
	vb, bClosed, _ := b.state()
	if !aClosed || bClosed || !near(vb, defaultMusicVolume) || len(m.layers) != 1 || m.CurrentMusic() != MusicNight {
		t.Errorf("after crossfade: old closed %v, new vol %v, layers %d, current %s", aClosed, vb, len(m.layers), m.CurrentMusic())
	}
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
}

func TestReturningToAFadingThemeReusesIt(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicDark)
	m.PlayMusic(MusicDark)
	clk.advance(2 * time.Second)
	m.Update()
	m.PlayMusic(MusicNight)
	clk.advance(500 * time.Millisecond)
	m.Update()
	m.PlayMusic(MusicDark)
	clk.advance(2 * time.Second)
	m.Update()
	loops, _ := out.counts()
	vol, aClosed, _ := out.loopAt(0).state()
	_, bClosed, _ := out.loopAt(1).state()
	if loops != 2 || aClosed || !bClosed || !near(vol, defaultMusicVolume) {
		t.Errorf("loops %d, dark closed %v vol %v, night closed %v", loops, aClosed, vol, bClosed)
	}
}

func TestStopMusicFadesOut(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicTitle)
	m.StopMusic()
	if m.CurrentMusic() != "" {
		t.Error("stop clears the current theme")
	}
	clk.advance(crossfade)
	m.Update()
	if _, closed, _ := out.loopAt(0).state(); !closed || len(m.layers) != 0 {
		t.Error("stopped music is released once silent")
	}
}

func TestIdenticalEffectsAreThrottled(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlaySFX(SFXClick)
	clk.advance(30 * time.Millisecond)
	m.PlaySFX(SFXClick)
	m.PlaySFX(SFXCoin)
	clk.advance(40 * time.Millisecond)
	m.PlaySFX(SFXClick)
	if _, shots := out.counts(); shots != 3 {
		t.Errorf("want click, coin, click after the window; got %d voices", shots)
	}
}

func TestVoiceCapStealsTheLeastImportant(t *testing.T) {
	m, out, clk := fakeManager()
	first := []SFX{SFXClick, SFXCoin, SFXBuild, SFXPodStart, SFXRollout, SFXBoat, SFXError, SFXReveal}
	for _, id := range first {
		m.PlaySFX(id)
		clk.advance(FamilyGap)
	}
	m.PlaySFX(SFXChronicle)
	if _, closed, _ := out.shotAt(0).state(); !closed || len(m.effects) != maxVoices {
		t.Fatalf("chronicle steals the click: closed %v, voices %d", closed, len(m.effects))
	}
	clk.advance(100 * time.Millisecond)
	m.PlaySFX(SFXClick)
	if _, shots := out.counts(); shots != 9 {
		t.Errorf("a click cannot steal from more important effects; voices started %d", shots)
	}
	m.PlaySFX(SFXVictory)
	if _, closed, _ := out.shotAt(1).state(); !closed {
		t.Error("victory steals the oldest of the least important voices")
	}
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
}

func TestFinishedEffectsAreReclaimed(t *testing.T) {
	m, out, _ := fakeManager()
	m.PlaySFX(SFXCoin)
	out.shotAt(0).finish()
	m.Update()
	if _, closed, _ := out.shotAt(0).state(); !closed || len(m.effects) != 0 {
		t.Error("finished one-shots are closed on Update")
	}
}

func TestMuteAndVolumes(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicDark)
	clk.advance(crossfade)
	m.Update()
	music := out.loopAt(0)
	m.SetMuted(true)
	if vol, _, _ := music.state(); vol != 0 || !m.Muted() {
		t.Error("mute silences the music")
	}
	m.PlaySFX(SFXCoin)
	if _, shots := out.counts(); shots != 0 {
		t.Error("muted managers start no effects")
	}
	m.SetMuted(false)
	for _, c := range []struct{ in, want float64 }{{2, 1}, {math.NaN(), 0}, {-1, 0}, {0.25, 0.25}} {
		m.SetMusicVolume(c.in)
		if vol, _, _ := music.state(); !near(vol, c.want) {
			t.Errorf("SetMusicVolume(%v) applied %v", c.in, vol)
		}
	}
	m.PlaySFX(SFXCoin)
	shot := out.shotAt(0)
	if vol, _, _ := shot.state(); !near(vol, defaultSFXVolume) {
		t.Errorf("effect volume %v", vol)
	}
	m.SetSFXVolume(0.3)
	if vol, _, _ := shot.state(); !near(vol, 0.3) {
		t.Errorf("live effect volume %v", vol)
	}
}

func TestPlaybackFailuresAreReported(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicDark)
	clk.advance(crossfade)
	m.Update()
	boom := errors.New("boom")
	out.fail(boom, boom)
	m.PlayMusic(MusicCrisis)
	if !errors.Is(m.Err(), boom) || m.CurrentMusic() != MusicDark {
		t.Errorf("failed switch keeps the old theme: err %v current %s", m.Err(), m.CurrentMusic())
	}
	m.Update()
	if vol, _, _ := out.loopAt(0).state(); !near(vol, defaultMusicVolume) {
		t.Error("old theme keeps playing at full volume")
	}
	m.PlaySFX(SFXCoin)
	if _, shots := out.counts(); shots != 0 || !errors.Is(m.Err(), boom) {
		t.Error("effect failure is reported")
	}
	m.PlayMusic("polka")
	if m.Err() == nil || !strings.Contains(m.Err().Error(), "unknown music") {
		t.Errorf("unknown music: %v", m.Err())
	}
	m.PlaySFX("kazoo")
	if m.Err() == nil || !strings.Contains(m.Err().Error(), "unknown effect") {
		t.Errorf("unknown effect: %v", m.Err())
	}
}

func TestCloseReleasesEverything(t *testing.T) {
	m, out, _ := fakeManager()
	m.PlayMusic(MusicFeudal)
	m.PlaySFX(SFXVictory)
	bad := errors.New("stuck")
	out.shotAt(0).closeErr = bad
	if err := m.Close(); !errors.Is(err, bad) {
		t.Errorf("close reports voice errors: %v", err)
	}
	if _, closed, _ := out.loopAt(0).state(); !closed || !m.Disabled() || m.CurrentMusic() != "" {
		t.Error("close releases every voice and disables the manager")
	}
	if err := m.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
	m.PlayMusic(MusicDark)
	m.PlaySFX(SFXCoin)
	m.Update()
	if loops, shots := out.counts(); loops != 1 || shots != 1 {
		t.Error("a closed manager starts nothing")
	}
}

func TestSilentAndNilManagersAreSafe(t *testing.T) {
	var nilManager *Manager
	for _, m := range []*Manager{NewSilent(), New(nil), nilManager} {
		if !m.Disabled() {
			t.Error("silent managers report disabled")
		}
		m.PlayMusic(MusicTitle)
		m.PlaySFX(SFXCoin)
		m.Update()
		m.StopMusic()
		m.SetMuted(true)
		m.SetMusicVolume(2)
		m.SetSFXVolume(-1)
		m.PlayMusic(MusicNight)
		if m.Err() != nil {
			t.Errorf("silent manager error: %v", m.Err())
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	}
	s := NewSilent()
	s.PlayMusic(MusicFeudal)
	if s.CurrentMusic() != MusicFeudal || s.Muted() {
		t.Error("silent manager still tracks the requested theme")
	}
	if nilManager.Muted() || nilManager.CurrentMusic() != "" {
		t.Error("nil manager reports zero values")
	}
}

func TestManagerWithARealBank(t *testing.T) {
	out, clk := &fakeOutput{}, newFakeClock()
	m := newManager(out, newBank(minRate), clk.now)
	m.PlayMusic(MusicTitle)
	m.PlaySFX(SFXReveal)
	if out.loopAt(0).pcm == 0 || out.shotAt(0).pcm == 0 || m.Err() != nil {
		t.Errorf("rendered buffers reach the output: %v", m.Err())
	}
}

func TestConcurrentUse(t *testing.T) {
	m, _, clk := fakeManager()
	tracks := AllMusic()
	effects := AllSFX()
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				switch (g + i) % 6 {
				case 0:
					m.PlayMusic(tracks[(g+i)%len(tracks)])
				case 1:
					m.PlaySFX(effects[(g*7+i)%len(effects)])
				case 2:
					m.Update()
				case 3:
					m.SetMuted(i%2 == 0)
				case 4:
					m.SetMusicVolume(float64(i%10) / 10)
				default:
					clk.advance(5 * time.Millisecond)
					_ = m.CurrentMusic()
				}
			}
		})
	}
	wg.Wait()
	if err := m.Close(); err != nil || m.Err() != nil {
		t.Errorf("close %v, err %v", err, m.Err())
	}
}
