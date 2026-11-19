package world

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

var zoneWash = color.RGBA{255, 218, 105, 46}

type portArt struct {
	level, age     int
	busy, selected bool
}

func (r *Renderer) portArtOf(sc Scene, p HavenPlan) portArt {
	clock := r.agesOf()
	clock.track(sc.Age, sc.Time)
	age, busy := clock.phase(p.ID)
	return portArt{level: p.Level, age: age, busy: busy, selected: p.ID == sc.Selected}
}

func (r *Renderer) portArts(sc Scene) []portArt {
	plans := r.HarborOf().Plans
	out := make([]portArt, len(plans))
	for i, p := range plans {
		out[i] = r.portArtOf(sc, p)
	}
	return out
}

func deckVariant(d Point) int { return int(tileHash(d.X, d.Y) % uint32(pieceKinds)) }

func (h *Harbor) deckNames(d Point, a portArt) []string {
	return h.pieceNames(HavenProp{Kind: "quay", Variant: deckVariant(d)}, a.level, a.age)
}

func (r *Renderer) drawDecks(dst *ebiten.Image, cam render.Camera, sc Scene) {
	h := r.HarborOf()
	arts := r.portArts(sc)
	zoom := cam.Scale()
	for i, p := range h.Plans {
		if arts[i].selected {
			r.outlineZone(dst, cam, sc, p, arts[i])
		}
	}
	r.deckHits = r.deckHits[:0]
	for _, d := range h.deckOrder {
		a := arts[h.deckPlan[d]]
		if sx, sy := ScreenPointOf(cam, sc.Site, d.Center()); onScreen(sx, sy, zoom) {
			names := h.deckNames(d, a)
			r.drawPortPiece(dst, names, HavenProp{Kind: "quay", Variant: deckVariant(d)}, sx, sy, zoom)
			if DeckPickable(sc) {
				r.deckHits = append(r.deckHits, r.deckHit(h.Plans[h.deckPlan[d]], names, sx, sy, zoom))
			}
		}
	}
	for i, p := range h.Plans {
		if arts[i].selected && !p.Open {
			washZone(dst, cam, sc, p)
		}
	}
}

const HandGrace = 2.0

func DeckPickable(sc Scene) bool {
	return sc.Freight.Waiting < 1 && (sc.ServedAt <= 0 || sc.Time-sc.ServedAt >= HandGrace)
}

func (r *Renderer) deckHit(p HavenPlan, names []string, sx, sy, zoom float64) Hit {
	hit := r.propShape(HavenProp{Kind: "quay"}, names, sx, sy, zoom)
	hit.Kind, hit.ID = HitHaven, p.ID
	return hit
}

func (r *Renderer) drawPortPiece(dst *ebiten.Image, names []string, fallback HavenProp, sx, sy, zoom float64) {
	r.drawPortPieceAt(dst, names, fallback, sx, sy, zoom, 1)
}

func (r *Renderer) drawPortPieceAt(dst *ebiten.Image, names []string, fallback HavenProp, sx, sy, zoom float64, alpha float32) {
	if _, ok := r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{Flip: fallback.Flip, Alpha: alpha}); !ok && r.Lib == nil {
		drawPieceFallback(dst, fallback.Kind, fallback.Variant, sx, sy, zoom)
	}
}

func (r *Renderer) outlineNames(dst *ebiten.Image, names []string, sx, sy, zoom float64, flip bool) {
	for _, n := range names {
		if sheet, reg, ok := r.Lib.Lookup(n); ok {
			render.OutlineFlipAt(dst, sheet, reg, sx, sy, zoom, selectLine, flip)
			return
		}
	}
}

func (r *Renderer) outlineZone(dst *ebiten.Image, cam render.Camera, sc Scene, p HavenPlan, a portArt) {
	zoom, h := cam.Scale(), r.HarborOf()
	for _, d := range p.Deck {
		sx, sy := ScreenPointOf(cam, sc.Site, d.Center())
		r.outlineNames(dst, h.deckNames(d, a), sx, sy, zoom, false)
	}
	kind := pierKind(p.Out)
	for i, v := range p.Pier {
		sx, sy := ScreenPointOf(cam, sc.Site, v)
		r.outlineNames(dst, h.pieceNames(HavenProp{Kind: kind, Variant: i % pieceKinds}, a.level, a.age), sx, sy, zoom, false)
	}
}

func washZone(dst *ebiten.Image, cam render.Camera, sc Scene, p HavenPlan) {
	zoom, lift := cam.Scale(), 0.0
	if p.Open {
		lift = art.HavenDeck
	}
	for _, d := range ZoneTiles(p) {
		sx, sy := ScreenPointOf(cam, sc.Site, d.Center())
		drawDiamond(dst, sx, sy-lift*zoom, zoom, zoneWash)
	}
}

func ZoneTiles(p HavenPlan) []Point {
	if !p.Open {
		out := []Point{}
		for j := 0; j < DeckDepth; j++ {
			for i := -DeckHalf; i <= DeckHalf; i++ {
				out = append(out, TileOf(offset(offset(p.Quay, p.Side, float64(i)), p.Out, -float64(j))))
			}
		}
		return out
	}
	out := append([]Point(nil), p.Deck...)
	for _, v := range p.Pier {
		out = append(out, TileOf(v))
	}
	return out
}
