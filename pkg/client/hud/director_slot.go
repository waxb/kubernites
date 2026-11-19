package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	CalmGap     = 10.0
	TroubleWait = 4.0
	actDone     = "done:"
)

type SlotBoxOf struct {
	Key    string `json:"-"`
	text   string
	From   float64 `json:"-"`
	Read   float64 `json:"-"`
	Waits  bool    `json:"-"`
	idea   bool
	hidden bool
	seen   float64
}

func (b SlotBoxOf) ReadEnd() float64 { return max(b.From, b.seen) + b.Read }

func (h *HUD) slotFree() bool {
	s := &h.Director.Stage
	b := s.Slot
	if b.Key == "" {
		return true
	}
	doneAt := s.StepDoneAt
	switch {
	case strings.HasPrefix(b.Key, actTeacher):
		doneAt = h.Yard.doneAt
	case strings.HasPrefix(b.Key, actDone):
		doneAt = s.pulled
	}
	answered := (b.Key == ActStep+s.StepOn && s.stepDoneOn == s.StepOn || strings.HasPrefix(b.Key, actDone)) && doneAt >= b.From || strings.HasPrefix(b.Key, actTeacher) && (h.Yard.faceDone || b.Key != actTeacher+h.Yard.face)
	switch {
	case b.Waits && s.Has(b.Key):
		return false
	case !b.Waits && (b.hidden || h.Now < b.ReadEnd()) && !answered:
		return false
	case strings.HasPrefix(b.Key, actToast) && h.toastShown(b.text):
		return false
	case strings.HasPrefix(b.Key, ActBanner) && h.BannerActive():
		return false
	}
	gap, end := CalmGap, b.ReadEnd()
	if b.Waits && (s.answered(h.lastSeen(b)) || strings.HasPrefix(b.Key, ActTrouble)) {
		end = h.lastSeen(b)
	}
	if b.Waits && !b.idea && s.closedBy(b.Key) {
		gap = BoxFloor
	}
	if answered {
		gap, end = BoxFloor, min(end, doneAt)
	}
	if strings.HasPrefix(b.Key, actDone) {
		s.Free = max(s.Free, b.From+BeatQuiet)
	}
	if strings.HasPrefix(b.Key, actToast) || strings.HasPrefix(b.Key, ActBanner) {
		end = max(end, h.lastSeen(b))
	}
	s.Free, s.lastEnd = max(s.Free, end+gap), end
	s.Slot = SlotBoxOf{}
	return true
}

func (s stageState) answered(left float64) bool {
	return left >= s.pressedAt && left-s.pressedAt <= attnAnswerWindow
}

func (h *HUD) pressed(at float64) {
	if s := &h.Director.Stage; s.Primed {
		s.pressedAt = at
	}
}

func (h *HUD) lastSeen(b SlotBoxOf) float64 {
	s := h.Director.Stage
	if at, ok := s.gone[b.Key]; ok && at >= b.From {
		return at
	}
	return h.Now
}

func (s stageState) closedBy(key string) bool {
	return !strings.HasPrefix(key, ActTrouble) || s.read[strings.TrimPrefix(key, ActTrouble)]
}

func (h *HUD) toastShown(text string) bool {
	return slices.ContainsFunc(h.ToastList, func(t Toast) bool { return t.Text == text })
}

func (h *HUD) slotOpen(f Frame) bool {
	if !h.slotFree() {
		return false
	}
	s := h.Director.Stage
	switch {
	case !h.untimed() && h.Now < s.Free:
		return false
	case !h.settled():
		return false
	case h.busyMinute():
		return false
	case h.OverlayID != OverlayNone && h.OverlayID != OverlayMenu && !h.ShortStone():
		return false
	}
	return true
}

func (h *HUD) ShortStone() bool { return h.OverlayID == OverlayStone && !h.Stone.Tall }

func (h *HUD) besideShortStone(a stageAct) bool {
	return !h.ShortStone() || strings.HasPrefix(a.key, ActStep) || strings.HasPrefix(a.key, actDone) || h.teachersOwn(a.key)
}

func (h *HUD) teachersOwn(key string) bool {
	switch {
	case !h.teacherShown():
		return false
	case h.Yard.lesson != "" && key == actLesson+h.Yard.lesson:
		return true
	}
	return false
}

func (h *HUD) busyMinute() bool {
	s := h.Director.Stage
	if h.Now >= FirstMinutes || h.untimed() {
		return false
	}
	n := 0
	for _, at := range s.Boxes {
		if h.Now-at < BusyWindow {
			n++
		}
	}
	return n >= BusyMost
}

