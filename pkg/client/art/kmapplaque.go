package art

import (
	"image/color"
	"math"
)

type plaqueStyle struct {
	fill, frame, frameLit, bevel color.NRGBA
	mark                         func(c *canvas, x, y int)
}

var PlaqueText = map[string]color.NRGBA{
	"locked": rgb(150, 160, 150), "available": rgb(238, 226, 196), "researching": rgb(238, 226, 196), "learned": rgb(238, 226, 196),
}

var plaqueStyles = map[string]plaqueStyle{
	"locked":      {fill: rgb(44, 46, 48), frame: rgb(96, 96, 98), frameLit: rgb(122, 119, 108), bevel: rgb(26, 28, 30), mark: markPadlock},
	"available":   {fill: rgb(96, 66, 44), frame: tkGold, frameLit: tkGoldBright, bevel: rgb(60, 40, 28), mark: markStar},
	"researching": {fill: rgb(40, 58, 66), frame: rgb(74, 137, 156), frameLit: rgb(132, 190, 204), bevel: rgb(24, 36, 42), mark: markHourglass},
	"learned":     {fill: rgb(74, 52, 38), frame: goldR[1], frameLit: tkGold, bevel: rgb(46, 32, 24), mark: markCheck},
}

var bannerCloths = map[string]ramp{
	"networking":  blueR,
	"scheduling":  ramp{rgb(70, 74, 36), rgb(106, 110, 52), rgb(140, 146, 70), rgb(186, 190, 110)},
	"reliability": redR,
	"upkeep":      greenR,
}

func paintPlaque(c *canvas, state string) {
	st := plaqueStyles[state]
	c.rect(1, 1, c.w-2, c.h-2, st.fill)
	ring(c, 0, ink, ink)
	ring(c, 1, st.frameLit, st.frame)
	ring(c, 2, st.frame, st.frame)
	ring(c, 3, st.bevel, mix(st.fill, st.bevel, 0.5))
	roundCorners(c, true)
	st.mark(c, c.w-8, 5)
}

func markStar(c *canvas, x, y int) {
	c.shadePoly([]pt{{float64(x) + 0.5, float64(y) - 4}, {float64(x) + 1.8, float64(y) - 0.6}, {float64(x) + 5, float64(y) - 0.3},
		{float64(x) + 2.4, float64(y) + 1.8}, {float64(x) + 3.4, float64(y) + 5}, {float64(x) + 0.5, float64(y) + 3},
		{float64(x) - 2.4, float64(y) + 5}, {float64(x) - 1.4, float64(y) + 1.8}, {float64(x) - 4, float64(y) - 0.3}, {float64(x) - 0.8, float64(y) - 0.6}}, flat(tkGoldBright))
	c.dot(x, y-1, white)
}

func markPadlock(c *canvas, x, y int) {
	c.rect(x-2, y-3, 1, 3, greyR[2])
	c.rect(x+2, y-3, 1, 3, greyR[1])
	c.rect(x-1, y-4, 3, 1, greyR[2])
	c.rect(x-3, y, 7, 5, greyR[2])
	c.rect(x-3, y, 7, 1, greyR[3])
	c.dot(x, y+2, ink)
	c.dot(x, y+3, ink)
}

func markHourglass(c *canvas, x, y int) {
	c.rect(x-3, y-4, 7, 1, woodR[2])
	c.rect(x-3, y+4, 7, 1, woodR[2])
	c.shadePoly([]pt{{float64(x) - 2, float64(y) - 3}, {float64(x) + 3, float64(y) - 3}, {float64(x) + 0.5, float64(y) + 0.5}}, flat(glassR[3]))
	c.shadePoly([]pt{{float64(x) + 0.5, float64(y) + 0.5}, {float64(x) + 3, float64(y) + 4}, {float64(x) - 2, float64(y) + 4}}, flat(tkGoldBright))
	c.rect(x-1, y-2, 3, 1, tkGoldBright)
}

