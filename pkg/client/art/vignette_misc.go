package art

import (
	"fmt"
	"image/color"
	"math"
)

func init() {
	vgRegister(SceneAgeUp, 0, vgAgeUp)
	vgRegister(SceneReforge, 10, vgReforge)
	vgRegister(SceneIncident, 0, vgIncident)
}

func vgAgeUp(p *vgPaint) {
	p.sky(vgDawnTop, vgDawnLow, 50)
	p.sun(float64(p.cx+62), 46, 11, goldR[3], orangeR[3])
	p.rays(p.cx+62, 46)
	p.hills(46, 8, rgb(120, 104, 120))
	p.land(52, vgRampTone(grassR, orangeR[2], 0.12), mix(vgDawnLow, vgFarGreen, 0.5))
	cx, gy, age := p.cx, 80, p.argAge()
	warm := vgSilhouette(orangeR[3], 0.12)
	p.putTone(vgKeep(age), cx-6, gy-6, warm)
	for i, x := range []int{cx - 96, cx - 70, cx + 44, cx + 100} {
		p.putTone(vgAgedHut(1+(i+p.intn(3))%3, age, i), x, gy-4+i%2*4, warm)
	}
	for i := 0; i < 3; i++ {
		p.put(fmt.Sprintf("walker/gull/walk/%s/%d", []string{"se", "sw"}[i%2], p.cyc(4)), cx-60+i*50+p.jit(6), 20+i*5+p.bob(2, float64(i)))
	}
	p.pennantAt(cx-40, gy+2, redR[2], 2)
	p.big(vgWalker("herald", "", "idle", "se", p.cyc(12)/6), cx-52, gy+2)
	p.put(vgFx("sparkle", p.cyc(6)), cx+62, 40)
}

func (p *vgPaint) rays(x, y int) {
	col := color.NRGBA{255, 230, 160, 46}
	for i := -4; i <= 4; i++ {
		a := float64(i) * 0.32
		for r := 16; r < 90; r++ {
			rx := x + int(float64(r)*math.Sin(a))
			ry := y - int(float64(r)*math.Cos(a))
			if (r+p.cyc(12))%6 < 4 {
				p.c.blend(rx, ry, col)
			}
		}
	}
}

func vgReforge(p *vgPaint) {
	p.sky(rgb(70, 60, 110), rgb(250, 200, 120), 36)
	p.sun(float64(p.cx+60), 34, 9, goldR[3], goldR[2])
	p.sea(36, 84, rgb(206, 164, 110), rgb(84, 96, 130))
	cx := p.cx
	p.isleAt(cx-80, 60, 34, vgSilhouette(rgb(120, 96, 90), 0.5))
	p.putTone(vgKeep(p.argAge()), cx-80, 58, vgSilhouette(rgb(110, 86, 90), 0.5))
	p.isleAt(cx+76, 54, 30, nil)
	p.put(vgDecor("tree-pine", p.intn(3)), cx+66, 52)
	p.put(vgDecor("tree-oak", p.intn(3)), cx+86, 52)
	p.thread(cx-60, 50, cx+6, 60)
	p.put(fmt.Sprintf("fx/wake/%d", p.cyc(6)), cx+2, 68)
	p.put(vgShip("galleon", p.intn(8), "ne", p.cyc(4)), cx+10, 70)
	p.put(vgFx("sparkle", p.cyc(6)), cx+76, 28)
}

func (p *vgPaint) isleAt(x, y, rx int, tone func(color.NRGBA) color.NRGBA) {
	s := p.stamp(2*rx+4, 20, rx+2, 10, func(c *canvas) {
		q := &vgPaint{c: c, frame: p.frame, w: c.w, h: c.h}
		q.isle(rx+2, 10, rx)
	})
	p.putSprite(s, x, y, vgPut{tone: tone})
}

func (p *vgPaint) thread(x0, y0, x1, y1 int) {
	for i := 0; i <= 40; i++ {
		t := float64(i) / 40
		x := x0 + int(float64(x1-x0)*t)
		y := y0 + int(float64(y1-y0)*t) - int(10*t*(1-t)*4)
		col := goldR[2]
		if (i+p.cyc(12))%4 == 0 {
			col = goldR[3]
		}
		p.c.dot(x, y, col)
	}
}

func vgIncident(p *vgPaint) {
	p.outdoors(vgDuskTop, vgDuskLow, vgFarBlue, grassR)
	p.treeLine(62, nil)
	cx, gy := p.cx, 82
	p.put(vgHut(2, "ready", ""), cx+24, gy-10)
	p.put(vgFx("alarm", p.cyc(6)), cx+24, gy-52)
	p.big(vgWalker("reeve", "", "idle", "sw", p.cyc(12)/6), cx-30, gy)
	p.bubbleAt(cx-30, gy-50, vgGlyphEx, vgSeal, 2)
}
