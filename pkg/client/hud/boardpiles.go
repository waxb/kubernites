package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
)

const (
	pileAnchorX = 8
	pileAnchorY = 10
	pileStoneDX = 12
	pileLogW    = 5
	pileLogH    = 2
	pileStoneW  = 3
	pilePerRow  = 3
	pileFlatY   = 4
)

var (
	logBark   = color.RGBA{122, 78, 42, 255}
	logEnd    = color.RGBA{214, 170, 110, 255}
	stoneFace = color.RGBA{150, 150, 144, 255}
	stoneLit  = color.RGBA{196, 196, 188, 255}
)

func slotPiles(v BoardView, s Slot) (int, int) {
	if s.Real && len(s.Pod.Requests) > 0 {
		return world.HutPiles(s.Pod)
	}
	ghost := model.PodView{QoS: v.Chain.QoS, State: model.HearthReady,
		Requests: []model.ContainerRequest{{CPUMilli: v.Facts.Per.CPU, MemoryBytes: v.Facts.Per.Memory}}}
	if ghost.QoS == "" {
		ghost.QoS = model.QoSBurstable
	}
	return world.HutPiles(ghost)
}

func pileArea(slot ui.Rect) ui.Rect {
	return ui.Rect{X: slot.X, Y: slot.Y + yardBase + yardGroundH + 1, W: pileStoneDX + pileLogW*pilePerRow, H: pileAnchorY + 2}
}

func (h *HUD) drawPiles(dst *ebiten.Image, v BoardView, s Slot, x, y int, o render.SpriteOptions) {
	wood, stone := slotPiles(v, s)
	h.drawPile(dst, art.PileWood, wood, x, y, o)
	h.drawPile(dst, art.PileStone, stone, x+pileStoneDX, y, o)
	if s.Kind == SlotWaits && wood+stone > 0 {
		ui.Fill(dst, ui.Rect{X: x, Y: y + pileAnchorY + 1, W: pileStoneDX + pileLogW*pilePerRow, H: 1}, ui.Palette.RedText)
	}
}

func (h *HUD) drawPile(dst *ebiten.Image, kind string, units, x, y int, o render.SpriteOptions) {
	if units <= 0 {
		return
	}
	if h.Lib.Draw(dst, art.RequestPileName(kind, min(units, art.RequestPileUnits)), float64(x+pileAnchorX), float64(y+pileAnchorY), 1, o) {
		return
	}
	for i := range units {
		row, col := i/pilePerRow, i%pilePerRow
		if kind == art.PileWood {
			lx, ly := x+col*pileLogW+row*2, y+pileFlatY-row*(pileLogH+1)
			ui.Fill(dst, ui.Rect{X: lx, Y: ly, W: pileLogW - 1, H: pileLogH}, logBark)
			ui.Fill(dst, ui.Rect{X: lx + pileLogW - 2, Y: ly, W: 1, H: pileLogH}, logEnd)
			continue
		}
		bx, by := x+col*(pileStoneW+1)+row*2, y+pileFlatY-row*(pileStoneW+1)
		ui.Fill(dst, ui.Rect{X: bx, Y: by, W: pileStoneW, H: pileStoneW}, stoneFace)
		ui.Fill(dst, ui.Rect{X: bx, Y: by, W: pileStoneW, H: 1}, stoneLit)
	}
}
