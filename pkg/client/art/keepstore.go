package art

import (
	"image/color"
	"math"
)

const (
	siloR      = 0.29
	siloH      = 24.0
	vaultH     = 8.0
	vaultSteps = 3
	blockSide  = 0.16
	blockRise  = 5.0
	poleH      = 30.0
)

func (r yardRect) ends() ([2]float64, [2]float64) {
	gx, gy := r.mid()
	if r.alongX() {
		return [2]float64{r.x0 + 0.3, gy}, [2]float64{r.x1 - 0.25, gy}
	}
	return [2]float64{gx, r.y0 + 0.3}, [2]float64{gx, r.y1 - 0.25}
}

func (s iso) keepGranary(r yardRect) {
	back, front := r.inset(0.04).ends()
	if s.lk.chance(130, 0.6) {
		s.keepSilo(back[0], back[1], siloR*0.75, siloH*0.8)
	} else {
		s.sacks(back[0], back[1])
	}
	gx, gy := r.cut(0.45, 0.9).mid()
	s.keepSilo(gx, gy, siloR, siloH)
	s.sacks(front[0]+0.08, front[1]+0.08)
}

func (s iso) keepSilo(cx, cy, r, h float64) {
	top := h * s.kit().scale
	s.turn(lathe{cx: cx, cy: cy, z: 0, h: 3, r: r + 0.03}, rubbleMat(stoneR, s.lk.seed+71), matStone)
	s.turn(lathe{cx: cx, cy: cy, z: 3, h: top - 3, r: r}, plankMat(woodR, s.lk.seed+73), matWall)
	s.roofCone(cx, cy, r+0.08, top, r*40, shapeRound, thatchMat(strawR, s.lk.seed+75))
	door := s.p(cx+r*0.35, cy+r*0.9, 3)
	rung := s.p(cx-r*0.2, cy+r*0.98, 3)
	s.pen(matTrim, func() {
		s.c.rect(int(door.x)-1, int(door.y)-6, 3, 6, woodR[0])
		s.c.dot(int(door.x), int(door.y)-7, strawR[3])
		for z := 0; z < int(top)-4; z += 3 {
			s.c.rect(int(rung.x)-1, int(rung.y)-z, 3, 1, woodR[3])
		}
	})
}

func (s iso) keepVault(r yardRect) {
	r = r.inset(0.08)
	k := s.kit()
	base := vaultH * k.scale
	stone := rubbleMat(stoneR, s.lk.seed+51)
	if s.lk.age >= ageCastle {
		stone = ashlarMat(stoneR, s.lk.seed+53)
	}
	s.block(r.x0, r.y0, r.x1, r.y1, 0, base, stone, stone, matStone)
	short := min(r.x1-r.x0, r.y1-r.y0) / 2
	for i := 1; i <= vaultSteps; i++ {
		t := float64(i) / float64(vaultSteps+1)
		in := short * (1 - math.Cos(t*math.Pi/2))
		z0 := base + 2*float64(i-1)
		b := r
		if r.alongX() {
			b.y0, b.y1 = r.y0+in, r.y1-in
		} else {
			b.x0, b.x1 = r.x0+in, r.x1-in
		}
		s.block(b.x0, b.y0, b.x1, b.y1, z0, z0+2, stone, toneMat(stoneR), matStone)
	}
	s.keepVaultDoor(r, base)
}

func (s iso) keepVaultDoor(r yardRect, h float64) {
	gx, gy := r.mid()
	dark := rgb(40, 32, 34)
	arch := func(uc float64, sh func(x, y int) (float64, float64)) shader {
		return func(x, y int) color.NRGBA {
			u, z := sh(x, y)
			if dz := z - h*0.55; dz > 0 && (u-uc)*(u-uc)/0.01+dz*dz/(h*h*0.2) > 1 {
				return stoneR[3]
			}
			return dark
		}
	}
	s.pen(matTrim, func() {
		if r.alongX() {
			s.onLeft(r.y1, gx-0.1, gx+0.1, 0, h, arch(gx, func(x, y int) (float64, float64) { return s.leftUV(x, y, r.y1) }))
			return
		}
		s.onRight(r.x1, gy-0.1, gy+0.1, 0, h, arch(gy, func(x, y int) (float64, float64) { return s.rightUV(x, y, r.x1) }))
	})
}

func (s iso) keepStones(gx, gy float64, n int) {
	spots := [][3]float64{{0, 0, 0}, {blockSide, 0, 0}, {0, blockSide, 0}, {blockSide * 0.5, blockSide * 0.5, blockRise}, {blockSide, blockSide, 0}}
	faces := faces{top: flat(cutR[3]), left: flat(cutR[2]), right: flat(cutR[1])}
	s.pen(matStone, func() {
		for _, p := range spots[:min(n, len(spots))] {
			x, y := gx+p[0]-blockSide, gy+p[1]-blockSide
			s.box(x, y, x+blockSide*0.92, y+blockSide*0.92, p[2], p[2]+blockRise, faces)
		}
	})
}

func (s iso) keepBanners(k KeepLook, a, b yardRect) {
	cloth := keepCloth(k)
	for i, r := range []yardRect{a, b} {
		_, front := r.ends()
		base, top := s.p(front[0], front[1], 0), s.p(front[0], front[1], poleH*s.kit().scale)
		s.pen(matTrim, func() {
			s.c.line(int(base.x), int(base.y), int(top.x), int(top.y), woodR[0])
			s.c.dot(int(top.x), int(top.y)-1, goldR[3])
			x := int(top.x) + 1 - 6*i
			for yy := int(top.y) + 1; yy < int(top.y)+13; yy++ {
				for xx := x; xx < x+5; xx++ {
					s.c.dot(xx, yy, []color.NRGBA{cloth[2], cloth[1], purpleR[2]}[(xx+yy)%3/2+keepBit(yy > int(top.y)+10)])
				}
			}
		})
	}
}

func keepBit(b bool) int {
	if b {
		return 1
	}
	return 0
}
