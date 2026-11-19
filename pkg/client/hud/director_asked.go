package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

func (h *HUD) untimed() bool { return h.teacherShown() }

func (h *HUD) noteAsked() {
	if s := &h.Director.Stage; s.Primed {
		s.askedAt = h.Now
	}
}

func (h *HUD) askPending() bool {
	s := h.Director.Stage
	return s.Primed && h.Now-s.askedAt <= PressMost
}

func (h *HUD) askedKey(key string) bool {
	return markedKey(h.Director.Stage.pulledKeys, key) || strings.HasPrefix(key, actDone) || strings.HasPrefix(key, actTeacher) || key == "panel:"+OverlayDebrief
}

func (h *HUD) dueKey(key string) bool { return markedKey(h.Director.Stage.dueKeys, key) }

func markedKey(marks map[string]bool, key string) bool {
	return marks[key] || strings.HasPrefix(key, actQuest) && !strings.HasPrefix(key, actQuest+recapPrefix) && marks[actIntro+view.BorgIntro]
}

func (h *HUD) AdmitAsked(f Frame, acts []stageAct) {
	s := &h.Director.Stage
	if h.untimed() {
		return
	}
	if a, ok := h.doneAct(f); ok {
		h.admitAskedAct(f, a)
	}
	passed := h.stepSetAside(f)
	if !h.askPending() && !passed {
		return
	}
	o, ok := f.current()
	if !ok || s.Has(ActStep+o.ID) || s.arrived != o.ID || !passed && (!h.stepEnjoyed(f) || s.DoneOn != s.StepOn) {
		return
	}
	if in := f.Guide.Intro; in != nil && !f.Unguided && (ownIntro(f, *in) || f.decreeIntro()) && !s.Has(actIntro+in.ID) {
		h.admitAskedAct(f, stageAct{key: actIntro + in.ID, prio: prioOwn, waits: true, idea: true})
	}
	if f.waitsForDecree(o) || h.ownCardFirst(f) {
		return
	}
	h.admitAskedAct(f, stageAct{key: ActStep + o.ID, prio: prioStep})
	s.askedAt = -100
}

func (h *HUD) stepSetAside(f Frame) bool {
	s := h.Director.Stage
	o, ok := f.current()
	if !ok || s.StepOn == "" || s.StepOn == o.ID || s.DoneOn == s.StepOn || strings.HasPrefix(s.StepOn, tryStep) {
		return false
	}
	_, listed := objectiveOf(f, s.StepOn)
	return !listed
}

func (h *HUD) admitAskedAct(f Frame, a stageAct) {
	s := &h.Director.Stage
	if !h.slotFree() && !h.asideForAsked(a.key) && !h.doneGivesWay(a) {
		return
	}
	h.trace("asked %s", a.key)
	s.pulledKeys = withKey(s.pulledKeys, a.key)
	h.putOn(f, a)
	s.LastNew = h.Now
	boxes := s.Boxes
	h.takeSlot(f, a)
	s.Boxes = boxes
	if !strings.HasPrefix(a.key, actIntro) {
		s.Slot.Waits = false
	}
}

func (h *HUD) asideForAsked(by string) bool {
	s := &h.Director.Stage
	b := s.Slot
	if strings.HasPrefix(b.Key, ActBanner) && h.BannerActive() {
		h.trace("banner %s replays after an asked card", b.Key)
		h.Director.Banners = append([]bannerItem{{text: h.Juice.Banner, cause: h.Juice.Cause, feat: h.Juice.feat}}, h.Director.Banners...)
		h.Juice.Banner = ""
		s.Slot = SlotBoxOf{}
		return true
	}
	if strings.HasPrefix(b.Key, ActBanner) && !h.BannerActive() || strings.HasPrefix(b.Key, actToast) && !h.toastShown(b.text) {
		h.trace("%s has left the screen: the asked card takes the slot", b.Key)
		s.Slot = SlotBoxOf{}
		return true
	}
	switch {
	case b.Key == "" || !b.Waits || !leaves(b.Key) || !s.Has(b.Key) || h.askedKey(b.Key):
		return false
	case strings.HasPrefix(b.Key, ActTrouble) && h.Now < b.From+b.Read:
		return false
	}
	h.trace("aside %s for an asked card", b.Key)
	h.SetAside(b.Key)
	delete(s.On, b.Key)
	delete(s.Waits, b.Key)
	if b.Key == ActTrouble+s.focus {
		s.focus = ""
		if step, ok := strings.CutPrefix(by, actDone); ok {
			s.doneHolds = [2]string{b.Key, step}
		}
	}
	s.Slot = SlotBoxOf{}
	return true
}

func (s stageState) heldByDone(key string) bool {
	trouble, step := s.doneHolds[0], s.doneHolds[1]
	return key == trouble && s.DoneOn == step && s.StepOn == step && s.pulled < s.doneAt
}

func (h *HUD) doneGivesWay(a stageAct) bool {
	s := &h.Director.Stage
	b := s.Slot
	if !strings.HasPrefix(a.key, actDone) || !strings.HasPrefix(b.Key, actDone) || b.Key == a.key || h.Now < b.From+view.ObjectiveSoak {
		return false
	}
	h.NoteAttention(EventStepDone, b.Key, h.attn.realm)
	s.Slot = SlotBoxOf{}
	return true
}
