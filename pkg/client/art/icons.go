package art

import (
	"image/color"
	"math"
)

var iconPainters = map[string]func(c *canvas){
	"gold": iconGold, "wisdom": iconWisdom, "renown": iconRenown, "pod": iconPod, "node": iconNode,
	"service": iconService, "build": iconHammer, "summon": iconSummon, "apply": iconSeal,
	"scale-up": func(c *canvas) { iconArrow(c, false, greenR) }, "scale-down": func(c *canvas) { iconArrow(c, true, redR) },
	"probe": iconProbe, "roll": iconRoll, "tech": iconTech, "codex": iconBook, "chronicle": iconChronicle,
	"soulstone": iconSoulstone, "crier": iconBell, "age": iconAge, "speed": iconSpeed, "pause": iconPause,
	"alert": iconAlert, "describe": iconLens, "logs": iconLogs, "yaml": iconYAML, "back": iconBack, "close": iconClose,
	"academy": iconAcademy, "diploma": iconDiploma, "thread": iconThread,
	"loom-treasury": iconLoomTreasury, "loom-builders": iconLoomBuilders, "loom-fame": iconLoomFame,
	"siege": iconSiege, "archive": iconArchive, "warhorn": iconWarhorn,
	"food": iconFood, "housing": iconHousing, "wheat": iconWheat,
}

func buildIcons() Atlas {
	b := &atlasBuilder{}
	for _, n := range append(append([]string{}, IconNames...), FreightIcons...) {
		paint := iconPainters[n]
		if paint == nil {
			paint = caesarIconPainters[n]
		}
		if paint == nil {
			paint = freightIconPainters[n]
		}
		b.add(iconName(n), IconSize, IconSize, 8, 8, func(c *canvas) {
			paint(c)
			c.outline(ink)
		})
	}
	b.add(AppIconName, AppIconSize, AppIconSize, AppIconSize/2, AppIconSize/2, paintAppIcon)
	addKnowledgeMap(b)
	addIncidentIcons(b)
	return b.build()
}

func litByX(r ramp, split float64) shader {
	return func(x, y int) color.NRGBA {
		if float64(x) < split {
			return r[3]
		}
		return r[2]
	}
}

func iconGold(c *canvas) {
	c.disc(11, 7, 3.6, goldR[1])
	c.disc(10.6, 6.6, 2.6, goldR[2])
	c.dot(10, 5, goldR[3])
	for i := 0; i < 3; i++ {
		y := float64(13 - i*2)
		c.ellipse(6, y, 4.5, 1.8, goldR[0])
		c.ellipse(6, y-0.6, 4.5, 1.4, goldR[2])
	}
	c.dot(4, 8, goldR[3])
}

func iconWisdom(c *canvas) {
	c.shadePoly([]pt{{3, 2}, {6, 5}, {3, 6}}, flat(woodR[1]))
	c.shadePoly([]pt{{13, 2}, {10, 5}, {13, 6}}, flat(woodR[1]))
	c.ellipse(8, 9, 5.5, 5.8, woodR[1])
	c.ellipse(8, 11, 3.2, 3.2, plasterR[1])
	c.disc(5.8, 7, 2, cream)
	c.disc(10.2, 7, 2, cream)
	c.dot(6, 7, ink)
	c.dot(10, 7, ink)
	c.dot(8, 9, goldR[2])
	c.dot(8, 10, goldR[1])
	c.dot(6, 12, woodR[2])
	c.dot(10, 12, woodR[2])
}

func iconRenown(c *canvas) {
	c.shadePoly([]pt{{4, 8}, {8, 8}, {6, 15}, {3, 13}}, flat(redR[1]))
	c.shadePoly([]pt{{8, 8}, {12, 8}, {13, 13}, {10, 15}}, flat(redR[2]))
	c.disc(8, 6.5, 5, goldR[1])
	c.disc(7.6, 6.1, 4, goldR[2])
	c.rect(7, 3, 1, 7, goldR[3])
	c.rect(4, 6, 7, 1, goldR[3])
	c.dot(7, 6, white)
}

