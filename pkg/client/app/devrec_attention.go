package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
)

type devAttn struct {
	boxes, frames, events int
	open                  map[int]bool
}

func (d *DevRecorder) noteAttention(s *Session, a devAttn) devAttn {
	boxes, frames, events := s.Hud.AttentionCounts()
	fresh := s.Hud.AttentionFrom(a.boxes, a.frames, a.events)
	next := devAttn{boxes: boxes, frames: frames, events: events, open: map[int]bool{}}
	for i := range a.open {
		next.open[i] = true
	}
	for i, b := range fresh.Boxes {
		next.open[a.boxes+i] = true
		d.write("box", b.Realm, map[string]any{"box": b.Kind, "key": b.Key, "text": b.Text, "slot": b.Slot, "rect": b.Rect, "at": round1(b.From),
			"waits": b.Waits, "idea": b.Idea, "face": b.Face, "urgent": b.Urgent, "read": round1(b.Read())})
	}
	for i := range next.open {
		b, ok := s.Hud.AttentionBox(i)
		if !ok || b.Open {
			continue
		}
		delete(next.open, i)
		d.write("box-left", b.RealmTo, map[string]any{"key": b.Key, "at": round1(b.To), "shown": round1(b.To - b.From), "changes": len(b.Texts)})
	}
	for _, f := range fresh.Frames {
		d.write("surfaces", f.Realm, map[string]any{"at": round1(f.At), "drawn": f.Drawn, "surfaces": compactSurfaces(f.Surfaces)})
	}
	s.Hud.DropAttentionFrames()
	for _, e := range fresh.Events {
		d.write("attention", e.Realm, map[string]any{"at": round1(e.At), "event": e.Kind, "key": e.Key})
	}
	return next
}

func compactSurfaces(list []hud.Surface) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, s := range list {
		m := map[string]any{"slot": s.Slot, "key": s.Key, "rect": []int{s.Rect.X, s.Rect.Y, s.Rect.W, s.Rect.H}}
		if s.Box {
			m["box"] = true
		}
		if s.Asked {
			m["asked"] = true
		}
		if s.Modal {
			m["modal"] = true
		}
		out = append(out, m)
	}
	return out
}
