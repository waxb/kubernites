package world

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	BubbleHold     = 3.0
	BubbleFade     = 0.6
	BubbleRegion   = "speech"
	bubbleWidth    = 22 * ui.Advance
	bubbleLines    = 4
	bubblePad      = 4
	bubbleTail     = 4
	bubbleTailBase = 6
	bubbleReach    = 2 * bubbleTail
	bubbleGap      = 2
	bubbleMargin   = 2
	bubbleDefault  = 30.0
)

var (
	bubbleFont   = ui.Font{Literal: true}
	BubbleFill   = color.RGBA{250, 250, 246, 255}
	BubbleInk    = ui.Palette.Ink
	bubbleCorner = []int{2, 1}
)

type Speech struct {
	Person string
	Text   string
	Start  float64
	CPS    float64
	Whole  bool
}

func (sp Speech) cps() float64 {
	if sp.CPS <= 0 {
		return bubbleDefault
	}
	return sp.CPS
}

func (sp Speech) Done() float64 {
	return float64(len([]rune(sp.Text))) / sp.cps()
}

func Typed(sp Speech, now float64, reduce bool) int {
	n := len([]rune(sp.Text))
	if reduce || sp.Whole {
		return n
	}
	return min(max(int((now-sp.Start)*sp.cps()), 0), n)
}

func BubbleAlpha(sp Speech, now float64, reduce bool) float64 {
	age := now - sp.Start
	if age < 0 {
		return 0
	}
	end := sp.Done() + BubbleHold
	if sp.Whole && age < end+BubbleFade {
		return 1
	}
	if reduce {
		end = BubbleHold + sp.Done()
		if age >= end {
			return 0
		}
		return 1
	}
	switch {
	case age < end:
		return 1
	case age < end+BubbleFade:
		return 1 - (age-end)/BubbleFade
	}
	return 0
}

func BubbleText(sp Speech, typed int) []string {
	font := bubbleFont
	text := []rune(sp.Text)
	out := []string{}
	pos := 0
	for _, line := range font.WrapLines(sp.Text, bubbleWidth, 1, bubbleLines) {
		for pos < len(text) && text[pos] == ' ' {
			pos++
		}
		if pos >= typed {
			break
		}
		runes := []rune(line)
		out = append(out, string(runes[:min(typed-pos, len(runes))]))
		pos += len(runes)
	}
	return out
}

func BubbleRect(lines []string, sx, headY float64) ui.Rect {
	font := bubbleFont
	w := 0
	for _, l := range lines {
		lw, _ := font.Measure(l, 1)
		w = max(w, lw)
	}
	boxW := max(w, ui.Advance) + 2*bubblePad
	boxH := max(len(lines), 1)*ui.LineHeight - (ui.LineHeight - ui.CapHeight) + 2*bubblePad
	x := min(max(int(sx)-boxW/2, bubbleMargin), ViewWidth-boxW-bubbleMargin)
	return ui.Rect{X: x, Y: int(headY) - bubbleGap - bubbleTail - boxH, W: boxW, H: boxH}
}

func bubbleSpot(box ui.Rect, sx, headY, footY float64, avoid []ui.Rect) (ui.Rect, bool, bool) {
	clear := func(r ui.Rect, above bool) bool {
		shape := bubbleBounds(r, sx, above)
		if shape.Y < 0 || shape.Y+shape.H > ViewHeight {
			return false
		}
		for _, a := range avoid {
			if Overlaps(shape, a) {
				return false
			}
		}
		return true
	}
	below := box
	below.Y = int(footY) + bubbleGap + bubbleTail
	for _, shift := range []int{0, -box.W / 2, box.W / 2, -box.W, box.W} {
		for _, spot := range []struct {
			r     ui.Rect
			above bool
		}{{box, true}, {below, false}} {
			r := spot.r
			r.X = min(max(r.X+shift, bubbleMargin), ViewWidth-r.W-bubbleMargin)
			if clear(r, spot.above) {
				return r, spot.above, true
			}
		}
	}
	return box, true, false
}

type FocusMark struct {
	Person   string
	Pos      Vec
	Activity string
	Load     string
	Head     float64
	Seen     bool
}

var loadWords = map[string]string{"sealed": "sealed box"}

func loadWord(kind string) string {
	if w, ok := loadWords[kind]; ok {
		return w
	}
	return kind
}

func (r *Renderer) Focus() (FocusMark, bool) {
	return r.focus, r.focus.Person != ""
}

func (r *Renderer) markFocus(sc Scene, figs []Figure) {
	if sc.Focus == "" {
		r.focus = FocusMark{}
		return
	}
	if r.focus.Person != sc.Focus {
		r.focus = FocusMark{Person: sc.Focus}
	}
	r.focus.Seen = false
	if owner, pet, ok := strings.Cut(sc.Focus, "#"); ok {
		r.markPet(sc, figs, owner, pet)
		return
	}
	for _, f := range figs {
		if PersonKey(f.Pod) != sc.Focus {
			continue
		}
		r.focus.Pos, r.focus.Seen, r.focus.Load = f.Pose.Pos, true, ""
		r.focus.Activity = activityOf(f)
		if f.Load != nil {
			r.focus.Load = loadWord(f.Load.Kind)
		}
		return
	}
	if home, ok := r.Homes[podOfPerson(sc.Focus)]; ok && r.focus.Pos == (Vec{}) {
		r.focus.Pos = Vec{home.X + .35, home.Y + .35}
	}
}

