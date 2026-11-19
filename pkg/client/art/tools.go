package art

func drawTool(c *canvas, tool string, hx, hy int, up bool) {
	lift := 0
	if up {
		lift = -3
	}
	switch tool {
	case "lantern-pole", "lantern-pole-brass", "lantern-pole-twin":
		drawLanternPole(c, tool, hx, hy, lift)
	case "lantern":
		c.rect(hx, hy, 1, 2, ink)
		c.rect(hx-1, hy+2, 3, 3, lampLit)
	case "staff":
		c.line(hx, hy+5, hx+1, hy-7+lift, woodR[2])
		c.dot(hx+1, hy-8+lift, greenR[3])
	case "hammer":
		c.line(hx, hy+1, hx+2, hy-4+lift, woodR[2])
		c.rect(hx+1, hy-6+lift, 3, 2, greyR[1])
	case "bindle":
		c.line(hx-5, hy-4, hx+1, hy, woodR[2])
		c.disc(float64(hx)-5, float64(hy)-5, 1.7, redR[2])
	case "pilgrim-staff", "bindle-oat", "bindle-slate", "bindle-moss":
		drawPilgrimTool(c, tool, hx, hy, lift)
	case "pennant":
		drawPennant(c, hx, hy+lift)
	case "axe", "pick":
		drawLaborTool(c, tool, hx, hy, lift)
	case "horn":
		c.shadePoly([]pt{{float64(hx), float64(hy + lift)}, {float64(hx) + 4, float64(hy+lift) - 3}, {float64(hx) + 5, float64(hy+lift) + 1}}, flat(goldR[2]))
	default:
		drawHeldItem(c, tool, hx, hy+lift)
	}
}

func drawHeldItem(c *canvas, tool string, x, y int) {
	switch tool {
	case "loaf":
		c.ellipse(float64(x)+1, float64(y)+1, 2.5, 1.5, woodR[3])
	case "quill":
		c.line(x, y+1, x+2, y-3, white)
	case "sack":
		c.disc(float64(x)+1, float64(y)+1, 2.2, strawR[2])
	case "tome", "book":
		c.rect(x-1, y-1, 4, 3, redR[1])
		c.rect(x-1, y-1, 4, 1, cream)
	case "satchel":
		c.rect(x-1, y, 3, 3, woodR[1])
		c.dot(x, y+1, goldR[2])
	case "tally":
		c.rect(x-1, y-2, 3, 4, woodR[2])
		c.dot(x, y-1, ink)
		c.dot(x, y+1, ink)
	case "orb":
		c.disc(float64(x)+0.5, float64(y)+0.5, 1.8, voidR[2])
		c.dot(x, y, voidR[3])
	case "seal":
		c.disc(float64(x)+0.5, float64(y)+0.5, 2, goldR[2])
		c.dot(x, y, redR[2])
	case "keys":
		c.disc(float64(x)+0.5, float64(y)+0.5, 1.5, goldR[2])
		c.dot(x+1, y+2, goldR[1])
	case "lens":
		c.disc(float64(x)+0.5, float64(y)-0.5, 1.8, glassR[3])
		c.dot(x, y+1, goldR[1])
	}
}
