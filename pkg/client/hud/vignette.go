package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	vigGap       = 4
	calamityVigH = art.VignetteH + 2*vigGap
	boxVigH      = art.VignetteH + 6
	boxVigPad    = 16
	contractPicW = art.ThumbW + 8
)

type Vig struct {
	Scene string
	Seed  uint64
}

func CalamityVig(v view.CalamityView, f Frame) Vig {
	scene := gfx.CalamityScene(v.Kind)
	if kind, ok := strings.CutPrefix(v.Object, "kind/"); ok {
		if v.Kind == view.CalamitySpoiling && quayBound(f, kind) {
			scene = gfx.CalamityScene(art.SceneQuayFull)
		}
		scene += ":" + kind
	}
	return Vig{Scene: art.AgedScene(scene, gfx.AgeTier(f.State.Age)), Seed: gfx.VignetteSeed(v.ID)}
}

func quayBound(f Frame, kind string) bool {
	for _, k := range f.State.Freight.Kinds {
		if k.Kind == kind {
			return k.Losses.QuayFull > k.Losses.Spoiled
		}
	}
	return false
}

func ContractVig(r ContractRow) Vig {
	box := strings.TrimPrefix(r.Icon, "icon/kind-")
	if box == r.Icon {
		box = ""
	}
	return Vig{Scene: gfx.ContractScene(box, r.Crowned, r.Sealed), Seed: gfx.VignetteSeed(r.ID)}
}

func SiegeVig(f Frame) Vig {
	return Vig{Scene: art.SceneRushTide, Seed: gfx.VignetteSeed(f.State.Age)}
}

func AgeVig(f Frame) Vig {
	return Vig{Scene: art.AgedScene(art.SceneAgeUp, gfx.AgeTier(f.State.AgeUp.NextAge)), Seed: gfx.VignetteSeed(f.State.AgeUp.NextAge)}
}

func ReforgeVig(f Frame) Vig {
	return Vig{Scene: art.AgedScene(art.SceneReforge, gfx.AgeTier(f.State.Age)), Seed: gfx.VignetteSeed(f.State.Age)}
}

func (h *HUD) VigFrame() int {
	return gfx.VignetteFrame(h.Now, h.Reduce)
}

func (h *HUD) DrawVig(dst *ebiten.Image, v Vig, r ui.Rect, scale int) {
	scale = max(scale, 1)
	img, ok := h.Lib.Vignette(v.Scene, v.Seed, r.W/scale, h.VigFrame())
	if !ok {
		ui.Fill(dst, r, ui.Palette.Shadow)
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(r.X), float64(r.Y))
	dst.DrawImage(img, op)
	ui.Stroke(dst, r.Inset(-1), ui.Palette.BlueDark)
}

func (h *HUD) DrawVigThumb(dst *ebiten.Image, v Vig, r ui.Rect, hot bool) {
	img, ok := h.Lib.Vignette(v.Scene, v.Seed, chronVigW, 0)
	if !ok {
		ui.Fill(dst, r, ui.Palette.Shadow)
		return
	}
	crop := art.ThumbRect(v.Scene, chronVigW)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(float64(r.X), float64(r.Y))
	dst.DrawImage(img.SubImage(crop).(*ebiten.Image), op)
	edge := ui.Palette.BlueDark
	if hot {
		edge = ui.Palette.Gold
	}
	ui.Stroke(dst, r.Inset(-1), edge)
}

func CalamityVigRect(card ui.Rect, textLines int) ui.Rect {
	y := card.Y + CalamityPad + CalamityHeadH + textLines*ui.LineHeight + vigGap
	return ui.Rect{X: card.X + CalamityPad, Y: y, W: card.W - 2*CalamityPad, H: art.VignetteH}
}

func (h *HUD) drawCalamityVig(dst *ebiten.Image, f Frame, v view.CalamityView, card ui.Rect) {
	cause, why := h.CalamityText(f, v)
	h.DrawVig(dst, CalamityVig(v, f), CalamityVigRect(card, len(cause)+len(why)).Offset(0, h.incidentHeight(f, v)+h.calamityHead()-CalamityHeadH), 1)
}

func BoxVigRect(box ui.Rect, top int) ui.Rect {
	return ui.Rect{X: box.X + boxVigPad, Y: top, W: box.W - 2*boxVigPad, H: art.VignetteH}
}

func SiegeVigRect() ui.Rect {
	return BoxVigRect(SiegeBox, SiegeBox.Y+34+SiegeRows*ui.LineHeight+vigGap)
}

func ReforgeVigRect() ui.Rect {
	return BoxVigRect(ReforgeBox, ReforgeBox.Y+156)
}

func AgeVigRect() ui.Rect {
	left := AgeLeft()
	return ui.Rect{X: left.X, Y: AgeBottom().Y - 10 - art.VignetteH, W: left.W, H: art.VignetteH}
}

func (h *HUD) drawAgeVig(dst *ebiten.Image, f Frame, textBottom int) {
	r := AgeVigRect()
	if f.Revealed("loom") || textBottom+vigGap > r.Y {
		return
	}
	h.DrawVig(dst, AgeVig(f), r, 1)
}

func ContractPicRect(row ui.Rect) ui.Rect {
	return ui.Rect{X: row.X + row.W - ContractRight - contractPicW + 2, Y: row.Y + (row.H-art.ThumbH)/2, W: art.ThumbW, H: art.ThumbH}
}

func (h *HUD) drawContractPic(dst *ebiten.Image, r ContractRow, row ui.Rect) {
	h.DrawVigThumb(dst, ContractVig(r), ContractPicRect(row), r.State == StateActive)
}
