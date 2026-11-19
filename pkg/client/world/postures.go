package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	hipLift    = 5
	logSeat    = 2
	groundSeat = 0
	snoreAlpha = .55
	snoreScale = .7
)

var (
	flushTint   = [3]float32{1, .74, .66}
	sickTint    = [3]float32{.72, .95, .62}
	fallTint    = [3]float32{.62, .62, .68}
	logBark     = color.RGBA{92, 60, 36, 255}
	logRing     = color.RGBA{176, 132, 84, 255}
	legShade    = color.RGBA{58, 40, 30, 255}
	inkColor    = color.RGBA{20, 25, 27, 255}
	sweatBlue   = color.RGBA{150, 214, 250, 255}
	dizzyGold   = color.RGBA{250, 222, 110, 255}
	pagePaper   = color.RGBA{240, 232, 210, 255}
	markGrey    = color.RGBA{178, 178, 172, 255}
	flameOrange = color.RGBA{255, 170, 60, 255}
)

type look struct {
	tint  [3]float32
	alpha float32
	air   gfx.Air
}

func (l look) scale(c color.RGBA) color.RGBA {
	a := l.alpha
	if a <= 0 {
		a = 1
	}
	return color.RGBA{uint8(float32(c.R) * a), uint8(float32(c.G) * a), uint8(float32(c.B) * a), uint8(float32(c.A) * a)}
}

func figureNames(k NameKey) []string {
	names := gfx.WalkerTierVariantNames(k.Role, k.variant, k.Tier, k.Action, k.Dir, k.frame)
	if k.fallback == "" || k.fallback == k.Role || len(names) < 2 {
		return names
	}
	out := append([]string{}, names[:len(names)-2]...)
	return append(out, gfx.WalkerNames(k.fallback, k.Action, k.Dir, k.frame)...)
}

func LookOf(p model.PodView) (int, int) {
	if p.System {
		return 0, 1
	}
	return int(PersonSeed(p) % uint32(max(gfx.WalkerVariants(p.Sprite), 1))), max(p.Tier, 1)
}

var (
	airWarm  = [...]float32{-1, -.5, 0, .5, 1}
	airLight = [...]float32{-1, -.5, 0, .5, 1}
)

func AirOf(p model.PodView, role string) gfx.Air {
	if p.System || !gfx.IsTrade(role) {
		return gfx.Air{}
	}
	seed := PersonSeed(p)
	return gfx.Air{
		On: true, Skin: int(mix32(seed^0x51) % uint32(gfx.AirSkins())), Hair: int(mix32(seed^0x7a) % uint32(gfx.AirHairs())),
		Warm: airWarm[mix32(seed^0x33)%uint32(len(airWarm))], Light: airLight[mix32(seed^0x99)%uint32(len(airLight))], Bob: int(mix32(seed^0x4d) % 2),
	}
}

func FaceOf(role, pod string) int {
	return int(model.Hash(pod) % uint32(max(gfx.WalkerVariants(role), 1)))
}

func firstRegion(lib *gfx.Library, names []string) (*ebiten.Image, render.Region, bool) {
	for _, n := range names {
		if sheet, reg, ok := lib.Lookup(n); ok {
			return sheet, reg, true
		}
	}
	return nil, render.Region{}, false
}

func (r *Renderer) drawSeated(dst *ebiten.Image, names []string, sx, sy, zoom, seat float64, dir string, l look) {
	sheet, reg, ok := firstRegion(r.Lib, names)
	if !ok {
		fallbackFigure(dst, sx, sy+3*zoom, zoom, 7, l)
		return
	}
	ps := r.props()
	o := render.SpriteOptions{Tint: l.tint, Alpha: l.alpha}
	if seat > 0 {
		ps.draw(dst, "log", sx-zoom, sy, zoom, o)
	}
	legs := render.SpriteOptions{Tint: l.tint, Alpha: l.alpha, Flip: dir == "sw"}
	legX := sx
	if legs.Flip {
		legX = sx - 6*zoom
	}
	ps.draw(dst, "legs", legX, sy-seat*zoom, zoom, legs)
	cut := min(max(reg.AnchorY-hipLift, 0), reg.H)
	top := render.Region{X: reg.X, Y: reg.Y, W: reg.W, H: cut, AnchorX: reg.AnchorX, AnchorY: cut}
	gfx.DrawRegionAir(dst, sheet, top, gfx.SpriteGeo(top, sx, sy-seat*zoom, zoom, o), o.Tint, o.Alpha, l.air)
}

func drawLying(dst *ebiten.Image, lib *gfx.Library, names []string, sx, sy, zoom, angle float64, l look) {
	sheet, reg, ok := firstRegion(lib, names)
	if !ok {
		fallbackFigure(dst, sx, sy, zoom, 10*math.Cos(angle), l)
		return
	}
	var geo ebiten.GeoM
	geo.Translate(-float64(reg.AnchorX), -float64(reg.AnchorY))
	geo.Rotate(angle)
	geo.Scale(zoom, zoom)
	geo.Translate(math.Round(sx), math.Round(sy))
	gfx.DrawRegionAir(dst, sheet, reg, geo, l.tint, l.alpha, l.air)
}

func fallbackFigure(dst *ebiten.Image, sx, sy, zoom, height float64, l look) {
	h := max(math.Abs(height), 3)
	vector.FillRect(dst, float32(sx-2*zoom), float32(sy-h*zoom), float32(4*zoom), float32(h*zoom), l.scale(color.RGBA{60, 90, 160, 255}), false)
}

func (r *Renderer) drawSweat(dst *ebiten.Image, sx, sy, zoom, t float64, still bool) {
	ps := r.props()
	for i := 0; i < 2; i++ {
		p := math.Mod(t*1.6+float64(i)*.5, 1)
		if still {
			p = .3 + .3*float64(i)
		}
		side := float64(1 - 2*i)
		ps.draw(dst, "drop", sx+side*(4+4*p)*zoom, sy-(17-5*p+6*p*p)*zoom, zoom, render.SpriteOptions{})
	}
}

func (r *Renderer) drawDizzy(dst *ebiten.Image, sx, sy, zoom, t float64) {
	ps := r.props()
	for i := 0; i < 3; i++ {
		a := t*3 + float64(i)*2*math.Pi/3
		ps.draw(dst, "star", sx+math.Cos(a)*5*zoom, sy-17*zoom+math.Sin(a)*1.6*zoom, zoom, render.SpriteOptions{})
	}
}

func (r *Renderer) drawSnore(dst *ebiten.Image, sx, sy, zoom, t float64) {
	p := math.Mod(t*.45, 1)
	if p > .8 {
		return
	}
	r.props().draw(dst, "z", sx+(3+5*p)*zoom, sy-(18+8*p)*zoom, zoom, render.SpriteOptions{Alpha: float32(min((.8-p)*5, 1) * snoreAlpha), Scale: snoreScale})
}
