package hud

import (
	"slices"
	"strings"
)

const maxAttnFrames = 200000

type attnTracker struct {
	on       bool
	log      AttnLog
	open     map[string]int
	sig      string
	drawSig  string
	dropped  int
	started  bool
	pressed  float64
	realm    float64
	done     map[string]bool
	metOpen  map[string]bool
	credited map[string]bool
	early    map[string]bool
}

const attnAnswerWindow = 1.0

func (h *HUD) SetAttention(on bool) {
	if on == h.attn.on {
		return
	}
	h.attn = attnTracker{on: on, open: map[string]int{}, pressed: -attnAnswerWindow - 1, metOpen: map[string]bool{}, credited: map[string]bool{}, early: map[string]bool{}}
}

func (h *HUD) Attention() AttnLog {
	l := h.attn.log
	return AttnLog{Boxes: slices.Clone(l.Boxes), Frames: slices.Clone(l.Frames), Events: slices.Clone(l.Events), Marks: slices.Clone(l.Marks)}
}

func (h *HUD) AttentionFrom(boxes, frames, events int) AttnLog {
	l := h.attn.log
	frames = max(frames-h.attn.dropped, 0)
	return AttnLog{Boxes: slices.Clone(l.Boxes[min(boxes, len(l.Boxes)):]), Frames: slices.Clone(l.Frames[min(frames, len(l.Frames)):]), Events: slices.Clone(l.Events[min(events, len(l.Events)):])}
}

func (h *HUD) AttentionCounts() (int, int, int) {
	l := h.attn.log
	return len(l.Boxes), len(l.Frames) + h.attn.dropped, len(l.Events)
}

func (h *HUD) DropAttentionFrames() {
	h.attn.dropped += len(h.attn.log.Frames)
	h.attn.log.Frames = nil
}

func (h *HUD) AttentionBox(i int) (AttnBox, bool) {
	if i < 0 || i >= len(h.attn.log.Boxes) {
		return AttnBox{}, false
	}
	return h.attn.log.Boxes[i], true
}

func (h *HUD) NoteAttention(kind, key string, realm float64) {
	if !h.attn.on {
		return
	}
	h.attn.log.Events = append(h.attn.log.Events, AttnEvent{At: h.Now, Realm: realm, Kind: kind, Key: key})
	if kind == EventAction || kind == EventNextStep {
		h.attn.pressed = h.Now
	}
	if i, open := h.attn.open[key]; open && (kind == EventStepDone || kind == EventNextStep) && h.attn.log.Boxes[i].Answered == 0 {
		h.attn.log.Boxes[i].Answered = h.Now
	}
}

func (h *HUD) noteAttention(f Frame) {
	t := &h.attn
	if !t.on {
		return
	}
	surfaces := []Surface{}
	for _, s := range h.paintedSurfaces(f) {
		surfaces = append(surfaces, s.Surface)
	}
	t.realm = f.State.RealmTime
	h.noteSurfaces(surfaces, f.State.RealmTime, f.State.Age)
	h.notePace(f, surfaces)
}

func (h *HUD) NoteDrawn(world []Surface, realm float64) {
	t := &h.attn
	if !t.on {
		return
	}
	all := append(slices.Clone(h.drawn), world...)
	sig := surfaceSig(all)
	if sig == t.drawSig || len(t.log.Frames) >= maxAttnFrames {
		return
	}
	t.drawSig = sig
	t.log.Frames = append(t.log.Frames, AttnFrame{At: h.Now, Realm: realm, Surfaces: all, Drawn: true})
}

func (h *HUD) noteSurfaces(surfaces []Surface, realm float64, age string) {
	t := &h.attn
	l := &t.log
	present := map[string]bool{}
	for _, s := range surfaces {
		if !s.Box {
			continue
		}
		present[s.Key] = true
		i, open := t.open[s.Key]
		if !open {
			t.open[s.Key] = len(l.Boxes)
			l.Boxes = append(l.Boxes, AttnBox{Kind: s.Kind, Key: s.Key, Text: s.Text, Status: s.Status, Slot: s.Slot, Rect: s.Rect, From: h.Now, Realm: realm, Age: age,
				Waits: s.Waits, Face: s.Face, Idea: s.Idea, Urgent: s.Urgent, Taught: s.Taught, Tray: s.Tray, Open: true, Entry: !t.started && s.Face, Asked: h.askedKey(s.Key), Due: h.dueKey(s.Key)})
			continue
		}
		b := &l.Boxes[i]
		last, status, rect := b.Text, b.Status, b.Rect
		if n := len(b.Texts); n > 0 {
			last, status, rect = b.Texts[n-1].Text, b.Texts[n-1].Status, b.Texts[n-1].Rect
		}
		if s.Text != last || s.Status != status || s.Rect.W != rect.W || s.Rect.H != rect.H {
			b.Texts = append(b.Texts, AttnText{At: h.Now, Text: s.Text, Status: s.Status, Rect: s.Rect, Pressed: h.Now-t.pressed <= attnAnswerWindow})
		}
	}
	held := h.OverlayID != OverlayNone && !autoOverlays[h.OverlayID] && !h.ShortStone() || slices.ContainsFunc(surfaces, func(s Surface) bool { return s.Modal && s.Box })
	for key, i := range t.open {
		if present[key] || held && h.heldUnder(key) {
			continue
		}
		b := &l.Boxes[i]
		b.To, b.RealmTo, b.Open = h.Now, realm, false
		b.Aside = h.Aside(key) && !b.Tray
		b.ByPlayer = b.Waits && h.Now-t.pressed <= attnAnswerWindow
		delete(t.open, key)
	}
	t.started = true
	sig := surfaceSig(surfaces)
	if sig != t.sig && len(l.Frames) < maxAttnFrames {
		t.sig = sig
		l.Frames = append(l.Frames, AttnFrame{At: h.Now, Realm: realm, Surfaces: surfaces})
	}
}

func (h *HUD) heldUnder(key string) bool {
	if text, ok := strings.CutPrefix(key, ActBanner); ok {
		return h.BannerActive() && text == h.Juice.Banner
	}
	return true
}

func surfaceSig(list []Surface) string {
	b := make([]byte, 0, 32*len(list))
	for _, s := range list {
		b = append(b, s.Key...)
		b = append(b, byte(s.Rect.X), byte(s.Rect.X>>8), byte(s.Rect.Y), byte(s.Rect.Y>>8), byte(s.Rect.W), byte(s.Rect.W>>8), byte(s.Rect.H), byte(s.Rect.H>>8), '|')
	}
	return string(b)
}

func (l AttnLog) Closed(now, realm float64) AttnLog {
	out := AttnLog{Boxes: slices.Clone(l.Boxes), Frames: l.Frames, Events: l.Events, Marks: l.Marks}
	for i := range out.Boxes {
		if out.Boxes[i].Open {
			out.Boxes[i].To, out.Boxes[i].RealmTo = now, realm
		}
	}
	return out
}