func (h *HUD) settled() bool {
	s := h.Director.Stage
	return h.untimed() || s.actedAt <= 0 || s.settledAt >= 0 && h.Now >= max(s.actedAt, s.settledAt)+BoxFloor
}

func (h *HUD) noteSettle(f Frame) {
	s := &h.Director.Stage
	switch {
	case effectPending(f):
		s.settledAt = -1
	case s.settledAt < 0:
		s.settledAt = h.Now
	}
}

func effectPending(f Frame) bool {
	return slices.ContainsFunc(f.Site.Structures, func(st view.Structure) bool { return st.State == view.StateConstruction })
}

func (h *HUD) NoteAct(action string, realm float64) {
	h.NoteAttention(EventAction, action, realm)
	h.pressed(h.Now)
	s := &h.Director.Stage
	if !s.Primed || action == "ack" {
		return
	}
	s.actedAt = h.Now
}

func (h *HUD) takeSlot(f Frame, a stageAct) {
	s := &h.Director.Stage
	text := h.BoxText(h.Stage(f), a.key)
	s.Slot = SlotBoxOf{Key: a.key, text: actText(a.key), From: h.Now, Read: ReadTime(text), Waits: a.waits, idea: a.idea && !a.urgent, hidden: objectiveKey(a.key) && !h.CardsDrawn(f)}
	h.trace("slot %s read %.1f", a.key, s.Slot.Read)
	s.Boxes = append(slices.DeleteFunc(slices.Clone(s.Boxes), func(at float64) bool { return h.Now-at >= BusyWindow }), h.Now)
	if a.idea && !a.urgent {
		s.Idea = h.Now
	}
}

func objectiveKey(key string) bool {
	return strings.HasPrefix(key, ActStep) || strings.HasPrefix(key, actDone)
}

func (h *HUD) noteSlotShown(f Frame) {
	b := &h.Director.Stage.Slot
	if !b.hidden || !h.CardsDrawn(f) {
		return
	}
	b.hidden, b.seen = false, h.Now
	b.Read = max(b.Read, ReadTime(h.BoxText(h.Stage(f), b.Key)))
	h.trace("slot %s shown, read %.1f", b.Key, b.Read)
}

func actText(key string) string {
	_, rest, _ := strings.Cut(key, ":")
	return rest
}

func (h *HUD) BoxText(f Frame, key string) string {
	switch {
	case strings.HasPrefix(key, ActStep) || strings.HasPrefix(key, actDone):
		return SurfaceText(h.ObjectiveLines(f))
	case strings.HasPrefix(key, actIntro):
		if f.Guide.Intro == nil {
			return ""
		}
		title, why, try := h.introText(*f.Guide.Intro)
		return title + " " + why + " " + try
	case strings.HasPrefix(key, actLesson):
		return SurfaceText(h.LessonLines())
	case strings.HasPrefix(key, actTeacher) && h.teacherShown():
		return h.TeacherText()
	case strings.HasPrefix(key, ActTrouble):
		if v, _, _, ok := h.CurrentCalamity(f); ok {
			cause, why := h.CalamityText(f, v)
			return strings.Join(append(append(append([]string{h.asideLine()}, cause...), why...), remedyLabels(h.calamityLines(f))), " ")
		}
	case strings.HasPrefix(key, actQuest):
		if c, ok := h.QuestCard(f); ok {
			return c.title + " " + c.Body
		}
	case strings.HasPrefix(key, ActBanner):
		return strings.TrimSpace(h.Juice.Banner + " " + h.Juice.Cause)
	case strings.HasPrefix(key, actToast):
		return actText(key)
	case strings.HasPrefix(key, ActNotice):
		for _, n := range f.Notices {
			if ActNotice+n.ID == key {
				return n.Title + " " + n.Detail
			}
		}
	}
	return ""
}

func ideaGapOf(age string) float64 {
	if age == "" || age == view.AgeDark {
		return IdeaGapDark
	}
	return IdeaGapFeudal
}

func (h *HUD) ideaWaits(f Frame, a stageAct) bool {
	s := h.Director.Stage
	return a.idea && !a.urgent && s.Idea > 0 && h.Now < s.Idea+ideaGapOf(f.State.Age)
}

func (h *HUD) stepEnjoyed(f Frame) bool {
	s := h.Director.Stage
	return s.doneRealm < 0 || f.State.RealmTime >= s.doneRealm+EnjoyMin || s.pulled >= s.doneAt || s.View.Ahead && s.View.Done
}

