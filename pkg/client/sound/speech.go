package sound

func (m *Manager) Speak(v Voice, text string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	quiet := m.out == nil || m.closed || m.muted
	if quiet {
		m.stopSpeech()
	}
	m.mu.Unlock()
	if quiet {
		return
	}
	pcm, err := RenderVoice(v, text, m.rate())
	if err != nil {
		m.fail(err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.stopSpeech()
	line, err := m.out.oneShot(pcm)
	if err != nil {
		m.err = err
		return
	}
	line.SetVolume(m.effectLevel())
	line.Play()
	m.speech = line
}

func (m *Manager) StopSpeech() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopSpeech()
}

func (m *Manager) Speaking() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.speech != nil && m.speech.IsPlaying()
}

func (m *Manager) stopSpeech() {
	if m.speech != nil {
		m.closeVoice(m.speech)
		m.speech = nil
	}
}

func (m *Manager) reapSpeech() {
	if m.speech != nil && !m.speech.IsPlaying() {
		m.stopSpeech()
	}
}

func (m *Manager) rate() int {
	if m.bank != nil {
		return m.bank.rate
	}
	return SampleRate
}
