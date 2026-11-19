package art

import (
	"image/color"
	"math"
)

var (
	marbleR = ramp{rgb(168, 158, 144), rgb(212, 204, 188), rgb(234, 228, 212), rgb(250, 246, 232)}
	copperR = ramp{rgb(36, 88, 84), rgb(58, 126, 110), rgb(92, 164, 138), rgb(156, 208, 178)}
	daubR   = ramp{rgb(150, 124, 92), rgb(192, 164, 124), rgb(216, 192, 150), rgb(236, 216, 176)}
)

func level(light float64) float64 { return 1.4 + light*1.1 }

func rampAt(r ramp, f float64) color.NRGBA {
	return r[min(max(int(math.Round(f)), 0), 3)]
}

func fmod(a, b float64) float64 {
	m := math.Mod(a, b)
	if m < 0 {
		m += b
	}
	return m
}

type courses struct {
	row   [256]int16
	start [256]float64
}

func newCourses(seed int, lo, span int) *courses {
	c := &courses{}
	pos, k := 0, 0
	for pos < len(c.row) {
		h := lo + int(hash(k, seed, 311)%uint32(span))
		for i := pos; i < min(pos+h, len(c.row)); i++ {
			c.row[i], c.start[i] = int16(k), float64(pos)
		}
		pos, k = pos+h, k+1
	}
	return c
}

func (c *courses) at(v float64) (int, float64) {
	i := min(max(int(v), 0), len(c.row)-1)
	return int(c.row[i]), v - c.start[i]
}

func rubbleMat(r ramp, seed int) mat {
	c := newCourses(seed, 3, 3)
	return func(p surf) color.NRGBA {
		row, fv := c.at(p.v)
		w := 5 + float64(hash(row, seed, 313)%4)
		o := float64(hash(row, seed, 317) % 7)
		col := int(math.Floor((p.u + o) / w))
		fu := fmod(p.u+o, w)
		if fv < 1 || fu < 1 {
			return r[0]
		}
		j := (float64(hash(row, col, seed)%5) - 2) * 0.32
		f := level(p.light) + j
		if fu < 2 && fv > 2 && hash(col, row, seed+1)%3 == 0 {
			f++
		}
		return rampAt(r, min(max(f, 0.8), 3))
	}
}

func ashlarMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		row := int(math.Floor(p.v / 4))
		o := float64(row%2) * 4.5
		col := int(math.Floor((p.u + o) / 9))
		fu := fmod(p.u+o, 9)
		if fmod(p.v, 4) < 1 || fu < 1 {
			return mix(r[0], r[1], 0.45)
		}
		j := (float64(hash(row, col, seed)%3) - 1) * 0.28
		return rampAt(r, min(max(level(p.light)+j, 1), 3))
	}
}

func marbleMat(seed int) mat {
	stone := ashlarMat(marbleR, seed)
	return func(p surf) color.NRGBA {
		v := vnoise(p.u*0.25, p.v*0.3, seed)
		if math.Abs(v-0.5) < 0.015 {
			return marbleR[1]
		}
		return stone(p)
	}
}

func wattleMat(seed int) mat {
	return func(p surf) color.NRGBA {
		n := vnoise(p.u/7, p.v/5, seed)
		switch {
		case n > 0.74:
			if (int(p.u)/2+int(p.v)/2)%2 == 0 {
				return rampAt(woodR, level(p.light)+0.3)
			}
			return rampAt(woodR, level(p.light)-0.6)
		case n > 0.7:
			return daubR[0]
		case pick(p.x, p.y, seed) > 0.95:
			return rampAt(daubR, level(p.light)-1)
		}
		return tone(daubR, p.light*0.8, p.x, p.y)
	}
}

func plasterMat(seed int) mat {
	return func(p surf) color.NRGBA {
		n := vnoise(p.u/9, p.v/7, seed)
		f := level(p.light) + 0.35
		if n < 0.3 {
			f -= 0.6
		}
		if pick(p.x, p.y, seed+3) > 0.97 {
			f--
		}
		return rampAt(plasterR, f)
	}
}

