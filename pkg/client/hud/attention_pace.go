package hud

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	EventAsk      = "ask"
	EventMet      = "met"
	EventCredited = "credited"

	SilenceMost = 1.0
	PressMost   = 1.0
	MetMost     = 1.0
	IdleMost    = 60.0

	BusyPanel = "panel"
	BusyBuild = "build"
	BusyWatch = "watch"

	RuleSilence = "a chapter with no Teacher card, step card or lesson on screen for more than 1 s"
	RulePress   = "a card more than 1 s after NEXT STEP, NEXT, GOT IT or BEGIN"
	RuleMet     = "a step met more than 1 s before its DONE is on screen"
	RuleIdle    = "more than 60 s with nothing to do or watch"
)

type PaceMark struct {
	At      float64 `json:"at"`
	Realm   float64 `json:"realm"`
	Card    bool    `json:"card,omitempty"`
	Busy    string  `json:"busy,omitempty"`
	Idle    bool    `json:"idle,omitempty"`
	Chapter bool    `json:"chapter,omitempty"`
	Speed   float64 `json:"speed,omitempty"`
}

type PaceSpan struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Realm float64 `json:"realm"`
	Key   string  `json:"key,omitempty"`
	Open  bool    `json:"open,omitempty"`
}

func (s PaceSpan) Seconds() float64 { return s.To - s.From }

func (s PaceSpan) String() string {
	sec := int(s.Realm)
	open := ""
	if s.Open {
		open = ", never came"
	}
	return fmt.Sprintf("%02d:%02d %.1f s %s%s", sec/60, sec%60, s.Seconds(), s.Key, open)
}

type Pace struct {
	Silences []PaceSpan `json:"silences"`
	Presses  []PaceSpan `json:"presses"`
	Credits  []PaceSpan `json:"credits"`
	Idles    []PaceSpan `json:"idles"`
}

func Longest(spans []PaceSpan) PaceSpan {
	if len(spans) == 0 {
		return PaceSpan{}
	}
	return slices.MaxFunc(spans, func(a, b PaceSpan) int {
		switch {
		case a.Seconds() < b.Seconds():
			return -1
		case a.Seconds() > b.Seconds():
			return 1
		}
		return 0
	})
}

func Over(spans []PaceSpan, most float64) []PaceSpan {
	out := []PaceSpan{}
	for _, s := range spans {
		if s.Seconds() > most+1e-6 {
			out = append(out, s)
		}
	}
	return out
}

func Mean(spans []PaceSpan) float64 {
	if len(spans) == 0 {
		return 0
	}
	sum := 0.0
	for _, s := range spans {
		sum += s.Seconds()
	}
	return sum / float64(len(spans))
}

func (p Pace) Summary() string {
	line := func(name string, spans []PaceSpan, most float64) string {
		l := Longest(spans)
		return fmt.Sprintf("%s: %d, mean %.1f s, longest %.1f s (%s), over %.0f s: %d", name, len(spans), Mean(spans), l.Seconds(), strings.TrimSpace(l.String()), most, len(Over(spans, most)))
	}
	return strings.Join([]string{
		line("silence", p.Silences, SilenceMost),
		line("press to card", p.Presses, PressMost),
		line("met to credited", p.Credits, MetMost),
		line("nothing to do or watch", p.Idles, IdleMost),
	}, "\n")
}

