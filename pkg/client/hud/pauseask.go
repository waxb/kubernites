package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	PauseAll      = "all"
	PauseTroubles = "troubles"
	PauseChoices  = "choices"
	PauseOff      = "off"
	AskTrouble    = "trouble"
	AskChoice     = "choice"
)

type PauseAsk struct {
	Kind  string
	Key   string
	Title string
	Why   string
	At    *[2]float64
}

func (h *HUD) SetPauseFor(mode string) { h.pauseFor = PauseMode(mode) }

func (h *HUD) PauseFor() string { return PauseMode(h.pauseFor) }

func PauseMode(mode string) string {
	switch mode {
	case PauseTroubles, PauseChoices, PauseOff:
		return mode
	}
	return PauseAll
}

func NextPauseMode(mode string) string {
	switch PauseMode(mode) {
	case PauseAll:
		return PauseTroubles
	case PauseTroubles:
		return PauseChoices
	case PauseChoices:
		return PauseOff
	}
	return PauseAll
}

func PauseCovers(mode, kind string) bool {
	switch PauseMode(mode) {
	case PauseAll:
		return true
	case PauseTroubles:
		return kind == AskTrouble
	case PauseChoices:
		return kind == AskChoice
	}
	return false
}

func PauseLabel(mode string) string {
	switch PauseMode(mode) {
	case PauseTroubles:
		return "PAUSE FOR: TROUBLES"
	case PauseChoices:
		return "PAUSE FOR: CHOICES"
	case PauseOff:
		return "PAUSE FOR: NOTHING"
	}
	return "PAUSE FOR: BOTH"
}

const pauseTip = "The realm pauses the first time each kind of trouble strikes, and when a choice needs you. TROUBLES, CHOICES, BOTH or NOTHING. Space goes on."

func (h *HUD) askPauseFor(f Frame, id string) {
	for _, v := range f.State.Calamities {
		if v.ID == id && v.First && v.Mended == "" && !h.Director.Stage.asked[id] {
			h.Director.Stage.asked = withKey(h.Director.Stage.asked, id)
			if TroubleFixable(v, f) {
				h.pauseAsks = append(h.pauseAsks, troubleAsk(v, f))
			}
		}
	}
}

func TroubleFixable(v view.CalamityView, f Frame) bool {
	for _, r := range v.Remedies[:min(len(v.Remedies), maxRemedies)] {
		if (r.Command != "" || r.Action != nil) && r.Blocked == "" {
			return true
		}
	}
	inv := v.Invite
	return inv != nil && f.State.Gold >= inv.Gold && f.State.Wisdom >= inv.Wisdom
}

func withKey(set map[string]bool, key string) map[string]bool {
	next := make(map[string]bool, len(set)+1)
	for k, v := range set {
		next[k] = v
	}
	next[key] = true
	return next
}

func troubleAsk(v view.CalamityView, f Frame) PauseAsk {
	ask := PauseAsk{Kind: AskTrouble, Key: "trouble:" + v.ID, Title: model.CauseTitle(v.Kind), Why: v.Cause}
	if at, ok := f.Places[v.Object]; ok {
		ask.At = &at
	}
	return ask
}

func (h *HUD) TakePauseAsks() []PauseAsk {
	out := h.pauseAsks
	h.pauseAsks = nil
	return out
}

func (h *HUD) PauseBlocked(f Frame) bool {
	return f.Introducing() || h.OverlayID == OverlayWelcome || h.QuestCardShown(f) || h.IntroShown(f)
}

func (h *HUD) AskPause(a PauseAsk) { h.pauseAsks = append(h.pauseAsks, a) }

func (h *HUD) FoldToast(old, text string, bad bool) {
	next := make([]Toast, 0, len(h.ToastList)+1)
	kept := Toast{Text: text, Bad: bad}
	for _, t := range h.ToastList {
		if t.Text != old {
			next = append(next, t)
			continue
		}
		kept.News, kept.Fanfare = t.News, t.Fanfare
	}
	h.ToastList = next
	h.pushToastAs(kept)
}
