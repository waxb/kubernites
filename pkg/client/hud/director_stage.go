package hud

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	StepRead    = 6.0
	stepSettle  = 1.0
	AgeButtonID = "cmd:age"
	AgeAdvance  = "age:advance"
	troubleWait = 10.0

	prioTrouble = 0
	prioOwn     = 1
	prioTimely  = 2
	prioStep    = 3
	prioLesson  = 4
	prioIntro   = 5
	prioNews    = 6
	prioReveal  = 7

	ActTrouble = "trouble:"
	actIntro   = "intro:"
	actQuest   = "quest:"
	ActStep    = "step:"
	actLesson  = "lesson:"
	actTeacher = "teacher:"
	ActBanner  = "banner:"
	actToast   = "toast:"
	ActNotice  = "notice:"
	actReveal  = "reveal:"
	actKnown   = "known:"
	actEnding  = "ending:demo"
)

type stageAct struct {
	key    string
	prio   int
	waits  bool
	rank   int
	idea   bool
	urgent bool
}

var urgentTroubles = map[string]bool{
	view.CalamitySpoiling: true, view.CalamityNoDepot: true, view.CalamityNoCrew: true, view.CalamityNines: true, view.TagCrashLoop: true,
}

func urgentTrouble(kind string) bool { return urgentTroubles[kind] }

func timelyPrio(timely bool, prio int) int {
	if timely {
		return prioTimely
	}
	return prio
}

func (h *HUD) soaksWhenGone(key string) {
	s := &h.Director.Stage
	s.soaks = withKey(s.soaks, key)
}

type stageState struct {
	off        bool
	Primed     bool `json:"-"`
	arrived    string
	arrivedAt  float64
	On         map[string]float64 `json:"-"`
	Waits      map[string]bool    `json:"-"`
	wanted     map[string]float64
	LastNew    float64 `json:"-"`
	lastGone   float64
	quiet      float64
	soak       float64
	soaks      map[string]bool
	readyOn    string
	readyAt    float64
	StepAt     float64 `json:"-"`
	known      map[string]bool
	revealed   map[string]bool
	StepOn     string             `json:"-"`
	DoneOn     string             `json:"-"`
	View       view.ObjectiveView `json:"-"`
	focus      string
	mended     string
	OverlayID  string `json:"-"`
	toasts     []Toast
	fanfares   []Toast
	asked      map[string]bool
	read       map[string]bool
	acted      map[string]bool
	Shown      map[string]bool `json:"-"`
	gone       map[string]float64
	silent     silentSig
	pulled     float64
	Slot       SlotBoxOf `json:"-"`
	Free       float64   `json:"-"`
	lastEnd    float64
	Boxes      []float64 `json:"-"`
	Idea       float64   `json:"-"`
	doneAt     float64
	doneRealm  float64
	actedAt    float64
	settledAt  float64
	asided     map[string]bool
	StepDoneAt float64 `json:"-"`
	stepDoneOn string
	revealAt   float64
	pressedAt  float64
	tray       int
	askedAt    float64
	pulledKeys map[string]bool
	dueKeys    map[string]bool
	doneShown  map[string]bool
	doneHolds  [2]string
}

func (s stageState) clone() stageState {
	out := s
	out.On, out.Waits, out.wanted, out.gone = maps.Clone(s.On), maps.Clone(s.Waits), maps.Clone(s.wanted), maps.Clone(s.gone)
	out.soaks, out.known, out.revealed = maps.Clone(s.soaks), maps.Clone(s.known), maps.Clone(s.revealed)
	out.asked, out.read, out.acted, out.Shown = maps.Clone(s.asked), maps.Clone(s.read), maps.Clone(s.acted), maps.Clone(s.Shown)
	out.toasts, out.fanfares = slices.Clone(s.toasts), slices.Clone(s.fanfares)
	out.Boxes, out.asided, out.pulledKeys, out.dueKeys = slices.Clone(s.Boxes), maps.Clone(s.asided), maps.Clone(s.pulledKeys), maps.Clone(s.dueKeys)
	out.doneShown = maps.Clone(s.doneShown)
	return out
}

func (s stageState) Has(key string) bool {
	_, ok := s.On[key]
	return ok
}

func (h *HUD) onStage(key string) bool {
	return !h.Director.Stage.Primed || h.Director.Stage.Has(key)
}

