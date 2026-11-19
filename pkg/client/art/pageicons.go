package art

import (
	"image/color"
	"math"
)

var PageIcons = []string{
	"speaking-stone", "marks", "muster", "scriptorium", "augury", "pledges", "keeps", "war-horns", "bindings", "masonry",
	"charters", "familiars", "ranks", "keep-floor", "palimpsest",
	"crews", "depots", "carters", "road-signs", "sealed-docks", "lighthouse", "gateways", "rollouts", "budgets", "rebalancer",
	"territory", "cert-manager", "herald", "keycloak", "goldpinger", "squid", "contracts", "blank",
	"daemons", "vaults", "stateful-crews", "vault-wrights", "bell", "watch", "treasure",
}

var pagePainters = map[string]func(c *canvas){
	"speaking-stone": pageStone, "marks": pageMarks, "muster": pageMuster, "scriptorium": pageScriptorium,
	"augury": pageAugury, "pledges": pagePledges, "keeps": pageKeeps, "war-horns": pageWarHorns,
	"bindings": pageBindings, "masonry": pageMasonry, "charters": pageCharters, "familiars": pageFamiliars,
	"ranks": pageRanks, "keep-floor": pageKeepFloor, "palimpsest": pagePalimpsest,
	"crews": pageCrews, "depots": pageDepots, "carters": pageCarters, "road-signs": pageRoadSigns,
	"sealed-docks": pageSealedDocks, "lighthouse": pageLighthouse, "gateways": pageGateways, "rollouts": pageRollouts,
	"budgets": pageBudgets, "rebalancer": pageRebalancer, "territory": pageTerritory, "cert-manager": pageCertManager,
	"herald": pageHerald, "keycloak": pageKeycloak, "goldpinger": pageGoldpinger, "squid": pageSquid,
	"contracts": pageContracts, "blank": pageBlank,
	"daemons": pageDaemons, "vaults": pageVaults, "stateful-crews": pageStatefulCrews, "vault-wrights": pageVaultWrights,
	"bell": pageBell, "watch": pageShutters, "treasure": pageTreasure,
}

func scrollSheet(c *canvas, x, y, w, h int) {
	c.rect(x, y+2, w, h-4, tkParchment)
	c.rect(x+w-2, y+2, 2, h-4, tkParchmentDark)
	for _, yy := range []int{y, y + h - 3} {
		c.rect(x-1, yy, w+2, 3, plasterR[2])
		c.rect(x-1, yy, w+2, 1, cream)
		c.rect(x-1, yy+2, w+2, 1, plasterR[1])
	}
}

func inkLines(c *canvas, x, y, w, n int, col color.NRGBA) {
	for i := 0; i < n; i++ {
		indent := []int{0, 2, 2, 0, 4}[i%5]
		c.rect(x+indent, y+i*3, w-indent-(i%2)*2, 1, col)
	}
}

func goldCoin(c *canvas, cx, cy, r float64) {
	c.disc(cx, cy, r, goldR[1])
	c.disc(cx-0.5, cy-0.5, r-1, goldR[2])
	c.dot(int(cx-r/2), int(cy-r/2), goldR[3])
}

func pageStone(c *canvas) {
	c.shadePoly([]pt{{4, 22}, {5, 8}, {9, 2}, {15, 2}, {19, 8}, {20, 22}}, func(x, y int) color.NRGBA {
		switch {
		case x < 9:
			return rockR[3]
		case x > 16:
			return rockR[1]
		}
		return rockR[2]
	})
	glow := rgb(150, 236, 255)
	for _, dx := range []int{0, 1} {
		c.line(7+dx, 8, 10+dx, 11, glow)
		c.line(10+dx, 11, 7+dx, 14, glow)
	}
	c.rect(12, 14, 5, 2, glow)
	for _, p := range [][2]int{{6, 8}, {12, 13}, {17, 14}, {6, 14}} {
		c.dot(p[0], p[1], rgb(84, 170, 210))
	}
	c.rect(2, 21, 20, 2, grassR[2])
	c.dot(3, 20, grassR[3])
	c.dot(20, 20, grassR[3])
}

func pageMarks(c *canvas) {
	c.line(3, 1, 11, 6, twineR[2])
	c.line(4, 1, 12, 6, twineR[1])
	c.shadePoly([]pt{{8, 8}, {13, 4}, {18, 8}, {18, 22}, {8, 22}}, flat(tkParchment))
	c.rect(17, 8, 1, 14, tkParchmentDark)
	c.disc(13, 8, 1.3, ink)
	c.rect(10, 12, 6, 1, mapInk)
	c.rect(10, 15, 4, 1, mapInk)
	c.disc(12.5, 19, 1.8, greenR[2])
	c.rect(15, 18, 2, 2, blueR[2])
}

