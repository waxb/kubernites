package art

import "image/color"

var (
	vgDryR    = ramp{rgb(150, 112, 70), rgb(186, 146, 92), rgb(208, 172, 110), rgb(230, 200, 140)}
	vgMarks   = []color.NRGBA{rgb(208, 64, 60), rgb(70, 120, 200), rgb(90, 170, 90), rgb(220, 170, 40)}
	vgOvercTo = rgb(108, 116, 134)
	vgOvercLo = rgb(190, 190, 186)
)

func init() {
	vgRegister(SceneDrought, -10, vgDrought)
	vgRegister(SceneAdoption, -16, vgAdoption)
	vgRegister(SceneDiskPressure, -10, vgDiskPressure)
	vgRegister(SceneFloorLimit, 0, vgFloorLimit)
	vgRegister(SceneNoSpace, 0, vgNoSpace)
	vgRegister(ScenePreempted, 0, vgPreempted)
}

func vgDrought(p *vgPaint) {
	p.sky(vgHazeTop, vgHazeLow, 46)
	p.sun(float64(p.cx+74), 14, 8, goldR[3], orangeR[2])
	p.hills(44, 7, rgb(176, 146, 100))
	p.land(48, vgDryR, vgHazeLow)
	p.treeLine(62, vgWithered)
	cx, gy := p.cx, 82
	p.cracks(cx-20, gy-10, 8)
	p.putTone(vgDecor("tree-oak", p.intn(3)), cx-88, 66, vgWithered)
	p.putTone(vgDecor("tree-maple", p.intn(3)), cx+92, 64, vgWithered)
	p.big(vgDecor("log", 0), cx+62, gy-4)
	slow := p.cyc(12) / 3
	kind := p.pick(vgKinds)
	p.shimmer(cx-100, 40, 200)
	p.big(vgWalker("porter", p.pick(vgLooks7), "carry", "sw", slow), cx-30, gy)
	p.sweat(cx-42, gy-44)
	p.big(vgWalker("porter", p.pick(vgLooks7), "carry", "sw", slow+2), cx+16, gy)
	p.sweat(cx+4, gy-44)
	_ = kind
}

func vgAdoption(p *vgPaint) {
	p.outdoors(vgNoonTop, vgNoonLow, vgFarGreen, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 82
	mark := vgMarks[p.intn(len(vgMarks))]
	p.put(vgHut(1, "ready", ""), cx-84, 62)
	p.put(vgHut(1, "ready", ""), cx-54, 58)
	p.pennantAt(cx+66, gy, mark, 2)
	p.big(vgWalker("helmbearer", "", "work", "sw", p.cyc(4)/2), cx+44, gy)
	walk := p.cyc(4)
	for i, x := range []int{cx + 10, cx - 20, cx - 50} {
		p.big(vgWalker("porter", p.pick(vgLooks7), "walk", "se", walk+i), x, gy)
		p.markAt(x, gy-54, mark, 2)
	}
	p.rope(cx+38, gy-24, cx-50, gy-22, 4)
	p.markAt(cx+84, gy-46, mark, 2)
}

func vgDiskPressure(p *vgPaint) {
	p.sky(vgOvercTo, vgOvercLo, 44)
	p.clouds(4, 2, 20, vgCloudSh, vgStormCl)
	p.land(44, vgRampTone(grassR, vgOvercLo, 0.25), vgOvercLo)
	cx, gy := p.cx, 82
	p.put(vgKeep(p.argAge()), cx-24, gy-4)
	p.put("midden/3/0", cx-50, gy)
	p.put("midden/3/1", cx-6, gy+2)
	p.put("midden/2/"+[]string{"0", "1"}[p.intn(2)], cx+28, gy)
	p.flies(cx-48, gy-40, 3)
	p.flies(cx-4, gy-38, 3)
	p.stink(cx+26, gy-22)
	p.put(vgHut(1, "banished", ""), cx+72, gy-8)
	p.big(vgWalker("reeve", "", "idle", "sw", p.cyc(12)/6), cx+58, gy+2)
	p.armAt(cx+50, gy-24, -1, blueR[1], 2)
}

func vgFloorLimit(p *vgPaint) {
	p.outdoors(vgOvercTo, vgOvercLo, vgFarGreen, grassR)
	p.plaza(56, 84)
	cx, gy := p.cx, 80
	p.diamond(cx-26, gy-6, 54, 14, false)
	p.big(vgHut(2, "ready", ""), cx-30, gy-8)
	p.put("codexbloat/3", cx-24, gy+22)
	p.put("midden/1/0", cx-58, gy+2)
	p.diamond(cx-26, gy-6, 54, 14, true)
	p.fallingScroll(cx-34, 4, 26, 0)
	p.fallingScroll(cx-20, 8, 26, 9)
	p.big(vgWalker("reeve", "", "idle", "sw", p.cyc(12)/6), cx+62, gy)
	p.armAt(cx+54, gy-24, -1, blueR[1], 2)
}

func vgNoSpace(p *vgPaint) {
	p.sky(vgNightTop, rgb(80, 64, 96), 46)
	p.starry(24)
	p.hills(46, 8, mix(vgFarBlue, vgNightTop, 0.5))
	p.plaza(56, 84)
	p.softGlow(p.cx+14, 50, 80, 40, color.NRGBA{255, 200, 120, 70})
	cx, gy := p.cx, 84
	p.put(BuildingLookName("codex", "ready", p.argAge(), 0), cx+30, gy+8)
	p.put("codexbloat/3", cx+14, gy+10)
	for i := 0; i < 3; i++ {
		p.fallingScroll(cx-10+i*22, 2+i*3, 44, i*7)
	}
	p.big(vgWalker("clerk", p.pick(vgLooks7), "idle", "se", p.cyc(12)/6), cx-66, gy)
	p.scroll(cx-44, gy-26, true)
	p.bubbleAt(cx-66, gy-50, vgGlyphNo, vgSeal, 2)
}

func vgPreempted(p *vgPaint) {
	p.outdoors(vgNoonTop, vgNoonLow, vgFarGreen, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 82
	p.put(vgHut(3, "kindling", "landed"), cx+86, 64)
	p.put(vgHut(1, "ready", "vagrant"), cx+8, 64)
	p.big(vgWalker("wanderer", p.pick(vgLooks7), "carry", "sw", p.cyc(4)), cx-44, gy)
	p.big("ambient/bundle/0", cx-60, gy-2)
	p.pennantAt(cx+70, gy-2, goldR[2], 2)
	p.big(vgWalker("chancellor", "", "idle", "sw", p.cyc(12)/6), cx+48, gy)
	p.armAt(cx+40, gy-26, -1, purpleR[1], 2)
	p.big("icon/rank-high", cx+48, gy-50)
}
