package sound

import (
	"fmt"
	"math"
)

const SampleRate = 44100

const (
	minRate    = 8000
	maxRate    = 192000
	nightEnd   = 0.2
	nightStart = 0.8
	ageFeudal  = "feudal"
)

type Music string

type SFX string

const (
	MusicTitle       Music = "title"
	MusicDark        Music = "dark"
	MusicNight       Music = "night"
	MusicFeudal      Music = "feudal"
	MusicFeudalNight Music = "feudal-night"
	MusicCrisis      Music = "crisis"
)

const (
	SFXClick     SFX = "click"
	SFXCoin      SFX = "coin"
	SFXBuild     SFX = "build"
	SFXBuilt     SFX = "built"
	SFXPodStart  SFX = "pod_start"
	SFXError     SFX = "error"
	SFXUnlock    SFX = "unlock"
	SFXRollout   SFX = "rollout"
	SFXVictory   SFX = "victory"
	SFXTremor    SFX = "tremor"
	SFXCalamity  SFX = "calamity"
	SFXBoat      SFX = "boat"
	SFXChronicle SFX = "chronicle"
	SFXReveal    SFX = "reveal"
	SFXDrums     SFX = "drums"
	SFXHorn      SFX = "horn"
	SFXReady     SFX = "ready"
	SFXPage      SFX = "page"
	SFXGleamLast SFX = "gleam_last"
)

var songBook = map[Music]func() song{
	MusicTitle:       titleSong,
	MusicDark:        darkSong,
	MusicNight:       nightSong,
	MusicFeudal:      feudalSong,
	MusicFeudalNight: feudalNightSong,
	MusicCrisis:      crisisSong,
}

func AllMusic() []Music {
	return []Music{MusicTitle, MusicDark, MusicNight, MusicFeudal, MusicFeudalNight, MusicCrisis}
}

func AllSFX() []SFX {
	return []SFX{
		SFXClick, SFXCoin, SFXBuild, SFXBuilt, SFXPodStart, SFXError, SFXUnlock,
		SFXRollout, SFXVictory, SFXTremor, SFXCalamity, SFXBoat, SFXChronicle, SFXReveal,
		SFXDrums, SFXHorn, SFXReady, SFXPage, SFXGleamLast,
	}
}

func RenderMusic(track Music, rate int) ([]byte, error) {
	if err := checkRate(rate); err != nil {
		return nil, err
	}
	compose, ok := songBook[track]
	if !ok {
		return nil, fmt.Errorf("sound: unknown music %q", track)
	}
	pcm, err := renderSong(compose(), rate)
	if err != nil {
		return nil, fmt.Errorf("sound: render music %q: %w", track, err)
	}
	return pcm, nil
}

func RenderSFX(effect SFX, rate int) ([]byte, error) {
	if err := checkRate(rate); err != nil {
		return nil, err
	}
	return renderSFX(effect, rate)
}

func ThemeFor(age string, dayPhase float64, crisis bool) Music {
	switch {
	case crisis:
		return MusicCrisis
	case age == ageFeudal && isNight(dayPhase):
		return MusicFeudalNight
	case age == ageFeudal:
		return MusicFeudal
	case isNight(dayPhase):
		return MusicNight
	default:
		return MusicDark
	}
}

func isNight(phase float64) bool {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return false
	}
	p := phase - math.Floor(phase)
	return p < nightEnd || p > nightStart
}

func checkRate(rate int) error {
	if rate < minRate || rate > maxRate {
		return fmt.Errorf("sound: unsupported sample rate %d", rate)
	}
	return nil
}

func knownMusic(track Music) bool {
	_, ok := songBook[track]
	return ok
}
