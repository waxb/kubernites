package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	NextStepID    = "obj:next"
	NextStepLabel = "NEXT STEP"
	NextStepRowH  = 20
	nextStepW     = 84
)

func (h *HUD) nextStepShown(f Frame) bool {
	if h.teacherShown() || h.trialShown() || h.QuestCardShown(f) {
		return false
	}
	o, ok := f.ShownObjective()
	return ok && o.Done && (f.State.Enjoying || f.Guide.Held != "")
}

func (h *HUD) nextStepButton(f Frame, card ui.Rect) []Button {
	if !h.nextStepShown(f) {
		return nil
	}
	r := ui.Rect{X: card.X + card.W - nextStepW - 6, Y: card.Y + card.H - NextStepRowH + 2, W: nextStepW, H: NextStepRowH - 6}
	return []Button{{ID: NextStepID, Rect: r, Label: NextStepLabel}}
}

func (h *HUD) pressNextStep(f Frame) []Intent {
	h.Director.Stage.pulled = h.Now
	key := ""
	if o, ok := h.Stage(f).ShownObjective(); ok && o.Done {
		key = actDone + o.ID
	}
	h.NoteAttention(EventNextStep, key, f.State.RealmTime)
	h.noteAsk(key, f.State.RealmTime)
	out := []Intent{{Kind: IntentSound, Command: "click"}}
	if f.State.Enjoying || f.State.Held {
		out = append(out, act(view.Action{Action: view.ActNextStep}))
	}
	return out
}

func (h *HUD) drawNextStep(dst *ebiten.Image, f Frame, card ui.Rect) {
	for _, b := range h.nextStepButton(f, card) {
		h.drawPrimary(dst, b)
	}
}
