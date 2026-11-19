package art

import (
	"image/color"
	"math"
)

func init() {
	vgRegister(SceneOOMKilled, -4, vgOOMKilled)
	vgRegister(SceneCrashLoop, 0, vgCrashLoop)
	vgRegister(SceneImagePull, -30, vgImagePull)
	vgRegister(SceneEviction, -8, vgEviction)
	vgRegister(SceneStorm, 16, vgStorm)
	vgRegister(SceneTremor, 4, vgTremor)
	vgRegister(SceneUnschedulable, 0, vgUnschedulable)
}

func vgRampTone(r ramp, to color.NRGBA, t float64) ramp {
	return ramp{mix(r[0], to, t), mix(r[1], to, t), mix(r[2], to, t), mix(r[3], to, t)}
}

func vgOOMKilled(p *vgPaint) {
	p.outdoors(vgDuskTop, vgDuskLow, vgFarBlue, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 80
	kind := p.pick(vgKinds)
	p.shadow(cx+52, gy-2, 26)
	p.big(vgBox(2, kind, p.intn(4), false), cx+54, gy-4)
	p.softGlow(cx+6, gy-16, 56, 34, color.NRGBA{170, 100, 255, 190})
	p.sack(cx+6, gy-2, 2)
	p.burst(cx+10, gy-26)
	p.spill(cx-16, gy+3, 5, 3)
	p.motes(cx+6, gy-30, voidR)
	look := p.pick(vgLooks7)
	p.shadow(cx-50, gy-2, 22)
	p.putWith(vgWalker("porter", look, "idle", "se", 0), cx-32, gy-8, vgPut{angle: -math.Pi / 2, scale: 2})
	p.stars(cx-68, gy-24)
}

func vgCrashLoop(p *vgPaint) {
	night := vgRampTone(grassR, vgNightTop, 0.45)
	p.sky(vgNightTop, vgNightLow, 46)
	p.starry(40)
	p.hills(44, 8, mix(vgFarBlue, vgNightTop, 0.5))
	p.land(48, night, vgNightLow)
	p.treeLine(62, vgSilhouette(vgNightTop, 0.45))
	cx, gy := p.cx, 86
	tier := 1 + p.intn(2)
	step := p.cyc(12)
	if step < 7 {
		p.softGlow(cx+28, gy-44, 70, 44, color.NRGBA{255, 150, 60, 110})
		p.big(vgHut(tier, "guttering", ""), cx+28, gy)
		p.big(vgFx("fire", p.cyc(6)), cx+18, gy-58)
		p.big(vgFx("fire", p.cyc(6)+3), cx+40, gy-50)
	} else {
		p.putWith(vgHut(tier, "ready", ""), cx+28, gy, vgPut{scale: 2, tone: vgCharred})
	}
	p.big(vgFx("smoke", p.cyc(6)), cx+26, gy-80)
	p.tally(cx-70, gy-6)
	p.big(vgWalker("reeve", "", "work", "se", p.cyc(4)/2), cx-28, gy-4)
	p.torchAt(cx-18, gy-26, 2)
	p.sweat(cx-40, gy-50)
}

func (p *vgPaint) starry(n int) {
	for i := 0; i < n; i++ {
		x, y := int(hash(i, 1, 99)%uint32(p.w)), int(hash(i, 2, 99)%34)
		col := rgb(200, 210, 240)
		if (i+p.cyc(12)/3)%5 == 0 {
			col = white
		}
		p.c.dot(x, y, col)
	}
}

func (p *vgPaint) pier(x0, y0, y1 int) {
	for y := y0; y < y1; y++ {
		for x := max(x0, 0); x < p.w; x++ {
			col := woodR[2]
			switch {
			case (x-x0)%6 == 5:
				col = woodR[0]
			case y == y0:
				col = woodR[3]
			case hash(x, y, 21)%7 == 0:
				col = woodR[1]
			}
			p.c.dot(x, y, col)
		}
	}
	p.c.rect(max(x0, 0), y1, p.w, 1, woodR[0])
	for x := x0 + 2; x < p.w; x += 16 {
		p.c.rect(x, y1+1, 3, 8, woodR[1])
		p.c.rect(x, y1+1, 1, 8, woodR[0])
	}
}

func vgImagePull(p *vgPaint) {
	p.sky(vgFogTop, vgFogLow, 36)
	p.sea(36, 84, rgb(118, 142, 150), rgb(70, 104, 124))
	cx := p.cx
	class := p.pick([]string{"caravel", "cog"})
	p.putWith(vgShip(class, p.intn(8), "sw", p.cyc(4)), cx-64+p.bob(1, 0), 50, vgPut{tone: vgSilhouette(vgFogLow, 0.55), alpha: 0.7})
	p.fog(24, 62, vgFogLow, 0.85)
	p.pier(cx-8, 70, 76)
	p.big("haven/bollard/0", cx, 74)
	p.big("ambient/lamp/"+[]string{"0", "1"}[p.cyc(12)/6], cx+48, 74)
	p.big(vgWalker(p.pick([]string{"pilgrim", "wanderer"}), p.pick(vgLooks7), "idle", "sw", p.cyc(12)/6), cx+24, 74)
	p.bubbleAt(cx+24, 26, vgGlyphQ, vgInk, 2)
	p.fog(60, 84, vgFogLow, 0.3)
}

func vgEviction(p *vgPaint) {
	p.outdoors(rgb(112, 124, 150), rgb(206, 204, 196), vgFarGreen, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 82
	p.path(cx+50, 66)
	p.put(vgHut(1+p.intn(3), "ready", ""), cx+62, 66)
	p.scroll(cx+60, 50, true)
	walk := p.cyc(4)
	p.put(vgFx("dust", p.cyc(6)), cx-4, gy+2)
	p.put(vgFx("dust", p.cyc(6)+3), cx-56, gy+2)
	p.big(vgWalker("pilgrim", p.pick(vgLooks7), "carry", "sw", walk+2), cx-64, gy)
	p.big(vgWalker("child", p.pick(vgLooks3), "walk", "sw", walk+1), cx-38, gy+1)
	p.big(vgWalker("farmer", p.pick(vgLooks3), "carry", "sw", walk), cx-10, gy)
	p.big(vgWalker("reeve", "", "idle", "sw", p.cyc(12)/6), cx+36, gy)
	p.armAt(cx+28, gy-26, -1, blueR[1], 2)
}

func (p *vgPaint) path(x, y int) {
	for yy := y; yy < p.h; yy++ {
		t := float64(yy-y) / float64(max(p.h-y, 1))
		half := 3 + int(t*18)
		mid := x - int(t*float64(x-p.cx+40))
		for xx := mid - half; xx <= mid+half; xx++ {
			p.c.dot(xx, yy, speckle(vgDryR, xx, yy, 5, 0.1))
		}
	}
}

func vgStorm(p *vgPaint) {
	p.sky(vgStormTop, vgStormLow, 34)
	p.clouds(6, 0, 16, vgStormCl, vgStormSh)
	p.sea(34, 84, rgb(52, 70, 86), rgb(30, 50, 70))
	cx := p.cx
	p.isle(cx+50, 52, 30)
	p.put(vgHut(1+p.intn(3), "ready", ""), cx+48, 52)
	p.put(vgWalker("porter", p.pick(vgLooks7), "idle", "sw", 0), cx+66, 54)
	p.shore(cx-40, 64)
	p.big(vgWalker("herald", "", "idle", "se", 0), cx-66, 82)
	p.rope(cx-42, 66, cx-16, 80, 2)
	p.rope(cx+22, 54, cx+6, 66, 1)
	flash := p.cyc(12) == 0 || p.cyc(12) == 2
	p.bolt(cx+6, 0, cx+10, 58, flash)
	p.rain(color.NRGBA{190, 200, 220, 120}, 60)
	if flash {
		p.tint(vgBolt, 0.12)
	}
}

func (p *vgPaint) isle(x, y, rx int) {
	p.c.ellipse(float64(x), float64(y+2), float64(rx), 7, sandR[1])
	p.c.ellipse(float64(x), float64(y), float64(rx-3), 5, sandR[2])
	p.c.ellipse(float64(x), float64(y-1), float64(rx-6), 4, grassR[1])
	p.c.ellipse(float64(x-3), float64(y-2), float64(rx-12), 2, grassR[2])
	foam := p.cyc(12) / 3
	for i := -rx; i <= rx; i += 3 {
		if (i/3+foam)%4 == 0 {
			p.c.dot(x+i, y+8, vgCloud)
		}
	}
}

func (p *vgPaint) shore(x1, y int) {
	for yy := y; yy < p.h; yy++ {
		edge := x1 - (yy-y)/2 + int(math.Round(1.5*math.Sin(float64(yy)*0.7)))
		for xx := 0; xx < edge; xx++ {
			col := speckle(grassR, xx, yy, 7, 0.08)
			if xx > edge-4 {
				col = sandR[2]
			}
			p.c.dot(xx, yy, col)
		}
	}
}

func vgTremor(p *vgPaint) {
	p.outdoors(vgNoonTop, vgNoonLow, vgFarGreen, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 86
	p.cracks(cx+14, gy-6, 6)
	tilt := 0.30 + 0.05*math.Sin(p.phase())
	p.putWith(vgHut(1+p.intn(2), "ready", ""), cx+22, gy, vgPut{angle: tilt, scale: 2})
	p.big(vgFx("dust", p.cyc(6)), cx-4, gy-2)
	p.big(vgFx("dust", p.cyc(6)+3), cx+56, gy-2)
	p.c.rect(cx+60, gy-8, 10, 3, woodR[1])
	p.c.rect(cx-14, gy-6, 8, 3, woodR[2])
	p.big(vgWalker("imp", "", "work", "se", p.cyc(4)), cx-24, gy-2)
	p.bubbleAt(cx-26, gy-50, vgGlyphEx, vgSeal, 2)
}

func vgUnschedulable(p *vgPaint) {
	p.sky(vgNoonTop, vgNoonLow, 40)
	p.clouds(3, 4, 20, vgCloud, vgCloudSh)
	p.land(38, grassR, vgNoonLow)
	for x, i := p.cx-150, 0; x <= p.cx+150; x, i = x+30, i+1 {
		p.put(vgHut(1+p.bgIntn(3), "ready", []string{"", "freeman", "landed"}[p.bgIntn(3)]), x+p.bgIntn(5)-2, 52)
		p.put("border/stone/"+[]string{"0", "1", "2"}[i%3], x+15, 54)
	}
	p.plaza(56, 84)
	cx, gy := p.cx, 82
	p.put(vgHut(1, "pending", ""), cx-20, 68)
	p.big(vgWalker("pilgrim", p.pick(vgLooks7), "idle", "se", p.cyc(12)/6), cx-58, gy)
	p.big("ambient/bundle/0", cx+2, gy-2)
	p.big("ambient/bundle/0", cx+14, gy+1)
	p.big(vgBox(0, p.pick(vgKinds), p.intn(4), false), cx-18, gy)
	p.big(vgWalker("quartermaster", p.pick(vgLooks7), "idle", "sw", p.cyc(12)/6), cx+50, gy)
	p.bubbleAt(cx+50, gy-50, vgGlyphQ, vgInk, 2)
}
