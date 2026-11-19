package hud

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	arrowPoint = 17
	arrowBase  = 11
	arrowWing  = 6
	pingFrom   = 6
	pingTo     = 26
)

func levelColor(level int) color.RGBA {
	switch level {
	case LevelDanger:
		return ui.Palette.Red
	case LevelWarn:
		return ui.Palette.GoldBright
	}
	return ui.Palette.Parchment
}

func (h *HUD) drawAlerts(dst *ebiten.Image, f Frame) {
	for i, a := range h.Arrows(f) {
		h.drawArrow(dst, a, h.Hovering && h.Hover.ID == "alert:"+strconv.Itoa(i))
	}
	h.drawPing(dst, f)
}

func (h *HUD) drawArrow(dst *ebiten.Image, a AlertArrow, hot bool) {
	clr := levelColor(a.Level)
	cx, cy := float32(a.X), float32(a.Y)
	dx, dy := float32(a.DX), float32(a.DY)
	var p vector.Path
	p.MoveTo(cx+dx*arrowPoint, cy+dy*arrowPoint)
	p.LineTo(cx+dx*arrowBase-dy*arrowWing, cy+dy*arrowBase+dx*arrowWing)
	p.LineTo(cx+dx*arrowBase+dy*arrowWing, cy+dy*arrowBase-dx*arrowWing)
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, &vector.FillOptions{}, op)
	r := ArrowRect(a)
	ui.Fill(dst, r.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 235))
	border := clr
	if a.Level == LevelDanger {
		border = ui.Tint(clr, uint8(120+135*pulse(h.Now, h.Reduce)))
	}
	if hot {
		border = ui.Palette.Cream
	}
	ui.Stroke(dst, r, border)
	if !h.Lib.Icon(dst, alertIconName(a.Kind), r.X+2, r.Y+2) {
		h.Font.DrawCentered(dst, "!", r, 1, clr)
	}
	if a.Count > 1 {
		h.drawCount(dst, a.Count, r, clr)
	}
}

func alertIconName(kind string) string {
	name := alertIcon(kind)
	return name[len("icon/"):]
}

func (h *HUD) countBadgeRect(n int, r ui.Rect) ui.Rect {
	w, _ := h.Font.Measure(countBadge(n)+" ALERTS", 1)
	x := min(max(r.X+r.W-w/2-4, 0), Width-w-4)
	return ui.Rect{X: x, Y: r.Y - 5, W: w + 4, H: 11}
}

func (h *HUD) drawCount(dst *ebiten.Image, n int, r ui.Rect, clr color.RGBA) {
	text := countBadge(n) + " ALERTS"
	badge := h.countBadgeRect(n, r)
	ui.Fill(dst, badge, clr)
	ui.Stroke(dst, badge, ui.Palette.Ink)
	h.Font.Draw(dst, text, badge.X+2, badge.Y+1, 1, ui.Palette.Ink)
}

func (h *HUD) drawPing(dst *ebiten.Image, f Frame) {
	age := h.Now - h.ping.start
	if !h.ping.on || age < 0 || age > pingLife {
		return
	}
	sx, sy, ok := ViewPoint(f.View, h.ping.at[0], h.ping.at[1])
	if !ok || !worldView().Contains(int(sx), int(sy)) {
		return
	}
	t := age / pingLife
	radius := pingFrom + (pingTo-pingFrom)*t
	if h.Reduce {
		radius = (pingFrom + pingTo) / 2
	}
	alpha := uint8(255 * (1 - t*t))
	vector.StrokeCircle(dst, float32(sx), float32(sy), float32(radius), 2, ui.Tint(ui.Palette.GoldBright, alpha), false)
	vector.StrokeCircle(dst, float32(sx), float32(sy), float32(radius*.55), 1, ui.Tint(ui.Palette.Cream, alpha), false)
}

func (h *HUD) drawBlips(dst *ebiten.Image, f Frame, v MinimapView) {
	blink := h.Reduce || math.Mod(h.Now*2, 1) < .6
	for _, a := range f.Alerts {
		level := SeverityOf(a)
		if level == LevelDanger && !blink {
			continue
		}
		x, y := v.FromGrid(a.At[0], a.At[1])
		vector.FillRect(dst, x-2, y-2, 5, 5, ui.Palette.Ink, false)
		vector.FillRect(dst, x-1, y-1, 3, 3, levelColor(level), false)
	}
}
