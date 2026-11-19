package art

import (
	"fmt"
	"image/color"
	"strings"
)

var vgClasses = []string{"cog", "caravel", "galleon", "carrack"}

func init() {
	vgRegister(SceneNoDepot, -10, vgNoDepot)
	vgRegister(SceneNoCrew, 0, vgNoCrew)
	vgRegister(SceneSpoiling, -6, vgSpoiling)
	vgRegister(SceneQuayFull, 20, vgQuayFull)
	vgRegister(SceneRushTide, 0, vgRushTide)
	vgRegister(SceneContract, 0, vgContract)
}

func (p *vgPaint) harbor(top, low color.NRGBA, quay int) {
	p.sky(top, low, 34)
	p.clouds(3, 2, 18, mix(vgCloud, low, 0.2), mix(vgCloudSh, top, 0.2))
	p.sea(34, quay, mix(waterR[2], low, 0.3), waterR[1])
	p.planks(quay, p.h)
}

func (p *vgPaint) argKind() string {
	for _, part := range strings.Split(p.arg, ":") {
		for _, k := range vgKinds {
			if part == k {
				return k
			}
		}
	}
	return p.pick(vgKinds)
}

func (p *vgPaint) argHas(flag string) bool {
	for _, part := range strings.Split(p.arg, ":") {
		if part == flag {
			return true
		}
	}
	return false
}

func vgNoDepot(p *vgPaint) {
	p.harbor(vgNoonTop, vgNoonLow, 64)
	cx, gy := p.cx, 82
	kind := p.argKind()
	p.put(vgShip(p.pick(vgClasses), p.intn(8), "sw", p.cyc(4)), cx+74, 52)
	p.put(fmt.Sprintf("boxpile/%s/3", kind), cx-8, gy)
	p.big(vgBox(1, kind, p.intn(4), true), cx+18, gy+2)
	p.stink(cx-8, gy-50)
	p.flies(cx-6, gy-44, 3)
	p.signAt(cx+46, gy, 2)
	p.big(vgWalker("porter", p.pick(vgLooks7), "idle", "se", p.cyc(12)/6), cx-52, gy)
	p.bubbleAt(cx-52, gy-50, vgGlyphQ, vgInk, 2)
}

func vgNoCrew(p *vgPaint) {
	p.outdoors(vgOvercTo, vgOvercLo, vgFarGreen, grassR)
	p.plaza(70, 84)
	cx, gy := p.cx, 84
	kind := p.argKind()
	p.put(BuildingLookName(StallArtKind(StallForBox(kind), "sw"), "dark", p.argAge(), 0), cx+8, gy+2)
	p.cobweb(cx-14, 36)
	p.cobweb(cx-6, 44)
	for i, x := range []int{cx - 70, cx - 48, cx + 56, cx + 76} {
		p.big(vgBox(i%2, kind, p.intn(4), false), x, gy-2+i%2*2)
	}
	p.fallenPennant(cx-34, gy-4, vgMarks[p.intn(len(vgMarks))])
	p.bubbleAt(cx+2, 30, vgGlyphQ, vgInk, 2)
	p.put("fx/spoil/"+[]string{"0", "1", "2", "3", "4", "5"}[p.cyc(6)], cx-60, gy-26)
	p.put("fx/spoil/"+[]string{"0", "1", "2", "3", "4", "5"}[(p.cyc(6)+3)%6], cx+66, gy-26)
}

func (p *vgPaint) cobweb(x, y int) {
	col := color.NRGBA{230, 230, 236, 170}
	for i := 0; i < 7; i++ {
		p.c.blend(x+i, y, col)
		p.c.blend(x, y+i, col)
		p.c.blend(x+i, y+i/2+3-i/2, col)
	}
	p.c.blend(x+2, y+2, col)
	p.c.blend(x+4, y+1, col)
	p.c.blend(x+1, y+4, col)
}

