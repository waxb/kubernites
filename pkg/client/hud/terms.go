package hud

import (
	"strings"
	"sync"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	termKeyPrefix = "term:"
	termGap       = 20.0
)

type termHint struct {
	Key     string `json:"-"`
	needles []string
	title   string
	Term    string `json:"-"`
	staging bool
}

var TermHints = []termHint{
	{termKeyPrefix + "burning", []string{"Burning"}, "BURNING", "Burning", false},
	{termKeyPrefix + "kindling", []string{"Kindling"}, "KINDLING", "Kindling", false},
	{termKeyPrefix + "unhoused", []string{"Unhoused"}, "UNHOUSED", "Unhoused", false},
	{termKeyPrefix + "banished", []string{"Banished"}, "BANISHED", "Banished", true},
	{termKeyPrefix + "forum", []string{"Forum"}, "THE FORUM", "Forum", false},
	{termKeyPrefix + "rallycall", []string{"badge it looks for"}, "THE BADGE IT LOOKS FOR", "badge it looks for", false},
	{termKeyPrefix + "badge", []string{"badge", "Badge"}, "BADGE", "badge", false},
	{termKeyPrefix + "pitch", []string{"Pitch"}, "THE HORN'S PITCH", "Pitch", false},
	{termKeyPrefix + "guttering", []string{"Guttering"}, "GUTTERING", "Guttering", false},
	{termKeyPrefix + "tremor", []string{"Tremor"}, "THE IMP'S TREMOR", "Tremor", false},
	{termKeyPrefix + "voidtaken", []string{"Voidtaken"}, "VOIDTAKEN", "Voidtaken", false},
	{termKeyPrefix + "unfit", []string{"Unfit for Duty"}, "UNFIT FOR DUTY", "Unfit for Duty", true},
	{termKeyPrefix + "pressgang", []string{"Press-Gang"}, "THE PRESS-GANG", "Press-Gang", true},
}

func (t termHint) lowerNeedles() []string {
	out := make([]string, 0, len(t.needles))
	for _, n := range t.needles {
		out = append(out, strings.ToLower(n))
	}
	return out
}

func (h *HUD) guidanceText(f Frame) string {
	parts := []string{}
	if o, ok := f.ShownObjective(); ok && h.ObjectiveShown(f) {
		parts = append(parts, o.Text, o.Progress)
	}
	if in := f.Guide.Intro; in != nil && h.IntroShown(f) {
		parts = append(parts, in.Title, in.Purpose, in.Try)
	}
	if v, ok := h.ShownCalamity(f); ok && h.CardsDrawn(f) {
		parts = append(parts, v.Cause, v.Why)
		for _, r := range v.Remedies {
			parts = append(parts, r.Label)
		}
	}
	return strings.Join(parts, "\n")
}

func (t termHint) Text() string {
	if word, ok := lexicon.PlainOf(t.Term); ok {
		return word.Sentence()
	}
	return ""
}

type termWatch struct {
	mu   sync.Mutex
	seen map[string]bool
}

func (w *termWatch) note(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.seen == nil {
		w.seen = map[string]bool{}
	}
	w.seen[key] = true
}

func (w *termWatch) take() map[string]bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.seen
	w.seen = nil
	return out
}

func (h *HUD) SeeText(text string) {
	if !ui.StoryNaming() || h.OverlayID != OverlayNone {
		return
	}
	for _, t := range TermHints {
		if h.taught[t.Key] {
			continue
		}
		for _, n := range t.needles {
			if strings.Contains(text, n) {
				h.terms.note(t.Key)
				break
			}
		}
	}
}

func (h *HUD) stepTerms(f Frame) {
	seen := h.terms.take()
	if len(seen) == 0 || !h.teachReady(f) || len(h.Results) > 0 || h.termAt > 0 && h.Now-h.termAt < termGap {
		return
	}
	guide := h.guidanceText(f)
	for _, t := range TermHints {
		if !seen[t.Key] || h.taught[t.Key] || t.staging && f.State.Age == view.AgeDark || t.Text() == "" || !namesAny([]string{guide}, t.lowerNeedles()) {
			continue
		}
		h.taughtDirty = h.teach1(t.Key, t.title, t.Text()) || h.taughtDirty
		h.termAt = max(h.Now, 1e-9)
		return
	}
}
