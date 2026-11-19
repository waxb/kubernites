package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	goalShare  = 0.9
	shareAfter = 5.0
)

func gateMode(f Frame) bool { return HandsNeeded(f.State.Freight) }

func ServedShare(f Frame) (float64, bool) {
	p := f.State.Pilgrims
	total := p.TotalServed + p.TotalDropped
	if total < shareAfter {
		return 0, false
	}
	return p.TotalServed / total, true
}

func GateLines(f Frame) []string {
	p, fv := f.State.Pilgrims, f.State.Freight
	first := "The quay is empty: a ship is on its way"
	if next := fv.Docking.Next - f.State.RealmTime; next > 0 {
		first = "The quay is empty: the next ship docks in " + Duration(next)
	}
	if waiting := int(parcelsWaiting(fv) + .5); waiting > 0 {
		first = fmt.Sprintf("%d parcels wait: click the pile", waiting)
	}
	return []string{
		first,
		fmt.Sprintf("delivered %d, spoiled %d", int(p.TotalServed), int(p.TotalDropped)),
		ui.Bridge("GOAL: 9 in 10 fresh: the Age's first ask", "GOAL: 9 in 10 on-time: Dev's first ask"),
	}
}

func parcelsWaiting(fv view.FreightView) float64 {
	for _, k := range fv.Kinds {
		if k.Kind == "parcel" {
			return k.Waiting
		}
	}
	return 0
}

func cardRows(f Frame) int {
	if gateMode(f) {
		return 1 + len(GateLines(f))
	}
	return 2 + len(Guilds(f))
}

func (h *HUD) drawGateCard(dst *ebiten.Image, f Frame) {
	x, y := CardBox.X+8, CardBox.Y+6
	h.Font.DrawShadow(dst, "THE QUAY", x, y, 1, ui.Palette.Gold)
	if share, ok := ServedShare(f); ok {
		clr := ui.Palette.Green
		if share < goalShare {
			clr = ui.Palette.RedText
		}
		text := fmt.Sprintf("FRESH %d%%", int(share*100))
		w, _ := h.Font.Measure(text, 1)
		h.Font.Draw(dst, text, CardBox.X+CardBox.W-8-w, y, 1, clr)
	}
	colors := []color.RGBA{ui.Palette.Cream, ui.Palette.TextDim, ui.Palette.Gold}
	for i, l := range GateLines(f) {
		y += ui.LineHeight
		if i == 0 {
			y += 2
		}
		h.Font.Draw(dst, l, x, y, 1, colors[i])
	}
	h.drawNext(dst, f)
}
