package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	ForumPaveVariants = 6
	ForumRoseReach    = 1
	forumPaveW        = TileWidth + 2
	forumPaveH        = TileHeight + 2
	forumPaveAX       = forumPaveW / 2
	forumPaveAY       = forumPaveH / 2
	forumPaveSeed     = 7331
	devFill           = 0.74
	stoneSpan         = 0.46
)

var (
	ForumSides   = []string{"+x", "+y", "-x", "-y"}
	porphyryR    = ramp{rgb(70, 26, 34), rgb(108, 40, 46), rgb(142, 58, 58), rgb(178, 92, 80)}
	verdeR       = ramp{rgb(28, 58, 46), rgb(44, 86, 64), rgb(68, 118, 84), rgb(112, 156, 116)}
	tesseraR     = ramp{rgb(196, 176, 140), rgb(222, 206, 172), rgb(238, 226, 198), rgb(250, 242, 222)}
	terracottaR  = ramp{rgb(120, 58, 36), rgb(164, 84, 50), rgb(196, 112, 68), rgb(222, 148, 96)}
	nigraR       = ramp{rgb(40, 34, 32), rgb(58, 50, 46), rgb(76, 66, 60), rgb(98, 88, 80)}
	ochreR       = ramp{rgb(146, 100, 46), rgb(180, 132, 64), rgb(202, 158, 84), rgb(224, 188, 118)}
	wildflowers  = []color.NRGBA{rgb(250, 226, 92), rgb(246, 244, 236), rgb(236, 142, 178), rgb(148, 170, 236), rgb(250, 176, 84)}
	forumPainter = [AgeTiers]func(u, v float64, x, y, variant int) color.NRGBA{devPave, stagingPave, uatPave, productionPave}
)

func ForumPaveName(age, v int) string { return fmt.Sprintf("forum/pave/%d/%d", clampAge(age), v) }
func ForumRimName(age int, side string) string {
	return fmt.Sprintf("forum/rim/%d/%s", clampAge(age), side)
}
func ForumBandName(age int, axis string) string {
	return fmt.Sprintf("forum/band/%d/%s", clampAge(age), axis)
}
func ForumRoseName(age, dx, dy int) string {
	return fmt.Sprintf("forum/rose/%d/%d/%d", clampAge(age), dx, dy)
}

type tileTex func(gx, gy float64, x, y int) (color.NRGBA, bool)

func paintTile(c *canvas, tex tileTex) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			a, b := (float64(x)+0.5-forumPaveAX)/16, (float64(y)+0.5-forumPaveAY)/8
			gx, gy := (a+b)/2, (b-a)/2
			if math.Abs(gx) > 0.5 || math.Abs(gy) > 0.5 {
				continue
			}
			if col, ok := tex(gx, gy, x, y); ok {
				c.dot(x, y, col)
			}
		}
	}
}

func addForumPaving(b *atlasBuilder) {
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < ForumPaveVariants; v++ {
			aa, vv := age, v
			b.add(ForumPaveName(age, v), forumPaveW, forumPaveH, forumPaveAX, forumPaveAY, func(c *canvas) {
				paintTile(c, func(gx, gy float64, x, y int) (color.NRGBA, bool) {
					col := forumPainter[aa](gx+0.5, gy+0.5, x, y, vv)
					return col, col.A > 0
				})
			})
		}
		if age < ageCastle {
			continue
		}
		addForumTrims(b, age)
	}
}

func addForumTrims(b *atlasBuilder, age int) {
	for i, side := range ForumSides {
		aa, ii := age, i
		b.add(ForumRimName(age, side), forumPaveW, forumPaveH, forumPaveAX, forumPaveAY, func(c *canvas) { paintTile(c, rimTex(aa, ii)) })
	}
	for _, axis := range []string{"x", "y"} {
		aa, ax := age, axis
		b.add(ForumBandName(age, axis), forumPaveW, forumPaveH, forumPaveAX, forumPaveAY, func(c *canvas) { paintTile(c, bandTex(aa, ax)) })
	}
	for dy := -ForumRoseReach; dy <= ForumRoseReach; dy++ {
		for dx := -ForumRoseReach; dx <= ForumRoseReach; dx++ {
			aa, ddx, ddy := age, dx, dy
			b.add(ForumRoseName(age, dx, dy), forumPaveW, forumPaveH, forumPaveAX, forumPaveAY, func(c *canvas) {
				paintTile(c, func(gx, gy float64, x, y int) (color.NRGBA, bool) {
					return roseTex(aa, gx+float64(ddx), gy+float64(ddy), x, y)
				})
			})
		}
	}
}

