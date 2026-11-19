package world

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	HitHaven      = "haven"
	GateHavenID   = "haven-0"
	postSide      = plankSide * 1.4
	postLampIn    = .18
	shipEvery     = 2
	laneStep      = .5
	havenPrefix   = "haven/"
	HarborSignAt  = "sign"
	propFallbackW = 16
	propFallbackH = 24
)

func BerthSpots(p HavenPlan) []Vec {
	out := []Vec{}
	for k := len(p.Pier) - 2; k >= 0; k-- {
		for _, side := range []float64{-1, 1} {
			out = append(out, offset(p.Pier[k], p.Side, side*postSide))
		}
	}
	return out
}

func PostsPerHaven(plans []HavenPlan, open int) [][]Vec {
	out := make([][]Vec, len(plans))
	left := max(open, 0)
	for i, p := range plans {
		if !p.Open || left == 0 {
			continue
		}
		spots := BerthSpots(p)
		n := min(left, len(spots))
		out[i] = spots[:n]
		left -= n
	}
	return out
}

func PortPosts(plans []HavenPlan, havens []view.Haven, global int) [][]Vec {
	own := false
	for i, h := range havens {
		own = own || i < len(plans) && h.Open && h.Moored > 0
	}
	if !own {
		return PostsPerHaven(plans, global)
	}
	out := make([][]Vec, len(plans))
	for i, p := range plans {
		if i < len(havens) && p.Open {
			spots := BerthSpots(p)
			out[i] = spots[:min(max(havens[i].Moored, 0), len(spots))]
		}
	}
	return out
}

func WithPosts(site view.Site, p HavenPlan, posts []Vec) HavenPlan {
	next := p
	next.Posts = append([]Vec(nil), posts...)
	next.Berths = append([]Berth(nil), p.Berths...)
	for j := range posts {
		if b, ok := postBerth(site, p, j); ok {
			next.Berths = append(next.Berths, b)
		}
	}
	return next
}

func postBerth(site view.Site, p HavenPlan, j int) (Berth, bool) {
	inward := j/2 + 1
	k := len(p.Pier) - 1 - inward
	if inward%shipEvery != 0 || k < 0 || len(p.Pier) == 0 {
		return Berth{}, false
	}
	side := float64(2*(j%2) - 1)
	b := Berth{Moor: offset(p.Pier[k], p.Side, side*BerthGap), Plank: offset(p.Pier[k], p.Side, side*plankSide),
		Lane: BerthAt(p, side, len(p.Pier)).Lane}
	return b, WetLine(site, b.Lane, b.Moor)
}

func WetLine(site view.Site, a, b Vec) bool {
	steps := int(Dist(a, b)/laneStep) + 1
	for i := 0; i <= steps; i++ {
		if !WetAt(site, Lerp(a, b, float64(i)/float64(steps))) {
			return false
		}
	}
	return true
}

func postLamp(p HavenPlan, post Vec) Vec {
	return offset(post, p.Out, -postLampIn)
}

func (r *Renderer) berthLamps() []lampSpot {
	out := []lampSpot{}
	for _, p := range r.HarborOf().Plans {
		for _, post := range p.Posts {
			out = append(out, lampSpot{at: postLamp(p, post), lit: true})
		}
	}
	return out
}

func (r *Renderer) postItems(out []Drawable, cam render.Camera, sc Scene, p HavenPlan) []Drawable {
	zoom := cam.Scale()
	a := r.portArtOf(sc, p)
	names := r.HarborOf().pieceNames(HavenProp{Kind: "bollard"}, a.level, a.age)
	for _, post := range p.Posts {
		sx, sy := ScreenPointOf(cam, sc.Site, post)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		sy -= art.HavenDeck * zoom
		out = append(out, Drawable{Depth: render.DepthKey(post.X, post.Y), Draw: func(dst *ebiten.Image) {
			r.drawPortPiece(dst, names, HavenProp{Kind: "bollard"}, sx, sy, zoom)
		}})
	}
	return out
}

var harborBuildings = map[string]bool{HarborSignAt: true, "warehouse": true, "lighthouse": true, "crane": true, houseKind: true, StakeKind: true}

func HarborBuilding(name string) bool {
	rest, ok := strings.CutPrefix(name, havenPrefix)
	if !ok {
		rest, ok = strings.CutPrefix(name, art.PortPrefix)
	}
	kind, _, _ := strings.Cut(rest, "/")
	return ok && harborBuildings[kind]
}

func (r *Renderer) propHit(p HavenPlan, prop HavenProp, sx, sy, zoom float64) *Hit {
	return r.propHitNamed(p, prop, PortPieceNames(prop, p.Level, 0), sx, sy, zoom)
}

func (r *Renderer) propHitNamed(p HavenPlan, prop HavenProp, names []string, sx, sy, zoom float64) *Hit {
	if !harborBuildings[prop.Kind] || !p.Open && prop.Kind != StakeKind {
		return nil
	}
	hit := r.propShape(prop, names, sx, sy, zoom)
	hit.Kind, hit.ID = HitHaven, p.ID
	return &hit
}

func (r *Renderer) propShape(prop HavenProp, names []string, sx, sy, zoom float64) Hit {
	name := r.ShownName(names)
	return Hit{Rect: SpriteRect(r.Lib, name, sx, sy, zoom, propFallbackW, propFallbackH),
		Shapes: []Shape{{Name: name, X: sx, Y: sy, Zoom: zoom, Flip: prop.Flip}}}
}

func HarborSign(site view.Site, id string) (Vec, bool) {
	for _, h := range SiteHavens(site) {
		if h.ID != id {
			continue
		}
		for _, prop := range PlanHaven(site, h).Props {
			if prop.Kind == HarborSignAt {
				return prop.At, true
			}
		}
		return VecOf(h.Quay), true
	}
	return Vec{}, false
}
