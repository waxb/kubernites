package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	portsRowH     = 12
	portsLabel    = "PORTS"
	portsNameMax  = 8
	portsChipPad  = 8
	freightPortID = FreightTip + "port:"
)

func PortsRowShown(f Frame) bool {
	return len(f.State.Ports) >= 2 && BerthsOffered(f) && (!f.BoardCompact || openPorts(f) >= 2)
}

func openPorts(f Frame) int {
	n := 0
	for _, p := range f.State.Ports {
		if p.Open {
			n++
		}
	}
	return n
}

func portsRowY(f Frame, r ui.Rect) int {
	rows := boardBodyRows(f) * ui.LineHeight
	if !f.BoardCompact && hasChips(f.State.Freight) {
		rows += chipRowH
	}
	return r.Y + BoardPad + rows
}

func portChipName(p view.PortView) string {
	name := p.Name
	if first, _, found := strings.Cut(name, " "); found {
		name = first
	}
	runes := []rune(strings.ToUpper(name))
	return string(runes[:min(len(runes), portsNameMax)])
}

func (h *HUD) portChips(f Frame, r ui.Rect) []Button {
	if !PortsRowShown(f) {
		return nil
	}
	start := r.X + BoardPad + h.Font.Width(portsLabel, 1) + ui.Advance
	y := portsRowY(f, r)
	pad := portsChipPad
	for pad > 2 && h.portsWidth(f, pad) > r.X+r.W-BoardPad-start {
		pad -= 2
	}
	x, out := start, []Button{}
	for _, p := range f.State.Ports {
		label := portChipName(p)
		w := h.Font.Width(label, 1) + pad
		if x+w > r.X+r.W-BoardPad {
			break
		}
		out = append(out, Button{ID: freightPortID + p.ID, Rect: ui.Rect{X: x, Y: y, W: w, H: portsRowH}, Label: label, Tip: portChipTip(p)})
		x += w + 2
	}
	return out
}

func (h *HUD) portsWidth(f Frame, pad int) int {
	w := 0
	for _, p := range f.State.Ports {
		w += h.Font.Width(portChipName(p), 1) + pad + 2
	}
	return w - 2
}

func portChipTip(p view.PortView) string {
	if !p.Open {
		return "Go to " + p.Name + ": a bare shore, not open yet."
	}
	return "Go to " + p.Name + ": level " + Compact(float64(p.Level)) + ", " + plural(p.Berths, "berth", "berths") + "."
}

func (h *HUD) drawPortsRow(dst *ebiten.Image, f Frame, r ui.Rect) {
	if !PortsRowShown(f) {
		return
	}
	y := portsRowY(f, r)
	h.Font.Draw(dst, portsLabel, r.X+BoardPad, y+2, 1, ui.Palette.TextDim)
	for _, b := range h.portChips(f, r) {
		ink := ui.Palette.Cream
		if h.Hovering && h.Hover.ID == b.ID {
			ui.Fill(dst, b.Rect, rowHover)
		}
		h.Font.Draw(dst, b.Label, b.Rect.X+(b.Rect.W-h.Font.Width(b.Label, 1))/2, y+2, 1, ink)
	}
}

func (h *HUD) pressPortChip(rest string) []Intent {
	id, ok := strings.CutPrefix(rest, "port:")
	if !ok {
		return nil
	}
	return []Intent{{Kind: IntentHarbor, Target: id}, {Kind: IntentSound, Command: "click"}}
}
