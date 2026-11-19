package art

var (
	laborTools = map[string]bool{"axe": true, "pick": true}
	chipR      = ramp{rgb(150, 110, 64), rgb(196, 150, 92), rgb(226, 186, 124), rgb(246, 220, 168)}
)

func paintSapling(c *canvas, s size, v int) {
	c.rect(7, 13, 1, s.ay-12, barkR[1])
	c.dot(8, s.ay-1, barkR[0])
	leaf := []ramp{pineLeaf, oakLeaf, spruceLeaf}[v%3]
	if v == 1 {
		blob(c, 7.5, 10, 4.2, leaf, 90+v)
		blob(c, 5.5, 13, 2.6, leaf, 93+v)
		return
	}
	conifer(c, 7.5, []tier{{2, 8, 2.6}, {6, 13, 3.8}, {10, 17, 5}}, leaf, 96+v)
}

func drawLaborTool(c *canvas, tool string, hx, hy, lift int) {
	c.line(hx, hy+2, hx+2, hy-5+lift, woodR[2])
	head := hy - 6 + lift
	if tool == "axe" {
		c.rect(hx+1, head-1, 3, 3, greyR[1])
		c.dot(hx+3, head-1, greyR[3])
		c.dot(hx+4, head, greyR[2])
		return
	}
	c.line(hx-1, head+1, hx+2, head-1, greyR[1])
	c.line(hx+2, head-1, hx+5, head+1, greyR[2])
}

func carryLoad(c *canvas, tool string, top int) bool {
	switch tool {
	case "axe":
		c.rect(3, top+5, 10, 3, woodR[1])
		c.rect(3, top+5, 10, 1, woodR[2])
		c.ellipse(12.5, float64(top)+6.5, 1.6, 1.6, woodR[3])
		c.dot(12, top+6, woodR[1])
		return true
	case "pick":
		c.rect(5, top+6, 6, 4, cutR[2])
		c.rect(5, top+6, 6, 1, cutR[3])
		c.rect(10, top+7, 1, 3, cutR[1])
		return true
	}
	return false
}