func (h *HUD) Staged() bool { return !h.Director.Stage.off }

func (h *HUD) SetStaging(on bool) {
	if h.Director.Stage.off == !on && (on || !h.Director.Stage.Primed) {
		return
	}
	h.Director.Stage = stageState{off: !on}
	h.Ledger = beatLedger{Beats: h.Ledger.Beats, drawn: h.Ledger.drawn}
}

func (h *HUD) manageStage(f Frame) {
	if h.Director.Stage.off {
		return
	}
	raw := f.RawFrame()
	if h.Director.Stage.Primed {
		h.noteSettle(raw)
		h.admitSilent(raw)
		h.admitStepTarget(raw)
	}
	acts := h.stageActs(raw)
	s := &h.Director.Stage
	if !s.Primed {
		h.primeStage(raw, acts)
		return
	}
	h.noteArrival(raw)
	h.leaveStage(acts)
	defer h.noteTray(acts)
	h.noteReady(raw)
	h.NotePanel()
	h.noteSlotShown(raw)
	h.noteMended(acts)
	h.NoteDone(raw)
	h.wantActs(acts)
	h.AdmitAsked(raw, acts)
	h.releaseFeats(raw)
	if !h.mayAdmit(raw) {
		h.preemptTrouble(raw, acts)
		h.admitReveal(raw, acts)
		h.ReachStage(raw)
		return
	}
	h.releaseFanfares()
	if best, ok := h.bestAct(raw, acts); ok {
		h.admit(raw, best)
	} else {
		h.admitReveal(raw, acts)
	}
	h.ReachStage(raw)
}

func (h *HUD) primeStage(f Frame, acts []stageAct) {
	s := stageState{Primed: true, On: map[string]float64{}, Waits: map[string]bool{}, wanted: map[string]float64{},
		LastNew: -100, lastGone: -100, quiet: -100, soak: -100, StepAt: -100, known: map[string]bool{}, revealed: map[string]bool{},
		lastEnd: -100, doneRealm: -1, doneAt: -100, revealAt: -100, pressedAt: -100, askedAt: -100, OverlayID: h.OverlayID}
	h.Director.Stage = s
	card, carded := firstAct(waitingActs(acts))
	welcome := autoOverlays[h.OverlayID]
	for _, a := range acts {
		switch {
		case strings.HasPrefix(a.key, ActStep) && welcome:
		case strings.HasPrefix(a.key, ActStep):
			h.putOn(f, a)
			h.Director.Stage.Boxes = append(h.Director.Stage.Boxes, h.Now)
		case strings.HasPrefix(a.key, actReveal) || strings.HasPrefix(a.key, actKnown) || strings.HasPrefix(a.key, ActNotice):
			h.putOn(f, a)
		case carded && a.key == card.key:
			h.putOn(f, a)
			h.takeSlot(f, a)
		}
	}
	if st := &h.Director.Stage; st.Slot.Key == "" && h.teacherShown() && !h.Yard.faceWait {
		h.takeSlot(f, stageAct{key: actTeacher + h.Yard.face})
		st.Slot.From = h.Yard.faceFrom
	}
	if st := &h.Director.Stage; st.StepOn == "" {
		if o, ok := lastDoneOf(f); ok {
			st.StepOn, st.DoneOn, st.View, st.doneAt, st.doneRealm = o.ID, o.ID, o, h.Now, f.State.RealmTime
		}
	}
	riders, own := h.pendingReach(f)
	h.admitReach(f, append(riders, own...))
	h.Director.Stage.LastNew = -100
}

func lastDoneOf(f Frame) (view.ObjectiveView, bool) {
	if _, open := f.current(); open {
		return view.ObjectiveView{}, false
	}
	for i := len(f.State.Objectives) - 1; i >= 0; i-- {
		if o := f.State.Objectives[i]; o.Done {
			return o, true
		}
	}
	return view.ObjectiveView{}, false
}

func firstAct(acts []stageAct) (stageAct, bool) {
	if len(acts) == 0 {
		return stageAct{}, false
	}
	return slices.MinFunc(acts, func(a, b stageAct) int {
		if c := cmp.Compare(a.prio, b.prio); c != 0 {
			return c
		}
		if c := cmp.Compare(a.rank, b.rank); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	}), true
}

func waitingActs(acts []stageAct) []stageAct {
	out := []stageAct{}
	for _, a := range acts {
		if a.waits {
			out = append(out, a)
		}
	}
	return out
}