func (r *Renderer) markPet(sc Scene, figs []Figure, owner, name string) {
	for _, f := range figs {
		if PersonKey(f.Pod) != owner {
			continue
		}
		for _, pet := range r.PetPoses(sc, f) {
			if pet.name == name {
				r.focus.Pos, r.focus.Seen, r.focus.Activity, r.focus.Load = pet.Pose.Pos, true, "familiar", ""
				return
			}
		}
	}
}

func podOfPerson(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < 2 {
		return key
	}
	return parts[1]
}

func activityOf(f Figure) string {
	switch {
	case f.Load != nil:
		return "carrying"
	case f.Mood == MoodRest:
		return "resting"
	case f.Mood == MoodSick:
		return "stopped"
	case f.Pose.Action == "walk":
		return "returning"
	case f.Pose.Action == "work":
		return "serving"
	}
	return "waiting"
}

func (r *Renderer) DrawSpeech(dst *ebiten.Image, cam render.Camera, sc Scene) {
	sp := sc.Speech
	if sp == nil || r.focus.Person == "" || r.focus.Person != sp.Person {
		return
	}
	alpha := BubbleAlpha(*sp, sc.Real, sc.Reduce)
	lines := BubbleText(*sp, Typed(*sp, sc.Real, sc.Reduce))
	if alpha <= 0 || len(lines) == 0 {
		return
	}
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, r.focus.Pos)
	sy -= r.HarborOf().Lift(r.focus.Pos) * zoom
	head := sy - bubbleHead*zoom
	box, above, ok := bubbleSpot(BubbleRect(lines, sx, head), sx, head, sy, sc.Avoid)
	if !ok {
		return
	}
	r.note(DrawnSpeech, "speech:"+sp.Person, sp.Text, bubbleBounds(box, sx, above))
	ui.InRegion(BubbleRegion, func() { drawBubble(dst, box, lines, sx, above, alpha) })
}

const bubbleHead = 24.0

type bubbleShape struct {
	fill, ink []ui.Rect
}

func cornerInset(row, rows int) int {
	switch {
	case row < len(bubbleCorner):
		return bubbleCorner[row]
	case rows-1-row < len(bubbleCorner):
		return bubbleCorner[rows-1-row]
	}
	return 0
}

func shapeBubble(box ui.Rect, sx float64, above bool) bubbleShape {
	var sh bubbleShape
	for row := range box.H {
		in := cornerInset(row, box.H)
		sh.fill = append(sh.fill, ui.Rect{X: box.X + in, Y: box.Y + row, W: box.W - 2*in, H: 1})
		sh.ink = append(sh.ink, ui.Rect{X: box.X + in - 1, Y: box.Y + row, W: 1, H: 1}, ui.Rect{X: box.X + box.W - in, Y: box.Y + row, W: 1, H: 1})
	}
	edge := bubbleCorner[0]
	sh.ink = append(sh.ink, ui.Rect{X: box.X + edge, Y: box.Y - 1, W: box.W - 2*edge, H: 1}, ui.Rect{X: box.X + edge, Y: box.Y + box.H, W: box.W - 2*edge, H: 1})
	base := min(max(int(sx), box.X+bubblePad+bubbleTailBase/2), box.X+box.W-bubblePad-bubbleTailBase/2-1)
	tip := min(max(int(sx), box.X-bubbleReach), box.X+box.W-1+bubbleReach)
	step, y := 1, box.Y+box.H
	if !above {
		step, y = -1, box.Y-1
	}
	for i := range bubbleTail {
		c := base + (tip-base)*(i+1)/bubbleTail
		w := max(1, bubbleTailBase-(bubbleTailBase-1)*(i+1)/bubbleTail)
		row := y + step*i
		sh.fill = append(sh.fill, ui.Rect{X: c - w/2, Y: row, W: w, H: 1})
		sh.ink = append(sh.ink, ui.Rect{X: c - w/2 - 1, Y: row, W: 1, H: 1}, ui.Rect{X: c - w/2 + w, Y: row, W: 1, H: 1})
	}
	sh.ink = append(sh.ink, ui.Rect{X: tip, Y: y + step*bubbleTail, W: 1, H: 1})
	return sh
}

func bubbleBounds(box ui.Rect, sx float64, above bool) ui.Rect {
	sh := shapeBubble(box, sx, above)
	x0, y0, x1, y1 := box.X, box.Y, box.X+box.W, box.Y+box.H
	for _, rs := range [][]ui.Rect{sh.fill, sh.ink} {
		for _, r := range rs {
			x0, y0 = min(x0, r.X), min(y0, r.Y)
			x1, y1 = max(x1, r.X+r.W), max(y1, r.Y+r.H)
		}
	}
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func drawBubble(dst *ebiten.Image, box ui.Rect, lines []string, sx float64, above bool, alpha float64) {
	a := uint8(255 * alpha)
	fill, ink := ui.Tint(BubbleFill, a), ui.Tint(BubbleInk, a)
	sh := shapeBubble(box, sx, above)
	for _, r := range sh.ink {
		ui.Fill(dst, r, ink)
	}
	for _, r := range sh.fill {
		ui.Fill(dst, r, fill)
	}
	font := bubbleFont
	for i, l := range lines {
		font.Draw(dst, l, box.X+bubblePad, box.Y+bubblePad+i*ui.LineHeight, 1, ink)
	}
}