func iconPod(c *canvas) {
	c.rect(11, 2, 2, 4, stoneR[1])
	c.shadePoly([]pt{{1, 8.5}, {8, 1.5}, {15, 8.5}}, litByX(redR, 8))
	c.rect(3, 8, 10, 6, plasterR[2])
	c.rect(8, 8, 5, 6, plasterR[1])
	c.rect(5, 10, 2, 4, woodR[1])
	c.rect(10, 10, 2, 2, lampLit)
}

func iconNode(c *canvas) {
	c.shadePoly([]pt{{1, 5.5}, {8, 9}, {8, 14.5}, {1, 11}}, flat(dirtR[2]))
	c.shadePoly([]pt{{8, 9}, {15, 5.5}, {15, 11}, {8, 14.5}}, flat(dirtR[1]))
	c.shadePoly([]pt{{8, 2}, {15, 5.5}, {8, 9}, {1, 5.5}}, flat(grassR[2]))
	c.dot(5, 5, grassR[3])
	c.dot(9, 4, grassR[3])
	c.rect(10, 6, 2, 1, rockR[2])
	c.dot(4, 10, rockR[1])
	c.dot(12, 10, rockR[0])
}

func iconService(c *canvas) {
	c.rect(7, 3, 2, 12, woodR[1])
	c.shadePoly([]pt{{2, 3}, {12, 3}, {14.5, 5}, {12, 7}, {2, 7}}, flat(woodR[3]))
	c.shadePoly([]pt{{4, 8}, {14, 8}, {14, 12}, {4, 12}, {1.5, 10}}, flat(woodR[2]))
	c.rect(4, 5, 6, 1, woodR[1])
	c.rect(6, 10, 6, 1, woodR[1])
}

func erase(c *canvas, x, y int) {
	if c.in(x, y) {
		c.img.SetNRGBA(x, y, clear)
	}
}

func iconHammer(c *canvas) {
	c.line(7, 7, 13, 14, woodR[2])
	c.line(8, 7, 14, 13, woodR[1])
	c.shadePoly([]pt{{2, 5}, {7, 1}, {11, 5}, {6, 9}}, flat(greyR[2]))
	c.shadePoly([]pt{{2, 5}, {7, 1}, {8, 2}, {3, 6}}, flat(greyR[3]))
}

func iconSummon(c *canvas) {
	c.line(2, 14, 9, 7, woodR[0])
	c.line(3, 14, 10, 7, woodR[2])
	twinkle(c, 11, 5, 3, goldR[3])
	c.dot(4, 4, voidR[3])
	c.dot(14, 11, voidR[3])
	c.dot(13, 1, goldR[2])
}

func iconSeal(c *canvas) {
	for k := 0; k < 10; k++ {
		a := float64(k) * math.Pi / 5
		c.disc(8+math.Cos(a)*5.2, 8+math.Sin(a)*5.2, 1.4, redR[1])
	}
	c.disc(8, 8, 5.4, redR[1])
	c.disc(8, 8, 3.8, redR[2])
	c.rect(6, 7, 5, 1, redR[0])
	c.rect(7, 6, 3, 3, redR[0])
	c.dot(8, 7, redR[3])
	c.dot(4, 4, redR[3])
}

func iconArrow(c *canvas, down bool, r ramp) {
	pts := []pt{{8, 1}, {14.5, 7.5}, {10.5, 7.5}, {10.5, 14.5}, {5.5, 14.5}, {5.5, 7.5}, {1.5, 7.5}}
	if down {
		for i := range pts {
			pts[i].y = 16 - pts[i].y
		}
	}
	c.shadePoly(pts, litByX(r, 8))
}

func iconProbe(c *canvas) {
	c.disc(5.5, 6, 3.4, redR[2])
	c.disc(10.5, 6, 3.4, redR[2])
	c.shadePoly([]pt{{2.2, 7}, {13.8, 7}, {8, 14}}, flat(redR[2]))
	c.dot(4, 4, redR[3])
	for _, s := range [][4]int{{2, 8, 5, 8}, {5, 8, 7, 5}, {7, 5, 9, 11}, {9, 11, 10, 8}, {10, 8, 13, 8}} {
		c.line(s[0], s[1], s[2], s[3], white)
	}
}
