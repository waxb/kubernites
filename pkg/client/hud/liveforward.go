package hud

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	liveForwardPrefix     = "liveforward"
	IntentLiveForwardStop = "live:forward:stop"
	liveForwardStop       = "STOP"
)

type LiveForward struct {
	ID   int
	Text string
	Tip  string
}

func LiveForwardID(id int) string {
	return liveForwardPrefix + ":" + strconv.Itoa(id)
}

func (c *LiveCard) forwardLines() []liveLine {
	var out []liveLine
	room := liveCardChars - len(liveForwardStop) - 2
	for _, f := range c.Forwards {
		out = append(out, liveLine{text: liveFont.Clip(f.Text, room), literal: true, clr: ui.Palette.GoldBright, id: LiveForwardID(f.ID), tip: f.Tip, stop: true})
	}
	return out
}

func liveStopRect(y int) ui.Rect {
	w := liveFont.Width(liveForwardStop, 1) + 6
	return ui.Rect{X: CardBox.X + CardBox.W - 8 - w, Y: y - 1, W: w, H: ui.LineHeight}
}

func (h *HUD) drawLiveStop(dst *ebiten.Image, y int) {
	r := liveStopRect(y)
	ui.Fill(dst, r, ui.Palette.RedDark)
	ui.Stroke(dst, r, ui.Palette.RedText)
	liveFont.Draw(dst, liveForwardStop, r.X+3, y, 1, ui.Palette.Cream)
}

func (h *HUD) pressLiveForward(rest string, _ Frame) []Intent {
	n, ok := strings.CutPrefix(rest, "stop:")
	if !ok {
		return nil
	}
	id, err := strconv.Atoi(n)
	if err != nil {
		return nil
	}
	return []Intent{{Kind: IntentSound, Command: "click"}, {Kind: IntentLiveForwardStop, Target: strconv.Itoa(id)}}
}

func (h *HUD) ShowLiveOrderPage(page int) {
	h.lw.page = page
}
