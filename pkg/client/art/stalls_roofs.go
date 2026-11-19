package art

func (t stall) roof(p stallPlan) {
	switch t.lk.age {
	case ageDark:
		t.awning(p, true)
	case ageFeudal:
		t.awning(p, false)
	case ageCastle:
		t.pentRoof(p)
	default:
		t.portico(p)
	}
}

func (t stall) awning(p stallPlan, striped bool) {
	cloth, strips := t.cloth(), 8
	stripe := mix(plasterR[3], cloth[3], 0.25)
	for i := 0; i < strips; i++ {
		a, b := 0.02+float64(i)*1.96/float64(strips), 0.02+float64(i+1)*1.96/float64(strips)
		col := cloth[2]
		if striped && i%2 == 1 {
			col = stripe
		}
		if !striped && i%2 == 1 {
			col = cloth[1]
		}
		t.pen(matRoof, func() {
			t.quad(t.at(a, eaveV, p.eave), t.at(b, eaveV, p.eave), t.at(b, frontV, p.posts), t.at(a, frontV, p.posts), flat(col))
		})
	}
	t.valance(p, cloth, striped)
}

func (t stall) valance(p stallPlan, cloth ramp, striped bool) {
	f := frontAt(frontV)
	if !t.sees(f) {
		return
	}
	for i := 0; i < 8; i++ {
		a := 0.02 + float64(i)*0.245
		col, drop := cloth[1], 2.0
		if i%2 == 1 {
			col = plasterR[3]
			if striped {
				col = mix(plasterR[3], cloth[3], 0.25)
			}
		}
		if !striped {
			drop = 3
		}
		t.pen(matTrim, func() { t.panel(f, a, a+0.245, p.posts-drop, p.posts, flat(col)) })
	}
}

func (t stall) pentRoof(p stallPlan) {
	r := mixRamp(slateR, t.lk.id, 0.62)
	m := slateMat(r, t.lk.seed+17)
	t.pen(matRoof, func() {
		t.quad(t.at(0.02, eaveV, p.eave), t.at(1.98, eaveV, p.eave), t.at(1.98, frontV, p.posts), t.at(0.02, frontV, p.posts), t.pentShade(p, m))
	})
	t.lintel(p)
}

func (t stall) pentShade(p stallPlan, m mat) shader {
	dz := p.eave - p.posts
	switch t.face {
	case "sw":
		return t.planeX(frontV, eaveV-frontV, p.posts, dz, litFront, m)
	case "ne":
		return t.planeX(1-frontV, frontV-eaveV, p.posts, dz, litBack, m)
	case "se":
		return t.planeY(frontV, eaveV-frontV, p.posts, dz, litEnd, m)
	}
	return t.planeY(1-frontV, frontV-eaveV, p.posts, dz, litBack, m)
}

func (t stall) lintel(p stallPlan) {
	f := frontAt(frontV)
	if !t.sees(f) {
		return
	}
	t.pen(matStone, func() {
		t.panel(f, 0.02, 1.98, p.posts-3, p.posts, t.stoneOn(f))
		t.panel(f, 0.7, 1.3, p.posts-4, p.posts-2, flat(stoneR[3]))
	})
}

func (t stall) stoneOn(f plane) shader {
	nx, _ := t.dir(f.du, f.dv)
	ax, ay, _, _ := t.span(f, 0, 2)
	if nx > 0.5 {
		return t.rightMat(ax, ashlarMat(stoneR, t.lk.seed+8))
	}
	return t.leftMat(ay, ashlarMat(stoneR, t.lk.seed+8))
}

func (t stall) portico(p stallPlan) {
	m := marbleMat(t.lk.seed + 9)
	top := p.posts + 4
	t.velarium(p)
	t.slab(0.02, 0.82, 1.98, 1.0, p.posts, top-1, m, m, matStone)
	cloth := t.cloth()
	t.slab(0.02, 0.36, 1.98, 1.0, top-1, top, m, stripeMat(cloth, mixRamp(cloth, marbleR, 0.55), 5), matStone)
	t.pen(matTrim, func() {
		t.panel(frontAt(1.0), 0.02, 1.98, p.posts+1, p.posts+2, flat(goldR[2]))
	})
}

func (t stall) velarium(p stallPlan) {
	f := frontAt(0.9)
	if !t.sees(f) {
		return
	}
	cloth := t.cloth()
	for i := 0; i < 6; i++ {
		a := 0.2 + float64(i)*0.267
		drop := 4.0 + float64(i%2)
		col := cloth[2]
		if i%2 == 1 {
			col = cloth[1]
		}
		t.pen(matTrim, func() { t.panel(f, a, a+0.267, p.posts-drop, p.posts, flat(col)) })
	}
}
