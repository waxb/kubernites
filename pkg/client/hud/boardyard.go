package hud

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const (
	yardBase    = 46
	yardGroundH = 4
	hutShift    = 5
	personDX    = 12
	personStep  = 5
	petDX       = -15
	popDrop     = 14.0
	ghostAlpha  = 0.45
	leaveAlpha  = 0.3
	miniHutH    = 12
	miniBase    = 40
	captionPad  = 2
	stakeH      = 5
	foremanZoom = 2
	foremanLeft = 4
	flagLift    = 10
	flagDX      = 7
	foremanFPS  = 3.0
	whistleTime = 1.2
	whistleRise = 8.0
	whistleSpan = 6.0
	whistleTop  = 24
)

const foremanSprite = "tallyman"

var (
	landGround   = color.RGBA{86, 112, 58, 255}
	landEdge     = color.RGBA{58, 78, 40, 255}
	landLot      = color.RGBA{132, 164, 96, 255}
	noLandGround = color.RGBA{96, 52, 48, 255}
	ghostTint    = [3]float32{0.7, 0.85, 1.1}
	waitTint     = [3]float32{1.1, 0.6, 0.55}
)

func (h *HUD) drawYard(dst *ebiten.Image, v BoardView, f Frame) {
	rects, mode := v.plots()
	for i, r := range rects {
		h.drawGround(dst, r, v.plotKind(i), mode)
	}
	for i, r := range rects[:min(len(rects), len(v.Slots))] {
		h.drawSlot(dst, v, i, r, mode, f)
	}
	h.drawWaitCaption(dst, v, rects)
	h.drawForeman(dst, v)
	if extra := len(v.Slots) - len(rects); extra > 0 {
		h.Font.Draw(dst, fmt.Sprintf("+%d", extra), v.yard.X+v.yard.W+cbTrackGap, v.yard.Y+yardBase-ui.GlyphHeight, 1, ui.Palette.Cream)
	}
	if len(v.Slots) == 0 {
		h.Font.Draw(dst, ui.Bridge("No huts: drag right.", "0 replicas: drag right."), v.yard.X+2, v.yard.Y+2, 1, ui.Palette.TextDim)
	}
}

func groundLine(mode plotMode) int {
	if mode == plotSprites {
		return yardBase
	}
	return miniBase
}

func (h *HUD) drawGround(dst *ebiten.Image, r ui.Rect, kind string, mode plotMode) {
	ground := ui.Rect{X: r.X, Y: r.Y + groundLine(mode), W: max(r.W-1, 1), H: yardGroundH}
	if kind == SlotWaits || kind == PlotNoLand {
		ui.Fill(dst, ground, noLandGround)
		for x := ground.X; x < ground.X+ground.W; x += 4 {
			ui.Fill(dst, ui.Rect{X: x, Y: ground.Y + 1, W: 2, H: 1}, ui.Palette.RedText)
		}
		return
	}
	ui.Fill(dst, ground, landGround)
	ui.Fill(dst, ui.Rect{X: ground.X, Y: ground.Y + ground.H - 1, W: ground.W, H: 1}, landEdge)
	if kind == PlotOpen && mode != plotBars {
		lot := ui.Rect{X: ground.X + 3, Y: ground.Y - stakeH, W: max(ground.W-6, 2), H: stakeH}
		for x := lot.X; x < lot.X+lot.W; x += 2 {
			ui.Fill(dst, ui.Rect{X: x, Y: lot.Y, W: 1, H: 1}, landLot)
		}
		ui.Fill(dst, ui.Rect{X: lot.X, Y: lot.Y, W: 1, H: lot.H}, landLot)
		ui.Fill(dst, ui.Rect{X: lot.X + lot.W - 1, Y: lot.Y, W: 1, H: lot.H}, landLot)
	}
}

func (h *HUD) PopOffset(i int) (float64, float32) {
	if h.Reduce || i >= len(h.Board.born) {
		return 0, 1
	}
	t := min(max((h.Now-h.Board.born[i])/PopTime, 0), 1)
	ease := 1 - math.Pow(1-t, 3)
	return -(1 - ease) * popDrop, float32(max(ease, 0.05))
}

