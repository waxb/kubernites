package hud

import (
	"slices"
	"sort"
	"strings"
)

type namedControl struct {
	phrases []string
	reveal  string
	control string
}

var namedControls = []namedControl{
	{[]string{"grimoire"}, "grimoire", "cmd:grimoire"},
	{[]string{"hire porters", "hire a porter"}, "summon", CmdSummonID},
}

var shownPrefixes = []string{"cmd:", "surface:", "meter:"}

const reachStoneLines = 12

var smallReach = []string{MeterGold, MeterWisdom, MeterUpkeep, MeterRenown, MeterNines, MeterHands, MeterLore, MeterCodex, SurfaceSpeed}

func (h *HUD) ReachStage(f Frame) {
	s := &h.Director.Stage
	riders, own := h.pendingReach(f)
	riding := s.LastNew == h.Now
	beat := h.revealBeatOpen()
	open := !riding && h.mayAdmit(f) && beat
	if !open {
		own = nil
	}
	if !riding && !beat {
		riders = nil
	}
	if len(own) > 1 {
		own = own[:1]
	}
	ids := append(riders, own...)
	if len(ids) == 0 {
		return
	}
	before := h.revealSet(h.stagedWith(f, "", ""))
	h.admitReach(f, ids)
	if !riding && !sameSet(before, h.revealSet(h.stagedWith(f, "", ""))) {
		h.Director.Stage.LastNew = h.Now
		h.revealBeat()
	}
}

func (s stageState) lastBoxAt() float64 {
	if len(s.Boxes) == 0 {
		return -100
	}
	return s.Boxes[len(s.Boxes)-1]
}

func (h *HUD) revealBeatOpen() bool {
	s := h.Director.Stage
	return h.Now >= s.lastBoxAt()+BoxFloor && h.Now >= s.quiet && h.Now >= s.revealAt+BoxFloor
}

func (h *HUD) revealBeat() {
	s := &h.Director.Stage
	s.Free, s.revealAt = max(s.Free, h.Now+BoxFloor), h.Now
}

func (h *HUD) cardWaitsOnScreen() bool {
	for _, b := range h.Ledger.Shown {
		if b.Waits {
			return true
		}
	}
	return false
}

func (h *HUD) admitReach(f Frame, ids []string) {
	for _, id := range ids {
		h.trace("reach %s", id)
		h.admitNamed(f, id)
	}
}

func (h *HUD) pendingReach(f Frame) ([]string, []string) {
	s := h.Director.Stage
	riders, own := []string{}, []string{}
	add := func(list *[]string, id string) {
		if !s.Shown[id] && !slices.Contains(riders, id) && !slices.Contains(own, id) {
			*list = append(*list, id)
		}
	}
	if f.Revealed("gold") && f.State.Gold > 0 {
		add(&riders, MeterGold)
	}
	if f.Revealed("wisdom") {
		add(&riders, MeterWisdom)
	}
	if f.Revealed("speed") {
		add(&riders, SurfaceSpeed)
	}
	texts := h.reachTexts(f)
	for _, c := range namedControls {
		if f.Revealed(c.reveal) && namesAny(texts, c.phrases) {
			add(&riders, c.control)
		}
	}
	waits := h.cardWaitsOnScreen()
	for _, base := range h.pendingKnown(f) {
		for _, t := range f.Guide.Introduced {
			if introBase(t) != base || !shownTarget(t) {
				continue
			}
			switch {
			case !slices.Contains(smallReach, t):
				add(&own, t)
			case !waits:
				add(&riders, t)
			}
		}
	}
	return riders, own
}

func namesAny(texts, phrases []string) bool {
	for _, t := range texts {
		lower := strings.ToLower(t)
		for _, p := range phrases {
			if strings.Contains(lower, p) {
				return true
			}
		}
	}
	return false
}

func (h *HUD) reachTexts(f Frame) []string {
	out := []string{}
	if o, ok := f.current(); ok {
		out = append(out, o.Text, o.Progress)
	}
	for _, q := range f.State.Quests {
		if q.Closed {
			continue
		}
		for _, p := range q.Parts {
			if p.Done {
				out = append(out, p.Gained)
			}
		}
	}
	lines := h.Stone.Output[max(len(h.Stone.Output)-reachStoneLines, 0):]
	for _, l := range lines {
		out = append(out, l.Text)
	}
	for _, t := range f.Events[max(len(f.Events)-3, 0):] {
		out = append(out, t.Text)
	}
	return out
}

func shownTarget(id string) bool {
	for _, p := range shownPrefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

func (h *HUD) noteShown(id string) {
	if !shownTarget(id) {
		return
	}
	s := &h.Director.Stage
	if s.Shown[id] {
		return
	}
	next := make(map[string]bool, len(s.Shown)+1)
	for k, v := range s.Shown {
		next[k] = v
	}
	next[id] = true
	s.Shown = next
}

func (h *HUD) withShown(list []string) []string {
	out := slices.Clone(list)
	extra := []string{}
	for id := range h.Director.Stage.Shown {
		if !slices.Contains(out, id) {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}
