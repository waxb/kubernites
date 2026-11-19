package app

import "github.com/waxb/kubernites/pkg/view"

const stepHoldSeconds = 10.0

type stepHold struct {
	id, shape string
	at        float64
	view      view.ObjectiveView
	fresh     bool
	shown     map[string]float64
}

func stepShape(v view.ObjectiveView) string {
	return v.ID + "|" + v.Text + "|" + v.Target + "|" + v.Button + "|" + digitRun.ReplaceAllString(v.Command, "#") + "|" + digitRun.ReplaceAllString(v.Progress, "#")
}

func (h *stepHold) release() {
	h.fresh = true
}

func (h *stepHold) returning(shape string, now float64) bool {
	at, ok := h.shown[shape]
	return ok && now-at < stepHoldSeconds
}

func (h *stepHold) steady(views []view.ObjectiveView, now float64) []view.ObjectiveView {
	n := len(views)
	if n == 0 || views[n-1].Done {
		*h = stepHold{}
		return views
	}
	v := views[n-1]
	shape := stepShape(v)
	switch {
	case h.id != v.ID || h.fresh || shape == h.shape || now-h.at >= stepHoldSeconds || !h.returning(shape, now):
		if h.id != v.ID {
			h.shown = map[string]float64{}
		}
		if shape != h.shape || h.id != v.ID {
			if h.shape != "" && h.shown != nil {
				h.shown[h.shape] = now
			}
			h.at = now
		}
		h.id, h.shape, h.view, h.fresh = v.ID, shape, v, false
		return views
	}
	out := append([]view.ObjectiveView(nil), views...)
	out[n-1] = h.view
	return out
}
