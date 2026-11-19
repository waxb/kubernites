package art

import (
	"image"
	"image/color"
	"math"
	"sync"
)

const (
	voidGround = Ground(255)
	bufMargin  = 4
	roadWobble = 0.035
)

type groundBuf struct {
	rect   image.Rectangle
	mat    []Ground
	lvl    []uint8
	own    []bool
	road   []uint8
	margin []float32
	side   []float32
	power  []float32
	depth  []float32
	wild   []bool
	peak   uint8
}

var bufPool = sync.Pool{New: func() any { return &groundBuf{} }}

func newGroundBuf(b image.Rectangle) *groundBuf {
	r := image.Rect(b.Min.X-bufMargin, b.Min.Y-bufMargin, b.Max.X+bufMargin, b.Max.Y+MaxLift+bufMargin)
	n := r.Dx() * r.Dy()
	g := bufPool.Get().(*groundBuf)
	if cap(g.mat) < n {
		*g = groundBuf{
			mat: make([]Ground, n), lvl: make([]uint8, n), own: make([]bool, n), road: make([]uint8, n),
			margin: make([]float32, n), side: make([]float32, n), power: make([]float32, n), depth: make([]float32, n),
			wild: make([]bool, n),
		}
	}
	g.rect, g.peak = r, 0
	g.mat, g.lvl, g.own, g.road = g.mat[:n], g.lvl[:n], g.own[:n], g.road[:n]
	g.margin, g.side, g.power, g.depth, g.wild = g.margin[:n], g.side[:n], g.power[:n], g.depth[:n], g.wild[:n]
	zero(g.lvl)
	zero(g.own)
	zero(g.road)
	zero(g.power)
	zero(g.wild)
	return g
}

func zero[T any](s []T) {
	var z T
	for i := range s {
		s[i] = z
	}
}

func (b *groundBuf) at(x, y int) (int, bool) {
	if x < b.rect.Min.X || y < b.rect.Min.Y || x >= b.rect.Max.X || y >= b.rect.Max.Y {
		return 0, false
	}
	return (y-b.rect.Min.Y)*b.rect.Dx() + (x - b.rect.Min.X), true
}

func (b *groundBuf) matAt(x, y int) Ground {
	if i, ok := b.at(x, y); ok {
		return b.mat[i]
	}
	return voidGround
}

func (b *groundBuf) lvlAt(x, y int) int {
	if i, ok := b.at(x, y); ok && b.mat[i] != voidGround {
		return int(b.lvl[i])
	}
	return -1
}

func (l *Land) fill(b *groundBuf, area image.Rectangle) {
	for y := b.rect.Min.Y; y < b.rect.Max.Y; y++ {
		for x := b.rect.Min.X; x < b.rect.Max.X; x++ {
			i, _ := b.at(x, y)
			gx, gy := groundPoint(x, y)
			tx, ty := int(math.Floor(gx+0.5)), int(math.Floor(gy+0.5))
			if tx < area.Min.X-1 || ty < area.Min.Y-1 || tx > area.Max.X || ty > area.Max.Y {
				b.mat[i] = voidGround
				continue
			}
			b.own[i] = tx >= area.Min.X && ty >= area.Min.Y && tx < area.Max.X && ty < area.Max.Y
			l.sample(b, i, gx, gy)
		}
	}
}

func (l *Land) sample(b *groundBuf, i int, gx, gy float64) {
	c := l.cellAt(gx, gy)
	m := c.material(gx, gy)
	b.mat[i] = m
	if m == GroundWater {
		b.depth[i] = openSea
		if l.nearCoast(gx, gy) {
			b.depth[i] = float32(l.depthField(gx, gy))
		}
		return
	}
	b.lvl[i] = c.level(gx, gy)
	b.peak = max(b.peak, b.lvl[i])
	b.wild[i] = l.wildAt(gx, gy)
	b.power[i] = float32(l.powerField(gx, gy))
	if m == GroundPlaza {
		return
	}
	if h := l.roadNear(gx, gy); h.level > 0 {
		margin := h.margin + fbm(gx*2.5, gy*2.5, 400)*roadWobble
		if margin > 0 {
			b.road[i], b.margin[i], b.side[i] = h.level, float32(margin), float32(h.side)
		}
	}
}

func (l *Land) Chunk(cx, cy int) *image.NRGBA {
	bounds := ChunkBounds(cx, cy)
	buf := newGroundBuf(bounds)
	l.fill(buf, ChunkArea(cx, cy))
	img := image.NewNRGBA(bounds)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		l.paintColumn(img, buf, x)
	}
	bufPool.Put(buf)
	return img
}

type ray struct {
	i     int
	level int
	lift  int
	top   bool
}

func (b *groundBuf) cast(x, y int) (ray, bool) {
	for h := int(b.peak) * ElevationStep; h >= 0; h-- {
		i, ok := b.at(x, y+h)
		if !ok || !b.own[i] || b.mat[i] == voidGround {
			continue
		}
		lv := int(b.lvl[i])
		if lv*ElevationStep >= h {
			return ray{i: i, level: lv, lift: h, top: lv*ElevationStep == h}, true
		}
	}
	return ray{}, false
}

func (l *Land) paintColumn(img *image.NRGBA, b *groundBuf, x int) {
	faceRow, faceLevel := -99, -1
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		r, ok := b.cast(x, y)
		if !ok {
			continue
		}
		gy := y + r.lift
		if !r.top {
			img.SetNRGBA(x, y, faceColor(b, x, gy, r))
			faceRow, faceLevel = y, r.level
			continue
		}
		col := l.topColor(b, x, gy, r)
		if since := y - faceRow; faceLevel > r.level && since >= 1 && since <= len(footShade) {
			col = darken(col, footShade[since-1])
		}
		img.SetNRGBA(x, y, col)
	}
}

var footShade = []float64{0.3, 0.2, 0.12, 0.05}

func (l *Land) topColor(b *groundBuf, x, y int, r ray) color.NRGBA {
	m := b.mat[r.i]
	if m == GroundWater {
		return waterTint(b, x, y, r.i)
	}
	gx, gy := groundPoint(x, y)
	col := groundTex(m, x, y, gx, gy)
	if lv := b.road[r.i]; lv > 0 {
		col = roadTex(lv, float64(b.margin[r.i]), float64(b.side[r.i]), col, x, y, gx, gy)
	} else {
		col = rimmed(b, m, x, y, col)
	}
	if r.level > 0 {
		col = brighten(col, hillLift*float64(r.level))
	}
	col = leyVein(col, float64(b.power[r.i]), x, y, gx, gy)
	if b.wild[r.i] {
		col = wildTint(col)
	}
	return col
}

const hillLift = 0.06

func brighten(c color.NRGBA, t float64) color.NRGBA {
	f := func(v uint8) uint8 { return uint8(min(float64(v)*(1+t), 255)) }
	return color.NRGBA{f(c.R), f(c.G), f(c.B), c.A}
}

func wildTint(c color.NRGBA) color.NRGBA {
	m := mix(c, grey(c), 0.3)
	f := func(v uint8, k, add float64) uint8 { return uint8(min(float64(v)*k+add, 255)) }
	return color.NRGBA{f(m.R, 0.92, 6), f(m.G, 0.96, 9), f(m.B, 1.04, 18), c.A}
}
