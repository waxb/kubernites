package art

import "fmt"

const (
	ChestW = 20
	ChestH = 19
)

var TreasureKinds = []string{"ledger", "crate", "dispatch", "sealed"}

func TreasureChestName(kind string, open bool) string {
	st := "shut"
	if open {
		st = "open"
	}
	return fmt.Sprintf("treasure/%s/%s", kind, st)
}

func addTreasureArt(b *atlasBuilder) {
	for _, k := range TreasureKinds {
		for _, open := range []bool{false, true} {
			kind, op := k, open
			b.add(TreasureChestName(k, open), ChestW, ChestH, ChestW/2, ChestH-2, padded(func(c *canvas) { paintChest(c, kind, op) }))
		}
	}
}

func paintChest(c *canvas, kind string, open bool) {
	c.ellipse(float64(c.w)/2, float64(c.h)-2, 8.5, 2, withAlpha(shadow, 80))
	inked(c, func(c *canvas) {
		if open {
			chestOpenLid(c)
		} else {
			chestLid(c)
		}
		chestBody(c)
		chestMark(c, kind)
	})
	if open {
		chestGleam(c)
	}
}

func chestBody(c *canvas) {
	c.rect(2, 9, 14, 6, woodR[1])
	c.rect(2, 9, 14, 1, woodR[2])
	c.rect(2, 14, 14, 1, woodR[0])
	for _, x := range []int{3, 14} {
		c.rect(x, 9, 1, 6, vaultIron[1])
		c.dot(x, 10, vaultIron[3])
	}
}

func chestLid(c *canvas) {
	c.rect(3, 4, 12, 1, woodR[3])
	c.rect(2, 5, 14, 4, woodR[2])
	c.rect(2, 8, 14, 1, woodR[1])
	for _, x := range []int{3, 14} {
		c.rect(x, 4, 1, 5, vaultIron[1])
	}
	c.rect(8, 7, 2, 3, goldR[2])
	c.dot(8, 9, goldR[1])
}

func chestOpenLid(c *canvas) {
	c.rect(3, 1, 12, 5, woodR[0])
	c.rect(3, 1, 12, 1, woodR[2])
	for _, x := range []int{4, 13} {
		c.rect(x, 1, 1, 5, vaultIron[1])
	}
	c.ellipse(9, 8.5, 6.5, 2.6, goldR[2])
	for i := 0; i < 9; i++ {
		c.dot(4+int(hash(i, 1, 907)%11), 6+int(hash(i, 2, 907)%3), goldR[3])
	}
	c.rect(6, 7, 1, 1, cream)
	c.rect(12, 6, 1, 1, cream)
	c.rect(8, 9, 2, 2, goldR[1])
}

func chestMark(c *canvas, kind string) {
	x, y := 7, 10
	switch kind {
	case "ledger":
		c.rect(x, y, 5, 4, clerkR[2])
		c.rect(x, y, 5, 1, clerkR[3])
		c.rect(x+4, y+1, 1, 3, cream)
	case "crate":
		c.rect(x, y, 5, 4, haulerR[2])
		c.line(x, y, x+4, y+3, haulerR[0])
		c.line(x+4, y, x, y+3, haulerR[0])
	case "dispatch":
		c.rect(x, y, 5, 4, cream)
		c.line(x, y, x+2, y+2, purpleR[1])
		c.line(x+4, y, x+2, y+2, purpleR[1])
	default:
		c.disc(float64(x)+2.5, float64(y)+2, 2.3, sealR[2])
		c.dot(x+1, y+1, sealR[3])
		c.dot(x+2, y+2, sealR[0])
	}
}

func chestGleam(c *canvas) {
	for _, s := range [][3]int{{15, 2, 2}, {4, 4, 1}} {
		x, y, r := s[0], s[1], s[2]
		c.dot(x, y, white)
		for d := 1; d <= r; d++ {
			c.dot(x-d, y, goldR[3])
			c.dot(x+d, y, goldR[3])
			c.dot(x, y-d, goldR[3])
			c.dot(x, y+d, goldR[3])
		}
	}
}
