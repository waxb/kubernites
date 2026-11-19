package sound

import (
	"fmt"
	"time"
)

func (m *Manager) PlaySFX(id SFX) {
	m.PlayAll(id)
}

func (m *Manager) PlayAll(ids ...SFX) {
	if m == nil {
		return
	}
	for _, id := range ids {
		if _, ok := sfxSpecs[id]; !ok {
			m.fail(fmt.Errorf("sound: unknown effect %q", id))
			return
		}
	}
	for _, id := range m.admit(ids) {
		pcm, err := m.bank.sfxPCM(id)
		if err != nil {
			m.fail(err)
			return
		}
		m.start(id, pcm)
	}
}

func (m *Manager) start(id SFX, pcm []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.startEffect(id, pcm, m.now())
	}
}

func (m *Manager) admit(ids []SFX) []SFX {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.out == nil || m.closed || m.muted {
		return nil
	}
	now := m.now()
	budget, picked := m.budget.Pick(ids, now)
	m.budget = budget
	out := make([]SFX, 0, len(picked))
	for _, id := range picked {
		last, seen := m.lastSFX[id]
		if throttled(last, seen, now) {
			continue
		}
		m.lastSFX[id] = now
		m.duckUnder(id, now)
		out = append(out, id)
	}
	return out
}

func (m *Manager) Hush(d time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	m.budget = m.budget.Hushed(now, now.Add(d))
}

func (m *Manager) Hushed() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.budget.hushing(m.now())
}

func (m *Manager) startEffect(id SFX, pcm []byte, now time.Time) {
	priority := priorityOf(id)
	live := m.reapEffects()
	if len(live) >= maxVoices {
		victim := pickVictim(slotsOf(live), priority)
		if victim < 0 {
			m.effects = live
			return
		}
		m.closeVoice(live[victim].voice)
		live = removeEffect(live, victim)
	}
	v, err := m.out.oneShot(pcm)
	if err != nil {
		m.err = err
		m.effects = live
		return
	}
	v.SetVolume(m.effectLevel())
	v.Play()
	m.effects = append(live, effect{id: id, voice: v, slot: slot{priority: priority, started: now}})
}

func (m *Manager) reapEffects() []effect {
	live := make([]effect, 0, len(m.effects)+1)
	for _, e := range m.effects {
		if e.voice.IsPlaying() {
			live = append(live, e)
			continue
		}
		m.closeVoice(e.voice)
	}
	return live
}

func slotsOf(effects []effect) []slot {
	out := make([]slot, len(effects))
	for i, e := range effects {
		out[i] = e.slot
	}
	return out
}

func removeEffect(effects []effect, i int) []effect {
	out := make([]effect, 0, len(effects))
	out = append(out, effects[:i]...)
	return append(out, effects[i+1:]...)
}

func (m *Manager) SetMuted(muted bool) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.muted = muted
	m.applyVolumes()
}

func (m *Manager) Muted() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.muted
}

func (m *Manager) SetMusicVolume(level float64) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.musicVol = clampVolume(level)
	m.applyVolumes()
}

func (m *Manager) SetSFXVolume(level float64) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sfxVol = clampVolume(level)
	m.applyVolumes()
}

func (m *Manager) applyVolumes() {
	now := m.now()
	for _, l := range m.layers {
		l.voice.SetVolume(m.musicLevel(l.fade.at(now), now))
	}
	for _, e := range m.effects {
		e.voice.SetVolume(m.effectLevel())
	}
	if m.speech != nil {
		m.speech.SetVolume(m.effectLevel())
	}
}

func (m *Manager) musicLevel(gain float64, now time.Time) float64 {
	if m.muted {
		return 0
	}
	return m.masterVol * m.musicVol * gain * m.duck.gain(now)
}

func (m *Manager) effectLevel() float64 {
	if m.muted {
		return 0
	}
	return m.masterVol * m.sfxVol
}
