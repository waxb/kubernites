package art

var toolIcons = map[string]func(c *canvas){
	"lantern-pole": iconLantern, "lantern": iconLantern, "staff": iconStaff, "hammer": iconHammer,
	"bindle": iconBindle, "horn": iconHorn, "loaf": iconLoaf, "quill": iconQuill, "sack": iconSack,
	"tome": iconTome, "book": iconBook, "satchel": iconSatchel, "tally": iconTally, "orb": iconOrb,
	"seal": iconSeal, "keys": iconKeys, "lens": iconLens, "pennant": iconPennant,
	"quill-ledger": iconQuillLedger, "parcel": func(c *canvas) { kindIcon(c, "parcel", 1) },
	"crate-back": func(c *canvas) { kindIcon(c, "crate", 1) }, "seal-press": iconSealPress,
}

func iconLantern(c *canvas) {
	c.rect(6, 0, 4, 1, greyR[1])
	c.shadePoly([]pt{{4, 4}, {6, 2}, {10, 2}, {12, 4}}, flat(greyR[1]))
	c.rect(4, 4, 8, 8, lampLit)
	c.rect(6, 6, 3, 4, white)
	c.rect(4, 4, 1, 8, greyR[0])
	c.rect(11, 4, 1, 8, greyR[0])
	c.rect(4, 12, 8, 2, greyR[1])
}

func iconStaff(c *canvas) {
	c.line(4, 15, 11, 3, woodR[2])
	c.line(5, 15, 12, 3, woodR[1])
	c.disc(12, 3, 2.2, greenR[3])
	c.dot(11, 2, white)
}

func iconBindle(c *canvas) {
	c.line(1, 14, 11, 5, woodR[2])
	c.disc(11.5, 5.5, 4, redR[2])
	c.dot(10, 4, white)
	c.dot(13, 7, white)
	c.dot(12, 3, redR[3])
	c.dot(9, 8, redR[1])
}

func iconHorn(c *canvas) {
	c.shadePoly([]pt{{1.5, 12}, {2.5, 11}, {11, 3}, {15, 7.5}, {3, 13}}, flat(goldR[2]))
	c.ellipse(13, 5, 2.2, 3.4, goldR[3])
	c.dot(13, 5, goldR[0])
	c.shadePoly([]pt{{5, 11}, {10, 8}, {10, 14}, {6, 15}}, flat(redR[2]))
}

func iconLoaf(c *canvas) {
	c.ellipse(8, 9.5, 6.5, 4, woodR[2])
	c.ellipse(7.5, 8.5, 5.5, 3, orangeR[2])
	for _, s := range [][4]int{{4, 7, 6, 10}, {7, 6, 9, 9}, {10, 6, 12, 9}} {
		c.line(s[0], s[1], s[2], s[3], woodR[1])
	}
}

func iconQuill(c *canvas) {
	c.shadePoly([]pt{{13, 1}, {15, 3}, {6, 12}, {4, 11}}, flat(plasterR[3]))
	c.line(13, 2, 5, 11, plasterR[1])
	c.line(5, 11, 2, 14, inkSft)
	c.rect(9, 11, 5, 4, blueR[0])
	c.rect(10, 10, 3, 1, blueR[1])
}

func iconSack(c *canvas) {
	c.disc(8, 10, 5, strawR[2])
	c.shadePoly([]pt{{6, 3}, {10, 3}, {9, 6}, {7, 6}}, flat(strawR[1]))
	c.rect(6, 5, 4, 1, woodR[1])
	c.disc(6, 9, 1.6, strawR[3])
	c.dot(2, 14, goldR[2])
	c.dot(14, 14, goldR[2])
}

func iconSatchel(c *canvas) {
	c.line(4, 6, 8, 1, woodR[0])
	c.line(8, 1, 12, 6, woodR[0])
	c.rect(3, 6, 10, 8, woodR[1])
	c.shadePoly([]pt{{3, 6}, {13, 6}, {12, 10}, {4, 10}}, flat(woodR[2]))
	c.rect(7, 9, 2, 2, goldR[2])
}

func iconTally(c *canvas) {
	c.rect(6, 1, 4, 14, woodR[3])
	c.rect(9, 1, 1, 14, woodR[2])
	for y := 3; y <= 11; y += 2 {
		c.rect(6, y, 2, 1, ink)
	}
}

func iconOrb(c *canvas) {
	c.disc(8, 7, 5, voidR[1])
	c.disc(7, 6, 3.5, voidR[2])
	c.dot(6, 5, voidR[3])
	c.dot(5, 4, white)
	c.shadePoly([]pt{{4, 13}, {12, 13}, {10, 11}, {6, 11}}, flat(goldR[1]))
	c.rect(3, 13, 10, 2, goldR[2])
}

func iconKeys(c *canvas) {
	c.disc(5, 5, 3.6, goldR[1])
	c.disc(5, 5, 1.8, clear)
	for _, x := range []int{4, 5, 6} {
		for _, y := range []int{4, 5, 6} {
			erase(c, x, y)
		}
	}
	c.line(7, 7, 14, 14, goldR[2])
	c.line(6, 8, 10, 15, goldR[3])
	c.rect(12, 13, 1, 2, goldR[2])
	c.rect(9, 13, 2, 1, goldR[3])
}
