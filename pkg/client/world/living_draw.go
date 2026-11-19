package world

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	boardSideX   = 1.5
	boardSideY   = -.3
	boardBack    = .7
	scaffoldBias = .02
	cairnBias    = .01
	systemSpace  = "kube-system"
)

func (r *Renderer) drawLivingGround(dst *ebiten.Image, cam render.Camera, sc Scene) {
	l := r.livingOf()
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	zoom := cam.Scale()
	h := r.hamletOf()
	r.drawWorn(dst, cam, sc, l, x0, y0, x1, y1, h)
	for _, f := range l.folk {
		plan, ok := l.plans.of(f.s)
		if !ok {
			continue
		}
		for _, t := range plan.wheat {
			if t.at.X < x0 || t.at.X > x1 || t.at.Y < y0 || t.at.Y > y1 || h.occAt(t.at) != occFree {
				continue
			}
			level := WheatLevel(f.free, t.mid)
			p := l.piece(fmt.Sprintf("wheat%d/%d", level, t.variant), func() art.Piece { return art.PaintWheat(level, t.variant) })
			sx, sy := cam.GridToScreenElevated(float64(t.at.X), float64(t.at.Y), float64(sc.Site.ElevationAt(t.at.X, t.at.Y)))
			p.draw(dst, sx, sy, zoom, render.SpriteOptions{})
		}
	}
}

func (r *Renderer) livingItems(cam render.Camera, sc Scene) []Drawable {
	l := r.livingOf()
	l.glows = l.glows[:0]
	out := r.bushItems(nil, cam, sc, l)
	out = r.cairnItems(out, cam, sc, l)
	out = r.lostRollItems(out, cam, sc, l)
	out = r.rubbleItems(out, cam, sc, l)
	out = r.boardItems(out, cam, sc, l)
	out = r.scaffoldItems(out, cam, sc, l)
	out = r.lanternItems(out, cam, sc, l)
	out = r.buntingItems(out, cam, sc, l)
	out = r.laundryItems(out, cam, sc, l)
	out = r.stockItems(out, cam, sc, l)
	return r.quayGullItems(out, cam, sc, l)
}

func (r *Renderer) pieceItem(out []Drawable, cam render.Camera, sc Scene, p livingPiece, at Vec, bias float64, alpha float32, flip bool) []Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return out
	}
	opts := HillLight(uint8(elevationAt(sc.Site, at)))
	opts.Alpha, opts.Flip = alpha, flip
	return append(out, Drawable{Depth: render.DepthKey(at.X, at.Y) + bias, Draw: func(dst *ebiten.Image) {
		p.draw(dst, sx, sy, zoom, opts)
	}})
}

func (r *Renderer) bushItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	h := r.hamletOf()
	for _, f := range l.folk {
		plan, ok := l.plans.of(f.s)
		if !ok {
			continue
		}
		fruit := BerryFruit(f.fruit)
		for _, b := range plan.bushes {
			if h.occAt(TileOf(b.at)) == occHut {
				continue
			}
			v := b.variant
			p := l.piece(fmt.Sprintf("bush%d/%d", fruit, v), func() art.Piece { return art.PaintBush(fruit, v) })
			out = r.pieceItem(out, cam, sc, p, b.at, 0, 1, v == 1)
		}
	}
	return out
}

func (r *Renderer) cairnItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	for _, c := range l.cairns {
		stage, v := CairnStage(sc.Time-c.since), int(c.seed%2)
		p := l.piece(fmt.Sprintf("cairn%d/%d", stage, v), func() art.Piece { return art.PaintCairn(stage, v) })
		out = r.pieceItem(out, cam, sc, p, c.at, cairnBias, 1, false)
	}
	return out
}

func (r *Renderer) rubbleItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	for _, m := range l.rubble {
		stage, v := RubbleStage(sc.Time-m.since), int(model.Hash(m.s.ID)%2)
		p := l.piece(fmt.Sprintf("rubble%d/%d", stage, v), func() art.Piece { return art.PaintRubble(stage, v) })
		out = r.pieceItem(out, cam, sc, p, FootprintCentre(m.s), 0, 1, false)
	}
	return out
}

func (r *Renderer) boardItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	if _, ok := FirstOfKind(sc.Site, view.StructChancery, true); !ok {
		return out
	}
	n := BoardParchments(declaredCrews(sc), declaredDepots(sc.Site))
	shown := min(n, art.BoardSlots)
	at, ok := boardSpot(sc.Site, r.hamletOf())
	if !ok {
		return out
	}
	p := l.piece(fmt.Sprintf("board%d", shown), func() art.Piece { return art.PaintBoard(shown, 0) })
	return r.standingPiece(out, cam, sc, p, at)
}

func boardSpot(site view.Site, h *Hamlet) (Vec, bool) {
	ch, ok := FirstOfKind(site, view.StructChancery, true)
	if !ok {
		return Vec{}, false
	}
	lo, hi := Footprint(ch)
	x0, y0, x1, y1 := float64(lo.X), float64(lo.Y), float64(hi.X), float64(hi.Y)
	return firstClear(site, h, []Vec{
		{x1 + boardSideX, y0 + boardSideY}, {x1 + boardSideX, y1 - boardSideY}, {x1 + boardBack, y1 + boardSideX},
		{x0 + boardSideY, y1 + boardSideX}, {x0 - boardSideX, y0 - boardSideY}, {x0 - boardSideY, y0 - boardSideX},
	})
}

func (r *Renderer) standingPiece(out []Drawable, cam render.Camera, sc Scene, p livingPiece, at Vec) []Drawable {
	n := len(out)
	out = r.pieceItem(out, cam, sc, p, at, 0, 1, false)
	if len(out) > n {
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		out[n] = propDrawable(out[n], at, sx, sy)
	}
	return out
}

func declaredCrews(sc Scene) int {
	n := 0
	for _, c := range sc.Crews {
		if c.Namespace != systemSpace {
			n++
		}
	}
	return n
}

func declaredDepots(site view.Site) int {
	n := 0
	for _, s := range site.Structures {
		if s.Object.Kind == ServiceKind && s.Object.Namespace != systemSpace && s.State == view.StateReady {
			n++
		}
	}
	return n
}

func (r *Renderer) scaffoldItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	if len(l.scaffold) == 0 {
		return out
	}
	p := l.piece("scaffold", art.PaintScaffold)
	for _, pod := range sc.Pods {
		home, ok := r.Homes[pod.Name]
		if !ok || !l.scaffold[pod.Name] {
			continue
		}
		at := Vec{home.X + .5, home.Y + .5}
		out = r.pieceItem(out, cam, sc, p, at, scaffoldBias, .92, false)
	}
	return out
}
