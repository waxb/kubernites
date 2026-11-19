package hud

import (
	"hash/fnv"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

var (
	roadTint      = color.RGBA{196, 160, 104, 255}
	roadStrength  = [4]float64{0, .35, .6, .85}
	unownedShade  = .5
	minimapShadow = ui.Palette.ConsoleEdge
)

type minimapCache struct {
	img     *ebiten.Image
	key     uint64
	src     *view.TileKind
	wear    *uint16
	view    MinimapView
	viewSrc *view.TileKind
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + .5) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255}
}

func owned(site view.Site, x, y int) (bool, bool) {
	is, ok := site.IslandAt(x, y)
	return is.Owned, ok
}

func MinimapColor(site view.Site, x, y int) color.RGBA {
	c := world.TileColor(site.Tile(x, y))
	if level := site.RoadAt(x, y); level > 0 && site.Tile(x, y) != view.TileRoad {
		c = mix(c, roadTint, roadStrength[min(level, 3)])
	}
	if mine, island := owned(site, x, y); island && !mine {
		c = mix(c, minimapShadow, unownedShade)
	}
	return c
}

func MinimapPixels(site view.Site) *image.RGBA {
	w, h := site.Width+site.Height, (site.Width+site.Height)/2
	img := image.NewRGBA(image.Rect(0, 0, w+1, h+1))
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			c := MinimapColor(site, x, y)
			px, py := world.MinimapPoint(site, x, y, 1)
			img.SetRGBA(px-1, py, c)
			img.SetRGBA(px, py, c)
		}
	}
	return img
}

func siteKey(site view.Site) uint64 {
	h := fnv.New64a()
	buf := make([]byte, 0, len(site.Tiles)*3+len(site.Islands)+2)
	buf = append(buf, byte(site.Width), byte(site.Height))
	for i, t := range site.Tiles {
		region := byte(0)
		if i < len(site.Region) {
			region = site.Region[i]
		}
		buf = append(buf, byte(t), region, byte(site.RoadAt(i%max(site.Width, 1), i/max(site.Width, 1))))
	}
	for _, is := range site.Islands {
		if is.Owned {
			buf = append(buf, 1)
			continue
		}
		buf = append(buf, 0)
	}
	h.Write(buf)
	return h.Sum64()
}

func (c minimapCache) fresh(site view.Site) bool {
	return c.img != nil && len(site.Tiles) > 0 && c.src == &site.Tiles[0] && c.wear == firstWear(site)
}

func firstWear(site view.Site) *uint16 {
	if len(site.Wear) == 0 {
		return nil
	}
	return &site.Wear[0]
}

func (h *HUD) minimapImage(site view.Site) *ebiten.Image {
	if h.minimap.fresh(site) {
		return h.minimap.img
	}
	key := siteKey(site)
	if h.minimap.img == nil || h.minimap.key != key {
		if h.minimap.img != nil {
			h.minimap.img.Deallocate()
		}
		h.minimap.img, h.minimap.key = ebiten.NewImageFromImage(MinimapPixels(site)), key
	}
	h.minimap.src, h.minimap.wear = &site.Tiles[0], firstWear(site)
	return h.minimap.img
}

type MinimapView struct {
	OX, OY, S float64
	Height    int
}

const minimapMargin = 3

func landBounds(site view.Site) (float64, float64, float64, float64, bool) {
	minX, minY, maxX, maxY, any := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1), false
	for i, t := range site.Tiles {
		if t == view.TileWater {
			continue
		}
		x, y := i%site.Width, i/site.Width
		px, py := float64(x-y+site.Height), float64(x+y)/2
		minX, maxX, minY, maxY, any = math.Min(minX, px), math.Max(maxX, px), math.Min(minY, py), math.Max(maxY, py), true
	}
	return minX - minimapMargin, minY - minimapMargin, maxX + minimapMargin, maxY + minimapMargin, any
}

func FitMinimap(site view.Site) MinimapView {
	inner := Minimap.Inset(5)
	span := float64(site.Width + site.Height)
	minX, minY, maxX, maxY, ok := landBounds(site)
	if !ok {
		minX, minY, maxX, maxY = 0, 0, span, span/2
	}
	minX, minY = math.Max(minX, 0), math.Max(minY, 0)
	maxX, maxY = math.Min(maxX, span), math.Min(maxY, span/2)
	w, h := math.Max(maxX-minX, 1), math.Max(maxY-minY, 1)
	s := math.Min(float64(inner.W)/w, float64(inner.H)/h)
	return MinimapView{
		OX: float64(inner.X) + (float64(inner.W)-w*s)/2 - minX*s,
		OY: float64(inner.Y) + (float64(inner.H)-h*s)/2 - minY*s,
		S:  s, Height: site.Height,
	}
}

