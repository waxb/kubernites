package world

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	MaxCampsShown = 4
	HitSignpost   = "signpost"
	postLift      = 34
)

var postSprite = gfx.HavenName("sign", 0)

type CampPlan struct {
	Spots  map[string]Vec
	Shown  map[string]bool
	Hidden int
	Post   Vec
	First  model.PodView
}

func SignpostSpot(site view.Site) Vec {
	return signpostSpotAmong(site, campSpots(site))
}

func signpostSpotAmong(site view.Site, spots []Vec) Vec {
	if len(spots) > MaxCampsShown {
		return spots[len(spots)-1]
	}
	return pendingSpotAmong(site, spots, MaxCampsShown)
}

func SignpostLabel(hidden int) string {
	return "+" + strconv.Itoa(hidden) + " waiting for room"
}

func PlanCamps(site view.Site, pods []model.PodView, homes map[string]Vec) CampPlan {
	spots := campSpots(site)
	plan := CampPlan{Spots: map[string]Vec{}, Shown: map[string]bool{}, Post: signpostSpotAmong(site, spots)}
	index, camps := 0, 0
	for _, p := range pods {
		if p.System {
			continue
		}
		if _, housed := homes[p.Name]; housed {
			continue
		}
		waiting := p.State == model.HearthPending
		if waiting && camps >= MaxCampsShown {
			if plan.Hidden == 0 {
				plan.First = p
			}
			plan.Spots[p.Name] = plan.Post
			plan.Hidden++
			continue
		}
		if waiting {
			camps++
		}
		plan.Spots[p.Name], plan.Shown[p.Name] = pendingSpotAmong(site, spots, index), true
		index++
	}
	return plan
}

func signpostBox(cam render.Camera, sc Scene, plan CampPlan) (float64, float64, ui.Rect) {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{plan.Post.X + .5, plan.Post.Y + .5})
	var font ui.Font
	w, _ := font.Measure(SignpostLabel(plan.Hidden), 1)
	return sx, sy, ui.Rect{X: int(sx) - w/2 - 4, Y: int(sy-postLift*zoom) - 14, W: w + 8, H: 12}
}

func (r *Renderer) signpostItem(cam render.Camera, sc Scene, plan CampPlan) Drawable {
	zoom := cam.Scale()
	sx, sy, home := signpostBox(cam, sc, plan)
	rect := SpriteRect(r.Lib, postSprite, sx, sy, zoom, 14, 26)
	if box, clear := SettleLabel(home, int(sx), sc.Panels); clear {
		rect = union(rect, box)
	}
	hit := &Hit{Kind: HitSignpost, ID: plan.First.Namespace + "/" + plan.First.Name, Pod: plan.First, Rect: rect}
	return Drawable{Depth: render.DepthKey(plan.Post.X+.5, plan.Post.Y+.5), Hit: hit, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, postSprite, sx, sy, zoom, render.SpriteOptions{})
	}}
}

func (r *Renderer) signpostLabel(cam render.Camera, sc Scene) (ui.Rect, bool) {
	if r.post.Hidden == 0 {
		return ui.Rect{}, false
	}
	sx, _, box := signpostBox(cam, sc, r.post)
	return SettleLabel(box, int(sx), sc.Panels)
}

func (r *Renderer) drawSignpostLabel(dst *ebiten.Image, cam render.Camera, sc Scene) {
	box, ok := r.signpostLabel(cam, sc)
	if !ok {
		return
	}
	if box, ok = r.settleLabel(box, midX(box), nil); !ok {
		return
	}
	var font ui.Font
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 220))
	ui.Stroke(dst, box, ui.Palette.Gold)
	font.Draw(dst, SignpostLabel(r.post.Hidden), box.X+4, box.Y+3, 1, ui.Palette.Cream)
}

func union(a, b ui.Rect) ui.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}
