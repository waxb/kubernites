package hud

const backStep = "cmd:back"

func (h *HUD) stepKey(f Frame) string {
	switch h.Voice(f) {
	case VoiceIntro:
		return VoiceIntro + ":" + f.Guide.Intro.ID
	case VoiceObjective:
		o, _ := f.current()
		return VoiceObjective + ":" + o.ID
	}
	return ""
}

func (h *HUD) returnToGoal(f Frame) {
	key := h.stepKey(f)
	if key == h.Director.step {
		return
	}
	h.Director.step = key
	for range MaxNavDepth {
		path := h.GuidePath(f)
		if h.PageID == PageMain || h.OverlayID != OverlayNone || len(path) == 0 || path[0] != backStep {
			return
		}
		h.PageID = pageParent(h.PageID)
	}
}
