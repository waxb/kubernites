package art

import (
	"fmt"
	"image/color"
)

const (
	BellFrames      = 2
	HourglassFrames = 4
	HourglassW      = 11
	HourglassH      = 15
	KnockMax        = 3
)

type hutDoor struct {
	face, u0, u1, z0, z1 float64
}

func doorOf(qos string, age int) hutDoor {
	switch qos {
	case "vagrant":
		return hutDoor{shackY1, doorU0, doorU1, 0, doorH}
	case "landed":
		return hutDoor{manorY1, gateU0 + 0.02, gateU1 - 0.02, 1.5, 9.5}
	}
	sill := hutCotSills[min(max(age, 0), AgeTiers-1)]
	return hutDoor{cotY1, 0.28, 0.44, sill, sill + 8}
}

type hutOpening struct {
	f              side
	u0, u1, z0, z1 float64
}

func windowsOf(qos string, age int) []hutOpening {
	switch qos {
	case "vagrant":
		return []hutOpening{{sideR(shackX1), 0.49, 0.65, 4, 9}}
	case "landed":
		low, up := hutManorFloors[age], manorY1+manorJetty*float64(boolInt(age == ageFeudal))
		return []hutOpening{
			{sideL(manorY1), 0.46, 0.62, 3, 9}, {sideL(up), 0.14, 0.3, low + 2, low + 8}, {sideL(up), 0.48, 0.64, low + 2, low + 8},
			{sideR(manorX1), 0.28, 0.44, 3, 9}, {sideR(manorX1), 0.22, 0.44, low + 2, low + 8},
		}
	}
	sill := hutCotSills[age]
	return []hutOpening{{sideL(cotY1), 0.5, 0.74, sill + 2, sill + 8}, {sideR(cotX1), 0.38, 0.54, sill + 2, sill + 8}}
}

func ShutterName(qos string, age int, open bool) string {
	st := "shut"
	if open {
		st = "open"
	}
	return fmt.Sprintf("probe/shutter/%s/%d/%s", qos, age, st)
}

func BellName(qos string, age, frame int) string {
	return fmt.Sprintf("probe/bell/%s/%d/%d", qos, age, ((frame%BellFrames)+BellFrames)%BellFrames)
}

func HourglassName(frame int) string {
	return fmt.Sprintf("probe/hourglass/%d", ((frame%HourglassFrames)+HourglassFrames)%HourglassFrames)
}

func KnockName(n int) string {
	return fmt.Sprintf("probe/knock/%d", min(max(n, 1), KnockMax))
}

func FrostName() string { return "probe/frost" }

func addProbeArt(b *atlasBuilder) {
	s := hearthSize()
	for _, q := range HearthQoS {
		for age := 0; age < AgeTiers; age++ {
			d, wins := doorOf(q, age), windowsOf(q, age)
			for _, open := range []bool{false, true} {
				op := open
				b.add(ShutterName(q, age, open), s.w, s.h, s.ax, s.ay, func(c *canvas) { inked(c, func(c *canvas) { paintShutters(hearthFrame(c, s), d, wins, op) }) })
			}
			for f := 0; f < BellFrames; f++ {
				fr := f
				b.add(BellName(q, age, f), s.w, s.h, s.ax, s.ay, func(c *canvas) { inked(c, func(c *canvas) { paintBell(hearthFrame(c, s), d, fr) }) })
			}
		}
	}
	for f := 0; f < HourglassFrames; f++ {
		fr := f
		b.add(HourglassName(f), HourglassW, HourglassH, HourglassW/2, HourglassH-1, padded(func(c *canvas) { inked(c, func(c *canvas) { paintHourglass(c, fr) }) }))
	}
	for n := 1; n <= KnockMax; n++ {
		count := n
		b.add(KnockName(n), MarkTagW, MarkTagH, MarkTagAX, MarkTagAY, padded(func(c *canvas) { paintKnocks(c, count) }))
	}
	b.add(FrostName(), MarkTagW, MarkTagH, MarkTagAX, MarkTagAY, padded(paintFrost))
}

func paintFrost(c *canvas) {
	cx, cy := c.w/2, 5
	for _, d := range [][2]int{{-3, 0}, {3, 0}, {0, -3}, {0, 3}, {-2, -2}, {2, 2}, {2, -2}, {-2, 2}} {
		c.line(cx, cy, cx+d[0], cy+d[1], glassR[3])
	}
	for _, d := range [][2]int{{-3, 0}, {3, 0}, {0, -3}, {0, 3}} {
		c.dot(cx+d[0], cy+d[1], glassR[2])
	}
	c.dot(cx, cy, cream)
	c.rect(tagAX, tagAY-3, 1, 3, cream)
	c.outline(ink)
}

func paintShutters(s iso, d hutDoor, wins []hutOpening, open bool) {
	for _, w := range wins {
		windowShutter(s, w, open)
	}
	paintShutter(s, d, open)
}

