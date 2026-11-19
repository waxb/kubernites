package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	HavenDeck      = 4
	havenTileW     = 36
	havenTileH     = 31
	havenTileOX    = 18
	havenTileOY    = 12
	havenBeamDepth = 1.5
)

var HavenKinds = []string{"pier-x", "pier-y", "quay", "bollard", "crates", "crane", "warehouse", "lighthouse", "sign"}

var havenVariantCount = map[string]int{
	"pier-x": 3, "pier-y": 3, "quay": 3, "bollard": 1, "crates": 3, "crane": 1, "warehouse": 2, "lighthouse": 1, "sign": 1,
}

var (
	pierR = ramp{rgb(84, 64, 50), rgb(126, 98, 72), rgb(162, 130, 94), rgb(196, 168, 124)}
	quayR = ramp{rgb(96, 62, 40), rgb(140, 96, 62), rgb(176, 130, 84), rgb(208, 168, 114)}
	ropeR = ramp{rgb(120, 96, 60), rgb(168, 140, 92), rgb(204, 180, 128), rgb(230, 212, 164)}
)

func HavenVariants(kind string) int { return havenVariantCount[kind] }

func HavenFootprint(kind string) int {
	if kind == "warehouse" {
		return 2
	}
	return 1
}

func havenName(kind string, v int) string { return fmt.Sprintf("haven/%s/%d", kind, v) }

type havenPiece struct {
	s     size
	paint func(c *canvas, v int)
}

var havenTileSize = size{havenTileW, havenTileH, havenTileOX, havenTileOY + 8}

func havenPieces() map[string]havenPiece {
	return map[string]havenPiece{
		"pier-x":     {havenTileSize, func(c *canvas, v int) { paintPier(c, v, true) }},
		"pier-y":     {havenTileSize, func(c *canvas, v int) { paintPier(c, v, false) }},
		"quay":       {havenTileSize, paintQuay},
		"bollard":    {size{11, 12, 4, 10}, paintBollard},
		"crates":     {size{24, 28, 12, 19}, paintCrates},
		"crane":      {craneSize, paintCrane},
		"warehouse":  {buildingLikeSize(2, warehouseHeight), paintWarehouse},
		"lighthouse": {buildingLikeSize(1, lighthouseHeight), paintLighthouse},
		"sign":       {size{20, 30, 6, 27}, paintHarborSign},
	}
}

func buildingLikeSize(n, height int) size {
	w := n*TileWidth + 4
	h := n*TileHeight + height + 4
	return size{w: w, h: h, ax: w / 2, ay: h - 2}
}

func addHaven(b *atlasBuilder) {
	pieces := havenPieces()
	for _, k := range HavenKinds {
		pc := pieces[k]
		for v := 0; v < havenVariantCount[k]; v++ {
			vv := v
			b.add(havenName(k, v), pc.s.w, pc.s.h, pc.s.ax, pc.s.ay, func(c *canvas) { pc.paint(c, vv) })
		}
	}
}

func tileIso(c *canvas) iso { return iso{c: c, ox: havenTileOX, oy: havenTileOY} }

func (s iso) ground(x, y int, z float64) (float64, float64) {
	d := (float64(x) + 0.5 - s.ox) / 16
	t := (float64(y) + 0.5 - s.oy + z) / 8
	return (t + d) / 2, (t - d) / 2
}

type pierAxis bool

func (m pierAxis) g(a, b float64) (float64, float64) {
	if m {
		return a, b
	}
	return b, a
}

func (m pierAxis) along(gx, gy float64) float64 {
	if m {
		return gx
	}
	return gy
}

func plankShade(r ramp, a float64, planks, seed int) color.NRGBA {
	f := a * float64(planks)
	if f-math.Floor(f) < 0.2 {
		return r[0]
	}
	switch hash(int(math.Floor(f)), seed, 431) % 7 {
	case 0:
		return r[1]
	case 1, 2:
		return r[3]
	}
	return r[2]
}

