package hud

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	bannerIn       = 0.3
	bannerHold     = 2.5
	bannerOut      = 0.3
	BannerY        = 120
	vignetteTime   = 0.4
	vignetteAlpha  = 0.35
	vignetteStride = 4
	causeW         = 380
	pinnedH        = 16
	PinnedResume   = "pause:resume"
	resumeW        = 96
	resumeH        = 20
	resumeGap      = 4
	pinnedPad      = 10
	pinnedFix      = "Close or fix its card, or Space."
	pinnedShort    = "Space goes on."
)

type juice struct {
	pinned     string
	pinCause   string
	Banner     string `json:"-"`
	Cause      string `json:"-"`
	feat       bool
	BannerAt   float64 `json:"-"`
	spot       [2]int
	spotFor    float64
	spotSet    bool
	vignetteAt float64
	ranks      map[string]int
	flash      map[string]float64
	locked     map[int]bool
	Tiers      map[int]float64 `json:"-"`
	vignette   *ebiten.Image
}

func newJuice() juice {
	return juice{BannerAt: -100, vignetteAt: -100, ranks: map[string]int{}, flash: map[string]float64{}, locked: map[int]bool{}, Tiers: map[int]float64{}}
}

func (h *HUD) Banner(text string) {
	h.queueBanner(text, "", false)
}

func (h *HUD) BannerCause(text, cause string) {
	h.queueBanner(text, cause, false)
}

func (h *HUD) BannerWhile(text, cause string, still func() bool) {
	h.queueBannerItem(bannerItem{text: text, cause: cause, still: still})
}

func (h *HUD) FeatBanner(text, cause string) {
	h.queueBanner(text, cause, true)
}

func (h *HUD) PinPause(title, cause string) {
	h.Juice.pinned, h.Juice.pinCause = title, cause
}

func (h *HUD) PinnedPause() (string, string) { return h.Juice.pinned, h.Juice.pinCause }

func (h *HUD) pinnedRect(f Frame) (ui.Rect, bool) {
	if h.Juice.pinned == "" || !f.Paused {
		return ui.Rect{}, false
	}
	left, right := h.pinnedGap(f)
	w := h.Font.Width(h.pinnedText(f), 1) + 2*pinnedPad
	return ui.Rect{X: (left+right)/2 - w/2, Y: TopH + 2, W: w, H: pinnedH}, true
}

func (h *HUD) pinnedGap(f Frame) (int, int) {
	left, right := 0, Width
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		left = h.ObjectiveRectOf(f).X + h.ObjectiveRectOf(f).W + pinnedPad
	}
	if BoardShown(f) {
		right = FreightRect(f).X - pinnedPad
	}
	return left, right
}

func (h *HUD) pinnedText(f Frame) string {
	left, right := h.pinnedGap(f)
	room := right - left - 2*pinnedPad
	for _, text := range []string{h.Juice.pinned + ". " + h.Juice.pinCause, h.Juice.pinned + ". " + pinnedFix, h.Juice.pinned + ". " + pinnedShort} {
		if h.Font.Width(text, 1) <= room {
			return text
		}
	}
	return h.Juice.pinned
}

func (h *HUD) pinnedButtons(f Frame) []Button {
	r, ok := h.pinnedRect(f)
	if !ok {
		return nil
	}
	return []Button{{ID: PinnedResume, Rect: ui.Rect{X: r.X + r.W/2 - resumeW/2, Y: r.Y + r.H + resumeGap, W: resumeW, H: resumeH}, Label: "RESUME",
		Tip: "The realm paused itself for you: RESUME lets time run again [Space]"}}
}

func (h *HUD) drawPinned(dst *ebiten.Image, f Frame) {
	r, ok := h.pinnedRect(f)
	if !ok {
		return
	}
	ui.Fill(dst, r.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Palette.ConsoleEdge)
	ui.Stroke(dst, r, ui.Palette.Gold)
	h.Font.DrawCentered(dst, h.pinnedText(f), ui.Rect{X: r.X, Y: r.Y + 4, W: r.W, H: ui.CapHeight}, 1, ui.Palette.GoldBright)
	for _, b := range h.pinnedButtons(f) {
		h.drawPrimary(dst, b)
	}
}

func (h *HUD) Vignette() {
	h.Juice.vignetteAt = h.events
}

