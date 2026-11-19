package world

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	causeLift = 26
	causePadX = 3
	causePadY = 2
	causeTail = 3
)

var strainTags = map[string]bool{view.TagThrottled: true, view.TagUnhoused: true}

type CauseMark struct {
	Pod    string
	Tag    string
	Label  string
	Text   string
	Rect   ui.Rect
	Strain bool
	Tail   int
}

func CauseLabel(tag string) string {
	return model.CauseTitle(tag)
}

func CauseRect(label string, sx, sy, zoom float64) ui.Rect {
	var font ui.Font
	w, _ := font.Measure(label, 1)
	boxW, boxH := w+2*causePadX, ui.CapHeight+2*causePadY
	return ui.Rect{X: int(sx) - boxW/2, Y: int(sy-causeLift*zoom) - boxH - causeTail, W: boxW, H: boxH}
}

func causeShape(box ui.Rect) ui.Rect {
	return ui.Rect{X: box.X, Y: box.Y, W: box.W + 1, H: box.H + causeTail + 1}
}

func podName(key string) string {
	if _, name, ok := strings.Cut(key, "/"); ok {
		return name
	}
	return key
}

func (r *Renderer) CauseMarks(cam render.Camera, sc Scene) []CauseMark {
	out := []CauseMark{}
	zoom := cam.Scale()
	seen := map[string]bool{}
	avoid := r.labelObstacles(cam, sc)
	for _, t := range sc.Causes {
		home, ok := r.Homes[podName(t.Pod)]
		key := t.Tag + "/" + t.Node
		if !ok || strainTags[t.Tag] && seen[key] {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{home.X + .5, home.Y + .5})
		if !onScreen(sx, sy, zoom) {
			continue
		}
		label := CauseLabel(t.Tag)
		shape, clear := StackLabel(causeShape(CauseRect(label, sx, sy, zoom)), int(sx), avoid, r.placed)
		box := ui.Rect{X: shape.X, Y: shape.Y, W: shape.W - 1, H: shape.H - causeTail - 1}
		m := CauseMark{Pod: t.Pod, Tag: t.Tag, Label: label, Text: t.Text, Rect: box, Strain: strainTags[t.Tag], Tail: int(sx)}
		if clear && !stacked(out, m.Rect) {
			seen[key] = true
			out = append(out, m)
		}
	}
	return out
}

func stacked(marks []CauseMark, r ui.Rect) bool {
	for _, m := range marks {
		if Overlaps(m.Rect, r) {
			return true
		}
	}
	return false
}

func (r *Renderer) DrawnCauseMarks() []CauseMark { return r.marks }

func (r *Renderer) DrawCauseTags(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	r.marks = r.CauseMarks(cam, sc)
	for _, m := range r.marks {
		b := m.Rect
		r.note(DrawnTag, "tag:"+m.Label, m.Label, causeShape(b))
		fill, edge := ui.Palette.RedDark, ui.Palette.RedText
		if m.Strain {
			fill, edge = ui.Palette.Console, ui.Palette.GoldBright
		}
		ui.Fill(dst, b.Offset(1, 1), ui.Palette.Shadow)
		ui.Fill(dst, b, fill)
		ui.Stroke(dst, b, edge)
		mid := min(max(m.Tail, b.X+causeTail), b.X+b.W-causeTail)
		for i := 0; i < causeTail; i++ {
			ui.Fill(dst, ui.Rect{X: mid - (causeTail - 1 - i), Y: b.Y + b.H + i, W: 2*(causeTail-1-i) + 1, H: 1}, edge)
		}
		font.Draw(dst, m.Label, b.X+causePadX, b.Y+causePadY, 1, ui.Palette.Cream)
	}
}
