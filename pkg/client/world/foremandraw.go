package world

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	foremanRole   = "tallyman"
	boardZoomMin  = .75
	historyShown  = 5
	historyAlpha  = 128
	threadDash    = 4.0
	threadSpeed   = 18.0
	pennantZoom   = .75
	departedAlpha = .4
	whistleW      = 7
	groundShare   = .7
	groundAlpha   = 110
	whistleH      = 10
)

var (
	postWood  = color.RGBA{74, 52, 32, 255}
	boardWood = color.RGBA{118, 84, 52, 255}
	clothTone = color.RGBA{232, 216, 176, 255}
	pausedRed = color.RGBA{196, 58, 48, 255}
)

func (r *Renderer) ForemenOf() *Foremen {
	if r.foremen == nil {
		r.foremen = NewForemen()
	}
	return r.foremen
}

func (r *Renderer) birthOf(h *Hamlet) Birth {
	return func(pod string) (float64, Vec, bool) {
		s, ok := h.SpotList[pod]
		return s.Born, Vec{X: float64(s.At.X) + .5, Y: float64(s.At.Y) + .5}, ok
	}
}

func (r *Renderer) foremanItems(cam render.Camera, sc Scene) []Drawable {
	posts := PostsOf(sc.Site, sc.Crews, sc.Pods, r.Home)
	f := r.ForemenOf()
	f.Step(posts, sc.Pods, r.birthOf(r.hamletOf()), sc.Time)
	f.Marks = nil
	out := []Drawable{}
	for _, p := range posts {
		if !p.Crew.Bare {
			out = append(out, r.bannerItem(cam, sc, p))
		}
		for i, fm := range p.Crew.Standing() {
			out = append(out, r.foremanItem(cam, sc, fm, p.Foremen[i]))
		}
	}
	for _, d := range f.Departures() {
		if it, ok := r.DepartureItem(cam, sc, d); ok {
			out = append(out, it)
		}
	}
	return append(out, r.PennantItems(cam, sc)...)
}

func px(zoom float64) float32 { return float32(max(zoom, 1)) }

func zoomed(v, zoom float64) float32 { return float32(v * zoom) }

func (r *Renderer) bannerItem(cam render.Camera, sc Scene, p Post) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, p.Banner)
	cloth := ui.Rect{X: int(sx), Y: int(sy - 26*zoom), W: int(9 * zoom), H: int(11 * zoom)}
	hit := &Hit{Kind: HitCrew, ID: p.Crew.ID(), Rect: ui.Rect{X: int(sx - 2*zoom), Y: cloth.Y, W: cloth.W + int(3*zoom), H: int(27 * zoom)}}
	selected := sc.Selected == p.Crew.ID()
	newest := newestRevision(p.Crew)
	history := p.Crew.History()
	return Drawable{Depth: render.DepthKey(p.Banner.X, p.Banner.Y), Hit: hit, stand: stand(p.Banner, sx, sy), Draw: func(dst *ebiten.Image) {
		vector.FillRect(dst, float32(sx), float32(sy-27*zoom), px(zoom), zoomed(27, zoom), postWood, false)
		vector.FillRect(dst, float32(cloth.X+1), float32(cloth.Y), float32(cloth.W), float32(cloth.H), clothTone, false)
		vector.FillRect(dst, float32(cloth.X+1), float32(cloth.Y+cloth.H)-zoomed(3, zoom), float32(cloth.W), zoomed(3, zoom), model.RevisionColor(newest), false)
		if p.Crew.Paused {
			drawPaused(dst, cloth, zoom)
		}
		drawHistory(dst, history, sx, cloth.Y+cloth.H, zoom)
		if selected {
			ui.Stroke(dst, cloth.Inset(-1), selectColor)
		}
	}}
}

func newestRevision(c model.CrewChain) int64 {
	for _, f := range c.Foremen {
		if f.Newest {
			return f.Revision
		}
	}
	return c.Revision
}

func drawPaused(dst *ebiten.Image, cloth ui.Rect, zoom float64) {
	w, h := zoomed(1.5, zoom), zoomed(5, zoom)
	x, y := float32(cloth.X)+zoomed(3, zoom), float32(cloth.Y)+zoomed(2, zoom)
	vector.FillRect(dst, x, y, w, h, pausedRed, false)
	vector.FillRect(dst, x+zoomed(3, zoom), y, w, h, pausedRed, false)
}