func (h *HUD) stageActs(f Frame) []stageAct {
	out := h.troubleActs(f)
	if a, ok := h.doneAct(f); ok {
		out = append(out, a)
	}
	if in := f.Guide.Intro; in != nil && !f.Unguided {
		prio := prioIntro
		if ownIntro(f, *in) {
			prio = prioOwn
		}
		out = append(out, stageAct{key: actIntro + in.ID, prio: prio, waits: true, idea: true})
	}
	if key, ok := h.recapWanted(f); ok {
		out = append(out, stageAct{key: actQuest + key, prio: prioOwn, waits: true, idea: true})
	}
	if v, ok := endingOf(f); ok && !v.Read && !h.Kube.Ending.opened {
		out = append(out, stageAct{key: actEnding, prio: prioOwn})
	}
	if o, ok := f.current(); ok {
		out = append(out, stageAct{key: ActStep + o.ID, prio: prioStep})
	}
	if h.LessonCard.Shown && h.LessonCard.Title != "" {
		out = append(out, stageAct{key: actLesson + h.LessonCard.Title, prio: timelyPrio(h.LessonCard.timely, prioLesson), waits: true, idea: !h.LessonCard.done, urgent: h.LessonCard.timely})
	}
	if len(h.Director.Banners) > 0 && !h.BannerActive() {
		out = append(out, stageAct{key: ActBanner + h.Director.Banners[0].text, prio: prioNews})
	}
	if len(h.Director.Stage.toasts) > 0 {
		out = append(out, stageAct{key: actToast + h.Director.Stage.toasts[0].Text, prio: prioNews})
	}
	for _, n := range f.Notices {
		if !n.Quiet {
			out = append(out, stageAct{key: ActNotice + n.ID, prio: timelyPrio(n.Timely, prioNews), idea: n.Kind == NoticeDecision, urgent: n.Timely})
		}
	}
	return append(out, h.revealActs(f)...)
}

func (h *HUD) troubleActs(f Frame) []stageAct {
	out := []stageAct{}
	for _, v := range h.visibleCalamities(f) {
		if v.Mended != "" && !h.Director.Stage.read[v.ID] {
			continue
		}
		out = append(out, stageAct{key: ActTrouble + v.ID, prio: prioTrouble, waits: v.Mended == "", rank: calamityRank(f, v.ID), idea: v.Mended == "", urgent: urgentTrouble(v.Kind)})
	}
	return out
}

func calamityRank(f Frame, id string) int {
	for i, v := range f.State.Calamities {
		if v.ID == id {
			return i
		}
	}
	return len(f.State.Calamities)
}

func ownIntro(f Frame, in Intro) bool {
	if in.ID == view.BorgIntro {
		return true
	}
	o, ok := f.current()
	if !ok {
		return false
	}
	if id, found := view.Content.IntroFor(o.Target); found && id == in.ID {
		return true
	}
	if o.Build != "" && (in.ID == o.Build || in.Target == "build:"+o.Build) {
		return true
	}
	return in.Target != "" && (in.Target == o.Target || in.Target == o.Button)
}

func (h *HUD) leaveStage(acts []stageAct) {
	s := &h.Director.Stage
	live := map[string]stageAct{}
	for _, a := range acts {
		live[a.key] = a
	}
	for key := range maps.Clone(s.On) {
		if !leaves(key) {
			continue
		}
		if a, ok := live[key]; ok {
			if h.reopened(key, a) {
				h.trace("reopened %s", key)
				h.reopen(key)
				continue
			}
			s.Waits[key] = a.waits
			continue
		}
		if s.Waits[key] {
			s.lastGone = h.Now
			s.gone = withTime(s.gone, key, h.Now)
		}
		h.trace("leave %s", key)
		delete(s.On, key)
		delete(s.Waits, key)
		if key == ActTrouble+s.focus {
			s.focus = ""
		}
	}
}

func (h *HUD) reopened(key string, a stageAct) bool {
	return strings.HasPrefix(key, ActTrouble) && a.waits && !h.Director.Stage.Waits[key]
}

func (h *HUD) reopen(key string) {
	s := &h.Director.Stage
	if key == ActTrouble+s.mended {
		s.mended = ""
	}
	if h.slotFree() {
		s.Waits[key] = true
		return
	}
	delete(s.On, key)
	delete(s.Waits, key)
	if key == ActTrouble+s.focus {
		s.focus = ""
	}
}