func flagstone(u, v float64, cells, seed int, fill float64) (float64, int, bool) {
	cu, cv := int(math.Floor(u*float64(cells))), int(math.Floor(v*float64(cells)))
	h := hash(cu, cv, seed)
	if float64(h%100)/100 > fill {
		return 0, 0, false
	}
	size := 1 / float64(cells)
	jx, jy := (float64(h>>8%9)-4)*size*0.03, (float64(h>>12%9)-4)*size*0.03
	du := (u - (float64(cu)+0.5)*size - jx) / (size * stoneSpan)
	dv := (v - (float64(cv)+0.5)*size - jy) / (size * stoneSpan)
	wob := 1 + (vnoise(u*9, v*9, seed+3)-0.5)*0.35
	r := math.Pow(math.Pow(math.Abs(du), 3)+math.Pow(math.Abs(dv), 3), 1.0/3)
	return -(du*0.7 + dv) / 1.4, int(h >> 20 % 3), r <= wob
}

func devPave(u, v float64, x, y, variant int) color.NRGBA {
	seed := forumPaveSeed + variant*97
	if light, tone, ok := flagstone(u, v, 3, seed, devFill); ok {
		switch {
		case light > 0.42:
			return plazaR[3]
		case light < -0.46:
			return plazaR[0]
		case pick(x, y, seed+5) > 0.9:
			return mossR[2]
		}
		return plazaR[1+tone%2]
	}
	if n := pick(x, y, seed+7); n > 0.97 {
		return wildflowers[int(hash(x, y, seed+9)%uint32(len(wildflowers)))]
	} else if n > 0.84 {
		return grassR[3]
	} else if n < 0.1 {
		return grassR[0]
	}
	return color.NRGBA{}
}

func stagingPave(u, v float64, x, y, variant int) color.NRGBA {
	seed := forumPaveSeed + 11 + variant*89
	row := int(math.Floor(v * 4))
	off := float64(row%2) * 0.25
	col := int(math.Floor((u + off) * 2))
	fu, fv := fmod((u+off)*2, 1), fmod(v*4, 1)
	switch {
	case fv < 0.14 || fu < 0.07:
		if pick(x, y, seed) < 0.55 {
			return mossR[1]
		}
		return plazaR[0]
	case fv > 0.82:
		return darken(plazaR[1], 0.06)
	case fv < 0.3 && fu < 0.5:
		return plazaR[3]
	}
	tone := int(hash(col, row, seed) % 3)
	if pick(x, y, seed+3) > 0.93 {
		return plazaR[tone]
	}
	return plazaR[1+tone%2]
}

func uatPave(u, v float64, x, y, variant int) color.NRGBA {
	if crosslet(u, v) {
		return tesseraAt(nigraR, x, y, variant)
	}
	if u < 0.035 || v < 0.035 {
		return tesseraR[1]
	}
	du, dv := u-0.5, v-0.5
	switch variant % 3 {
	case 1:
		if col, ok := lozenge(du, dv, x, y, variant); ok {
			return col
		}
	case 2:
		if col, ok := rosette(du, dv, x, y, variant); ok {
			return col
		}
	}
	return tesseraAt(tesseraR, x, y, variant)
}

func crosslet(u, v float64) bool {
	cu, cv := math.Min(u, 1-u), math.Min(v, 1-v)
	return (cu < 0.04 && cv < 0.14) || (cv < 0.04 && cu < 0.14)
}

func lozenge(du, dv float64, x, y, variant int) (color.NRGBA, bool) {
	switch m := math.Max(math.Abs(du), math.Abs(dv)); {
	case m < 0.04:
		return tesseraAt(ochreR, x, y, variant), true
	case m < 0.16:
		return tesseraAt(terracottaR, x, y, variant), true
	case m < 0.2:
		return tesseraAt(nigraR, x, y, variant), true
	}
	return color.NRGBA{}, false
}

const petalStretch = 1.5

var petalCentres = [4][2]float64{{0.22, 0}, {-0.22, 0}, {0, 0.22}, {0, -0.22}}

func rosette(du, dv float64, x, y, variant int) (color.NRGBA, bool) {
	if math.Hypot(du, dv) < 0.07 {
		return tesseraAt(verdeR, x, y, variant), true
	}
	near := math.Inf(1)
	for _, p := range petalCentres {
		near = math.Min(near, petalDistance(du-p[0], dv-p[1], p[1] == 0))
	}
	switch {
	case near < 0.1:
		return tesseraAt(ochreR, x, y, variant), true
	case near < 0.13:
		return tesseraAt(nigraR, x, y, variant), true
	}
	return color.NRGBA{}, false
}

func petalDistance(du, dv float64, alongU bool) float64 {
	if alongU {
		return math.Hypot(du/petalStretch, dv)
	}
	return math.Hypot(du, dv/petalStretch)
}

