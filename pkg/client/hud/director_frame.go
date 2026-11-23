package hud

import (
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

const StepMetLine = "done: STEP DONE comes after the card up now"

var (
	introAliases  = map[string]string{"academy": "cmd:soulvault", "speed": "surface:speed"}
	introPrefixes = []string{"cmd:", "surface:", "meter:"}
)

type rawParts struct {
	guide   Guide
	notices []Notice
	reveals []string
}

func IntroTargets(ids []string) []string {
	out := make([]string, 0, len(ids)*len(introPrefixes)+len(ids))
	for _, id := range ids {
		out = append(out, id)
		if alias, ok := introAliases[id]; ok {
			out = append(out, alias)
		}
		for _, p := range introPrefixes {
			out = append(out, p+id)
		}
	}
	return out
}

func introBase(target string) string {
	for id, alias := range introAliases {
		if alias == target {
			return id
		}
	}
	for _, p := range introPrefixes {
		if rest, ok := strings.CutPrefix(target, p); ok {
			return rest
		}
	}
	return target
}

func (f Frame) RawFrame() Frame {
	if f.raw == nil {
		return f
	}
	out := f
	out.Guide, out.Notices = f.raw.guide, f.raw.notices
	out.State.Reveals = f.raw.reveals
	out.raw = nil
	return out
}

func (h *HUD) Stage(f Frame) Frame {
	if h.live != nil {
		f.Live, f.LiveCouncil = true, h.live.Council
	}
	raw := h.withUnreadTrouble(f.RawFrame())
	if !h.Director.Stage.Primed {
		return h.withBoard(raw)
	}
	return h.withBoard(h.stagedWith(raw, "", ""))
}

func (h *HUD) stagedWith(raw Frame, reveal, known string) Frame {
	out := raw
	out.raw = &rawParts{guide: raw.Guide, notices: raw.Notices, reveals: raw.State.Reveals}
	out.Guide = h.stageGuide(raw.Guide, raw.State, known)
	out.Notices = append(h.stagedNotices(raw.Notices), h.shelvedNotices(raw)...)
	if !out.Shown(SurfaceRail) {
		out.Notices = slices.DeleteFunc(out.Notices, func(n Notice) bool { return !n.Timely })
	}
	out.State.Reveals = h.stagedReveals(raw.State.Reveals, reveal)
	return out
}

func (h *HUD) StageGuide(g Guide, st view.GameState) Guide {
	if !h.Director.Stage.Primed {
		return g
	}
	return h.stageGuide(g, st, "")
}

func (h *HUD) stageGuide(g Guide, st view.GameState, known string) Guide {
	s := h.Director.Stage
	out := g
	out.Introduced = h.withShown(h.stagedIntroduced(g.Introduced, known))
	if in := g.Intro; in != nil && !s.Has(actIntro+in.ID) {
		out.Intro, out.Queue = nil, nil
	}
	o, ok := currentOf(st)
	switch {
	case ok && s.StepOn == o.ID && (s.Has(ActStep+o.ID) || s.DoneOn != s.StepOn):
		return out
	case s.StepOn == "" && !ok:
		return out
	case ok:
		out.Held = o.ID
	default:
		out.Held = s.StepOn
	}
	out.Target, out.Button, out.Why = "", "", ""
	if last, found := h.LastStep(st); found {
		out.Last = &last
	}
	return out
}

func currentOf(st view.GameState) (view.ObjectiveView, bool) {
	for _, o := range st.Objectives {
		if !o.Done {
			return o, true
		}
	}
	return view.ObjectiveView{}, false
}

func (h *HUD) LastStep(st view.GameState) (view.ObjectiveView, bool) {
	s := h.Director.Stage
	if s.StepOn == "" {
		return view.ObjectiveView{}, false
	}
	for _, o := range st.Objectives {
		if o.ID == s.StepOn {
			if o.Done && s.DoneOn != s.StepOn {
				v := s.View
				v.Progress = StepMetLine
				return v, true
			}
			if !o.Done && s.DoneOn == s.StepOn && !s.Has(ActStep+o.ID) {
				v := s.View
				v.Done = true
				return v, true
			}
			if o.Done && o.Lesson == "" {
				o.Lesson = s.View.Lesson
			}
			return o, true
		}
	}
	v := s.View
	v.Done = s.DoneOn == s.StepOn
	return v, true
}

func (h *HUD) stagedIntroduced(list []string, known string) []string {
	s := h.Director.Stage
	out := make([]string, 0, len(list))
	for _, t := range list {
		base := introBase(t)
		if s.known[base] || base == known {
			out = append(out, t)
		}
	}
	return out
}

func (h *HUD) stagedReveals(list []string, reveal string) []string {
	s := h.Director.Stage
	out := make([]string, 0, len(list))
	for _, id := range list {
		if s.revealed[id] || id == reveal {
			out = append(out, id)
		}
	}
	return out
}

func (h *HUD) stagedNotices(list []Notice) []Notice {
	s := h.Director.Stage
	out := make([]Notice, 0, len(list))
	for _, n := range list {
		if n.Quiet || s.Has(ActNotice+n.ID) {
			out = append(out, n)
		}
	}
	return out
}

func (h *HUD) revealActs(f Frame) []stageAct {
	out := []stageAct{}
	for _, id := range h.pendingReveals(f) {
		out = append(out, stageAct{key: actReveal + id, prio: prioReveal})
	}
	for _, base := range h.pendingKnown(f) {
		out = append(out, stageAct{key: actKnown + base, prio: prioReveal})
	}
	return out
}

func (h *HUD) pendingReveals(f Frame) []string {
	out := []string{}
	for _, id := range f.State.Reveals {
		if !h.Director.Stage.revealed[id] && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func (h *HUD) pendingKnown(f Frame) []string {
	out := []string{}
	for _, t := range f.Guide.Introduced {
		base := introBase(t)
		if !h.Director.Stage.known[base] && !slices.Contains(out, base) {
			out = append(out, base)
		}
	}
	return out
}

func (h *HUD) admitSilent(f Frame) {
	s := &h.Director.Stage
	if len(h.pendingReveals(f))+len(h.pendingKnown(f)) == 0 {
		return
	}
	sig := silentSig{reveals: len(f.State.Reveals), known: len(f.Guide.Introduced), admitted: len(s.revealed) + len(s.known), page: h.PageID, overlay: h.OverlayID}
	if sig == s.silent {
		return
	}
	s.silent = sig
	before := h.revealSet(h.stagedWith(f, "", ""))
	for _, id := range h.pendingReveals(f) {
		if sameSet(before, h.revealSet(h.stagedWith(f, id, ""))) {
			s.revealed[id] = true
		}
	}
	for _, base := range h.pendingKnown(f) {
		if sameSet(before, h.revealSet(h.stagedWith(f, "", base))) {
			s.known[base] = true
		}
	}
}

type silentSig struct {
	reveals, known, admitted int
	page, overlay            string
}

func (h *HUD) revealSet(f Frame) map[string]bool {
	out := map[string]bool{}
	for _, k := range h.RevealKeys(f) {
		out[k] = true
	}
	return out
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func (h *HUD) admitNamed(f Frame, goal string) {
	s := &h.Director.Stage
	ids, reveals := namedReach(goal)
	for _, id := range ids {
		base := introBase(id)
		s.known[base] = true
		s.revealed[base] = true
		h.noteShown(id)
	}
	for _, r := range reveals {
		s.revealed[r] = true
	}
}

func namedReach(goal string) ([]string, []string) {
	if goal == "" {
		return nil, nil
	}
	ids := append([]string{goal}, NavPath(goal, NavState{Page: PageMain})...)
	reveals := []string{}
	for _, id := range ids {
		for _, e := range mainEntries {
			if e.button.ID == id {
				reveals = append(reveals, e.reveal)
			}
		}
		for _, t := range wellTabs {
			if t.surface == id {
				reveals = append(reveals, t.reveal)
			}
		}
	}
	return ids, reveals
}

func (h *HUD) heldReach(f Frame) map[string]bool {
	o, ok := f.current()
	if !ok || o.ID == h.Director.Stage.StepOn {
		return nil
	}
	out := map[string]bool{}
	for _, goal := range []string{ObjectiveTarget(o), o.Button} {
		ids, reveals := namedReach(goal)
		for _, id := range ids {
			out[introBase(id)] = true
		}
		for _, r := range reveals {
			out[r] = true
		}
	}
	return out
}