func leaves(key string) bool {
	for _, p := range []string{ActTrouble, actIntro, actQuest, actLesson, ActNotice, actEnding} {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func (h *HUD) NotePanel() {
	s := &h.Director.Stage
	if s.OverlayID != h.OverlayID && s.OverlayID != OverlayNone && s.OverlayID != OverlayMenu {
		s.lastGone = max(s.lastGone, h.Now)
		s.Free = max(s.Free, h.Now+BoxFloor)
		if autoOverlays[s.OverlayID] {
			s.Boxes = append(s.Boxes, h.Now)
		}
	}
	s.OverlayID = h.OverlayID
}

func (h *HUD) noteMended(acts []stageAct) {
	s := &h.Director.Stage
	if s.focus == "" || s.mended == s.focus {
		return
	}
	for _, a := range acts {
		if a.key == ActTrouble+s.focus && !a.waits {
			s.mended = s.focus
			h.feat()
		}
	}
}

func (h *HUD) noteArrival(f Frame) {
	s := &h.Director.Stage
	if o, ok := f.current(); ok && o.ID != s.arrived {
		s.arrived, s.arrivedAt = o.ID, h.Now
	}
}

func (h *HUD) NoteDone(f Frame) {
	s := &h.Director.Stage
	if s.StepOn == "" || s.DoneOn == s.StepOn {
		return
	}
	o, ok := objectiveOf(f, s.StepOn)
	switch {
	case ok && !o.Done:
		s.View = o
	case s.stepDoneOn != s.StepOn:
		s.StepDoneAt, s.stepDoneOn = h.Now, s.StepOn
		h.NoteAttention(EventStepDone, ActStep+s.StepOn, f.State.RealmTime)
	}
}

func (h *HUD) feat() {
	s := &h.Director.Stage
	s.quiet = max(s.quiet, h.Now+BeatQuiet)
	s.LastNew = h.Now
}

func (h *HUD) wantActs(acts []stageAct) {
	s := &h.Director.Stage
	next := map[string]float64{}
	for _, a := range acts {
		at, ok := s.wanted[a.key]
		if !ok {
			at = h.Now
		}
		next[a.key] = at
	}
	s.wanted = next
}

func (h *HUD) releaseFeats(f Frame) {
	if len(h.Director.Banners) == 0 || h.BannerActive() || !h.Director.Banners[0].feat {
		return
	}
	if _, done := h.doneAct(f); done || h.Director.Stage.tray > 0 {
		return
	}
	if h.OverlayID != OverlayNone && h.OverlayID != OverlayMenu && h.OverlayID != OverlayAge && !(h.ShortStone() && h.teachersOwn(ActBanner+h.Director.Banners[0].text)) {
		return
	}
	if h.cardAwaitsAnswer() || !h.slotFree() || !h.untimed() && h.Now < h.Director.Stage.Free || h.busyMinute() || !h.settled() {
		return
	}
	text := h.Director.Banners[0].text
	h.showBanner()
	h.feat()
	h.takeSlot(Frame{}, stageAct{key: ActBanner + text})
	h.releaseFanfares()
}

func (h *HUD) cardAwaitsAnswer() bool {
	s := h.Director.Stage
	for key, w := range s.Waits {
		if w && s.Has(key) && !strings.HasPrefix(key, ActTrouble) {
			return true
		}
	}
	return false
}

func (h *HUD) holdFanfare(t Toast) bool {
	s := &h.Director.Stage
	if !s.Primed || !h.cardAwaitsAnswer() {
		return false
	}
	s.fanfares = append(slices.Clone(s.fanfares), t)
	return true
}

func (h *HUD) releaseFanfares() {
	s := &h.Director.Stage
	for _, t := range s.fanfares {
		h.pushToastAs(t)
	}
	s.fanfares = nil
}

func (h *HUD) waitingOnStage() bool {
	for key, w := range h.Director.Stage.Waits {
		if w && h.Director.Stage.Has(key) {
			return true
		}
	}
	return false
}

func (h *HUD) beatOpen() bool {
	s := h.Director.Stage
	switch {
	case h.untimed() && h.OverlayID != OverlayNone && h.OverlayID != OverlayMenu:
		return false
	case h.untimed():
		return true
	case h.Now < s.LastNew+BeatGap || h.Now < s.lastGone+BeatGap || h.Now < s.quiet:
		return false
	case h.OverlayID != OverlayNone && h.OverlayID != OverlayMenu:
		return false
	}
	return true
}

func (h *HUD) stepBeatOpen() bool {
	s := h.Director.Stage
	switch {
	case h.untimed():
		return h.OverlayID == OverlayNone || h.OverlayID == OverlayMenu || h.ShortStone()
	case h.Now < s.LastNew+BeatGap || h.Now < s.lastGone+BeatGap || h.Now < s.quiet:
		return false
	case h.OverlayID == OverlayStone && !h.Stone.Tall:
		return true
	}
	return h.beatOpen()
}

func (h *HUD) mayAdmit(f Frame) bool {
	return h.slotOpen(f) && !h.waitingOnStage()
}

func (h *HUD) bestAct(f Frame, acts []stageAct) (stageAct, bool) {
	s := h.Director.Stage
	pending := []stageAct{}
	for _, a := range acts {
		if s.Has(a.key) || !h.actReady(f, a) || !h.besideShortStone(a) {
			continue
		}
		pending = append(pending, a)
	}
	if len(pending) == 0 {
		return stageAct{}, false
	}
	slices.SortStableFunc(pending, func(a, b stageAct) int {
		if c := cmp.Compare(a.prio, b.prio); c != 0 {
			return c
		}
		if c := cmp.Compare(a.rank, b.rank); c != 0 {
			return c
		}
		if c := cmp.Compare(s.wanted[a.key], s.wanted[b.key]); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})
	return pending[0], true
}

func (h *HUD) soakAfter(key string) {
	if s := &h.Director.Stage; s.soaks[key] {
		s.soak = max(s.soak, h.Now+SoakGap)
	}
}

func (h *HUD) soaking(a stageAct) bool {
	return !h.untimed() && a.idea && !a.urgent && !h.Ledger.drawn[a.key] && h.Now < h.Director.Stage.soak
}

func (h *HUD) actReady(f Frame, a stageAct) bool {
	s := h.Director.Stage
	if at, left := s.gone[a.key]; left && a.waits && !strings.HasPrefix(a.key, ActTrouble) && !h.Aside(a.key) && !h.untimed() && h.Now < at+BoxFloor+CalmGap {
		return false
	}
	if !h.Aside(a.key) && !h.untimed() && h.ideaWaits(f, a) && !h.stepDue(f, a) {
		return false
	}
	switch {
	case strings.HasPrefix(a.key, actReveal) || strings.HasPrefix(a.key, actKnown):
		return !h.heldReach(f)[actText(a.key)] && h.revealBeatOpen()
	case strings.HasPrefix(a.key, actDone):
		return true
	case strings.HasPrefix(a.key, ActTrouble):
		return s.focus == "" && !s.heldByDone(a.key)
	case strings.HasPrefix(a.key, ActNotice):
		return h.noticeSeen(f, strings.TrimPrefix(a.key, ActNotice))
	case strings.HasPrefix(a.key, actToast) && len(s.toasts) > 0:
		return h.toastRoom(h.Stage(f), s.toasts[0])
	case strings.HasPrefix(a.key, ActStep):
		o, _ := f.current()
		return !h.ownCardFirst(f) && !f.waitsForDecree(o) && h.stepEnjoyed(f) && s.DoneOn == s.StepOn && s.arrived == o.ID && (h.untimed() || h.Now >= s.arrivedAt+stepSettle)
	case strings.HasPrefix(a.key, actIntro) && a.prio == prioIntro:
		free := f.Selection.Empty() && h.PageID == PageMain || outsideGrid(f.Guide.Intro)
		return free && f.Placing == "" && (h.untimed() || h.Now >= s.StepAt+StepRead) && !h.heldForStep(f, a) && !ageReady(f)
	}
	return true
}

var gridPrefixes = []string{"cmd:", "build:", "summon:", stallPrefix + ":", "act:", crewNS}

func outsideGrid(in *Intro) bool {
	if in == nil || in.Target == "" {
		return false
	}
	return !slices.ContainsFunc(gridPrefixes, func(p string) bool { return strings.HasPrefix(in.Target, p) })
}

func ageReady(f Frame) bool {
	o, ok := f.current()
	return ok && o.Target == AgeButtonID && strings.Contains(o.Progress, readyMark)
}

func withTime(m map[string]float64, key string, at float64) map[string]float64 {
	next := maps.Clone(m)
	if next == nil {
		next = map[string]float64{}
	}
	next[key] = at
	return next
}

func (h *HUD) noteReady(f Frame) {
	s := &h.Director.Stage
	o, ok := f.current()
	switch {
	case !ok || !midStep(f):
		s.readyOn = ""
	case s.readyOn != o.ID:
		s.readyOn, s.readyAt = o.ID, h.Now
	}
}

func (h *HUD) heldForStep(f Frame, a stageAct) bool {
	s := h.Director.Stage
	return midStep(f) && s.wanted[a.key] >= s.readyAt
}

func midStep(f Frame) bool {
	o, ok := f.current()
	return ok && strings.Contains(o.Progress, readyMark)
}

func (h *HUD) ownCardFirst(f Frame) bool {
	in := f.Guide.Intro
	return in != nil && !f.Unguided && ownIntro(f, *in) && !h.Director.Stage.Has(actIntro+in.ID)
}

func (h *HUD) preemptTrouble(f Frame, acts []stageAct) {
	s := &h.Director.Stage
	if h.OverlayID != OverlayNone && h.OverlayID != OverlayMenu || h.busyMinute() || !h.settled() {
		return
	}
	trouble, ok := worstPending(acts, *s)
	if !ok || !h.troubleTaught(f, trouble.key) || !h.untimed() && h.Now < s.wanted[trouble.key]+TroubleWait {
		return
	}
	if s.focus != "" && trouble.rank >= calamityRank(f, s.focus) {
		return
	}
	b := s.Slot
	switch {
	case h.slotFree() && (h.untimed() || h.Now >= s.lastEnd+BoxFloor):
	case b.Key != "" && b.Waits && leaves(b.Key) && !strings.HasPrefix(b.Key, ActTrouble) && s.Has(b.Key) && h.Now >= b.From+b.Read:
		h.trace("aside %s", b.Key)
		h.SetAside(b.Key)
		delete(s.On, b.Key)
		delete(s.Waits, b.Key)
		s.Slot = SlotBoxOf{}
	default:
		return
	}
	for key := range maps.Clone(s.On) {
		if strings.HasPrefix(key, ActTrouble) {
			delete(s.On, key)
			delete(s.Waits, key)
		}
	}
	s.focus = ""
	h.admit(f, trouble)
}

func (h *HUD) troubleTaught(f Frame, key string) bool {
	for _, v := range h.visibleCalamities(f) {
		if ActTrouble+v.ID == key {
			return !v.First
		}
	}
	return false
}

func worstPending(acts []stageAct, s stageState) (stageAct, bool) {
	best, found := stageAct{}, false
	for _, a := range acts {
		if strings.HasPrefix(a.key, ActTrouble) && a.waits && a.urgent && !s.Has(a.key) && !s.read[strings.TrimPrefix(a.key, ActTrouble)] && (!found || a.rank < best.rank) {
			best, found = a, true
		}
	}
	return best, found
}

var StageTrace func(string)

func (h *HUD) trace(format string, args ...any) {
	if StageTrace != nil {
		StageTrace(fmt.Sprintf("%7.1f ", h.Now) + fmt.Sprintf(format, args...))
	}
}

func (h *HUD) admit(f Frame, a stageAct) {
	h.trace("admit %s prio %d", a.key, a.prio)
	if !h.Aside(a.key) && h.ideaWaits(f, a) && h.stepDue(f, a) {
		h.trace("due %s inside the idea gap", a.key)
		h.Director.Stage.dueKeys = withKey(h.Director.Stage.dueKeys, a.key)
	}
	h.putOn(f, a)
	h.Director.Stage.LastNew = h.Now
	if strings.HasPrefix(a.key, actReveal) || strings.HasPrefix(a.key, actKnown) {
		h.revealBeat()
	}
	if !strings.HasPrefix(a.key, actReveal) && !strings.HasPrefix(a.key, actKnown) {
		h.takeSlot(f, a)
	}
}

func (h *HUD) putOn(f Frame, a stageAct) {
	s := &h.Director.Stage
	s.On[a.key], s.Waits[a.key] = h.Now, a.waits
	if a.idea && !a.urgent {
		h.soaksWhenGone(a.key)
	}
	rest := a.key[strings.Index(a.key, ":")+1:]
	switch {
	case strings.HasPrefix(a.key, actDone):
		delete(s.On, a.key)
		h.putDone(f, rest)
		h.feat()
	case strings.HasPrefix(a.key, ActStep):
		h.putStep(f, rest)
	case strings.HasPrefix(a.key, ActTrouble):
		s.focus = rest
		s.read = withKey(s.read, rest)
		h.askPauseFor(f, rest)
	case strings.HasPrefix(a.key, actIntro):
		h.admitNamed(f, introGoal(f))
	case strings.HasPrefix(a.key, actLesson):
		h.LessonCard.start = h.Now
		h.markTaught(h.LessonCard.once)
	case strings.HasPrefix(a.key, ActBanner):
		delete(s.On, a.key)
		h.showBanner()
	case strings.HasPrefix(a.key, actToast):
		delete(s.On, a.key)
		h.showToast()
	case strings.HasPrefix(a.key, actReveal):
		s.revealed[rest] = true
		delete(s.On, a.key)
	case strings.HasPrefix(a.key, actKnown):
		s.known[rest] = true
		delete(s.On, a.key)
	}
}

func (h *HUD) admitStepTarget(f Frame) {
	s := h.Director.Stage
	if s.StepOn == "" || h.Now > s.StepAt && !h.revealBeatOpen() {
		return
	}
	if o, ok := objectiveOf(f, s.StepOn); ok && !o.Done {
		h.admitNamed(f, ObjectiveTarget(o))
		h.admitNamed(f, o.Button)
	}
}

func (h *HUD) putStep(f Frame, id string) {
	s := &h.Director.Stage
	if s.StepOn != "" && s.StepOn != id {
		delete(s.On, ActStep+s.StepOn)
	}
	s.StepOn, s.StepAt = id, h.Now
	if s.DoneOn == id {
		s.DoneOn = ""
	}
	if s.doneShown[id] {
		s.doneShown = maps.Clone(s.doneShown)
		delete(s.doneShown, id)
	}
	if o, ok := objectiveOf(f, id); ok {
		s.View = o
		h.admitNamed(f, ObjectiveTarget(o))
		h.admitNamed(f, o.Button)
	}
}

func introGoal(f Frame) string {
	if f.Guide.Intro == nil {
		return ""
	}
	return f.Guide.Intro.Target
}

func ObjectiveTarget(o view.ObjectiveView) string {
	if o.Build != "" {
		return "build:" + o.Build
	}
	return o.Target
}

func (h *HUD) showToast() {
	s := &h.Director.Stage
	if len(s.toasts) == 0 {
		return
	}
	t := s.toasts[0]
	s.toasts = slices.Clone(s.toasts[1:])
	h.pushToastAs(t)
}

func (h *HUD) queueNews(t Toast) {
	if !h.Director.Stage.Primed {
		h.pushToastAs(t)
		return
	}
	for _, old := range h.Director.Stage.toasts {
		if old.Text == t.Text {
			return
		}
	}
	h.Director.Stage.toasts = append(slices.Clone(h.Director.Stage.toasts), t)
}

func (h *HUD) StageDebug() string {
	s := h.Director.Stage
	return fmt.Sprintf("on=%v waits=%v lastNew=%.1f lastGone=%.1f quiet=%.1f soak=%.1f now=%.1f step=%s focus=%s overlay=%s", s.On, s.Waits, s.LastNew, s.lastGone, s.quiet, s.soak, h.Now, s.StepOn, s.focus, h.OverlayID)
}

func (h *HUD) ShowStepDone(o view.ObjectiveView, realm float64) {
	s := &h.Director.Stage
	if !s.Primed {
		return
	}
	delete(s.On, ActStep+s.StepOn)
	s.StepOn, s.DoneOn, s.View, s.doneAt, s.doneRealm = o.ID, o.ID, o, h.Now, realm
	s.On[ActStep+o.ID] = h.Now
	h.feat()
}

const StepDueMost = 60.0

func (h *HUD) stepDue(f Frame, a stageAct) bool {
	s := h.Director.Stage
	own := strings.HasPrefix(a.key, actIntro) && (a.prio == prioOwn || f.decreeIntro()) || strings.HasPrefix(a.key, ActStep)
	return own && s.doneRealm >= 0 && f.State.RealmTime >= s.doneRealm+view.EnjoyTime+StepDueMost-CalmGap
}
