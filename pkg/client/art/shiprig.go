package art

import (
	"image/color"
	"math"
)

func (s shipScene) mastBase(m mastSpec) float64 {
	return s.castleTop(m.at)
}

func (s shipScene) mastGroup(m mastSpec) {
	u, base := m.at*s.sp.half(), s.mastBase(m)
	front := s.v.toward(vec3{u: 1}) > 0
	if m.lateen {
		front = s.v.toward(vec3{w: 1}) > 0
	}
	sails := func() {
		if m.lateen {
			s.lateenSail(m, u, base)
			return
		}
		for _, y := range m.yards {
			s.squareSail(u, base, y)
		}
	}
	if front {
		s.mast(m, u, base)
		sails()
		return
	}
	sails()
	s.mast(m, u, base)
}

func (s shipScene) mast(m mastSpec, u, base float64) {
	top, bot := s.v.p(u, 0, base+m.height), s.v.p(u, 0, base)
	x := int(math.Floor(top.x))
	for y := int(math.Floor(top.y)); y <= int(math.Floor(bot.y)); y++ {
		s.v.c.dot(x, y, s.lk.wood[0])
		if m.width > 1 {
			s.v.c.dot(x-1, y, s.lk.wood[2])
		}
	}
	if m.nest {
		n := s.v.p(u, 0, base+m.height*0.74)
		nx, ny := int(math.Floor(n.x)), int(math.Floor(n.y))
		s.v.c.rect(nx-2, ny, 5, 2, s.lk.wood[1])
		s.v.c.rect(nx-2, ny, 5, 1, s.lk.wood[2])
	}
}

func (s shipScene) billow() float64 {
	return s.sp.billow * shipBillow[s.frame]
}

func (s shipScene) squareSail(u, base float64, y yardSpec) {
	b, top, foot := s.billow(), base+y.top, base+y.foot
	at := func(sv, t float64) vec3 {
		half := y.span * (1 - 0.1*t)
		bulge := b * math.Sin(math.Pi*sv) * (0.35 + 0.65*math.Sin(math.Pi*t))
		return vec3{u + 0.02 + bulge, (sv*2 - 1) * half, top + (foot-top)*t}
	}
	ns := 6
	if y.span*36 > 16 {
		ns = 12
	}
	m := mesh{at: at, ns: ns, nt: max(3, int((top-foot)/2)), cloth: s.lk.sail.color}
	s.v.drawMesh(m)
	s.hem(m)
	s.v.seg(vec3{u + 0.02, -y.span * 1.15, top}, vec3{u + 0.02, y.span * 1.15, top}, s.lk.wood[0])
}

func (s shipScene) hem(m mesh) {
	col := s.lk.sail.cloth[0]
	s.v.polyline(m, [2]float64{0, 1}, [2]float64{1, 1}, 8, col)
	s.v.polyline(m, [2]float64{0, 0.08}, [2]float64{0, 1}, 4, col)
	s.v.polyline(m, [2]float64{1, 0.08}, [2]float64{1, 1}, 4, col)
}

func (s shipScene) lateenSail(m mastSpec, u, base float64) {
	head := vec3{u, 0, base + m.height}
	tack := vec3{u + m.fore, 0, base + 2}
	peak := vec3{head.u + (head.u-tack.u)*m.over, 0, head.z + (head.z-tack.z)*m.over}
	clew := vec3{u - m.aft, 0, base + 3}
	b := s.billow() * 1.4
	at := func(sv, t float64) vec3 {
		x := vec3{
			tack.u + sv*((1-t)*(peak.u-tack.u)+t*(clew.u-tack.u)),
			0,
			tack.z + sv*((1-t)*(peak.z-tack.z)+t*(clew.z-tack.z)),
		}
		x.w = b * math.Sin(math.Pi*sv) * math.Sin(math.Pi*0.8*t)
		return x
	}
	mm := mesh{at: at, ns: 8, nt: 6, cloth: s.lk.sail.color}
	s.v.drawMesh(mm)
	col := s.lk.sail.cloth[0]
	s.v.polyline(mm, [2]float64{0, 1}, [2]float64{1, 1}, 8, col)
	s.v.polyline(mm, [2]float64{1, 0}, [2]float64{1, 1}, 6, col)
	s.v.seg(vec3{tack.u + m.fore*0.25, 0, tack.z - 1}, peak, s.lk.wood[0])
}

func (s shipScene) bowsprit() {
	if s.sp.bowsprit <= 0 {
		return
	}
	h := s.sp.half()
	root := vec3{h * 0.9, 0, s.castleTop(0.9) - 1}
	tip := vec3{h + s.sp.bowsprit, 0, s.sp.rail(1) + s.sp.bowsprit*14}
	s.v.seg(root, tip, s.lk.wood[1])
	s.v.seg(vec3{root.u, 0, root.z + 1}, vec3{tip.u, 0, tip.z + 1}, s.lk.wood[2])
}

func (s shipScene) pennants() {
	best := 0
	for i, m := range s.sp.masts {
		if m.height+s.mastBase(m) > s.sp.masts[best].height+s.mastBase(s.sp.masts[best]) {
			best = i
		}
	}
	for i, m := range s.sp.masts {
		if m.pennant <= 0 {
			continue
		}
		cloth := s.lk.flag
		if i != best {
			cloth = s.lk.trim
		}
		s.pennant(m.at*s.sp.half(), s.mastBase(m)+m.height, m.pennant, m.flutterAmp, cloth)
	}
}

func (s shipScene) pennant(u, z, length, amp float64, cloth ramp) {
	n := 8
	up, dn := make([]pt, 0, n+1), make([]pt, 0, n+1)
	for k := 0; k <= n; k++ {
		f := float64(k) / float64(n)
		wave := math.Sin(2*math.Pi*(f*1.1-float64(s.frame)/4)) * amp * f
		half := 1.0*(1-f) + 0.25
		c := z + wave - f*1.5
		up = append(up, s.v.p(u+f*length, 0, c+half))
		dn = append(dn, s.v.p(u+f*length, 0, c-half))
	}
	for i := len(dn) - 1; i >= 0; i-- {
		up = append(up, dn[i])
	}
	s.v.c.shadePoly(up, func(x, y int) color.NRGBA {
		if (x+y)%4 == 0 {
			return cloth[3]
		}
		return cloth[2]
	})
	s.v.seg(vec3{u, 0, z - 1}, vec3{u, 0, z + 2}, s.lk.wood[0])
}

func (s shipScene) rigging() {
	if !s.sp.stays || len(s.sp.masts) == 0 {
		return
	}
	col := withAlpha(inkSft, 150)
	h := s.sp.half()
	fore := s.sp.masts[0]
	tip := vec3{h + s.sp.bowsprit, 0, s.sp.rail(1) + s.sp.bowsprit*14}
	s.v.seg(vec3{fore.at * h, 0, s.mastBase(fore) + fore.height*0.9}, tip, col)
}
