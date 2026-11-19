package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	bannerBaseW  = 2 * bannerHalfW
	bannerCapW   = 80
	bannerTileW  = 80
	bannerMinW   = 2 * bannerCapW
	bannerMargin = 84
	bannerMaxW   = bannerBaseW + 5*bannerTileW
)

type bannerPart struct{ srcX, w, dx int }

func (h *HUD) bannerWidth() int {
	return bannerWidthFor(h.Font.Width(h.Juice.Banner, 1))
}

func bannerWidthFor(text int) int {
	extra := text + bannerMargin - bannerMinW
	if extra <= 0 {
		return bannerMinW
	}
	return min(bannerMinW+(extra+bannerTileW-1)/bannerTileW*bannerTileW, bannerMaxW)
}

func bannerParts(w int) []bannerPart {
	tiles := (w - 2*bannerCapW) / bannerTileW
	out := make([]bannerPart, 0, tiles+2)
	out = append(out, bannerPart{srcX: 0, w: bannerCapW, dx: 0})
	for i := 0; i < tiles; i++ {
		out = append(out, bannerPart{srcX: bannerCapW, w: bannerTileW, dx: bannerCapW + i*bannerTileW})
	}
	return append(out, bannerPart{srcX: bannerBaseW - bannerCapW, w: bannerCapW, dx: w - bannerCapW})
}

func (h *HUD) drawBannerCloth(dst *ebiten.Image, name string, cx, y float64, w int, fade float32) bool {
	sheet, r, ok := h.Lib.Lookup(name)
	if !ok {
		return false
	}
	left, top := cx-float64(w)/2, y-float64(r.AnchorY)
	for _, p := range bannerParts(w) {
		part := render.Region{X: r.X + p.srcX, Y: r.Y, W: p.w, H: r.H}
		render.DrawAt(dst, sheet, part, left+float64(p.dx), top, 1, render.SpriteOptions{Alpha: fade})
	}
	return true
}
