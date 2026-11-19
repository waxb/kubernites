package hud

import (
	"maps"
	"slices"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	teachMaxLines = 6
	TeachTopBar   = "topbar"
	topBarDelay   = 6.0
)

func (h *HUD) Taught() []string {
	return slices.Sorted(maps.Keys(h.taught))
}

func (h *HUD) SetTaught(keys []string) {
	h.taught = map[string]bool{}
	for _, k := range keys {
		h.taught[k] = true
	}
}

func (h *HUD) teach1(key, title, text string) bool {
	if h.taught[key] {
		return false
	}
	if h.taught == nil {
		h.taught = map[string]bool{}
	}
	h.taught[key] = true
	h.LessonCard = LessonCard{Title: title, Text: text, start: h.Now, Shown: true, Teach: true}
	return true
}

const GildedTaught = "gilded"

func (h *HUD) ResultOnce(key, title, text string, at *[2]float64) {
	if h.taught[key] {
		return
	}
	if h.taught == nil {
		h.taught = map[string]bool{}
	}
	h.taught[key], h.taughtDirty = true, true
	h.ResultAt(title, text, at)
}

func (h *HUD) ResultOnceWhile(key, title, text string, at *[2]float64, still func() bool) {
	if h.taught[key] {
		return
	}
	h.ResultAt(title, text, at)
	timely := func(c *LessonCard) {
		if c.Title == title {
			c.still, c.timely, c.once = still, true, key
		}
	}
	if h.LessonCard.Shown {
		timely(&h.LessonCard)
	}
	next := slices.Clone(h.Results)
	for i := range next {
		timely(&next[i])
	}
	h.Results = next
}

func (h *HUD) markTaught(key string) {
	if key == "" || h.taught[key] {
		return
	}
	if h.taught == nil {
		h.taught = map[string]bool{}
	}
	h.taught[key], h.taughtDirty = true, true
}

func (h *HUD) teachReady(f Frame) bool {
	return h.OverlayID == OverlayNone && !f.Introducing() && !h.LessonCard.Shown && !h.QuestCardShown(f) && !f.Unguided && !f.State.Enjoying && f.Guide.Held == ""
}

func (h *HUD) Result(title, text string) { h.ResultAt(title, text, nil) }

func (h *HUD) ResultWhile(title, text string, still func() bool) {
	h.ResultAt(title, text, nil)
	next := slices.Clone(h.Results)
	for i, r := range next {
		if r.Title == title {
			next[i].still = still
		}
	}
	h.Results = next
}

func (h *HUD) pruneResults() {
	h.Results = slices.DeleteFunc(slices.Clone(h.Results), LessonCard.stale)
}

func (h *HUD) ResultAt(title, text string, at *[2]float64) {
	if h.LessonCard.Shown && h.LessonCard.hold && h.LessonCard.Title == title {
		h.LessonCard.Text, h.LessonCard.at = text, at
		return
	}
	next := slices.Clone(h.Results)
	for i, r := range next {
		if r.Title == title {
			next[i].Text, next[i].at = text, at
			h.Results = next
			return
		}
	}
	h.Results = append(next, LessonCard{Title: title, Text: text, hold: true, at: at})
}

func (h *HUD) nextResult() {
	h.pruneResults()
	if h.LessonCard.Shown && h.LessonCard.stale() && !h.Director.Stage.Has(actLesson+h.LessonCard.Title) {
		h.LessonCard = LessonCard{}
	}
	if len(h.Results) == 0 {
		return
	}
	if h.LessonCard.Shown {
		return
	}
	next := h.Results[0]
	next.start, next.Shown = h.Now, true
	h.LessonCard, h.Results = next, slices.Clone(h.Results[1:])
}

func (h *HUD) afterUpdate(f Frame) {
	h.nextResult()
	h.stepTerms(f)
	h.watchDawn(f)
	if h.taught[TeachTopBar] || !f.Shown(MeterGold) || NetGoldRate(f.State) == 0 {
		h.rateSince = 0
		return
	}
	if h.rateSince == 0 {
		h.rateSince = max(h.Now, 1e-9)
	}
	if h.Now-h.rateSince < topBarDelay || !h.teachReady(f) {
		return
	}
	h.taughtDirty = h.teach1(TeachTopBar, TopBarTitle(), topBarText()) || h.taughtDirty
}

func (h *HUD) TaughtChanged() bool {
	changed := h.taughtDirty
	h.taughtDirty = false
	return changed
}

func TopBarTitle() string { return "GOLD, INCOME AND UPKEEP" }

func topBarText() string {
	return ui.Bridge(
		"The number is your gold. The +/S beside it is income: the gold your boxes earn each second. From the Feudal Age on a red -/S joins it: upkeep, what your Keeps cost each second. Income minus upkeep is what your gold really gains.",
		"The number is your money. The +/S beside it is income: the revenue your requests earn each second. In Staging a red -/S joins it: upkeep, what your Nodes cost each second. Income minus upkeep is what your balance really gains.")
}

func VeteranTips() []string {
	out := []string{TeachTopBar}
	for _, t := range TermHints {
		if !t.staging {
			out = append(out, t.Key)
		}
	}
	return out
}
