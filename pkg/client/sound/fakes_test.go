package sound

import (
	"errors"
	"sync"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type fakeVoice struct {
	mu       sync.Mutex
	pcm      int
	walk     phraseWalk
	volume   float64
	playing  bool
	plays    int
	closed   bool
	closeErr error
}

func (v *fakeVoice) Play() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.plays++
	v.playing = true
}

func (v *fakeVoice) IsPlaying() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.playing && !v.closed
}

func (v *fakeVoice) SetVolume(volume float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.volume = volume
}

func (v *fakeVoice) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return errors.New("fake voice closed twice")
	}
	v.closed = true
	return v.closeErr
}

func (v *fakeVoice) finish() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.playing = false
}

func (v *fakeVoice) state() (float64, bool, int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.volume, v.closed, v.plays
}

type fakeOutput struct {
	mu      sync.Mutex
	loops   []*fakeVoice
	shots   []*fakeVoice
	loopErr error
	shotErr error
}

func (o *fakeOutput) loop(pcm []byte, walk phraseWalk) (voice, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.loopErr != nil {
		return nil, o.loopErr
	}
	v := &fakeVoice{pcm: len(pcm), walk: walk}
	o.loops = append(o.loops, v)
	return v, nil
}

func (o *fakeOutput) oneShot(pcm []byte) (voice, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.shotErr != nil {
		return nil, o.shotErr
	}
	v := &fakeVoice{pcm: len(pcm)}
	o.shots = append(o.shots, v)
	return v, nil
}

func (o *fakeOutput) counts() (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.loops), len(o.shots)
}

func (o *fakeOutput) loopAt(i int) *fakeVoice {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.loops[i]
}

func (o *fakeOutput) shotAt(i int) *fakeVoice {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.shots[i]
}

func (o *fakeOutput) fail(loopErr, shotErr error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.loopErr, o.shotErr = loopErr, shotErr
}

func fakeManager() (*Manager, *fakeOutput, *fakeClock) {
	out, clk := &fakeOutput{}, newFakeClock()
	return newManager(out, nil, clk.now), out, clk
}
