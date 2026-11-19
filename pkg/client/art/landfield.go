package art

import "math"

const (
	coastWobble   = 0.46
	groundWobble  = 0.42
	terraceWobble = 0.3
)

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }

func lattice(x, y, seed int) float64 {
	return float64(hash(x, y, seed)&0xffff) / 65535
}

func vnoise(x, y float64, seed int) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	tx, ty := smoothstep(x-x0), smoothstep(y-y0)
	ix, iy := int(x0), int(y0)
	a, b := lattice(ix, iy, seed), lattice(ix+1, iy, seed)
	c, d := lattice(ix, iy+1, seed), lattice(ix+1, iy+1, seed)
	return a + (b-a)*tx + (c-a)*ty + (a-b-c+d)*tx*ty
}

func fbm(x, y float64, seed int) float64 {
	return 0.62*vnoise(x*1.6, y*1.6, seed) + 0.38*vnoise(x*4.3, y*4.3, seed+1) - 0.5
}

type cell struct {
	g [4]Ground
	h [4]uint8
	w [4]float64
}

var cellCorners = [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}

func (l *Land) cellAt(gx, gy float64) cell {
	x0, y0 := math.Floor(gx), math.Floor(gy)
	sx, sy := smoothstep(gx-x0), smoothstep(gy-y0)
	ix, iy := int(x0), int(y0)
	c := cell{w: [4]float64{(1 - sx) * (1 - sy), sx * (1 - sy), (1 - sx) * sy, sx * sy}}
	for k, d := range cellCorners {
		c.g[k] = baseGround(l.groundAt(ix+d[0], iy+d[1]))
		c.h[k] = l.heightAt(ix+d[0], iy+d[1])
	}
	return c
}

func baseGround(g Ground) Ground {
	switch g {
	case GroundRoad:
		return GroundGrass
	case GroundShallows:
		return GroundWater
	}
	return g
}

func (c cell) uniform() bool {
	return c.g[0] == c.g[1] && c.g[0] == c.g[2] && c.g[0] == c.g[3]
}

func (c cell) flat() bool {
	return c.h[0] == c.h[1] && c.h[0] == c.h[2] && c.h[0] == c.h[3]
}

func (c cell) material(gx, gy float64) Ground {
	if c.uniform() {
		return c.g[0]
	}
	land := 0.0
	for k, g := range c.g {
		if g != GroundWater {
			land += c.w[k]
		}
	}
	if land+fbm(gx*0.9, gy*0.9, 101)*coastWobble < 0.5 {
		return GroundWater
	}
	var weight [GroundPlaza + 1]float64
	for k, g := range c.g {
		if g != GroundWater {
			weight[g] += c.w[k]
		}
	}
	best, score := GroundSand, math.Inf(-1)
	for g, w := range weight {
		if w == 0 {
			continue
		}
		if s := w + fbm(gx, gy, 200+g*7)*groundWobble; s > score {
			best, score = Ground(g), s
		}
	}
	return best
}

func (c cell) level(gx, gy float64) uint8 {
	if c.flat() {
		return c.h[0]
	}
	n := fbm(gx*1.1, gy*1.1, 300) * terraceWobble
	lv := uint8(0)
	for step := uint8(1); step <= MaxLevel; step++ {
		s := 0.0
		for k, h := range c.h {
			if h >= step {
				s += c.w[k]
			}
		}
		if s == 0 || s+n < 0.5 {
			break
		}
		lv = step
	}
	return lv
}

func (l *Land) corners(gx, gy float64, at func(i int) float64, fallback float64) float64 {
	x0, y0 := math.Floor(gx), math.Floor(gy)
	tx, ty := gx-x0, gy-y0
	ix, iy := int(x0), int(y0)
	var v [4]float64
	for k, d := range cellCorners {
		v[k] = fallback
		if i, ok := l.index(ix+d[0], iy+d[1]); ok {
			v[k] = at(i)
		}
	}
	return v[0] + (v[1]-v[0])*tx + (v[2]-v[0])*ty + (v[0]-v[1]-v[2]+v[3])*tx*ty
}

func (l *Land) powerField(gx, gy float64) float64 {
	if len(l.Power) < l.W*l.H {
		return 0
	}
	return l.corners(gx, gy, func(i int) float64 { return float64(l.Power[i]) }, 0)
}

func (l *Land) depthField(gx, gy float64) float64 {
	if len(l.depth) < l.W*l.H {
		return seaReach
	}
	return l.corners(gx, gy, func(i int) float64 { return float64(l.depth[i]) }, seaReach)
}

func (l *Land) nearCoast(gx, gy float64) bool {
	ix, iy := int(math.Floor(gx)), int(math.Floor(gy))
	for _, d := range cellCorners {
		if l.depthAt(ix+d[0], iy+d[1]) < openSea {
			return true
		}
	}
	return false
}

func (l *Land) wildAt(gx, gy float64) bool {
	i, ok := l.index(int(math.Floor(gx+0.5)), int(math.Floor(gy+0.5)))
	return ok && i < len(l.Wild) && l.Wild[i]
}

func groundPoint(x, y int) (float64, float64) {
	u, v := (float64(x)+0.5)/16, (float64(y)+0.5)/8
	return (u + v) / 2, (v - u) / 2
}