func (h *HUD) drawSlot(dst *ebiten.Image, v BoardView, i int, r ui.Rect, mode plotMode, f Frame) {
	s := v.Slots[i]
	switch mode {
	case plotIcons:
		h.drawMiniSlot(dst, v, i, r)
		return
	case plotBars:
		ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y + miniBase - miniHutH, W: max(r.W-1, 1), H: miniHutH}, slotColor(s))
		return
	}
	dy, fade := h.PopOffset(i)
	cx, base := float64(r.X+r.W/2-hutShift), float64(r.Y+yardBase)+dy
	o := slotLook(s.Kind, fade)
	h.drawHut(dst, v, s, cx, base, o, gfx.AgeTier(f.State.Age))
	h.drawPeople(dst, v, s, cx, base, o)
	h.drawPiles(dst, v, s, r.X, r.Y+yardBase+yardGroundH+1, o)
	if s.Real && (f.Selection.Pod.Name == s.Pod.Name && f.Selection.Kind != "structure" || h.rowHot(CrewPrefix+":pick:pod:"+s.Pod.Name)) {
		ui.Stroke(dst, ui.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: yardBase + yardGroundH}, ui.Palette.GoldBright)
	}
}

func (h *HUD) drawForeman(dst *ebiten.Image, v BoardView) {
	reach := (flagDX + crewGlyphW) * foremanZoom
	x := (v.box.X+v.yard.X)/2 - (reach-foremanLeft)/2
	if x-foremanLeft < v.box.X+cbPad || x+reach > v.yard.X {
		return
	}
	base := float64(v.yard.Y + yardBase)
	tone := model.RevisionColor(newestOf(v.Chain))
	drawFlagAt(dst, x+flagDX*foremanZoom, int(base)-flagLift*foremanZoom, tone, tone, foremanZoom)
	frame := 0
	if !h.Reduce {
		frame = int(h.Now * foremanFPS)
	}
	h.Lib.DrawFirst(dst, gfx.WalkerTierVariantNames(foremanSprite, 0, 1, "idle", "se", frame), float64(x), base, foremanZoom, render.SpriteOptions{})
	if h.whistling() {
		rise := int(math.Mod(h.Now*whistleRise, whistleSpan))
		top := int(base) - whistleTop*foremanZoom
		ui.Fill(dst, ui.Rect{X: x + 5*foremanZoom, Y: top - rise, W: 2, H: 2}, ui.Palette.GoldBright)
		ui.Fill(dst, ui.Rect{X: x + 8*foremanZoom, Y: top - 3 - rise/2, W: 2, H: 2}, ui.Palette.Cream)
	}
}

func (h *HUD) whistling() bool {
	for _, at := range h.Board.born {
		if h.Now-at < whistleTime && h.Now >= at {
			return true
		}
	}
	return false
}

func slotLook(kind string, fade float32) render.SpriteOptions {
	switch kind {
	case SlotFits:
		return render.SpriteOptions{Alpha: ghostAlpha * fade, Tint: ghostTint}
	case SlotWaits:
		return render.SpriteOptions{Alpha: ghostAlpha * fade, Tint: waitTint}
	case SlotLeaving:
		return render.SpriteOptions{Alpha: leaveAlpha}
	}
	return render.SpriteOptions{Alpha: fade}
}

func slotState(s Slot) string {
	switch s.Kind {
	case SlotWaits:
		return model.HearthPending
	case SlotFits, SlotRising:
		return model.HearthKindling
	}
	if s.Real {
		return s.Pod.State
	}
	return model.HearthReady
}

func (h *HUD) drawHut(dst *ebiten.Image, v BoardView, s Slot, cx, base float64, o render.SpriteOptions, age int) {
	tier, qos, variant := model.TierOf(v.Chain.Image), v.Chain.QoS, 0
	if s.Real {
		tier, variant = s.Pod.Tier, gfx.LookVariant(s.Pod.Name)
		if s.Pod.QoS != "" {
			qos = s.Pod.QoS
		}
	}
	if _, ok := h.Lib.DrawFirst(dst, gfx.HearthLookNames(tier, slotState(s), qos, age, variant), cx, base, 1, o); ok {
		return
	}
	drawHutIcon(dst, ui.Rect{X: int(cx) - 10, Y: int(base) - 16, W: 20, H: 16}, slotColor(s), false)
}