func (v MinimapView) ToGrid(mx, my int) (float64, float64) {
	px, py := (float64(mx)-v.OX)/v.S, (float64(my)-v.OY)/v.S
	diff := px - float64(v.Height)
	sum := 2 * py
	return (sum + diff) / 2, (sum - diff) / 2
}

func (v MinimapView) FromGrid(gx, gy float64) (float32, float32) {
	return float32(v.OX + (gx-gy+float64(v.Height))*v.S), float32(v.OY + (gx+gy)*v.S/2)
}

func MinimapToGrid(site view.Site, mx, my int) (float64, float64) {
	return FitMinimap(site).ToGrid(mx, my)
}

func GridToMinimap(site view.Site, gx, gy float64) (float32, float32) {
	return FitMinimap(site).FromGrid(gx, gy)
}

func (h *HUD) minimapView(site view.Site) MinimapView {
	if len(site.Tiles) == 0 {
		return FitMinimap(site)
	}
	if h.minimap.viewSrc != &site.Tiles[0] {
		h.minimap.view, h.minimap.viewSrc = FitMinimap(site), &site.Tiles[0]
	}
	return h.minimap.view
}

func (h *HUD) MinimapJump(mx, my int, f Frame) []Intent {
	if f.Site.Width == 0 || !Minimap.Inset(5).Contains(mx, my) || !f.Shown(SurfaceMinimap) {
		return nil
	}
	gx, gy := h.minimapView(f.Site).ToGrid(mx, my)
	if gx < 0 || gy < 0 || gx >= float64(f.Site.Width) || gy >= float64(f.Site.Height) {
		return nil
	}
	return []Intent{{Kind: IntentJump, X: gx, Y: gy}}
}

func (h *HUD) drawMinimap(dst *ebiten.Image, f Frame) {
	if f.Site.Width == 0 || len(f.Site.Tiles) == 0 || !f.Shown(SurfaceMinimap) {
		return
	}
	inner := Minimap.Inset(4)
	clip := dst.SubImage(image.Rect(inner.X, inner.Y, inner.X+inner.W, inner.Y+inner.H)).(*ebiten.Image)
	v := h.minimapView(f.Site)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(v.S, v.S)
	op.GeoM.Translate(v.OX, v.OY)
	clip.DrawImage(h.minimapImage(f.Site), op)
	for _, st := range f.Site.Structures {
		x, y := v.FromGrid(float64(st.X)+float64(st.W)/2, float64(st.Y)+float64(st.H)/2)
		size, clr := float32(3), ui.Palette.GoldBright
		if st.Kind == view.StructKeep {
			size, clr = 4, ui.Palette.Cream
		}
		vector.FillRect(clip, x-size/2, y-size/2, size, size, clr, false)
	}
	h.drawHearthDots(clip, f, v)
	drawView(clip, f, v)
	h.drawBlips(clip, f, v)
}

func (h *HUD) drawHearthDots(dst *ebiten.Image, f Frame, v MinimapView) {
	for _, st := range f.Site.Structures {
		if st.Kind != view.StructKeep {
			continue
		}
		n := 0
		for _, p := range f.Pods {
			if p.Node == st.Object.Name && !p.System {
				n++
			}
		}
		if n == 0 {
			continue
		}
		x, y := v.FromGrid(float64(st.X)+float64(st.W)/2, float64(st.Y)+float64(st.H)+1)
		vector.FillRect(dst, x-1, y, float32(min(n, 12)), 2, ui.Palette.Green, false)
	}
}

func drawView(dst *ebiten.Image, f Frame, v MinimapView) {
	pts := make([][2]float32, 4)
	for i, c := range f.View {
		x, y := v.FromGrid(c[0], c[1])
		pts[i] = [2]float32{x, y}
	}
	for i := range pts {
		a, b := pts[i], pts[(i+1)%4]
		vector.StrokeLine(dst, a[0], a[1], b[0], b[1], 1, ui.Palette.Cream, false)
	}
}
