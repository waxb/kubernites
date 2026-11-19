package hud

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const cardLinePrefix = "cardline"

func init() {
	prefixHandlers[cardLinePrefix] = func(h *HUD, rest string, f Frame) []Intent { return h.Press(rest, f) }
}

func (h *HUD) cardBarRect(c Card, f Frame) (ui.Rect, int, bool) {
	box := PortraitBox()
	x, y, w := box.X+box.W+8, CardBox.Y+6, CardBox.X+CardBox.W-box.X-box.W-14
	y += ui.LineHeight
	if c.K8s != "" {
		y += ui.LineHeight
	}
	bottom := CardBottom(c, f)
	lines := h.FitLines(c.Lines, w, (bottom-y)/ui.LineHeight)
	room := (bottom - y) / ui.LineHeight
	for i, l := range lines {
		if l.Bar {
			return ui.Rect{X: x, Y: y, W: w, H: ui.LineHeight}, i, true
		}
		rows := len(h.Font.WrapLines(l.Text, w, 1, room))
		y += rows * ui.LineHeight
		if room -= rows; room <= 0 {
			break
		}
	}
	return ui.Rect{}, 0, false
}

func (h *HUD) cardLineButtons(f Frame) []Button {
	if !HavenSelected(f) {
		return nil
	}
	p, ok := SelectedPort(f)
	if !ok {
		return nil
	}
	verb, gate := PortVerbOpen, p.Gates.Open
	if p.Open {
		verb, gate = PortVerbRaise, p.Gates.Upgrade
		if p.Next == nil {
			return nil
		}
	}
	r, _, found := h.cardBarRect(h.CardFor(f), f)
	if !found {
		return nil
	}
	b := Button{ID: cardLinePrefix + ":" + PortButtonID(verb, p.ID), Rect: r, Label: "", Tip: portBarTip(p, verb)}
	return []Button{disabledBy(b, gate)}
}

func portBarTip(p view.PortView, verb string) string {
	if verb == PortVerbOpen {
		return PortTip(p.Effects.Open, portOpenK8s())
	}
	return PortTip(p.Effects.Upgrade, p.K8s)
}
