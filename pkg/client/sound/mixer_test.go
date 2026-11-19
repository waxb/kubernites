package sound

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

func TestFadeGain(t *testing.T) {
	cases := []struct {
		from, to      float64
		elapsed, span time.Duration
		want          float64
	}{
		{0, 1, 0, time.Second, 0},
		{0, 1, 500 * time.Millisecond, time.Second, math.Sin(math.Pi / 4)},
		{1, 0, 500 * time.Millisecond, time.Second, math.Cos(math.Pi / 4)},
		{0, 1, 2 * time.Second, time.Second, 1},
		{0.3, 0.8, 0, 0, 0.8},
		{0.3, 0.8, -time.Second, time.Second, 0.3},
	}
	for _, c := range cases {
		if got := fadeGain(c.from, c.to, c.elapsed, c.span); !near(got, c.want) {
			t.Errorf("fadeGain(%v→%v at %v/%v) = %v, want %v", c.from, c.to, c.elapsed, c.span, got, c.want)
		}
	}
	for x := 0.0; x <= 1; x += 0.1 {
		elapsed := time.Duration(x * float64(time.Second))
		in, out := fadeGain(0, 1, elapsed, time.Second), fadeGain(1, 0, elapsed, time.Second)
		if !near(in*in+out*out, 1) {
			t.Errorf("crossfade loses power at %.1f: %v", x, in*in+out*out)
		}
	}
	start := time.Now()
	f := newFade(0.5, 1, start)
	if f.dur != crossfade/2 || f.finished(start) || !f.finished(start.Add(crossfade/2)) {
		t.Errorf("partial fades take proportionally less time: %v", f.dur)
	}
}

func TestThrottleWindow(t *testing.T) {
	now := time.Now()
	cases := []struct {
		last time.Time
		seen bool
		want bool
	}{
		{now, false, false},
		{now.Add(-59 * time.Millisecond), true, true},
		{now.Add(-60 * time.Millisecond), true, false},
		{now.Add(-time.Second), true, false},
	}
	for _, c := range cases {
		if got := throttled(c.last, c.seen, now); got != c.want {
			t.Errorf("throttled(%v ago, %v) = %v", now.Sub(c.last), c.seen, got)
		}
	}
}

func TestPickVictim(t *testing.T) {
	t0 := time.Now()
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	slots := []slot{{2, at(0)}, {1, at(5)}, {1, at(3)}, {4, at(1)}}
	cases := []struct {
		incoming, want int
	}{{4, 2}, {1, 2}, {0, -1}}
	for _, c := range cases {
		if got := pickVictim(slots, c.incoming); got != c.want {
			t.Errorf("pickVictim(incoming %d) = %d, want %d", c.incoming, got, c.want)
		}
	}
	if pickVictim(nil, 4) != -1 {
		t.Error("nothing to steal")
	}
}

func TestClampVolume(t *testing.T) {
	for in, want := range map[float64]float64{-3: 0, 0: 0, 0.4: 0.4, 1: 1, 7: 1, math.Inf(1): 1} {
		if got := clampVolume(in); got != want {
			t.Errorf("clampVolume(%v) = %v", in, got)
		}
	}
}

func TestBankCachesAndWarms(t *testing.T) {
	b := newBank(minRate)
	first, err := b.sfxPCM(SFXReveal)
	if err != nil {
		t.Fatal(err)
	}
	again, err := b.sfxPCM(SFXReveal)
	if err != nil || &first[0] != &again[0] {
		t.Error("a clip renders once and is reused")
	}
	if _, err := b.musicPCM("polka"); err == nil {
		t.Error("unknown music")
	}
	if _, err := b.sfxPCM("kazoo"); err == nil {
		t.Error("unknown effect")
	}
	var none *bank
	if pcm, err := none.sfxPCM(SFXCoin); pcm != nil || err != nil {
		t.Error("nil bank is silent")
	}
	none.warm(make(chan struct{}), nil)
	var mu sync.Mutex
	var failures []error
	report := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		failures = append(failures, err)
	}
	renders := 0
	fake := &bank{}
	for range 5 {
		fake.order = append(fake.order, &lazyPCM{render: func() ([]byte, error) {
			mu.Lock()
			defer mu.Unlock()
			renders++
			return []byte{0, 0, 0, 0}, nil
		}})
	}
	fake.warm(make(chan struct{}), report)
	newBank(10).warm(make(chan struct{}), report)
	if want := len(AllMusic()) + len(AllSFX()); len(failures) != want || renders != 5 {
		t.Errorf("warm renders everything (%d of 5) and reports every broken render (%d of %d)", renders, len(failures), want)
	}
}

func TestWarmStopsWhenAsked(t *testing.T) {
	renders := 0
	b := &bank{order: []*lazyPCM{{render: func() ([]byte, error) { renders++; return nil, nil }}}}
	stop := make(chan struct{})
	close(stop)
	b.warm(stop, func(error) {})
	if renders != 0 {
		t.Error("a stopped warm-up renders nothing")
	}
}

var (
	contextOnce sync.Once
	testContext *audio.Context
)

func sharedContext() *audio.Context {
	contextOnce.Do(func() {
		testContext = audio.NewContext(SampleRate)
	})
	return testContext
}

func TestEbitenOutput(t *testing.T) {
	out := ebitenOutput{ctx: sharedContext()}
	if _, err := out.loop(nil, phraseWalk{}); !errors.Is(err, errEmptyPCM) {
		t.Error("empty loop is refused")
	}
	if _, err := out.oneShot(nil); !errors.Is(err, errEmptyPCM) {
		t.Error("empty effect is refused")
	}
	pcm := make([]byte, 4*441)
	loop := func(pcm []byte) (voice, error) { return out.loop(pcm, phraseWalk{}) }
	for _, create := range []func([]byte) (voice, error){loop, out.oneShot} {
		v, err := create(pcm)
		if err != nil {
			t.Fatal(err)
		}
		v.SetVolume(0)
		v.Play()
		if !v.IsPlaying() {
			t.Error("player reports playing once started")
		}
		if err := v.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestNewDrivesAnAudioContext(t *testing.T) {
	m := New(sharedContext())
	if m.Disabled() {
		t.Fatal("a context enables audio")
	}
	m.SetSFXVolume(0)
	m.PlaySFX(SFXClick)
	m.Update()
	if len(m.effects) != 1 || m.Err() != nil {
		t.Errorf("effects %d err %v", len(m.effects), m.Err())
	}
	if err := m.Close(); err != nil {
		t.Error(err)
	}
}