func slotColor(s Slot) color.RGBA {
	switch s.Kind {
	case SlotWaits:
		return ui.Palette.RedDark
	case SlotFits:
		return ui.Palette.BlueDark
	case SlotLeaving:
		return ui.Palette.Muted
	}
	if c, ok := hearthColors[slotState(s)]; ok {
		return c
	}
	return ui.Palette.Muted
}

func (h *HUD) drawPeople(dst *ebiten.Image, v BoardView, s Slot, cx, base float64, o render.SpriteOptions) {
	sprites := slotPeople(v.Chain, s)
	for i, sprite := range sprites {
		variant, tier := 0, max(model.TierOf(v.Chain.Image), 1)
		if s.Real {
			variant, tier = world.LookOf(s.Pod.Person(i))
		}
		x := cx + min(personDX, float64(v.Track.W/max(v.Top, 1))/2) + float64(i*personStep)
		h.Lib.DrawFirst(dst, gfx.WalkerTierVariantNames(sprite, variant, tier, "idle", "sw", 0), x, base+1, 1, o)
	}
	if !s.Real {
		return
	}
	for i, pet := range s.Pod.Familiars {
		h.Lib.Draw(dst, gfx.WalkerName(pet, "idle", "se", 0), cx+petDX-float64(i*personStep), base+1, 1, o)
	}
}

func slotPeople(c model.CrewChain, s Slot) []string {
	if s.Real && len(s.Pod.People) > 0 {
		out := make([]string, len(s.Pod.People))
		for i, k := range s.Pod.People {
			out[i] = k.Sprite
		}
		return out
	}
	if s.Real && s.Pod.Sprite != "" {
		return []string{s.Pod.Sprite}
	}
	_, sprite := soulstone.ProfessionOf(c.Image)
	return []string{sprite}
}

func (h *HUD) drawMiniSlot(dst *ebiten.Image, v BoardView, i int, r ui.Rect) {
	s := v.Slots[i]
	dy, _ := h.PopOffset(i)
	hut := ui.Rect{X: r.X + 1, Y: r.Y + miniBase - miniHutH + int(dy), W: max(r.W-3, 4), H: miniHutH}
	drawHutIcon(dst, hut, slotColor(s), s.Real && h.rowHot(CrewPrefix+":pick:pod:"+s.Pod.Name))
	wood, stone := slotPiles(v, s)
	y := r.Y + miniBase + yardGroundH + 1
	ui.Fill(dst, ui.Rect{X: r.X + 1, Y: y, W: min(wood*2, r.W-2), H: 2}, logBark)
	ui.Fill(dst, ui.Rect{X: r.X + 1, Y: y + 3, W: min(stone*2, r.W-2), H: 2}, stoneFace)
}

var waitCaptions = [][2]string{
	{"WAITING: NEEDS A KEEP", "PENDING: NO NODE FITS"},
	{"NEEDS A KEEP", "NO NODE FITS"},
	{"WAITING", "PENDING"},
}

var tooBigCaptions = [][2]string{
	{"WAITING: TOO BIG FOR A KEEP", "PENDING: TOO BIG FOR A NODE"},
	{"TOO BIG FOR A KEEP", "TOO BIG FOR A NODE"},
	{"WAITING", "PENDING"},
}

func waitCaptionsOf(v BoardView) [][2]string {
	if v.Facts.tooBig() {
		return tooBigCaptions
	}
	return waitCaptions
}

func (h *HUD) drawWaitCaption(dst *ebiten.Image, v BoardView, rects []ui.Rect) {
	first, last := -1, -1
	for i := range rects[:min(len(rects), len(v.Slots))] {
		if v.Slots[i].Kind == SlotWaits {
			last = i
			if first < 0 {
				first = i
			}
		}
	}
	if first < 0 {
		return
	}
	group := ui.Rect{X: rects[first].X, Y: v.yard.Y, W: rects[last].X + rects[last].W - rects[first].X, H: ui.LineHeight}
	for _, pair := range waitCaptionsOf(v) {
		text := ui.Bridge(pair[0], pair[1])
		if w := h.Font.Width(text, 1); w+2*captionPad <= group.W {
			tag := ui.Rect{X: group.X + (group.W-w)/2 - captionPad, Y: group.Y, W: w + 2*captionPad, H: ui.LineHeight}
			ui.Fill(dst, tag, ui.Palette.RedDark)
			h.Font.Draw(dst, text, tag.X+captionPad, tag.Y+1, 1, ui.Palette.Cream)
			return
		}
	}
}
