package world

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const unservedRise = 30.0

func KindIndex(kind string) (int, bool) {
	i := slices.Index(FreightKinds, kind)
	return i, i >= 0
}

func YardFocus(site view.Site, kind string) (Vec, bool) {
	havens := SiteHavens(site)
	i, ok := KindIndex(kind)
	if len(havens) == 0 || !ok {
		return HarborFocus(site)
	}
	across, depth := YardCentre(i)
	return QuayPoint(PlanHaven(site, havens[0]), across, depth), true
}

func UnservedLabel(u model.Unserved) string {
	amount, name := goldText(u.Gold), strings.ToUpper(u.Name)
	switch {
	case u.Gold == 0 && u.Queued:
		return ui.Bridge(name+" BACKING UP", ui.Name(name)+" QUEUEING")
	case u.Gold == 0:
		return ui.Bridge(name+" SPOILING", ui.Name(name)+" DROPPED")
	}
	if u.Queued {
		return ui.Bridge(name+" BACKING UP: "+amount+" GOLD/S AT RISK", ui.Name(name)+" QUEUEING: $"+amount+"/S AT RISK")
	}
	return ui.Bridge(name+" SPOILING: -"+amount+" GOLD/S", ui.Name(name)+" DROPPED: -$"+amount+"/S")
}

func goldText(v float64) string {
	switch {
	case v < 1:
		return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
	case v < 10:
		return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
	}
	return view.Compact(v)
}

type UnservedMark struct {
	Kind string
	Text string
	Box  ui.Rect
	At   [2]int
}

type keptPrice struct {
	at  [2]int
	box ui.Rect
}

func (r *Renderer) keptPrice(m UnservedMark, avoid ...[]ui.Rect) (ui.Rect, bool) {
	prev, ok := r.prices[m.Kind]
	if !ok || prev.at != m.At {
		return ui.Rect{}, false
	}
	box := ui.Rect{X: prev.box.X, Y: prev.box.Y, W: m.Box.W, H: m.Box.H}
	if box.X > m.At[0] || box.X+box.W < m.At[0] || box.X < 0 || box.X+box.W > ViewWidth {
		return ui.Rect{}, false
	}
	for _, rects := range avoid {
		if overlapsAnyRect(box, rects) {
			return ui.Rect{}, false
		}
	}
	return box, true
}

func (r *Renderer) keepPrice(m UnservedMark) {
	next := make(map[string]keptPrice, len(r.prices)+1)
	for k, v := range r.prices {
		next[k] = v
	}
	next[m.Kind] = keptPrice{at: m.At, box: m.Box}
	r.prices = next
}

func (r *Renderer) UnservedMarks(cam render.Camera, sc Scene) []UnservedMark {
	h := r.HarborOf()
	if len(h.Plans) == 0 || sc.Chapter {
		return nil
	}
	zoom := cam.Scale()
	avoid := append(append([]ui.Rect(nil), sc.Panels...), r.BeaconRects(cam, sc)...)
	out := []UnservedMark{}
	for _, u := range model.UnservedKinds(sc.Freight) {
		i, ok := KindIndex(u.Kind)
		if !ok {
			continue
		}
		across, depth := YardCentre(i)
		at := QuayPoint(h.Plan(h.pileHaven(i)), across, depth)
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		text := UnservedLabel(u)
		mark := UnservedMark{Kind: u.Kind, Text: text, Box: labelBox(text, sx, sy-(h.Lift(at)+unservedRise)*zoom), At: [2]int{int(sx), int(sy)}}
		if box, kept := r.keptPrice(mark, avoid, boxesOf(out)); kept {
			mark.Box = box
			out = append(out, mark)
			r.keepPrice(mark)
			continue
		}
		box, ok := settleAbove(mark.Box, int(sx), avoid)
		if !ok {
			continue
		}
		if box, ok = StackLabel(box, int(sx), avoid, boxesOf(out)); ok {
			mark.Box = box
			out = append(out, mark)
			r.keepPrice(mark)
		}
	}
	return out
}

func (r *Renderer) drawUnserved(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	for _, m := range r.UnservedMarks(cam, sc) {
		box, ok := r.settleLabel(m.Box, midX(m.Box), sc.Panels)
		if !ok {
			continue
		}
		m.Box = box
		r.keepPrice(m)
		ui.Fill(dst, m.Box.Offset(1, 1), ui.Palette.Shadow)
		ui.Fill(dst, m.Box, ui.Tint(ui.Palette.ConsoleEdge, 235))
		ui.Stroke(dst, m.Box, ui.Palette.RedText)
		font.Draw(dst, m.Text, m.Box.X+4, m.Box.Y+3, 1, ui.Palette.RedText)
	}
}

func (h *Harbor) pileHaven(kind int) int {
	for _, y := range h.Yards {
		if y.Kind == kind && (len(y.Boxes) > 0 || y.Overflow > 0) {
			return y.Haven
		}
	}
	return 0
}

func boxesOf(marks []UnservedMark) []ui.Rect {
	out := make([]ui.Rect, 0, len(marks))
	for _, m := range marks {
		out = append(out, m.Box)
	}
	return out
}

func settleAbove(box ui.Rect, anchorX int, avoid []ui.Rect) (ui.Rect, bool) {
	if settled, ok := SettleLabel(box, anchorX, avoid); ok {
		return settled, true
	}
	for _, a := range avoid {
		if !Overlaps(box, a) {
			continue
		}
		for _, y := range []int{a.Y - labelGap - box.H, a.Y + a.H + labelGap} {
			moved := box
			moved.Y = y
			if settled, ok := SettleLabel(moved, anchorX, avoid); ok {
				return settled, true
			}
		}
	}
	return ui.Rect{}, false
}
