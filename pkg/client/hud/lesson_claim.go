package hud

import "github.com/waxb/kubernites/pkg/client/ui"

func (h *HUD) ResultClaim(key, title, text string, at *[2]float64, still func() bool, claim Intent) {
	h.ResultOnceWhile(key, title, text, at, still)
	if h.taught[key] {
		return
	}
	if h.claims == nil {
		h.claims = map[string]Intent{}
	}
	h.claims[title] = claim
}

func (h *HUD) lessonAckButton(face ui.Rect) Button {
	if _, ok := h.claims[h.LessonCard.Title]; ok {
		return Button{ID: LessonAck, Rect: face, Label: "CLAIM", Tip: "Claim one gleaming box now: the others fade"}
	}
	return Button{ID: LessonAck, Rect: face, Label: "GOT IT"}
}

func (h *HUD) claimLesson(rest string) ([]Intent, bool) {
	claim, ok := h.claims[h.LessonCard.Title]
	if rest != "ack" || !ok {
		return nil, false
	}
	delete(h.claims, h.LessonCard.Title)
	h.LessonCard.Shown = false
	return []Intent{{Kind: IntentSound, Command: "click"}, claim}, true
}