func pageMuster(c *canvas) {
	for i := 0; i < 3; i++ {
		x := 5 + i*7
		c.rect(x-2, 10, 5, 9, orangeR[2])
		c.rect(x-2, 10, 1, 9, orangeR[3])
		c.rect(x+2, 10, 1, 9, orangeR[1])
		c.rect(x-2, 14, 5, 1, woodR[0])
		c.rect(x-2, 4, 5, 5, skinR[2])
		c.rect(x-2, 3, 5, 2, brownHair)
		c.dot(x-1, 6, ink)
		c.dot(x+1, 6, ink)
		c.rect(x-1, 19, 1, 2, woodR[0])
		c.rect(x+1, 19, 1, 2, woodR[0])
	}
	c.rect(2, 22, 20, 1, goldR[2])
	c.dot(2, 21, goldR[2])
	c.dot(21, 21, goldR[2])
}

func pageScriptorium(c *canvas) {
	scrollSheet(c, 3, 3, 14, 19)
	inkLines(c, 5, 7, 10, 4, mapInk)
	c.shadePoly([]pt{{22, 1}, {23, 3}, {14, 16}, {12.5, 15}}, flat(white))
	c.line(22, 2, 13, 15, plasterR[2])
	c.line(13, 15, 11, 19, inkSft)
}

func pageAugury(c *canvas) {
	for i, h := range []int{8, 13, 6} {
		col := []color.NRGBA{greenR[2], goldR[2], redR[2]}[i]
		c.rect(2+i*5, 21-h, 4, h, col)
		c.rect(2+i*5, 21-h, 4, 1, mix(col, white, 0.4))
	}
	c.rect(1, 21, 16, 1, mapInk)
	ringArc(c, 15, 9, 4.5, 6.5, func(float64) bool { return false }, greyR)
	c.disc(15, 9, 4.5, withAlpha(glassR[3], 200))
	c.dot(13, 7, white)
	c.line(19, 13, 22, 17, woodR[1])
	c.line(20, 13, 23, 17, woodR[2])
}

func pagePledges(c *canvas) {
	goldCoin(c, 12, 6, 4.5)
	c.dot(12, 6, goldR[0])
	c.shadePoly([]pt{{4, 14}, {20, 14}, {19, 21}, {5, 21}}, flat(skinR[2]))
	for i := 0; i < 4; i++ {
		c.rect(5+i*4, 11, 3, 4, skinR[2])
		c.dot(5+i*4, 11, skinR[3])
	}
	c.rect(4, 20, 16, 1, skinR[1])
	c.rect(3, 15, 2, 4, skinR[1])
	c.rect(6, 21, 12, 2, blueR[1])
}

func pageKeeps(c *canvas) {
	c.rect(6, 8, 12, 14, stoneR[2])
	c.rect(6, 8, 4, 14, stoneR[3])
	c.rect(15, 8, 3, 14, stoneR[1])
	for x := 6; x < 18; x += 4 {
		c.rect(x, 5, 2, 3, stoneR[2])
	}
	for _, y := range []int{12, 16, 20} {
		c.rect(6, y, 12, 1, stoneR[0])
	}
	c.rect(10, 16, 4, 6, woodR[0])
	c.rect(11, 10, 2, 3, lampLit)
	c.line(12, 1, 12, 5, woodR[0])
	c.shadePoly([]pt{{13, 1}, {19, 2.5}, {13, 4}}, flat(purpleR[2]))
}

func pageWarHorns(c *canvas) {
	for i := 0; i < 8; i++ {
		t := float64(i) / 7
		x := 3 + t*14
		y := 17 - math.Sin(t*math.Pi)*5
		c.disc(x, y, 1.2+t*2.4, goldR[2-i%2])
	}
	c.disc(18, 16, 3.6, goldR[1])
	c.disc(18, 16, 2, ink)
	for _, x := range []int{6, 12} {
		c.rect(x, 3, 2, 6, greenR[2])
		c.shadePoly([]pt{{float64(x) - 2, 4}, {float64(x) + 1, 0.5}, {float64(x) + 4, 4}}, flat(greenR[3]))
	}
}