func markCheck(c *canvas, x, y int) {
	c.disc(float64(x)+0.5, float64(y)+0.5, 4.6, rgb(45, 83, 50))
	c.disc(float64(x)+0.5, float64(y)+0.5, 3.6, rgb(112, 176, 88))
	c.line(x-2, y, x, y+2, white)
	c.line(x, y+2, x+3, y-2, white)
}

func paintMedallion(c *canvas, state string) {
	st := plaqueStyles[state]
	cx, cy := float64(MedallionSize)/2, float64(MedallionSize)/2
	if state == "learned" {
		laurel(c, cx, cy)
	}
	c.disc(cx, cy, 14, st.frame)
	c.disc(cx-0.5, cy-0.5, 13, st.frameLit)
	c.disc(cx, cy, 12.5, st.frame)
	fill := tkParchment
	if state == "locked" {
		fill = rgb(120, 116, 106)
	}
	c.disc(cx, cy, 11.5, fill)
	if state == "researching" {
		for i := 0; i < 12; i++ {
			a := float64(i) * math.Pi / 6
			c.dot(int(cx+13*math.Cos(a)), int(cy+13*math.Sin(a)), white)
		}
	}
	c.outline(ink)
}

func laurel(c *canvas, cx, cy float64) {
	for i := 0; i < 9; i++ {
		a := math.Pi*0.6 + float64(i)*0.2
		for _, side := range []float64{-1, 1} {
			x := cx + side*14.2*math.Cos(a)
			y := cy + 14.2*math.Sin(a)
			c.ellipse(x, y, 1.6, 1.1, greenR[2+i%2])
		}
	}
}

func paintBranchBanner(c *canvas, branch string) {
	paintInset(c, BannerW, BannerH, kmapMargin, func(t *canvas) { paintBannerCloth(t, branch) })
	c.outline(ink)
}

func paintBranchFlag(c *canvas, branch string) {
	paintInset(c, FlagW, FlagH, kmapMargin, func(t *canvas) { paintFlagCloth(t, branch) })
	c.outline(ink)
}

func paintFlagCloth(c *canvas, branch string) {
	cloth := bannerCloths[branch]
	mid := FlagH / 2
	for y := range FlagH {
		cut := flagNotch - abs(y-mid)*flagNotch/mid
		tone := cloth[1]
		switch {
		case y < 3:
			tone = cloth[2]
		case y >= FlagH-3:
			tone = cloth[0]
		}
		c.rect(0, y, FlagW-max(cut, 0), 1, tone)
	}
	c.rect(0, 0, FlagTuck, FlagH, cloth[0])
	c.rect(FlagTuck, 0, 1, FlagH, goldR[1])
	c.rect(FlagTuck+2, 1, FlagW-FlagTuck-flagNotch-4, 1, goldR[2])
	c.rect(FlagTuck+2, FlagH-2, FlagW-FlagTuck-flagNotch-4, 1, goldR[1])
	bannerEmblems[branch](c, FlagTuck+(FlagW-FlagTuck-flagNotch)/2, mid)
}

func paintBannerCloth(c *canvas, branch string) {
	cloth := bannerCloths[branch]
	c.rect(1, 1, BannerW-2, 3, woodR[2])
	c.rect(1, 1, BannerW-2, 1, woodR[3])
	c.disc(2, 2.5, 2, goldR[2])
	c.disc(float64(BannerW)-2.5, 2.5, 2, goldR[2])
	c.shadePoly([]pt{{4, 4}, {28, 4}, {28, 44}, {16, 37}, {4, 44}}, func(x, y int) color.NRGBA {
		switch {
		case x < 7:
			return cloth[2]
		case x > 25:
			return cloth[0]
		}
		return cloth[1]
	})
	c.rect(4, 7, 24, 1, goldR[2])
	c.rect(4, 34, 24, 1, goldR[1])
	bannerEmblems[branch](c, 16, 21)
}
