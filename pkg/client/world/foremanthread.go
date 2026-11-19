package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	handLift     = 12.0
	roofLift     = 10.0
	sparkShare   = .35
	sayLift      = 34
	SayGap       = 3
	sayTop       = 20
	sayFace      = 16
	sayPad       = 3
	sayLineStep  = ui.GlyphHeight + 1
	threadWidth  = 1.5
	threadShadow = 90
)

type Thread struct {
	From, To Vec
	Age      float64
	Tone     color.RGBA
}

func ThreadAlpha(age float64) float64 {
	if age < 0 || age >= ThreadLife {
		return 0
	}
	return 1 - age/ThreadLife
}

func (r *Renderer) Threads(sc Scene) []Thread {
	f := r.ForemenOf()
	h := r.hamletOf()
	revs := revisionsByRS(sc.Crews)
	out := []Thread{}
	for _, p := range sc.Pods {
		rs := OwnerRS(p)
		from, standing := f.Standing(rs)
		spot, housed := h.SpotList[p.Name]
		if !standing || !housed || ThreadAlpha(sc.Time-spot.Born) <= 0 {
			continue
		}
		to := Vec{X: float64(spot.At.X) + .5, Y: float64(spot.At.Y) + .5}
		out = append(out, Thread{From: from, To: to, Age: sc.Time - spot.Born, Tone: model.RevisionColor(revs[rs])})
	}
	return out
}

func (r *Renderer) drawForemanThreads(dst *ebiten.Image, cam render.Camera, sc Scene) {
	defer r.drawTallies(dst)
	zoom := cam.Scale()
	for _, t := range r.Threads(sc) {
		x0, y0 := ScreenPointOf(cam, sc.Site, t.From)
		x1, y1 := ScreenPointOf(cam, sc.Site, t.To)
		y0, y1 = y0-handLift*zoom, y1-roofLift*zoom
		a := ThreadAlpha(t.Age)
		drawDashes(dst, x0, y0+1, x1, y1+1, zoom, ui.Tint(ui.Palette.Shadow, uint8(threadShadow*a)), 0)
		drawDashes(dst, x0, y0, x1, y1, zoom, ui.Tint(t.Tone, uint8(255*a)), t.Age*threadSpeed*zoom)
		if spark := t.Age / (ThreadLife * sparkShare); spark < 1 && !sc.Reduce {
			sx, sy := x0+(x1-x0)*spark, y0+(y1-y0)*spark
			vector.FillCircle(dst, float32(sx), float32(sy), float32(2*max(zoom, 1)), ui.Palette.Cream, false)
		}
	}
}

func drawDashes(dst *ebiten.Image, x0, y0, x1, y1, zoom float64, c color.RGBA, phase float64) {
	length := math.Hypot(x1-x0, y1-y0)
	if length < 1 {
		return
	}
	dash := threadDash * max(zoom, 1)
	ux, uy := (x1-x0)/length, (y1-y0)/length
	for d := math.Mod(phase, 2*dash) - 2*dash; d < length; d += 2 * dash {
		a, b := max(d, 0), min(d+dash, length)
		if b <= a {
			continue
		}
		vector.StrokeLine(dst, float32(x0+ux*a), float32(y0+uy*a), float32(x0+ux*b), float32(y0+uy*b), float32(threadWidth*max(zoom, 1)), c, false)
	}
}

func SayLines(s ForemanSay) (string, string) {
	head, tail := SayText(s)
	khead, ktail := SayTextK8s(s)
	return ui.Bridge(head, khead), ui.Bridge(tail, ktail)
}

func SayRect(s ForemanSay, sx, sy, zoom float64, reduce bool) ui.Rect {
	var font ui.Font
	head, tail := SayLines(s)
	w := max(font.Width(head, 1), font.Width(tail, 1))
	rise := 0.0
	if !reduce {
		rise = calloutRise * min(s.Age()/CalloutLife, 1)
	}
	boxW := sayFace + 3*sayPad + w
	boxH := max(sayFace, sayLineStep+ui.GlyphHeight) + 2*sayPad
	return ui.Rect{X: int(sx) - boxW/2, Y: int(sy-sayLift*zoom-rise) - boxH, W: boxW, H: boxH}
}

func (s ForemanSay) Age() float64 { return s.Now - s.Start }

