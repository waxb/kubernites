package world

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	gateAhead  = 2
	gateJitter = .7
)

type GateMark struct {
	Node  string
	At    Vec
	Count int
	First model.PodView
}

func GateTile(keep view.Structure) Point {
	_, hi := Footprint(keep)
	return Point{hi.X + gateAhead, hi.Y + gateAhead}
}

func GateLabel(n int) string {
	return "+" + strconv.Itoa(n)
}

func gateHome(at Vec, name string) Vec {
	h := mix32(model.Hash(name))
	u := float64(h&0xffff)/0xffff - .5
	v := float64(h>>16)/0xffff - .5
	return Vec{at.X + u*gateJitter, at.Y + v*gateJitter}
}

func (h *Hamlet) Gates(site view.Site, pods []model.PodView) []GateMark {
	if len(h.Homeless) == 0 {
		return nil
	}
	var out []GateMark
	for _, p := range pods {
		if node, ok := h.Homeless[p.Name]; !ok || node != p.Node {
			continue
		}
		out = countAtGate(out, site, p)
	}
	return out
}

func countAtGate(out []GateMark, site view.Site, p model.PodView) []GateMark {
	for i := range out {
		if out[i].Node == p.Node {
			out[i].Count++
			return out
		}
	}
	keep, ok := KeepForNode(site, p.Node)
	if !ok {
		return out
	}
	return append(out, GateMark{Node: p.Node, At: GateTile(keep).Center(), Count: 1, First: p})
}

func withGates(housed map[string]Vec, pods []model.PodView, h *Hamlet, gates []GateMark) map[string]Vec {
	if len(gates) == 0 {
		return housed
	}
	out := make(map[string]Vec, len(housed)+len(h.Homeless))
	for name, v := range housed {
		out[name] = v
	}
	for _, p := range pods {
		if node, ok := h.Homeless[p.Name]; ok && node == p.Node {
			for _, g := range gates {
				if g.Node == p.Node {
					out[p.Name] = gateHome(g.At, p.Name)
				}
			}
		}
	}
	return out
}

func gateBox(cam render.Camera, sc Scene, g GateMark) (float64, float64, ui.Rect) {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{g.At.X + .5, g.At.Y + .5})
	var font ui.Font
	w, _ := font.Measure(GateLabel(g.Count), 1)
	return sx, sy, ui.Rect{X: int(sx) - w/2 - 4, Y: int(sy-postLift*zoom) - 14, W: w + 8, H: 12}
}

func (r *Renderer) gateItems(out []Drawable, cam render.Camera, sc Scene, gates []GateMark) []Drawable {
	zoom := cam.Scale()
	for _, g := range gates {
		sx, sy, box := gateBox(cam, sc, g)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		rect := SpriteRect(r.Lib, postSprite, sx, sy, zoom, 14, 26)
		hit := &Hit{Kind: HitSignpost, ID: PodKey(g.First), Pod: g.First, Rect: union(rect, box)}
		out = append(out, Drawable{Depth: render.DepthKey(g.At.X+.5, g.At.Y+.5), Hit: hit, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, postSprite, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func (r *Renderer) drawGateLabels(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	for _, g := range r.gates {
		sx, sy, home := gateBox(cam, sc, g)
		box, clear := SettleLabel(home, int(sx), sc.Panels)
		if !onScreen(sx, sy, cam.Scale()) || !clear {
			continue
		}
		if box, clear = r.settleLabel(box, midX(box), sc.Panels); !clear {
			continue
		}
		ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 220))
		ui.Stroke(dst, box, ui.Palette.Gold)
		font.Draw(dst, GateLabel(g.Count), box.X+4, box.Y+3, 1, ui.Palette.Cream)
	}
}