func windowShutter(s iso, w hutOpening, open bool) {
	if !open {
		s.on(w.f, w.u0, w.u1, w.z0, w.z1, boardedPlanks)
		s.on(w.f, w.u0, w.u1, (w.z0+w.z1)/2, (w.z0+w.z1)/2+1, flat(woodR[0]))
		return
	}
	s.on(w.f, w.u0-0.07, w.u0-0.01, w.z0, w.z1, shutterPlanks)
	s.on(w.f, w.u1+0.01, w.u1+0.07, w.z0, w.z1, shutterPlanks)
}

func paintShutter(s iso, d hutDoor, open bool) {
	w := (d.u1 - d.u0) / 2
	if !open {
		s.onLeft(d.face, d.u0-0.04, d.u1+0.04, d.z0, d.z1+1, boardedPlanks)
		lo, hi := s.p(d.u0-0.02, d.face, d.z0+1), s.p(d.u1+0.02, d.face, d.z1)
		s.c.line(int(lo.x), int(lo.y), int(hi.x), int(hi.y), woodR[0])
		bar0, bar1 := s.p(d.u0-0.05, d.face, (d.z0+d.z1)/2), s.p(d.u1+0.05, d.face, (d.z0+d.z1)/2)
		s.c.line(int(bar0.x), int(bar0.y), int(bar1.x), int(bar1.y), vaultIron[1])
		s.c.line(int(bar0.x), int(bar0.y)-1, int(bar1.x), int(bar1.y)-1, vaultIron[3])
		return
	}
	s.onLeft(d.face, d.u0, d.u1, d.z0, d.z1, doorGlow)
	s.onLeft(d.face, d.u0-w-0.03, d.u0-0.01, d.z0, d.z1+0.5, shutterPlanks)
	s.onLeft(d.face, d.u1+0.01, d.u1+w+0.03, d.z0, d.z1+0.5, shutterPlanks)
}

func shutterPlanks(x, y int) color.NRGBA {
	switch {
	case x%3 == 0:
		return woodR[0]
	case y%5 == 0:
		return woodR[1]
	}
	return woodR[2]
}

func boardedPlanks(x, y int) color.NRGBA {
	if x%3 == 0 {
		return woodR[1]
	}
	return woodR[3]
}

func doorGlow(x, y int) color.NRGBA {
	if (x+y)%4 == 0 {
		return goldR[3]
	}
	return lampLit
}

func paintBell(s iso, d hutDoor, frame int) {
	root := s.p(d.u1+0.08, d.face, d.z1+1)
	x, y := int(root.x), int(root.y)
	s.c.rect(x, y, 4, 1, vaultIron[1])
	s.c.dot(x, y-1, vaultIron[2])
	sway := []int{0, 1}[frame%BellFrames]
	bx := x + 3 + sway
	s.c.dot(bx, y+1, vaultIron[2])
	s.c.rect(bx-1, y+2, 3, 2, goldR[2])
	s.c.rect(bx-2, y+4, 5, 1, goldR[1])
	s.c.dot(bx-1, y+2, goldR[3])
	s.c.dot(bx+sway, y+5, vaultIron[0])
}

func paintHourglass(c *canvas, frame int) {
	w, h := c.w, c.h
	c.rect(0, 0, w, 2, woodR[2])
	c.rect(0, h-2, w, 2, woodR[1])
	c.rect(0, 0, w, 1, woodR[3])
	cx := w / 2
	for y := 2; y < h-2; y++ {
		half := hourglassHalf(y, h)
		c.rect(cx-half, y, 2*half+1, 1, glassR[2])
	}
	for i := 0; i < HourglassFrames-1-frame; i++ {
		sandRow(c, cx, h/2-1-i, h, sandR[2])
	}
	for i := 0; i <= frame; i++ {
		sandRow(c, cx, h-3-i, h, sandR[1])
	}
	if frame < HourglassFrames-1 {
		c.rect(cx, h/2, 1, h/2-2-frame, sandR[3])
	}
	c.rect(cx-hourglassHalf(3, h), 3, 1, 2, glassR[3])
}

func sandRow(c *canvas, cx, y, h int, col color.NRGBA) {
	half := max(hourglassHalf(y, h)-1, 0)
	c.rect(cx-half, y, 2*half+1, 1, col)
}

func hourglassHalf(y, h int) int {
	mid := h / 2
	return max(abs(y-mid)*3/4, 0)
}

func paintKnocks(c *canvas, n int) {
	for i := 0; i < n; i++ {
		cx, cy := 3+i*4, 5
		col := tagGoldEdge
		if i == KnockMax-1 {
			col = tagRedEdge
		}
		knockBurst(c, cx, cy, col)
	}
	c.rect(tagAX, tagAY-3, 1, 3, cream)
	c.outline(ink)
}

func knockBurst(c *canvas, cx, cy int, col color.NRGBA) {
	c.rect(cx-1, cy-1, 3, 3, cream)
	c.dot(cx, cy, col)
	for _, d := range [][2]int{{-2, 0}, {2, 0}, {0, -2}, {0, 2}, {-2, -2}, {2, 2}, {2, -2}, {-2, 2}} {
		c.dot(cx+d[0], cy+d[1], col)
	}
}
