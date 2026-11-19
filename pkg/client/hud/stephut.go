package hud

import "strings"

const hutGoalHead = "pod:"

func (h *HUD) StepHut(f Frame) (string, bool) {
	f = h.Stage(f)
	o, ok := f.current()
	if !ok || f.Guide.Held != "" || f.GoalAt == nil || f.GoalOfNext || h.Voice(f) != VoiceObjective || !strings.Contains(o.Progress, readyMark) {
		return "", false
	}
	goal := h.readyGoal(f)
	if !strings.HasPrefix(goal, hutGoalHead) {
		return "", false
	}
	return goal, !h.goalInView(f)
}
