package hud

import (
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayTrials = "trials"
	IntentTrial   = "trial"
	TrialsButton  = "menu:trials"

	TrialStart = "start"
	TrialBegin = "begin"
	TrialRetry = "retry"
	TrialLeave = "leave"
	TrialPace  = "pace"
	TrialBoard = "board"

	trialPrefix     = "trial"
	TrialPageList   = "list"
	TrialPageBrief  = "brief"
	TrialPageResult = "result"
	TrialPaceMax    = 3
)

type TrialView struct {
	Trial   view.Trial
	Status  view.TrialStatus
	Elapsed float64
	Verdict view.TrialVerdict
	Result  view.TrialResult
	Best    view.TrialScore
	HasBest bool
	NewBest bool
	Surge   int
	Gaps    int
}

type TrialPaceOf struct {
	Surge, Gaps int  `json:"-"`
	Stepped     bool `json:"-"`
}

type trialState struct {
	scores view.TrialScores
	PageID string `json:"-"`
	Focus  string `json:"-"`
	view   *TrialView
	Hint   bool        `json:"-"`
	Pace   TrialPaceOf `json:"-"`
}

func (h *HUD) SetTrialScores(s view.TrialScores) { h.Trial.scores = s }

func (h *HUD) SetTrial(v *TrialView) {
	if v == nil {
		h.Trial.view = nil
		return
	}
	next := *v
	h.Trial.view = &next
	if !h.Trial.Pace.Stepped {
		h.Trial.Pace = TrialPaceOf{Surge: v.Surge, Gaps: v.Gaps}
	}
}

func (h *HUD) InTrial() bool { return h.Trial.view != nil }

func (h *HUD) trialShown() bool { return h.Trial.view != nil }

func (h *HUD) TrialPage() string {
	if h.OverlayID != OverlayTrials {
		return ""
	}
	return h.Trial.PageID
}

func (h *HUD) OpenTrials(focus string) {
	if _, ok := view.FindTrial(focus); !ok {
		focus = view.Content.Trials()[0].ID
	}
	h.Trial.PageID, h.Trial.Focus = TrialPageList, focus
	h.OverlayID = OverlayTrials
}

func (h *HUD) OpenTrialBrief() {
	h.Trial.PageID, h.Trial.Hint = TrialPageBrief, false
	h.Trial.Pace = TrialPaceOf{}
	if v := h.Trial.view; v != nil {
		h.Trial.Pace = TrialPaceOf{Surge: v.Surge, Gaps: v.Gaps}
	}
	h.OverlayID = OverlayTrials
}

func (h *HUD) OpenTrialResult() {
	h.Trial.PageID = TrialPageResult
	h.OverlayID = OverlayTrials
}

func trialIntent(cmd, target string) Intent {
	return Intent{Kind: IntentTrial, Command: cmd, Target: target}
}

func (h *HUD) PressTrial(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	verb, arg, _ := strings.Cut(rest, ":")
	switch verb {
	case "pick":
		h.Trial.Focus = arg
		return []Intent{click}
	case TrialStart:
		h.OverlayID = OverlayNone
		return []Intent{click, trialIntent(TrialStart, arg)}
	case TrialBegin:
		h.OverlayID = OverlayNone
		return []Intent{click, trialIntent(TrialBegin, "")}
	case TrialRetry, TrialLeave:
		h.OverlayID = OverlayNone
		return []Intent{click, trialIntent(verb, "")}
	case TrialBoard:
		h.OpenTrials(arg)
		return []Intent{click}
	case "brief":
		h.Trial.PageID = TrialPageBrief
		h.OverlayID = OverlayTrials
		return []Intent{click}
	case "result":
		h.OpenTrialResult()
		return []Intent{click}
	case "close":
		h.OverlayID = OverlayNone
		return []Intent{click}
	case "hint":
		h.Trial.Hint = !h.Trial.Hint
		return []Intent{click}
	case "step":
		h.Trial.Pace = SteppedPace(h.Trial.Pace, arg)
		return []Intent{click}
	case TrialPace:
		p := h.Trial.Pace
		h.Trial.Pace.Stepped = false
		return []Intent{click, trialIntent(TrialPace, strconv.Itoa(p.Surge)+":"+strconv.Itoa(p.Gaps))}
	}
	return []Intent{click}
}

func SteppedPace(p TrialPaceOf, arg string) TrialPaceOf {
	field, dir, _ := strings.Cut(arg, ":")
	d := 1
	if dir == "down" {
		d = -1
	}
	next := p
	next.Stepped = true
	switch field {
	case "surge":
		next.Surge = min(max(p.Surge+d, 0), TrialPaceMax)
	case "gaps":
		next.Gaps = min(max(p.Gaps+d, 0), TrialPaceMax)
	}
	return next
}

func PaceTarget(target string) (int, int, bool) {
	a, b, ok := strings.Cut(target, ":")
	surge, err1 := strconv.Atoi(a)
	gaps, err2 := strconv.Atoi(b)
	if !ok || err1 != nil || err2 != nil || surge < 0 || gaps < 0 || surge+gaps == 0 {
		return 0, 0, false
	}
	return surge, gaps, true
}

func trialText(t view.TrialText) string { return t.In(ui.Naming()) }

func (h *HUD) trialFocus() view.Trial {
	if t, ok := view.FindTrial(h.Trial.Focus); ok {
		return t
	}
	return view.Content.Trials()[0]
}

func (h *HUD) trialOf() view.Trial {
	if v := h.Trial.view; v != nil {
		return v.Trial
	}
	return h.trialFocus()
}
