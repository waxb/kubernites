package hud

import (
	"fmt"
	"slices"
)

const unguidedNote = "New: %s. %s"

func (h *HUD) passUnguided(f Frame) []Intent {
	f = f.RawFrame()
	if !f.Unguided || !f.Introducing() || h.unguidedAck == f.Guide.Intro.ID {
		return nil
	}
	in := *f.Guide.Intro
	h.unguidedAck = in.ID
	h.queueNews(Toast{Text: fmt.Sprintf(unguidedNote, in.Title, in.Purpose), News: true, Opens: in.Target})
	return []Intent{act(Ack(in.ID))}
}

func (h *HUD) dropUsedNews(id string, f Frame) {
	used := func(t Toast) bool {
		return t.Opens != "" && (Opens(t.Opens, id) || id == ObjDo && t.Opens == f.Guide.Target)
	}
	h.Director.Stage.toasts = slices.DeleteFunc(slices.Clone(h.Director.Stage.toasts), used)
	h.ToastList = slices.DeleteFunc(slices.Clone(h.ToastList), used)
}
