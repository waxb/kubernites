package art

func init() {
	vgRegister(SceneEagerWatchman, 0, vgEagerWatchman)
}

func vgEagerWatchman(p *vgPaint) {
	p.outdoors(vgDuskTop, vgDuskLow, vgFarBlue, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 84
	p.shadow(cx+30, gy-2, 30)
	p.big(vgHut(2, "ready", ""), cx+30, gy)
	knocks := 1 + p.cyc(9)/3
	p.putSprite(p.stamp(MarkTagW, MarkTagH, MarkTagAX, MarkTagAY, func(c *canvas) { paintKnocks(c, knocks) }), cx+30, gy-64, vgPut{scale: 2})
	p.shadow(cx+2, gy-2, 16)
	p.big(vgWalker("porter", p.pick(vgLooks7), "carry", "sw", p.cyc(12)/3), cx+2, gy)
	p.shadow(cx-50, gy-2, 18)
	p.big(vgWalker("reeve", "", "work", "se", p.cyc(4)/2), cx-50, gy-4)
	p.torchAt(cx-40, gy-26, 2)
	p.burst(cx-16, gy-40)
}