func (h *HUD) doneAct(f Frame) (stageAct, bool) {
	s := h.Director.Stage
	switch {
	case s.StepOn == "":
		return stageAct{}, false
	case s.DoneOn != s.StepOn:
		if o, ok := objectiveOf(f, s.StepOn); !ok && strings.HasPrefix(s.StepOn, tryStep) || ok && o.Done {
			return stageAct{key: actDone + s.StepOn, prio: prioOwn}, true
		}
		return stageAct{}, false
	}
	if id, ok := doneBehind(f, s.StepOn, s.doneShown); ok {
		return stageAct{key: actDone + id, prio: prioOwn}, true
	}
	return stageAct{}, false
}

func doneBehind(f Frame, after string, shown map[string]bool) (string, bool) {
	list := f.State.Objectives
	at := slices.IndexFunc(list, func(o view.ObjectiveView) bool { return o.ID == after })
	if at < 0 {
		return "", false
	}
	for i := len(list) - 1; i > at; i-- {
		if list[i].Done && !list[i].Skipped && !shown[list[i].ID] {
			return list[i].ID, true
		}
	}
	return "", false
}

func (h *HUD) putDone(f Frame, id string) {
	s := &h.Director.Stage
	o, ok := objectiveOf(f, id)
	if !ok {
		o = s.View
		o.Done = true
	}
	if o.Lesson == "" && id == s.StepOn {
		o.Lesson = s.View.Lesson
	}
	delete(s.On, ActStep+s.StepOn)
	if id != s.StepOn {
		s.StepOn, s.StepAt = id, h.Now
	}
	s.DoneOn, s.View, s.doneAt, s.doneRealm = id, o, h.Now, f.State.RealmTime
	s.doneShown = withKey(s.doneShown, id)
}

func (h *HUD) admitReveal(f Frame, acts []stageAct) {
	s := &h.Director.Stage
	if !h.slotFree() || !h.untimed() && (h.Now < s.lastEnd+BoxFloor || h.Now < s.revealAt+BoxFloor || h.Now < s.lastBoxAt()+BoxFloor) || !h.settled() {
		return
	}
	for _, a := range acts {
		silent := strings.HasPrefix(a.key, actReveal) || strings.HasPrefix(a.key, actKnown)
		if silent && !s.Has(a.key) && h.actReady(f, a) && h.besideShortStone(a) {
			s.revealAt = h.Now
			h.admit(f, a)
			return
		}
	}
}

func (h *HUD) SetAside(key string) {
	s := &h.Director.Stage
	s.asided = withKey(s.asided, key)
	s.tray++
}

func (h *HUD) Aside(key string) bool { return h.Director.Stage.asided[key] }

func (h *HUD) noteTray(acts []stageAct) {
	s := &h.Director.Stage
	s.tray = 0
	for _, a := range acts {
		if s.asided[a.key] && !s.Has(a.key) {
			s.tray++
		}
	}
}

func AsideLine(n int) string {
	if n == 1 {
		return "1 card waits: it comes back after this"
	}
	return fmt.Sprintf("%d cards wait: they come back after this", n)
}

func (h *HUD) asideLine() string {
	if h.steadyTrouble.key != "" {
		return h.steadyTrouble.aside
	}
	return h.liveAsideLine()
}

func (h *HUD) liveAsideLine() string {
	if n := h.Director.Stage.tray; n > 0 {
		return AsideLine(n)
	}
	return ""
}

func (h *HUD) calamityHead() int {
	if h.asideLine() != "" {
		return CalamityHeadH + ui.LineHeight
	}
	return CalamityHeadH
}

func (h *HUD) StagePending(f Frame) bool {
	s := h.Director.Stage
	if !s.Primed || s.off {
		return false
	}
	if len(h.Results) > 0 && !h.LessonCard.Shown || len(h.Director.Banners) > 0 && !h.BannerActive() {
		return true
	}
	raw := h.Stage(f).RawFrame()
	now := h.Now
	h.Now += settleAhead
	defer func() { h.Now = now }()
	for _, a := range h.stageActs(raw) {
		if !s.Has(a.key) && h.actReady(raw, a) && h.besideShortStone(a) && !strings.HasPrefix(a.key, actReveal) && !strings.HasPrefix(a.key, actKnown) {
			return true
		}
	}
	return false
}

const settleAhead = 300.0

func (h *HUD) SettleStage(in Input, f Frame, limit float64) Frame {
	h.Update(in, f)
	for waited := 0.0; waited < limit && h.StagePending(f); waited += stageSettleStep {
		f.Clock += stageSettleStep
		h.Update(in, f)
	}
	return f
}

const stageSettleStep = 0.5