func logMat(seed int) mat {
	return func(p surf) color.NRGBA {
		row := int(math.Floor(p.v / 4))
		fv := fmod(p.v, 4)
		f := level(p.light) + (float64(hash(row, seed, 331)%3)-1)*0.3
		switch {
		case fv < 1:
			return woodR[0]
		case fv >= 3:
			f += 0.7
		case hash(row, int(p.u/3), seed)%29 == 0:
			return woodR[0]
		}
		return rampAt(woodR, f)
	}
}

func plankMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		col := int(math.Floor(p.u / 4))
		if fmod(p.u, 4) < 1 {
			return r[0]
		}
		f := level(p.light) + (float64(hash(col, seed, 337)%3)-1)*0.35
		if hash(col, int(p.v/2), seed)%9 == 0 {
			f -= 0.8
		}
		return rampAt(r, f)
	}
}

func thatchMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		col := int(math.Floor(p.u / 1.5))
		band := int(p.v/3) + int(hash(col, 0, seed)%3)
		f := level(p.light) + (float64(hash(col, band, seed)%5)-2)*0.24
		if fmod(p.v+float64(hash(col/3, 1, seed)%3), 6) < 1 && dither(p.x, p.y) < 0.7 {
			f -= 0.9
		}
		return rampAt(r, f)
	}
}

func shingleMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		row := int(math.Floor(p.v / 3))
		o := float64(hash(row, seed, 347) % 4)
		col := int(math.Floor((p.u + o) / 4))
		f := level(p.light)
		switch h := hash(row, col, seed) % 17; {
		case fmod(p.v, 3) < 1:
			return r[0]
		case fmod(p.u+o, 4) < 0.9:
			f -= 1
		case h == 0:
			f -= 1.2
		case h < 3:
			f += 0.8
		}
		return rampAt(r, f)
	}
}

func slateMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		row := int(math.Floor(p.v / 3))
		o := float64(row%2)*1.5 + float64(hash(row, seed, 349)%2)*0.5
		col := int(math.Floor((p.u + o) / 3))
		f := level(p.light) + (float64(hash(row, col, seed)%3)-1)*0.3
		switch {
		case fmod(p.v, 3) < 1:
			return r[0]
		case fmod(p.u+o, 3) < 0.8:
			f -= 0.8
		}
		return rampAt(r, f)
	}
}

func tileMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		row := int(math.Floor(p.v / 4))
		o := float64(hash(row, seed, 353) % 2)
		f := level(p.light) + (float64(hash(row, int(p.u/3), seed)%3)-1)*0.25
		switch fv := fmod(p.v, 4); {
		case fv < 1:
			return r[0]
		case fv >= 3:
			f += 0.6
		}
		if fmod(p.u+o, 3) < 1 {
			f -= 0.8
		}
		return rampAt(r, f)
	}
}

func ribbedMat(r ramp, seed int) mat {
	return func(p surf) color.NRGBA {
		f := level(p.light) + p.up*0.6
		if fmod(p.u, 7) < 1 {
			f -= 0.9
		}
		if vnoise(p.u*0.3, p.v*0.3, seed) > 0.72 {
			f -= 0.5
		}
		return rampAt(r, f)
	}
}

func flatMat(col color.NRGBA) mat { return func(surf) color.NRGBA { return col } }

func toneMat(r ramp) mat {
	return func(p surf) color.NRGBA { return tone(r, p.light, p.x, p.y) }
}

func stripeMat(a, b ramp, w float64) mat {
	return func(p surf) color.NRGBA {
		if int(math.Floor(p.u/w))%2 == 0 {
			return tone(a, p.light, p.x, p.y)
		}
		return tone(b, p.light, p.x, p.y)
	}
}
