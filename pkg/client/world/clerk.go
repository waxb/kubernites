package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	clerkAside = .6
	clerkBack  = .4
	clerkRole  = "clerk"
	maxFlecks  = 3
	fleckRise  = 12.0
	bannerPole = 27
)

var (
	deskWood = color.RGBA{120, 80, 46, 255}
	deskTop  = color.RGBA{164, 116, 70, 255}
	poleWood = color.RGBA{96, 64, 38, 255}
)

var bannerColors = []color.RGBA{{196, 52, 44, 255}, {52, 98, 188, 255}, {226, 176, 48, 255}, {70, 150, 78, 255}, {148, 70, 170, 255}}

func BannerHue(pod string) color.RGBA {
	return bannerColors[model.Hash(pod)%uint32(len(bannerColors))]
}

func ClerkSpot(codex view.Structure) Vec {
	p := Porch(codex)
	return Vec{p.X + clerkAside, p.Y - clerkBack}
}

func Scribbles(sc Scene) []float64 {
	out := []float64{}
	for _, e := range sc.Errands {
		age := sc.Time - e.Start
		if e.Kind == ErrandScribble && e.Life > 0 && age >= 0 && age <= e.Life {
			out = append(out, age/e.Life)
		}
	}
	return out
}

func (r *Renderer) ClerkItems(cam render.Camera, sc Scene) []Drawable {
	codex, ok := FirstOfKind(sc.Site, view.StructCodex, true)
	if !ok {
		return nil
	}
	spot := ClerkSpot(codex)
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, spot)
	busy := Scribbles(sc)
	pose := Pose{Pos: spot, Action: "idle", Dir: "sw", Frame: Frame(sc.Time, 2)}
	if len(busy) > 0 {
		pose.Action, pose.Frame = "work", Frame(sc.Time, 8)
	}
	names := r.namesFor(clerkRole, errandFallback[clerkRole], pose.Action, pose.Dir, pose.Frame)
	depth := render.DepthKey(spot.X, spot.Y) + .05
	return []Drawable{{person: "clerk", Depth: depth, Draw: func(dst *ebiten.Image) {
		r.props().draw(dst, "desk", sx-7*zoom, sy, zoom, render.SpriteOptions{})
		if _, ok := r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{}); !ok {
			fallbackFigure(dst, sx, sy, zoom, 10, look{})
		}
		if !sc.Reduce {
			r.DrawFlecks(dst, sx-7*zoom, sy-9*zoom, zoom, busy)
		}
	}}}
}

func (r *Renderer) DrawFlecks(dst *ebiten.Image, x, y, zoom float64, busy []float64) {
	ps := r.props()
	for i, p := range busy {
		if i >= maxFlecks {
			return
		}
		ps.draw(dst, "page", x+math.Sin(p*7+float64(i)*2)*3*zoom, y-fleckRise*p*zoom, zoom, render.SpriteOptions{Alpha: float32(min((1-p)*3, 1))})
	}
}

func (r *Renderer) drawBanner(dst *ebiten.Image, x, y, zoom float64, hue color.RGBA, t float64) {
	ps := r.props()
	ps.draw(dst, "pole", x, y, zoom, render.SpriteOptions{})
	flag := "flag"
	if Frame(t, 5)%2 == 1 {
		flag = "flag2"
	}
	tint := [3]float32{float32(hue.R) / 255, float32(hue.G) / 255, float32(hue.B) / 255}
	ps.draw(dst, flag, x+zoom, y-(bannerPole-2)*zoom, zoom, render.SpriteOptions{Tint: tint})
}