func paintPier(c *canvas, v int, alongX bool) {
	s, m := tileIso(c), pierAxis(alongX)
	x0, y0 := m.g(0, 0.2)
	x1, y1 := m.g(1, 0.8)
	top := func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, HavenDeck)
		a := m.along(gx, gy)
		if v == 1 && a > 0.52 && a < 0.64 {
			return deepR[0]
		}
		return plankShade(pierR, a, 6, v*13+boolInt(alongX))
	}
	beam := faces{top: top, left: beamShader(pierR, true), right: beamShader(pierR, false)}
	s.box(x0, y0, x1, y1, HavenDeck-havenBeamDepth, HavenDeck, beam)
	pierPosts(s, m)
	pierExtras(s, m, v)
	c.outline(ink)
	trimFarJoin(c, s, func(gx, gy float64) bool { return m.along(gx, gy) < 0 })
	pierRipples(s, m)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func beamShader(r ramp, lit bool) shader {
	return func(x, y int) color.NRGBA {
		if lit {
			return r[1]
		}
		return r[0]
	}
}

func pierPosts(s iso, m pierAxis) {
	for _, a := range []float64{0.15, 0.85} {
		gx, gy := m.g(a, 0.8)
		post := faces{top: flat(pierR[2]), left: flat(pierR[1]), right: flat(pierR[0])}
		s.box(gx-0.05, gy-0.05, gx+0.05, gy+0.05, -1, HavenDeck-havenBeamDepth, post)
	}
}

func pierExtras(s iso, m pierAxis, v int) {
	if v == 0 {
		return
	}
	gx, gy := m.g(0.85, 0.72)
	s.box(gx-0.06, gy-0.06, gx+0.06, gy+0.06, HavenDeck, HavenDeck+5, faces{top: flat(pierR[3]), left: flat(pierR[2]), right: flat(pierR[1])})
	if v == 2 {
		rx, ry := m.g(0.4, 0.45)
		p := s.p(rx, ry, HavenDeck)
		s.c.ellipse(p.x, p.y-0.5, 3, 1.6, ropeR[1])
		s.c.ellipse(p.x, p.y-0.8, 2, 1, ropeR[3])
		s.c.ellipse(p.x, p.y-0.8, 1, 0.5, ropeR[0])
	}
}

func trimFarJoin(c *canvas, s iso, far func(gx, gy float64) bool) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.at(x, y) != ink {
				continue
			}
			gx, gy := s.ground(x, y, HavenDeck)
			lx, ly := s.ground(x, y, HavenDeck-havenBeamDepth-1)
			if far(gx, gy) || far(lx, ly) {
				c.img.SetNRGBA(x, y, clear)
			}
		}
	}
}

func pierRipples(s iso, m pierAxis) {
	for _, a := range []float64{0.15, 0.85} {
		gx, gy := m.g(a, 0.85)
		p := s.p(gx, gy, 0)
		s.c.rect(int(p.x)-2, int(p.y), 4, 1, withAlpha(waterR[3], 150))
	}
}

func paintQuay(c *canvas, v int) {
	s := tileIso(c)
	top := func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, HavenDeck)
		return quayPlank(gx, gy, v)
	}
	s.box(0, 0, 1, 1, 0, HavenDeck, faces{top: top, left: quayFace(true), right: quayFace(false)})
	if v == 1 {
		p := s.p(0.62, 0.38, HavenDeck)
		s.c.shadePoly([]pt{{p.x - 4, p.y}, {p.x, p.y - 2}, {p.x + 4, p.y}, {p.x, p.y + 2}}, flat(inkSft))
		s.c.line(int(p.x)-2, int(p.y), int(p.x)+2, int(p.y), quayR[0])
	}
	c.outline(ink)
	trimFarJoin(c, s, func(gx, gy float64) bool { return gx < 0 || gy < 0 })
}

func quayPlank(gx, gy float64, v int) color.NRGBA {
	row := int(math.Floor(gy * 5))
	f := gy*5 - math.Floor(gy*5)
	if f < 0.2 {
		return quayR[0]
	}
	shift := float64(hash(row, v, 77)%3) / 3
	u := gx*2 + shift
	if u-math.Floor(u) < 0.06 {
		return quayR[0]
	}
	if v == 2 && hash(row, int(math.Floor(u)), 91)%3 == 0 {
		return quayR[1]
	}
	if hash(row, int(math.Floor(u)), 53)%4 == 0 {
		return quayR[3]
	}
	return quayR[2]
}

func quayFace(lit bool) shader {
	return func(x, y int) color.NRGBA {
		base, line := quayR[1], quayR[0]
		if !lit {
			base, line = quayR[0], darken(quayR[0], 0.25)
		}
		if y%3 == 0 {
			return line
		}
		return base
	}
}