func (p Pace) Faults() []AttnFault {
	out := []AttnFault{}
	add := func(rule string, spans []PaceSpan) {
		for _, s := range spans {
			out = append(out, AttnFault{Rule: rule, At: s.From, Realm: s.Realm, Keys: []string{fmt.Sprintf("%.1f s %s", s.Seconds(), s.Key)}})
		}
	}
	add(RuleSilence, Over(p.Silences, SilenceMost))
	add(RulePress, Over(p.Presses, PressMost))
	add(RuleMet, Over(p.Credits, MetMost))
	add(RuleIdle, Over(p.Idles, IdleMost))
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

func MeasurePace(log AttnLog, end float64) Pace {
	return Pace{
		Silences: silences(log.Marks, end),
		Presses:  pressSpans(log, end),
		Credits:  creditSpans(log.Events, end),
		Idles:    idleSpans(log.Marks, end),
	}
}

func markSpans(marks []PaceMark, end float64, in func(PaceMark) bool) []PaceSpan {
	out := []PaceSpan{}
	open := -1
	for i, m := range marks {
		switch {
		case in(m) && open < 0:
			open = i
		case !in(m) && open >= 0:
			out = append(out, PaceSpan{From: marks[open].At, To: m.At, Realm: marks[open].Realm})
			open = -1
		}
	}
	if open >= 0 {
		out = append(out, PaceSpan{From: marks[open].At, To: max(end, marks[open].At), Realm: marks[open].Realm, Open: true})
	}
	return out
}

func silences(marks []PaceMark, end float64) []PaceSpan {
	out := markSpans(marks, end, func(m PaceMark) bool { return m.Chapter && !m.Card && m.Busy == "" })
	for i := range out {
		out[i].Key = "no card"
	}
	return out
}

func idleSpans(marks []PaceMark, end float64) []PaceSpan {
	out := markSpans(marks, end, func(m PaceMark) bool { return m.Idle })
	for i := range out {
		out[i].Key = "nothing to do or watch"
	}
	return out
}

var cardPrefixes = []string{ActStep, actDone, actTeacher, actLesson, actIntro, actQuest, "debrief", "panel:" + OverlayDebrief}

func isCard(key string) bool {
	return slices.ContainsFunc(cardPrefixes, func(p string) bool { return strings.HasPrefix(key, p) })
}

func pressSpans(log AttnLog, end float64) []PaceSpan {
	asks := []AttnEvent{}
	for _, e := range log.Events {
		if e.Kind != EventAsk {
			continue
		}
		if n := len(asks); n > 0 && asks[n-1].Key == e.Key {
			if _, came := cardAfter(log.Boxes, asks[n-1], e.At); !came {
				continue
			}
		}
		asks = append(asks, e)
	}
	out := []PaceSpan{}
	for i, e := range asks {
		limit := end
		if i+1 < len(asks) {
			limit = asks[i+1].At
		}
		at, ok := cardAfter(log.Boxes, e, limit)
		span := PaceSpan{From: e.At, To: at, Realm: e.Realm, Key: e.Key}
		if !ok {
			span.To, span.Open = max(limit, e.At), true
		}
		out = append(out, span)
	}
	return out
}

func cardAfter(boxes []AttnBox, e AttnEvent, limit float64) (float64, bool) {
	at := min(e.At+PressMost, limit)
	for _, b := range boxes {
		if isCard(b.Key) && b.Key != e.Key && b.From <= e.At+1e-6 && (b.Open || b.To > at) {
			return e.At, true
		}
	}
	best, found := 0.0, false
	for _, b := range boxes {
		if !isCard(b.Key) || b.Key == e.Key || b.From < e.At-1e-6 || b.From > limit+1e-6 {
			continue
		}
		if !found || b.From < best {
			best, found = b.From, true
		}
	}
	return best, found
}

func creditSpans(events []AttnEvent, end float64) []PaceSpan {
	met := map[string]AttnEvent{}
	order := []string{}
	out := []PaceSpan{}
	for _, e := range events {
		switch e.Kind {
		case EventMet:
			if _, ok := met[e.Key]; !ok {
				met[e.Key] = e
				order = append(order, e.Key)
			}
		case EventCredited:
			if m, ok := met[e.Key]; ok {
				out = append(out, PaceSpan{From: m.At, To: max(e.At, m.At), Realm: m.Realm, Key: e.Key})
				delete(met, e.Key)
				order = slices.DeleteFunc(order, func(k string) bool { return k == e.Key })
			}
		}
	}
	for _, key := range order {
		m := met[key]
		out = append(out, PaceSpan{From: m.At, To: max(end, m.At), Realm: m.Realm, Key: key, Open: true})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].From < out[j].From })
	return out
}

func (h *HUD) noteAsk(answered string, realm float64) {
	h.noteAsked()
	h.NoteAttention(EventAsk, answered, realm)
}

func TeacherStepKey(v TeacherView) string {
	return fmt.Sprintf("teacher:%s:%d", v.Chapter.ID, v.Step)
}

