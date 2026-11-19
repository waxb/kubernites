package hud

import "github.com/waxb/kubernites/pkg/view"

func (f Frame) StoneStands() bool {
	return f.Revealed("stone") && f.Shown(SurfaceStone)
}

func (h *HUD) StoneAsleep() []Intent {
	if h.cardAsksForStone() {
		h.OpenStone(h.Yard.View.Chip)
		return nil
	}
	if h.InChapter() {
		h.Toast(StoneLater, false)
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	h.Toast(stoneSleeps, false)
	return []Intent{{Kind: IntentSound, Command: "error"}, TutorialIntent(TutorialNote, view.TutorialNoteDormant)}
}

func (h *HUD) OpenStoneFromWorld(f Frame) []Intent {
	if !f.StoneStands() {
		return nil
	}
	h.OpenStone(f.SpeakCommand())
	return h.IntroTried(f, SurfaceStone)
}
