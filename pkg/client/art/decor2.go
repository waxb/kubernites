package art

func paintHaystack(c *canvas, s size, v int) {
	hay := ramp{rgb(150, 110, 40), rgb(204, 160, 62), rgb(232, 196, 92), rgb(250, 228, 150)}
	blob(c, 9, 10, 7, hay, v+40)
	for x := 4; x < 15; x += 3 {
		c.dot(x, 13, hay[0])
	}
}

func paintFence(c *canvas, s size, v int) {
	for i := 0; i < 4; i++ {
		x := 3 + i*9
		y := 4 + i*4
		if x >= s.w-1 {
			break
		}
		c.rect(x, y, 2, 8, woodR[1])
		c.dot(x, y, woodR[3])
	}
	for _, off := range []int{6, 9} {
		c.line(3, 4+off-2, 31, 16+off-2, woodR[2])
	}
	if v == 1 {
		c.dot(20, 11, greenR[3])
	}
}