func tesseraAt(r ramp, x, y, variant int) color.NRGBA {
	if (x%2 == 0 && y%2 == 0) || pick(x/2, y, forumPaveSeed+variant) > 0.92 {
		return r[1]
	}
	if pick(x/2, y, forumPaveSeed+31+variant) > 0.8 {
		return r[3]
	}
	return r[2]
}

func productionPave(u, v float64, x, y, variant int) color.NRGBA {
	seed := forumPaveSeed + 41 + variant*53
	if u < 0.04 || v < 0.04 {
		return marbleR[0]
	}
	du, dv := u-0.52, v-0.52
	switch variant % 3 {
	case 1:
		if col, ok := roundel(du, dv, x, y); ok {
			return col
		}
	case 2:
		if col, ok := insetSquare(du, dv, x, y); ok {
			return col
		}
	}
	if n := vnoise(u*3+float64(variant), v*5, seed); math.Abs(n-0.5) < 0.02 {
		return marbleR[1]
	}
	if (u+v) < 0.3 && dither(x, y) < 0.5 {
		return marbleR[3]
	}
	return marbleR[2+variant%2]
}

func roundel(du, dv float64, x, y int) (color.NRGBA, bool) {
	switch r := math.Hypot(du, dv); {
	case r < 0.24:
		return speckledStone(porphyryR, x, y), true
	case r < 0.28:
		return ochreR[2], true
	}
	return color.NRGBA{}, false
}

func insetSquare(du, dv float64, x, y int) (color.NRGBA, bool) {
	switch m := math.Max(math.Abs(du), math.Abs(dv)); {
	case m < 0.2:
		return speckledStone(verdeR, x, y), true
	case m < 0.24:
		return ochreR[2], true
	}
	return color.NRGBA{}, false
}

func speckledStone(r ramp, x, y int) color.NRGBA {
	if pick(x, y, forumPaveSeed+61) > 0.9 {
		return r[3]
	}
	return r[1+(x/2+y)%2]
}

func rimTex(age, side int) tileTex {
	if age == ageCastle {
		return dentilRim(side)
	}
	return func(gx, gy float64, x, y int) (color.NRGBA, bool) {
		d := 0.5 - [4]float64{gx, gy, -gx, -gy}[side]
		switch {
		case d < 0.05:
			return porphyryR[0], true
		case d < 0.16:
			if d > 0.13 {
				return ochreR[2], true
			}
			return porphyryR[1+(x+y)%2], true
		}
		return color.NRGBA{}, false
	}
}

func dentilRim(side int) tileTex {
	return func(gx, gy float64, x, y int) (color.NRGBA, bool) {
		d := 0.5 - [4]float64{gx, gy, -gx, -gy}[side]
		switch {
		case d < 0.05:
			return nigraR[1], true
		case d < 0.14:
			if (x/3)%2 == 0 {
				return terracottaR[2], true
			}
			return tesseraR[2], true
		case d < 0.17:
			return nigraR[2], true
		}
		return color.NRGBA{}, false
	}
}

func bandTex(age int, axis string) tileTex {
	return func(gx, gy float64, x, y int) (color.NRGBA, bool) {
		d := math.Abs(gy)
		if axis == "y" {
			d = math.Abs(gx)
		}
		if age == ageCastle {
			return castleBand(d, x, y)
		}
		switch {
		case d < 0.07:
			if (x+y)%5 == 0 {
				return ochreR[2], true
			}
			return ochreR[3], true
		case d < 0.1:
			return marbleR[0], true
		}
		return color.NRGBA{}, false
	}
}

func castleBand(d float64, x, y int) (color.NRGBA, bool) {
	switch {
	case d < 0.03:
		return ochreR[2], true
	case d < 0.075:
		return terracottaR[1+(x+y)%2], true
	case d < 0.1:
		return nigraR[1], true
	}
	return color.NRGBA{}, false
}

func roseTex(age int, gx, gy float64, x, y int) (color.NRGBA, bool) {
	r := math.Hypot(gx, gy*1.0)
	ang := math.Atan2(gy, gx)
	rays := math.Abs(math.Cos(4*ang)) * (1.2 - r)
	inner, outer, ring := porphyryR, ochreR, ochreR[1]
	if age == ageCastle {
		inner, outer, ring = terracottaR, ochreR, nigraR[1]
	}
	switch {
	case r > 1.45:
		return color.NRGBA{}, false
	case r > 1.32:
		if age == ageCastle && int(math.Floor((ang+math.Pi)*6/math.Pi))%2 == 0 {
			return nigraR[1], true
		}
		return outer[1+(x+y)%2], true
	case r > 1.22:
		return color.NRGBA{}, false
	case rays > 0.55:
		return inner[2+(x/2+y)%2], true
	case r < 0.16:
		return outer[3], true
	case r > 0.52 && r < 0.62:
		return ring, true
	}
	return color.NRGBA{}, false
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