func vgSpoiling(p *vgPaint) {
	p.harbor(rgb(120, 132, 110), rgb(214, 206, 170), 62)
	cx, gy := p.cx, 82
	p.put(vgShip(p.pick(vgClasses), p.intn(8), "se", p.cyc(4)), cx-80, 54)
	for i, x := range []int{cx - 30, cx - 4, cx + 22, cx - 16, cx + 8} {
		p.putWith(vgBox(1, "crate", p.intn(4), i%2 == 1), x, gy-i/3*16, vgPut{scale: 2, tone: vgRotten})
	}
	p.big("fx/spoil/"+[]string{"0", "1", "2", "3", "4", "5"}[p.cyc(6)], cx-18, gy-40)
	p.big("fx/spoil/"+[]string{"0", "1", "2", "3", "4", "5"}[(p.cyc(6)+3)%6], cx+14, gy-42)
	p.flies(cx-4, gy-50, 4)
	p.big("haven/crates/0", cx+66, gy)
	p.hourglassAt(cx+66, gy-32, 2)
	p.big(vgWalker("porter", p.pick(vgLooks7), "carry", "se", p.cyc(12)/3), cx-70, gy)
	p.sweat(cx-82, gy-46)
}

func vgQuayFull(p *vgPaint) {
	p.harbor(vgNoonTop, vgNoonLow, 62)
	cx, gy := p.cx, 84
	for i, x := range []int{cx - 90, cx - 50, cx + 60} {
		p.put(fmt.Sprintf("boxpile/%s/3", vgKinds[(i+p.intn(4))%4]), x, gy-4)
	}
	p.put(fmt.Sprintf("fx/wake/%d", p.cyc(6)), cx+34, 52)
	p.put(vgShip(p.pick(vgClasses), p.intn(8), "ne", p.cyc(4)), cx+30, 50)
	p.put(vgShip(p.pick(vgClasses), p.intn(8), "nw", p.cyc(4)), cx-14, 44)
	p.big("ambient/ward/0", cx-22, gy-2)
	p.big(vgWalker("dockhand", p.pick(vgLooks3), "idle", "sw", p.cyc(12)/6), cx+6, gy)
	p.armAt(cx+14, gy-34, 1, redR[1], 2)
	p.bubbleAt(cx+6, gy-52, vgGlyphNo, vgSeal, 2)
}

func vgRushTide(p *vgPaint) {
	p.harbor(vgDuskTop, vgDuskLow, 66)
	for i, x := 0, 6; x < p.w-6; i, x = i+1, x+14+p.bgIntn(8) {
		name := vgShip(vgClasses[p.bgIntn(2)], p.bgIntn(8), "sw", p.cyc(4)+i)
		p.putTone(name, x, 40+p.bgIntn(4)+p.bob(1, float64(i)), vgSilhouette(vgDuskTop, 0.55))
	}
	for i, x := 0, p.cx-100; x < p.cx+110; i, x = i+1, x+50 {
		p.put(vgShip(vgClasses[2+p.bgIntn(2)], p.intn(8), "sw", p.cyc(4)+i), x+p.bgIntn(9)-4, 56+p.bob(1, float64(i)))
	}
	cx, gy := p.cx, 84
	p.big(vgWalker("herald", "", "work", "se", p.cyc(4)), cx-58, gy)
	p.big(fmt.Sprintf("fx/horn/%d", p.cyc(4)), cx-38, gy-30)
}

func vgContract(p *vgPaint) {
	p.harbor(vgNoonTop, vgNoonLow, 64)
	cx, gy := p.cx, 82
	kind := p.argKind()
	p.put(vgShip(p.pick([]string{"cog", "caravel"}), p.intn(8), "sw", p.cyc(4)), cx+44, 60)
	if p.argHas("crown") {
		p.pennantAt(cx+76, gy, goldR[2], 2)
		p.put("icon/priority", cx+18, gy-40)
	}
	p.big(vgBox(1, kind, p.intn(4), false), cx+20, gy)
	if p.argHas("sealed") {
		p.put("icon/sealed", cx+30, gy-22)
	}
	p.big(vgWalker("merchant", p.pick(vgLooks3), "idle", "se", p.cyc(12)/6), cx-20, gy)
	p.scrollAt(cx-6, gy-28+p.bob(1, 0), 2)
}