func drawHistory(dst *ebiten.Image, history []model.Foreman, sx float64, top int, zoom float64) {
	for i, f := range history[:min(len(history), historyShown)] {
		c := ui.Tint(model.RevisionColor(f.Revision), historyAlpha)
		y := float32(top) + float32(i)*zoomed(3, zoom) + zoomed(1, zoom)
		vector.FillRect(dst, float32(sx)+px(zoom), y, zoomed(4, zoom), zoomed(2, zoom), c, false)
	}
}

func (r *Renderer) foremanItem(cam render.Camera, sc Scene, fm model.Foreman, at Vec) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	whistling := r.ForemenOf().Whistling(fm.ID(), sc.Time)
	action := "idle"
	if whistling {
		action = "work"
	}
	names := r.namesFor(foremanRole, "", action, "sw", Frame(sc.Time, 3))
	body := SpriteRect(r.Lib, names[0], sx, sy, zoom, 12, 20)
	board, text := tallyBoard(fm, sx, body.Y)
	hit := &Hit{Kind: HitForeman, ID: fm.ID(), Rect: union(body, board)}
	selected := sc.Selected == fm.ID()
	tone := model.RevisionColor(fm.Revision)
	if zoom >= boardZoomMin {
		r.ForemenOf().Marks = append(r.ForemenOf().Marks, tallyMark{id: fm.ID(), board: board, text: text, tone: tone, ink: TallyColor(fm)})
	}
	if whistling {
		r.ForemenOf().Marks = append(r.ForemenOf().Marks, tallyMark{Whistle: true, x: sx + 4*zoom, top: body.Y + int(4*zoom), zoom: zoom})
	}
	return Drawable{person: foremanPerson, Depth: render.DepthKey(at.X, at.Y) + .04, Hit: hit, stand: stand(at, sx, sy), Draw: func(dst *ebiten.Image) {
		drawDiamond(dst, sx, sy, zoom*groundShare, ui.Tint(tone, groundAlpha))
		drawGuidon(dst, sx-5*zoom, sy, zoom, tone)
		if selected {
			r.outlineSprite(dst, names, sx, sy, zoom)
		}
		r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{})
	}}
}

type tallyMark struct {
	id        string
	board     ui.Rect
	text      string
	tone, ink color.RGBA
	Whistle   bool `json:"-"`
	x         float64
	top       int
	zoom      float64
}

func (f *Foremen) boardOf(id string) (ui.Rect, bool) {
	for _, m := range f.Marks {
		if m.id == id && !m.Whistle {
			return m.board, true
		}
	}
	return ui.Rect{}, false
}

func (r *Renderer) drawTallies(dst *ebiten.Image) {
	for _, m := range r.ForemenOf().Marks {
		switch {
		case !m.Whistle:
			drawBoard(dst, m.board, m.text, m.tone, m.ink)
		default:
			drawWhistle(dst, m.x, m.top, m.zoom)
		}
	}
}

func TallyText(fm model.Foreman) string {
	return Itoa(fm.Ready) + "/" + Itoa(fm.Desired)
}

func tallyBoard(fm model.Foreman, sx float64, head int) (ui.Rect, string) {
	text := TallyText(fm)
	var font ui.Font
	w := font.Width(text, 1) + 4
	h := ui.CapHeight + 4
	return ui.Rect{X: int(sx) - w/2, Y: head - h - 1, W: w, H: h}, text
}

func TallyColor(fm model.Foreman) color.RGBA {
	switch {
	case fm.Desired == 0 && fm.Ready == 0:
		return ui.Palette.Cream
	case fm.Ready >= fm.Desired:
		return ui.Palette.Green
	}
	return ui.Palette.GoldBright
}

func drawBoard(dst *ebiten.Image, board ui.Rect, text string, tone, ink color.RGBA) {
	ui.Fill(dst, board.Offset(1, 1), ui.Palette.Shadow)
	ui.Fill(dst, board, boardWood)
	ui.Stroke(dst, board, tone)
	var font ui.Font
	font.Draw(dst, text, board.X+2, board.Y+2, 1, ink)
}

