package art

var tradeItems = map[string]func(tf tradeFrame){
	"porter": porterItems, "hauler": haulerItems, "clerk": clerkItems, "sealwright": sealwrightItems, "courier": courierItems,
}

func miniParcel(c *canvas, x, y, w, h int) {
	c.rect(x, y, w, h, pineR[2])
	c.rect(x+w-1, y, 1, h, pineR[1])
	for i := 0; i < w; i++ {
		j := i * (h - 1) / max(w-1, 1)
		c.dot(x+i, y+j, battenR[1])
		c.dot(x+w-1-i, y+j, battenR[1])
	}
	c.rect(x, y, w, 1, pineR[3])
	c.dot(x+1, y, blueR[2])
}

func porterItems(tf tradeFrame) {
	c, top := tf.c, tf.top
	switch tf.action {
	case "carry":
		c.dot(11, top+5, tf.spec.tunic[0])
		c.dot(12, top+5, tf.spec.skin[1])
		miniParcel(c, 9, top+1, 5, 4)
	case "work":
		y := top + []int{6, 2, 5, 8}[tf.frame]
		miniParcel(c, 6, y, 5, 3)
		c.dot(5, y+1, tf.spec.skin[2])
		c.dot(11, y+1, tf.spec.skin[1])
	default:
		miniParcel(c, 2, top+6, 4, 3)
	}
}

func ropeCoil(c *canvas, top int) {
	c.line(6, top+5, 11, top+9, ropeR[1])
	c.line(6, top+6, 10, top+9, ropeR[2])
	c.rect(4, top+4, 3, 2, ropeR[2])
	c.dot(5, top+4, ropeR[0])
	c.dot(4, top+5, ropeR[3])
}

func backCrate(c *canvas, x, y int) {
	c.rect(x, y, 9, 8, crateR[2])
	c.rect(x, y, 9, 1, crateR[3])
	for _, yy := range []int{y + 3, y + 6} {
		c.rect(x, yy, 9, 1, crateR[0])
	}
	c.rect(x+8, y+1, 1, 7, crateR[1])
	c.rect(x+1, y+2, 2, 2, amberR[2])
}

func haulerItems(tf tradeFrame) {
	c, top := tf.c, tf.top
	switch tf.action {
	case "carry":
		c.rect(6, top+5, 1, 3, woodR[0])
		c.rect(10, top+5, 1, 3, woodR[0])
	case "work":
		y := top + []int{9, 7, 4, 7}[tf.frame]
		c.rect(5, y, 7, 5, crateR[2])
		c.rect(5, y, 7, 1, crateR[3])
		c.rect(5, y+2, 7, 1, crateR[0])
		c.rect(11, y+1, 1, 4, crateR[1])
		c.rect(7, y+3, 2, 1, amberR[2])
		c.dot(4, y+2, tf.spec.skin[2])
		c.dot(12, y+2, tf.spec.skin[1])
	default:
		ropeCoil(c, top)
	}
}

func ledgerBook(c *canvas, x, y int, cover ramp) {
	c.rect(x, y, 4, 3, cover[1])
	c.rect(x, y, 4, 1, cover[2])
	c.rect(x+3, y+1, 1, 2, cream)
	c.dot(x+1, y+1, goldR[2])
}

func clerkItems(tf tradeFrame) {
	c, top := tf.c, tf.top
	switch tf.action {
	case "carry":
		ledgerBook(c, 5, top+9, leatherR)
		ledgerBook(c, 6, top+7, redR)
		ledgerBook(c, 5, top+5, tealR)
	case "work":
		c.rect(5, top+7, 6, 2, cream)
		c.rect(5, top+9, 6, 1, leatherR[1])
		c.dot(8, top+7, inkSft)
		qx := 8 + tf.frame%2*2
		c.line(qx, top+6, qx+2, top+3, white)
		c.dot(qx+2, top+2, plasterR[2])
	default:
		ledgerBook(c, 2, top+6, leatherR)
		c.dot(11, top, white)
		c.dot(12, top-1, white)
	}
}

func sealPress(c *canvas, x, y int) {
	c.rect(x-1, y, 3, 1, woodR[2])
	c.rect(x, y+1, 1, 3, woodR[1])
	c.rect(x-1, y+4, 3, 2, goldR[1])
	c.dot(x-1, y+4, goldR[3])
}

func waxStick(c *canvas, x, y int, hot bool) {
	c.rect(x, y+1, 1, 3, sealR[2])
	c.dot(x, y, fireR[3])
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}} {
		c.dot(x+d[0], y+d[1], glowCol)
	}
	if hot {
		c.dot(x, y-2, withAlpha(fireR[2], 170))
		c.dot(x-1, y-1, glowCol)
		c.dot(x+1, y-1, glowCol)
	}
}

func sealwrightItems(tf tradeFrame) {
	c, top := tf.c, tf.top
	switch tf.action {
	case "carry":
		miniParcel(c, 6, top+6, 5, 4)
		c.disc(8.5, float64(top)+9, 1.4, sealR[2])
		c.dot(8, top+9, sealR[3])
	case "work":
		down := []int{0, 3, 1, 0}[tf.frame]
		sealPress(c, 11, top+2+down)
		if tf.frame == 1 || tf.frame == 2 {
			c.dot(11, top+9, sealR[2])
		}
		waxStick(c, 4, top+5, tf.frame >= 2)
	default:
		sealPress(c, 12, top+5-tf.p.armSwing)
		waxStick(c, 4, top+6+tf.p.armSwing, false)
	}
}

func satchel(c *canvas, top int) {
	c.line(6, top+5, 10, top+8, woodR[0])
	c.rect(10, top+8, 3, 3, woodR[1])
	c.rect(10, top+8, 3, 1, woodR[2])
	c.dot(11, top+9, goldR[2])
}

func miniCask(c *canvas, x, y int) {
	c.rect(x, y, 4, 5, caskR[2])
	c.rect(x, y, 1, 5, caskR[3])
	c.rect(x+3, y, 1, 5, caskR[1])
	c.rect(x, y+1, 4, 1, greyR[1])
	c.rect(x, y+3, 4, 1, purpleR[2])
	c.rect(x+1, y-1, 2, 1, caskR[3])
}

func courierItems(tf tradeFrame) {
	c, top := tf.c, tf.top
	satchel(c, top)
	switch tf.action {
	case "carry":
		c.dot(5, top+5, tf.spec.skin[2])
		miniCask(c, 2, top)
	case "work":
		hx := []int{4, 4, 12, 12}[tf.frame]
		y := top + 3 + tf.frame%2
		c.rect(hx-1, y, 3, 2, white)
		c.dot(hx, y+1, sealR[2])
	}
}
