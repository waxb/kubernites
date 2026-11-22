package gfx

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

type entry struct {
	sheet  *ebiten.Image
	region art.Region
	top    int
	bottom int
	mask   Mask
}

type Library struct {
	index map[string]entry
	lazy  *lazySprites
	vig   *vignettes
}

func Load(set art.Set) *Library {
	lib := &Library{index: map[string]entry{}, lazy: newLazySprites(), vig: newVignettes(art.NewSheet(set))}
	for _, atlas := range []art.Atlas{set.Terrain, set.Buildings, set.Walkers, set.Effects, set.Icons, set.Portraits, set.UI} {
		lib.add(atlas)
		lib.addGilded(atlas)
	}
	return lib
}

func (l *Library) add(a art.Atlas) {
	if a.Image == nil || len(a.Regions) == 0 {
		return
	}
	sheet := ebiten.NewImageFromImage(a.Image)
	for name, r := range a.Regions {
		l.index[name] = newEntry(sheet, a.Image, r)
	}
}

func ContentTop(img *image.NRGBA, r art.Region) int {
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if img.NRGBAAt(r.X+x, r.Y+y).A > 0 {
				return y
			}
		}
	}
	return r.H
}

func ContentBottom(img *image.NRGBA, r art.Region) int {
	for y := r.H - 1; y >= 0; y-- {
		for x := 0; x < r.W; x++ {
			if img.NRGBAAt(r.X+x, r.Y+y).A > 0 {
				return y + 1
			}
		}
	}
	return 0
}

func (l *Library) entry(name string) (entry, bool) {
	if l == nil {
		return entry{}, false
	}
	if e, ok := l.index[name]; ok {
		return e, true
	}
	return l.lazyEntry(name)
}

func (l *Library) Headroom(name string) (int, bool) {
	e, ok := l.entry(name)
	if !ok {
		return 0, false
	}
	return e.region.AnchorY - e.top, true
}

func (l *Library) Extent(name string) (int, int, bool) {
	e, ok := l.entry(name)
	if !ok {
		return 0, 0, false
	}
	return e.region.AnchorY - e.top, e.bottom - e.region.AnchorY, true
}

func (l *Library) Has(name string) bool {
	_, ok := l.entry(name)
	return ok
}

func (l *Library) Lookup(name string) (*ebiten.Image, render.Region, bool) {
	e, ok := l.entry(name)
	if !ok {
		return nil, render.Region{}, false
	}
	r := e.region
	return e.sheet, render.Region{X: r.X, Y: r.Y, W: r.W, H: r.H, AnchorX: r.AnchorX, AnchorY: r.AnchorY}, true
}

func (l *Library) Size(name string) (int, int) {
	_, r, ok := l.Lookup(name)
	if !ok {
		return 0, 0
	}
	return r.W, r.H
}

func (l *Library) Draw(dst *ebiten.Image, name string, sx, sy, zoom float64, o render.SpriteOptions) bool {
	sheet, r, ok := l.Lookup(name)
	if !ok {
		return false
	}
	render.DrawAt(dst, sheet, r, sx, sy, zoom, o)
	return true
}

func (l *Library) DrawFirst(dst *ebiten.Image, names []string, sx, sy, zoom float64, o render.SpriteOptions) (string, bool) {
	for _, n := range names {
		if l.Draw(dst, n, sx, sy, zoom, o) {
			return n, true
		}
	}
	return "", false
}

func (l *Library) Icon(dst *ebiten.Image, name string, x, y int) bool {
	sheet, r, ok := l.Lookup("icon/" + name)
	if !ok {
		return false
	}
	r.AnchorX, r.AnchorY = 0, 0
	render.DrawAt(dst, sheet, r, float64(x), float64(y), 1, render.SpriteOptions{})
	return true
}

func (l *Library) previewSprite(name string) (*ebiten.Image, render.Region, bool) {
	if sheet, r, ok := l.Lookup(name); ok {
		return sheet, r, true
	}
	if alt, ok := KeepFallback(name); ok {
		return l.Lookup(alt)
	}
	return nil, render.Region{}, false
}

func (l *Library) Portrait(dst *ebiten.Image, name string, box ui.Rect) bool {
	return l.Fitted(dst, "portrait/"+name, box)
}

func (l *Library) Fitted(dst *ebiten.Image, name string, box ui.Rect) bool {
	sheet, r, ok := l.previewSprite(name)
	if !ok {
		return false
	}
	r.AnchorX, r.AnchorY = 0, 0
	scale := PortraitScale(r.W, r.H, box.W, box.H)
	x := float64(box.X) + (float64(box.W)-float64(r.W)*scale)/2
	y := float64(box.Y) + (float64(box.H)-float64(r.H)*scale)/2
	render.DrawAt(dst, sheet, r, x, y, scale, render.SpriteOptions{})
	return true
}

func (l *Library) NineSlice(dst *ebiten.Image, name string, box ui.Rect) bool {
	if l == nil {
		return false
	}
	e, ok := l.index["ui/"+name]
	if !ok || e.region.Slice <= 0 || box.W < 2*e.region.Slice || box.H < 2*e.region.Slice {
		return false
	}
	for _, p := range NineParts(e.region, box) {
		sub := e.sheet.SubImage(p.Src).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(float64(p.Dst.Dx())/float64(max(p.Src.Dx(), 1)), float64(p.Dst.Dy())/float64(max(p.Src.Dy(), 1)))
		op.GeoM.Translate(float64(p.Dst.Min.X), float64(p.Dst.Min.Y))
		dst.DrawImage(sub, op)
	}
	return true
}

type Part struct {
	Src image.Rectangle
	Dst image.Rectangle
}

func NineParts(r art.Region, box ui.Rect) []Part {
	s := r.Slice
	sx := []int{r.X, r.X + s, r.X + r.W - s, r.X + r.W}
	sy := []int{r.Y, r.Y + s, r.Y + r.H - s, r.Y + r.H}
	dx := []int{box.X, box.X + s, box.X + box.W - s, box.X + box.W}
	dy := []int{box.Y, box.Y + s, box.Y + box.H - s, box.Y + box.H}
	out := make([]Part, 0, 9)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			src := image.Rect(sx[i], sy[j], sx[i+1], sy[j+1])
			dst := image.Rect(dx[i], dy[j], dx[i+1], dy[j+1])
			if src.Empty() || dst.Empty() {
				continue
			}
			out = append(out, Part{Src: src, Dst: dst})
		}
	}
	return out
}

func Missing(l *Library, names []string) []string {
	out := []string{}
	for _, n := range names {
		if !l.Has(n) {
			out = append(out, n)
		}
	}
	return out
}

func (l *Library) String() string {
	if l == nil {
		return "gfx.Library(nil)"
	}
	return fmt.Sprintf("gfx.Library(%d regions)", len(l.index))
}

func PortraitScale(w, h, boxW, boxH int) float64 {
	w, h = max(w, 1), max(h, 1)
	if fit := min(boxW/w, boxH/h); fit >= 1 {
		return float64(fit)
	}
	div := max((w+boxW-1)/max(boxW, 1), (h+boxH-1)/max(boxH, 1))
	return 1 / float64(div)
}
