package sound

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type output interface {
	loop(pcm []byte, walk phraseWalk) (voice, error)
	oneShot(pcm []byte) (voice, error)
}

type voice interface {
	Play()
	IsPlaying() bool
	SetVolume(volume float64)
	Close() error
}

type layer struct {
	track Music
	voice voice
	fade  fade
}

type effect struct {
	id    SFX
	voice voice
	slot  slot
}

type Manager struct {
	mu        sync.Mutex
	out       output
	bank      *bank
	now       func() time.Time
	current   Music
	layers    []layer
	effects   []effect
	speech    voice
	lastSFX   map[SFX]time.Time
	budget    Budget
	duck      duck
	muted     bool
	masterVol float64
	musicVol  float64
	sfxVol    float64
	err       error
	closed    bool
	stop      chan struct{}
}

func New(ctx *audio.Context) *Manager {
	if ctx == nil {
		return NewSilent()
	}
	m := newManager(ebitenOutput{ctx: ctx}, newBank(ctx.SampleRate()), time.Now)
	go m.bank.warm(m.stop, m.fail)
	return m
}

func NewSilent() *Manager {
	return newManager(nil, nil, time.Now)
}

func newManager(out output, b *bank, now func() time.Time) *Manager {
	return &Manager{
		out: out, bank: b, now: now, lastSFX: map[SFX]time.Time{}, budget: NewBudget(),
		masterVol: defaultMasterVolume, musicVol: defaultMusicVolume, sfxVol: defaultSFXVolume, stop: make(chan struct{}),
	}
}

func (m *Manager) Disabled() bool {
	if m == nil {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.out == nil || m.closed
}

func (m *Manager) PlayMusic(track Music) {
	if m == nil {
		return
	}
	if !knownMusic(track) {
		m.fail(fmt.Errorf("sound: unknown music %q", track))
		return
	}
	if m.CurrentMusic() == track {
		return
	}
	pcm, err := m.bank.musicPCM(track)
	if err != nil {
		m.fail(err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.current == track {
		return
	}
	next, err := m.retarget(track, pcm, m.now())
	if err != nil {
		m.err = err
		return
	}
	m.layers, m.current = next, track
}

func (m *Manager) retarget(track Music, pcm []byte, now time.Time) ([]layer, error) {
	next := make([]layer, 0, len(m.layers)+1)
	found := false
	for _, l := range m.layers {
		to := 0.0
		if l.track == track {
			to, found = 1, true
		}
		next = append(next, l.toward(to, now))
	}
	if found || m.out == nil {
		return next, nil
	}
	v, err := m.out.loop(pcm, m.bank.musicWalk(track))
	if err != nil {
		return nil, err
	}
	v.SetVolume(0)
	v.Play()
	return append(next, layer{track: track, voice: v, fade: newFade(0, 1, now)}), nil
}

func (l layer) toward(to float64, now time.Time) layer {
	return layer{track: l.track, voice: l.voice, fade: newFade(l.fade.at(now), to, now)}
}

func (m *Manager) StopMusic() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	next := make([]layer, 0, len(m.layers))
	for _, l := range m.layers {
		next = append(next, l.toward(0, now))
	}
	m.layers, m.current = next, ""
}

func (m *Manager) CurrentMusic() Music {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

func (m *Manager) Update() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.layers = m.stepLayers(m.now())
	m.effects = m.reapEffects()
	m.reapSpeech()
}

func (m *Manager) stepLayers(now time.Time) []layer {
	kept := make([]layer, 0, len(m.layers))
	for _, l := range m.layers {
		if l.fade.to == 0 && l.fade.finished(now) {
			m.closeVoice(l.voice)
			continue
		}
		l.voice.SetVolume(m.musicLevel(l.fade.at(now), now))
		kept = append(kept, l)
	}
	return kept
}

func (m *Manager) Err() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *Manager) fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *Manager) closeVoice(v voice) {
	if err := v.Close(); err != nil {
		m.err = err
	}
}

func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.stop)
	errs := make([]error, 0, len(m.layers)+len(m.effects))
	for _, l := range m.layers {
		errs = append(errs, l.voice.Close())
	}
	for _, e := range m.effects {
		errs = append(errs, e.voice.Close())
	}
	if m.speech != nil {
		errs = append(errs, m.speech.Close())
	}
	m.layers, m.effects, m.current, m.speech = nil, nil, "", nil
	return errors.Join(errs...)
}