func drawGuidon(dst *ebiten.Image, x, sy, zoom float64, tone color.RGBA) {
	vector.FillRect(dst, float32(x), float32(sy-20*zoom), px(zoom), zoomed(20, zoom), postWood, false)
	vector.FillRect(dst, float32(x)+px(zoom), float32(sy-20*zoom), zoomed(5, zoom), zoomed(4, zoom), tone, false)
}

func drawWhistle(dst *ebiten.Image, x float64, top int, _ float64) {
	bubble := ui.Rect{X: int(x), Y: top - whistleH, W: whistleW, H: whistleH}
	ui.Fill(dst, bubble.Offset(1, 1), ui.Palette.Shadow)
	ui.Fill(dst, bubble, ui.Palette.Cream)
	ui.Stroke(dst, bubble, ui.Palette.Ink)
	ui.Fill(dst, ui.Rect{X: bubble.X + whistleW/2, Y: bubble.Y + 2, W: 1, H: whistleH - 6}, pausedRed)
	ui.Fill(dst, ui.Rect{X: bubble.X + whistleW/2, Y: bubble.Y + whistleH - 3, W: 1, H: 1}, pausedRed)
	for i := range 2 {
		wx := float32(bubble.X + whistleW + 2 + 2*i)
		vector.StrokeLine(dst, wx, float32(bubble.Y+2), wx+1, float32(bubble.Y+whistleH/2), 1, ui.Palette.Cream, false)
		vector.StrokeLine(dst, wx+1, float32(bubble.Y+whistleH/2), wx, float32(bubble.Y+whistleH-2), 1, ui.Palette.Cream, false)
	}
}

func (r *Renderer) DepartureItem(cam render.Camera, sc Scene, d Departure) (Drawable, bool) {
	t := (sc.Time - d.Since) / WalkOffLife
	if t < 0 || t >= 1 {
		return Drawable{}, false
	}
	to := Vec{X: d.From.X + walkOffX, Y: d.From.Y + walkOffY}
	at := Lerp(d.From, to, t)
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	names := r.namesFor(foremanRole, "", "walk", Facing(d.From, to), Frame(sc.Time, 8))
	alpha := float32(min(1, (1-t)/departedAlpha))
	return Drawable{person: foremanPerson, Depth: render.DepthKey(at.X, at.Y) + .04, stand: stand(at, sx, sy), Draw: func(dst *ebiten.Image) {
		r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{Alpha: alpha, Flip: Facing(d.From, to) == "sw"})
	}}, true
}

func (r *Renderer) PennantItems(cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	if zoom < pennantZoom {
		return nil
	}
	revs := revisionsByRS(sc.Crews)
	h := r.hamletOf()
	out := []Drawable{}
	for _, p := range sc.Pods {
		rev, ok := revs[OwnerRS(p)]
		spot, housed := h.SpotList[p.Name]
		if !ok || !housed || RiseOf(spot.Born, sc.Time) < 1 {
			continue
		}
		anchor := Vec{X: float64(spot.At.X) + .5, Y: float64(spot.At.Y) + .5}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{anchor.X - .01, anchor.Y - .01})
		if !onScreen(sx, sy, zoom) {
			continue
		}
		tone := model.RevisionColor(rev)
		out = append(out, Drawable{Depth: render.DepthKey(anchor.X, anchor.Y) + .02, Draw: func(dst *ebiten.Image) {
			drawPennant(dst, sx+8*zoom, sy-zoom, zoom, tone)
		}})
	}
	return out
}

func revisionsByRS(chains []model.CrewChain) map[string]int64 {
	out := map[string]int64{}
	for _, c := range chains {
		if c.Bare {
			continue
		}
		for _, f := range c.Foremen {
			out[f.ID()] = f.Revision
		}
	}
	return out
}

func drawPennant(dst *ebiten.Image, x, base, zoom float64, tone color.RGBA) {
	vector.FillRect(dst, float32(x), float32(base-9*zoom), px(zoom), zoomed(9, zoom), postWood, false)
	vector.FillRect(dst, float32(x)+px(zoom), float32(base-9*zoom), zoomed(4, zoom), zoomed(3, zoom), tone, false)
}

func (r *Renderer) ForemanAt(id string) (Vec, bool) {
	return r.ForemenOf().Standing(id)
}
