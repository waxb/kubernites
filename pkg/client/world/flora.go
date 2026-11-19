package world

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	floraReach = 72
	floraSide  = 32
)

var wildTint = [3]float32{0.88, 0.93, 1.03}

type floraChunk struct {
	digest uint64
	items  []scenery
	beasts []critter
	flocks []flock
	fish   []fishSpot
}

func (c *landCache) habitat(site view.Site) *habitat {
	sig := StructureSig(site) ^ c.version*1099511628211
	if c.hb == nil || sig != c.hbSig {
		c.hb, c.hbSig = BuildHabitat(site, c.land), sig
		c.flora = map[chunkKey]floraChunk{}
	}
	return c.hb
}

func (c *landCache) wildAt(site view.Site, k chunkKey) floraChunk {
	hb := c.habitat(site)
	if f, ok := c.flora[k]; ok && f.digest == c.digests[k] {
		return f
	}
	a := art.ChunkArea(k.cx, k.cy)
	f := floraChunk{digest: c.digests[k]}
	for y := a.Min.Y; y < min(a.Max.Y, site.Height); y++ {
		for x := a.Min.X; x < min(a.Max.X, site.Width); x++ {
			f.items = append(f.items, sceneryFor(site, hb, x, y)...)
			f.beasts = append(f.beasts, crittersFor(site, hb, x, y)...)
			if fl, ok := flockFor(site, x, y); ok {
				f.flocks = append(f.flocks, fl)
			}
			if fs, ok := fishFor(site, x, y); ok {
				f.fish = append(f.fish, fs)
			}
		}
	}
	c.flora[k] = f
	return f
}

func sceneryOptions(s scenery) render.SpriteOptions {
	o := HillLight(uint8(s.lift))
	o.Flip = s.flip
	if !s.wild {
		return o
	}
	b := float32(1)
	if o.Tint[0] > 0 {
		b = o.Tint[0]
	}
	o.Tint = [3]float32{wildTint[0] * b, wildTint[1] * b, wildTint[2] * b}
	return o
}

func inView(screen image.Rectangle, sx, sy, zoom float64) bool {
	return sx > float64(screen.Min.X)-floraSide*zoom && sx < float64(screen.Max.X)+floraSide*zoom &&
		sy > float64(screen.Min.Y)-8*zoom && sy < float64(screen.Max.Y)+floraReach*zoom
}

func (r *Renderer) sceneryItems(cam render.Camera, sc Scene) []Drawable {
	lc := r.Land
	if lc == nil || lc.land == nil {
		return nil
	}
	zoom := cam.Scale()
	out := r.laborItems(cam, sc)
	felled, cleared := r.labor, r.livingOf().cleared
	for _, k := range lc.visible(cam, lc.view) {
		for _, s := range lc.wildAt(sc.Site, k).items {
			if s.tiny && zoom < 1 || !s.tiny && (felled.hiddenAt(s.tile) || cleared[s.tile]) {
				continue
			}
			sx, sy := cam.GridToScreenElevated(s.pos.X, s.pos.Y, s.lift)
			if !inView(lc.view, sx, sy, zoom) {
				continue
			}
			name, opts := s.name, sceneryOptions(s)
			out = append(out, Drawable{Depth: render.DepthKey(s.pos.X, s.pos.Y) - .3, Draw: func(dst *ebiten.Image) {
				r.Lib.Draw(dst, name, sx, sy, zoom, opts)
			}})
		}
	}
	return out
}
