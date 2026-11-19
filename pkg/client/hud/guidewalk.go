package hud

import (
	"fmt"
	"maps"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	maxWalk   = 8
	readyMark = "ready:"
)

type GuideWalk struct {
	Voice   string
	Goal    string
	Steps   []string
	Problem string
}

func (w GuideWalk) OK() bool { return w.Problem == "" }

func (h *HUD) WalkGuide(f Frame) GuideWalk {
	return h.walk(f, h.Voice(f), h.GuideGoal)
}

func (h *HUD) WalkReady(f Frame) GuideWalk {
	o, ok := f.current()
	if !ok || f.Guide.Held != "" || !strings.Contains(o.Progress, readyMark) || !h.ObjectiveDrawn(f) || f.Unguided {
		return GuideWalk{}
	}
	w := h.walk(f, VoiceObjective, h.readyGoal)
	if w.Goal == "" {
		w.Problem = "the ready line points at nothing to press"
	}
	return w
}

func (h *HUD) readyGoal(f Frame) string {
	return gridGoal(h.chooserGoal(ObjectiveGoal(f), f), f)
}

func (h *HUD) walk(f Frame, voice string, goal func(Frame) string) GuideWalk {
	saved := *h
	saved.Director.Stage, saved.Ledger = h.Director.Stage.clone(), h.Ledger.clone()
	saved.taught, saved.questSeen = maps.Clone(h.taught), maps.Clone(h.questSeen)
	saved.top.stats = maps.Clone(h.top.stats)
	defer func() { *h = saved }()
	h.attn.on = false
	w := GuideWalk{Voice: voice, Goal: goal(f)}
	if w.Goal == "" {
		return w
	}
	for range maxWalk {
		s, ok := h.stepOn(NavPath(goal(f), h.NavState(f)), f)
		if !ok {
			w.Problem = h.unresolved(w.Goal, f)
			return w
		}
		w.Steps = append(w.Steps, s.ID)
		if w.Problem = h.StepProblem(s, w.Goal, f); w.Problem != "" || s.Final || s.Map {
			return w
		}
		h.Update(Input{MX: s.Rect.X + s.Rect.W/2, MY: s.Rect.Y + s.Rect.H/2, Click: true}, f)
	}
	w.Problem = "the path never arrives"
	return w
}

func (h *HUD) unresolved(goal string, f Frame) string {
	target := goal
	if alias, ok := NavAliases[goal]; ok {
		target = alias
	}
	switch {
	case Arrived(target, h.NavState(f)):
		return ""
	case MapTarget(goal) && f.GoalAt != nil:
		return ""
	case MapTarget(goal):
		return "its place on the map is unknown"
	}
	return fmt.Sprintf("nothing on screen answers it (page %s, window %q)", h.PageID, h.OverlayID)
}

func OnScreen(r ui.Rect) bool {
	return r.W > 0 && r.H > 0 && r.X >= 0 && r.Y >= 0 && r.X+r.W <= Width && r.Y+r.H <= Height
}

func (h *HUD) StepProblem(s GuideStep, goal string, f Frame) string {
	if s.Map {
		return ""
	}
	if !OnScreen(s.Rect) {
		return fmt.Sprintf("%s at %+v is off the screen", s.ID, s.Rect)
	}
	if what, ok := h.CoveredBy(s.Rect, f); ok {
		return fmt.Sprintf("%s is covered by %s", s.ID, what)
	}
	return h.ClickProblem(s, goal, f)
}

func (h *HUD) ClickProblem(s GuideStep, goal string, f Frame) string {
	buttons := h.Buttons(f)
	hit, ok := HitButton(buttons, s.Rect.X+s.Rect.W/2, s.Rect.Y+s.Rect.H/2)
	_, isButton := HitID(buttons, s.ID)
	switch {
	case ok && hit.ID == s.ID && hit.Disabled:
		return fmt.Sprintf("%s is disabled: %s", s.ID, FirstLine(hit.Tip))
	case ok && hit.ID == s.ID:
		return ""
	case isButton:
		return fmt.Sprintf("%s lies under %q", s.ID, hit.ID)
	case !s.Final || h.actsOn(goal, f) && !(ok && Within(hit.Rect, s.Rect.Inset(-1))):
		return fmt.Sprintf("%s is no button the player can press", s.ID)
	}
	return ""
}

func (h *HUD) actsOn(goal string, f Frame) bool {
	in := f.Guide.Intro
	return in != nil && in.Target == goal && !Passive(*in)
}

func Within(r, box ui.Rect) bool {
	return r.X >= box.X && r.Y >= box.Y && r.X+r.W <= box.X+box.W && r.Y+r.H <= box.Y+box.H
}

func (h *HUD) CoveredBy(r ui.Rect, f Frame) (string, bool) {
	if h.OverlayID != OverlayNone && !Within(r, h.OverlayRect()) && Overlaps(r, h.OverlayRect()) {
		return "the open window " + h.OverlayID, true
	}
	if h.IntroShown(f) && Overlaps(r, h.IntroCard(f)) {
		return "the intro card", true
	}
	if b, ok := h.BannerRect(f); ok && Overlaps(r, b) {
		return "the banner " + h.Juice.Banner, true
	}
	for i, t := range h.ToastRects(f) {
		if Overlaps(r, t) {
			return fmt.Sprintf("the toast %q", h.ToastList[i].Text), true
		}
	}
	return "", false
}