type SayPlacement struct {
	Say   ForemanSay
	Box   ui.Rect
	Alpha float64
}

func (r *Renderer) SayObstacles(cam render.Camera, sc Scene) []ui.Rect {
	out := append([]ui.Rect(nil), sc.Avoid...)
	for _, b := range r.TallyBoards() {
		out = append(out, b)
	}
	for _, b := range r.Drawn() {
		out = append(out, b.Rect)
	}
	for _, p := range placeCallouts(cam, sc) {
		out = append(out, calloutShade(p.box))
	}
	return out
}

func (r *Renderer) SayPlacements(cam render.Camera, sc Scene) []SayPlacement {
	out := []SayPlacement{}
	if sc.HoldSays {
		return out
	}
	avoid := r.SayObstacles(cam, sc)
	for _, s := range r.ForemenOf().Says() {
		s.Now = sc.Time
		alpha := CalloutAlpha(s.Age(), sc.Reduce)
		sx, sy := ScreenPointOf(cam, sc.Site, s.At)
		if alpha <= 0 || !onScreen(sx, sy, cam.Scale()) {
			continue
		}
		rect := SayRect(s, sx, sy, cam.Scale(), sc.Reduce)
		if board, found := r.ForemenOf().boardOf(s.Foreman.ID()); found {
			rect.Y = min(rect.Y, board.Y-SayGap-rect.H)
		}
		if box, ok := ClearAbove(rect, avoid); ok {
			out = append(out, SayPlacement{Say: s, Box: box, Alpha: alpha})
			avoid = append(avoid, calloutShade(box))
		}
	}
	return out
}

func ClearAbove(r ui.Rect, avoid []ui.Rect) (ui.Rect, bool) {
	r.Y = max(r.Y, sayTop)
	for _, dx := range []int{0, -r.W / 4, r.W / 4, -r.W / 2, r.W / 2} {
		c := r.Offset(dx, 0)
		if c.X >= 0 && c.X+c.W <= ViewWidth && !overlapsAnyRect(calloutShade(c), avoid) {
			return c, true
		}
	}
	moved := r
	for range avoid {
		hit := false
		for _, a := range avoid {
			if Overlaps(calloutShade(moved), a) {
				moved.Y, hit = a.Y-moved.H-SayGap, true
			}
		}
		if !hit {
			break
		}
	}
	if moved.Y >= sayTop && !overlapsAnyRect(calloutShade(moved), avoid) {
		return moved, true
	}
	shade, ok := clearOf(calloutShade(r), avoid)
	return ui.Rect{X: shade.X, Y: shade.Y, W: r.W, H: r.H}, ok
}

func overlapsAnyRect(r ui.Rect, rects []ui.Rect) bool {
	for _, a := range rects {
		if Overlaps(r, a) {
			return true
		}
	}
	return false
}

func (r *Renderer) DrawForemanCallouts(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	for _, p := range r.SayPlacements(cam, sc) {
		box, a := p.Box, uint8(255*p.Alpha)
		ui.Fill(dst, box.Offset(1, 2), ui.Tint(ui.Palette.Shadow, a/2))
		ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, uint8(220*p.Alpha)))
		ui.Stroke(dst, box, ui.Tint(model.RevisionColor(p.Say.Foreman.Revision), a))
		face := ui.Rect{X: box.X + sayPad, Y: box.Y + sayPad, W: sayFace, H: sayFace}
		if p.Alpha > .5 {
			r.Lib.Portrait(dst, foremanRole, face)
		}
		head, tail := SayLines(p.Say)
		r.note(DrawnCallout, "foreman:"+p.Say.Foreman.ID(), head+" "+tail, calloutShade(box))
		x := face.X + face.W + sayPad
		font.Draw(dst, head, x, box.Y+sayPad, 1, ui.Tint(ui.Palette.GoldBright, a))
		font.Draw(dst, tail, x, box.Y+sayPad+sayLineStep, 1, ui.Tint(ui.Palette.Cream, a))
	}
}

func (r *Renderer) TallyBoards() map[string]ui.Rect {
	out := map[string]ui.Rect{}
	for _, m := range r.ForemenOf().Marks {
		if !m.Whistle {
			out[m.id] = m.board
		}
	}
	return out
}
