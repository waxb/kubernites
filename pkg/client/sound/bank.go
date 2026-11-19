package sound

import (
	"fmt"
	"runtime"
	"sync"
)

const warmWorkers = 3

type lazyPCM struct {
	once   sync.Once
	render func() ([]byte, error)
	pcm    []byte
	err    error
}

func (l *lazyPCM) get() ([]byte, error) {
	l.once.Do(func() {
		l.pcm, l.err = l.render()
	})
	return l.pcm, l.err
}

type bank struct {
	rate  int
	music map[Music]*lazyPCM
	sfx   map[SFX]*lazyPCM
	order []*lazyPCM
}

func newBank(rate int) *bank {
	b := &bank{rate: rate, music: map[Music]*lazyPCM{}, sfx: map[SFX]*lazyPCM{}}
	for _, effect := range AllSFX() {
		l := &lazyPCM{render: func() ([]byte, error) { return RenderSFX(effect, rate) }}
		b.sfx[effect] = l
		b.order = append(b.order, l)
	}
	for _, track := range AllMusic() {
		l := &lazyPCM{render: func() ([]byte, error) { return RenderMusic(track, rate) }}
		b.music[track] = l
		b.order = append(b.order, l)
	}
	return b
}

func (b *bank) musicPCM(track Music) ([]byte, error) {
	if b == nil {
		return nil, nil
	}
	l, ok := b.music[track]
	if !ok {
		return nil, fmt.Errorf("sound: unknown music %q", track)
	}
	return l.get()
}

func (b *bank) musicWalk(track Music) phraseWalk {
	if b == nil {
		return phraseWalk{}
	}
	return walkFor(songBook[track](), b.rate)
}

func (b *bank) sfxPCM(effect SFX) ([]byte, error) {
	if b == nil {
		return nil, nil
	}
	l, ok := b.sfx[effect]
	if !ok {
		return nil, fmt.Errorf("sound: unknown effect %q", effect)
	}
	return l.get()
}

func (b *bank) warm(stop <-chan struct{}, report func(error)) {
	if b == nil {
		return
	}
	jobs := make(chan *lazyPCM, len(b.order))
	for _, l := range b.order {
		jobs <- l
	}
	close(jobs)
	var wg sync.WaitGroup
	for range min(warmWorkers, runtime.NumCPU()) {
		wg.Go(func() { drain(jobs, stop, report) })
	}
	wg.Wait()
}

func drain(jobs <-chan *lazyPCM, stop <-chan struct{}, report func(error)) {
	for l := range jobs {
		select {
		case <-stop:
			return
		default:
		}
		if _, err := l.get(); err != nil {
			report(err)
		}
	}
}
