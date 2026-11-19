package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

var gridKeyCells = []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9}

const gridKeyNames = "123456789"

func (h *HUD) drawCellKey(dst *ebiten.Image, b Button, keyed bool) {
	cell := cellOf(b.Rect)
	if !keyed || b.Pips > 0 || cell >= len(gridKeyNames) {
		return
	}
	if IsLiveOrder(b.ID) {
		if chord := h.orderChord(b.ID); chord != "" {
			h.Font.Draw(dst, chord, b.Rect.X+3, b.Rect.Y+3, 1, ui.Palette.TextDim)
		}
		return
	}
	h.Font.Draw(dst, gridKeyNames[cell:cell+1], b.Rect.X+3, b.Rect.Y+3, 1, ui.Palette.TextDim)
}

func (h *HUD) GridKeysTaken(f Frame) bool {
	return h.OverlayID == OverlayNone && f.Placing == "" && (h.objectShown(f) || h.PageID != PageMain)
}

func (h *HUD) gridKeys(in Input, f Frame) ([]Intent, bool) {
	if !h.GridKeysTaken(f) {
		return nil, false
	}
	for cell, key := range gridKeyCells {
		if in.Pressed(key) {
			return h.pressCell(cell, f), true
		}
	}
	return nil, false
}

func (h *HUD) pressCell(cell int, f Frame) []Intent {
	b, ok := cellButton(h.CommandButtons(f), cell)
	switch {
	case !ok:
		return nil
	case IsLiveOrder(b.ID) && b.ID != LiveOrderPrefix+":"+liveOrderMore:
		h.Toast(liveDigitTip, true)
		return []Intent{{Kind: IntentSound, Command: "error"}}
	case b.Disabled:
		h.Toast(b.Tip, true)
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	return append(h.Press(b.ID, f), h.IntroTried(f, b.ID)...)
}

func cellButton(buttons []Button, cell int) (Button, bool) {
	for _, b := range buttons {
		if b.Rect == CellRect(cell) {
			return b, true
		}
	}
	return Button{}, false
}

func (h *HUD) openBuild(f Frame) []Intent {
	var out []Intent
	if h.objectShown(f) {
		out = h.leaveObject(f)
	}
	h.PageID, h.OverlayID = PageBuild, OverlayNone
	return append(out, h.IntroTried(f, "cmd:build")...)
}
