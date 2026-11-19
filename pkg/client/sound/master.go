package sound

func (m *Manager) SetMasterVolume(level float64) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.masterVol = clampVolume(level)
	m.applyVolumes()
}

func (m *Manager) Volumes() (master, music, sounds float64) {
	if m == nil {
		return defaultMasterVolume, defaultMusicVolume, defaultSFXVolume
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.masterVol, m.musicVol, m.sfxVol
}