func (h *HUD) notePaceMark(f Frame, surfaces []Surface) {
	t := &h.attn
	m := PaceMark{At: h.Now, Realm: f.State.RealmTime, Chapter: h.teacherShown(), Speed: f.State.Speed}
	for _, s := range surfaces {
		if s.Box && isCard(s.Key) && !strings.HasPrefix(s.Key, actIntro) && !strings.HasPrefix(s.Key, actQuest) {
			m.Card = true
		}
	}
	if h.OverlayID == OverlayDebrief {
		m.Card = true
	}
	m.Busy = h.paceBusy(f)
	m.Idle = !m.Chapter && m.Busy == "" && h.nothingToDo(surfaces, f.RawFrame().State.Enjoying)
	n := len(t.log.Marks)
	if n > 0 {
		last := t.log.Marks[n-1]
		if last.Card == m.Card && last.Busy == m.Busy && last.Idle == m.Idle && last.Chapter == m.Chapter {
			return
		}
	}
	t.log.Marks = append(t.log.Marks, m)
}

func (h *HUD) paceBusy(f Frame) string {
	switch {
	case h.OverlayID != OverlayNone && h.OverlayID != OverlayMenu && !autoOverlays[h.OverlayID] && !h.ShortStone():
		return BusyPanel + ":" + h.OverlayID
	case effectPending(f.RawFrame()):
		return BusyBuild
	}
	if v := h.Yard.View; v != nil && v.running() && v.step().Watch {
		return BusyWatch
	}
	return ""
}

func (h *HUD) nothingToDo(surfaces []Surface, enjoying bool) bool {
	for _, s := range surfaces {
		switch {
		case s.Box && strings.HasPrefix(s.Key, ActStep):
			return false
		case s.Box && enjoying && strings.HasPrefix(s.Key, actDone):
			return false
		case s.Box && s.Waits:
			return false
		}
	}
	return true
}

func (h *HUD) noteMetSteps(f Frame) {
	t := &h.attn
	if h.teacherShown() {
		return
	}
	if id := f.State.Met; id != "" {
		h.noteMet(ActStep + id)
	}
	done := map[string]bool{}
	for _, o := range f.RawFrame().State.Objectives {
		if !o.Done {
			continue
		}
		done[o.ID] = true
		if t.done != nil && !t.done[o.ID] {
			h.noteMet(ActStep + o.ID)
		}
	}
	t.done = done
}

func (h *HUD) noteTeacherMet(v TeacherView) {
	if v.Status == view.ChapterRunning && (v.Soaking || v.Met) {
		h.noteMet(TeacherStepKey(v))
	}
}

func (h *HUD) noteMet(key string) {
	t := &h.attn
	if !t.on || t.metOpen[key] || t.credited[key] {
		return
	}
	t.metOpen[key] = true
	h.NoteAttention(EventMet, key, t.realm)
	if t.early[key] {
		delete(t.early, key)
		h.noteCredited(key)
	}
}

func (h *HUD) noteCredited(key string) {
	t := &h.attn
	if !t.on || t.credited[key] {
		return
	}
	if !t.metOpen[key] {
		t.early[key] = true
		return
	}
	delete(t.metOpen, key)
	t.credited[key] = true
	h.NoteAttention(EventCredited, key, t.realm)
}

func (h *HUD) noteTeacherCredit(surfaces []Surface) {
	v := h.Yard.View
	if v == nil {
		return
	}
	if !slices.ContainsFunc(surfaces, func(s Surface) bool { return s.Box && strings.HasPrefix(s.Key, actTeacher) }) {
		return
	}
	if v.Status == view.ChapterRunning && v.Soaking {
		h.noteCredited(TeacherStepKey(*v))
	}
}

func (h *HUD) noteDoneCredit(surfaces []Surface) {
	for _, s := range surfaces {
		if s.Box && strings.HasPrefix(s.Key, actDone) {
			h.noteCredited(ActStep + strings.TrimPrefix(s.Key, actDone))
		}
	}
}

func (h *HUD) notePace(f Frame, surfaces []Surface) {
	h.noteMetSteps(f)
	h.noteTeacherCredit(surfaces)
	h.noteDoneCredit(surfaces)
	h.notePaceMark(f, surfaces)
}
