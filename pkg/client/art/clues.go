package art

import "image/color"

var ClueKinds = []string{"hut", "logs", "describe", "get-pods", "keep-room"}

var TimelineKinds = []string{"struck", "cause-found", "fixed", "boxes-lost"}

func ClueIconName(kind string, lit bool) string {
	if lit {
		return iconName("clue-" + kind)
	}
	return iconName("clue-" + kind + "-dim")
}

func TimelineIconName(kind string) string { return iconName("timeline-" + kind) }

var cluePainters = map[string]func(c *canvas){
	"hut": iconPod, "logs": iconCrashLog, "describe": iconClueDescribe, "get-pods": iconGetPods, "keep-room": iconKeepRoom,
}

var timelinePainters = map[string]func(c *canvas){
	"struck": iconStruck, "cause-found": iconCauseFound, "fixed": iconFixed, "boxes-lost": iconBoxesLost,
}

func addIncidentIcons(b *atlasBuilder) {
	for _, k := range ClueKinds {
		paint := cluePainters[k]
		b.add(ClueIconName(k, true), IconSize, IconSize, 8, 8, func(c *canvas) {
			paint(c)
			c.outline(ink)
		})
		b.add(ClueIconName(k, false), IconSize, IconSize, 8, 8, func(c *canvas) {
			paint(c)
			c.mapPixels(dimClue)
			c.outline(ink)
		})
	}
	for _, k := range TimelineKinds {
		paint := timelinePainters[k]
		b.add(TimelineIconName(k), IconSize, IconSize, 8, 8, func(c *canvas) {
			paint(c)
			c.outline(ink)
		})
	}
}

func dimClue(col color.NRGBA) color.NRGBA {
	return darken(mix(col, grey(col), 0.7), 0.35)
}

func iconCrashLog(c *canvas) {
	c.rect(2, 1, 11, 14, plasterR[3])
	c.rect(12, 1, 1, 14, plasterR[1])
	for i, w := range []int{7, 5, 6} {
		c.rect(4, 3+i*2, w, 1, inkSft)
	}
	c.rect(3, 9, 9, 3, tagRedFill)
	c.rect(4, 10, 7, 1, tagRedEdge)
	c.rect(4, 13, 4, 1, inkSft)
}

func iconClueDescribe(c *canvas) {
	c.rect(1, 2, 10, 12, cream)
	for i := 0; i < 4; i++ {
		c.rect(3, 4+i*2, 5, 1, inkSft)
		c.dot(9, 4+i*2, []color.NRGBA{greenR[2], greenR[2], goldR[2], redR[2]}[i])
	}
	c.disc(11, 9, 3.4, glassR[3])
	c.disc(11, 9, 2.2, glassR[2])
	c.dot(10, 8, white)
	c.line(13, 12, 15, 14, woodR[1])
	c.line(13, 11, 15, 13, woodR[2])
}

func iconGetPods(c *canvas) {
	c.rect(1, 2, 14, 12, tagConsole)
	c.rect(1, 2, 14, 1, slateR[2])
	for i, st := range []color.NRGBA{greenR[3], redR[2], greenR[3]} {
		y := 4 + i*3
		c.rect(3, y, 2, 2, plasterR[3])
		c.rect(6, y, 4, 1, cream)
		c.rect(11, y, 2, 2, st)
	}
}

func iconKeepRoom(c *canvas) {
	c.rect(3, 4, 8, 11, stoneR[2])
	c.rect(3, 4, 1, 11, stoneR[3])
	for x := 3; x < 11; x += 2 {
		c.rect(x, 2, 1, 2, stoneR[2])
	}
	c.rect(5, 7, 4, 7, ink)
	c.rect(5, 11, 4, 3, goldR[2])
	c.rect(5, 9, 4, 2, redR[2])
	c.rect(12, 5, 2, 10, slateR[1])
	c.rect(12, 9, 2, 6, greenR[2])
}

func iconStruck(c *canvas) {
	c.shadePoly([]pt{{9, 0.5}, {3, 9}, {7.5, 9}, {5, 15.5}, {13, 6}, {8.5, 6}, {11.5, 0.5}}, litByX(goldR, 7))
	c.dot(8, 7, white)
}

func iconCauseFound(c *canvas) {
	c.disc(6.5, 6.5, 5.2, glassR[3])
	c.disc(6.5, 6.5, 3.8, glassR[1])
	c.rect(4, 4, 5, 5, tagRedFill)
	c.dot(6, 6, tagRedEdge)
	c.dot(5, 5, white)
	c.line(10, 10, 14, 14, woodR[1])
	c.line(11, 10, 15, 14, woodR[2])
}

func iconFixed(c *canvas) {
	c.disc(8, 8, 6.8, greenR[1])
	c.disc(7.5, 7.5, 5.6, greenR[2])
	c.line(4, 8, 7, 11, cream)
	c.line(4, 9, 7, 12, cream)
	c.line(7, 11, 12, 4, cream)
	c.line(7, 12, 12, 5, cream)
}

func iconBoxesLost(c *canvas) {
	for _, b := range [][2]int{{1, 7}, {7, 9}, {5, 3}} {
		c.rect(b[0], b[1], 6, 5, haulerR[2])
		c.rect(b[0], b[1], 6, 1, haulerR[3])
		c.rect(b[0]+2, b[1], 1, 5, haulerR[0])
	}
	c.line(10, 1, 15, 6, redR[2])
	c.line(15, 1, 10, 6, redR[2])
	c.line(11, 1, 15, 5, redR[1])
}
