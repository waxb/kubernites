package hud

import (
	"slices"

	"github.com/waxb/kubernites/pkg/view"
)

const TroubleRead = 6.0

func (h *HUD) noteLiveTrouble(f Frame) {
	next := map[string]view.CalamityView{}
	for _, v := range f.State.Calamities {
		if v.Mended == "" {
			next[v.ID] = v
		}
	}
	if v, ok := h.Ops.live[h.Director.Stage.focus]; ok {
		if _, still := next[v.ID]; !still && h.troubleUnread(f, v.ID) {
			next[v.ID] = v
		}
	}
	h.Ops.live = next
}

func (h *HUD) troubleUnread(f Frame, id string) bool {
	s := h.Director.Stage
	at, on := s.On[ActTrouble+id]
	return id != "" && on && s.Has(ActTrouble+id) && !s.acted[id] && f.Clock-at < TroubleRead
}

func (h *HUD) withUnreadTrouble(f Frame) Frame {
	id := h.Director.Stage.focus
	held, ok := h.Ops.live[id]
	if !ok || !h.troubleUnread(f, id) {
		return f
	}
	i := slices.IndexFunc(f.State.Calamities, func(v view.CalamityView) bool { return v.ID == id })
	if i >= 0 && f.State.Calamities[i].Mended == "" {
		return f
	}
	out := f
	list := slices.Clone(f.State.Calamities)
	if i >= 0 {
		list[i] = held
	} else {
		list = append([]view.CalamityView{held}, list...)
	}
	out.State.Calamities = list
	return out
}