func BannerOffset(age float64) (float64, bool) {
	switch {
	case age < 0 || age > bannerIn+bannerHold+bannerOut:
		return 0, false
	case age < bannerIn:
		return age/bannerIn - 1, true
	case age < bannerIn+bannerHold:
		return 0, true
	}
	return -(age - bannerIn - bannerHold) / bannerOut, true
}

func (h *HUD) drawBanner(dst *ebiten.Image, f Frame) {
	off, ok := BannerOffset(h.Now - h.Juice.BannerAt)
	if !ok || h.Juice.Banner == "" || !h.CardsDrawn(f) && !(h.Juice.feat && h.OverlayID == OverlayAge) {
		return
	}
	cx, top := h.settleBanner(f)
	y, fade := float64(top), float32(min(max(1+off, 0), 1))
	if fade <= 0 {
		return
	}
	a := uint8(255 * fade)
	frame := int(h.events*6) % 3
	name := "fx/banner/" + string(rune('0'+frame))
	w := h.bannerWidth()
	if !h.drawBannerCloth(dst, name, float64(cx), y, w, fade) {
		ui.Fill(dst, ui.Rect{X: cx - w/2, Y: int(y) - 20, W: w, H: 40}, ui.Tint(ui.Palette.Gold, a))
	}
	h.Font.DrawCentered(dst, h.Juice.Banner, ui.Rect{X: cx - w/2, Y: int(y) - 12, W: w, H: 16}, 1, ui.Tint(ui.Palette.GoldBright, a))
	h.drawCause(dst, cx, int(y)+24, a)
}

func (h *HUD) drawCause(dst *ebiten.Image, cx, y int, a uint8) {
	if h.Juice.Cause == "" {
		return
	}
	rows := h.Font.WrapLines(h.Juice.Cause, causeW-16, 1, 3)
	r := ui.Rect{X: cx - causeW/2, Y: y, W: causeW, H: len(rows)*ui.LineHeight + 8}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, uint8(225*int(a)/255)))
	ui.Stroke(dst, r, ui.Tint(ui.Palette.Gold, a))
	for i, row := range rows {
		h.Font.DrawCentered(dst, row, ui.Rect{X: r.X, Y: r.Y + 4 + i*ui.LineHeight, W: r.W, H: ui.CapHeight}, 1, ui.Tint(ui.Palette.Cream, a))
	}
}

func VignetteAlpha(age float64) float64 {
	if age < 0 || age > vignetteTime {
		return 0
	}
	return vignetteAlpha * (1 - age/vignetteTime)
}

func vignetteImage() *ebiten.Image {
	w, h := Width/vignetteStride, Height/vignetteStride
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := ui.Palette.RedDark
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (float64(x)/float64(w-1) - .5) * 2
			dy := (float64(y)/float64(h-1) - .5) * 2
			d := math.Min(math.Hypot(dx, dy)/math.Sqrt2, 1)
			a := uint8(255 * math.Pow(d, 1.6))
			img.SetRGBA(x, y, color.RGBA{uint8(int(c.R) * int(a) / 255), uint8(int(c.G) * int(a) / 255), uint8(int(c.B) * int(a) / 255), a})
		}
	}
	return ebiten.NewImageFromImage(img)
}

func (h *HUD) drawVignette(dst *ebiten.Image) {
	a := VignetteAlpha(h.events - h.Juice.vignetteAt)
	if a <= 0 || h.Reduce {
		return
	}
	if h.Juice.vignette == nil {
		h.Juice.vignette = vignetteImage()
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(vignetteStride, vignetteStride)
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(h.Juice.vignette, op)
}

func (h *HUD) watchRanks(f Frame) {
	next := make(map[string]int, len(f.State.Reforge.Patterns))
	for _, p := range f.State.Reforge.Patterns {
		if prev, ok := h.Juice.ranks[p.ID]; ok && p.Rank > prev {
			h.Juice.flash[p.ID] = h.events
		}
		next[p.ID] = p.Rank
	}
	h.Juice.ranks = next
	h.watchTiers(f)
}

func (h *HUD) watchTiers(f Frame) {
	locked := map[int]bool{}
	for _, p := range f.State.Reforge.Patterns {
		locked[p.Tier] = locked[p.Tier] || p.Locked
	}
	for tier, now := range locked {
		if was, ok := h.Juice.locked[tier]; ok && was && !now {
			h.Juice.Tiers[tier] = h.events
		}
	}
	h.Juice.locked = locked
}

func pulse(now float64, reduce bool) float64 {
	if reduce {
		return 1
	}
	return .5 + .5*math.Sin(now*2*math.Pi)
}
