package art

import "image/color"

const (
	promptLift = 13
	promptGlow = promptLift - 3.5
)

var (
	menhirOutline = []pt{{-9, 3}, {-10, -5}, {-9, -15}, {-7, -24}, {-4, -31}, {0, -34}, {4, -33}, {7, -28}, {8, -19}, {9, -9}, {8, 3}, {4, 5}, {-3, 5}}
	promptDim     = rgb(84, 170, 210)
	promptLit     = rgb(150, 236, 255)
	promptCore    = rgb(226, 250, 255)
)

func paintMenhir(c *canvas, state string) {
	ax, ay := float64(SpeakStoneAX), float64(SpeakStoneAY)
	c.ellipse(ax, ay+2, 10.5, 4.5, dirtR[1])
	c.ellipse(ax-0.5, ay+1.5, 9.5, 3.5, grassR[1])
	inked(c, func(c *canvas) {
		body := make([]pt, len(menhirOutline))
		for i, p := range menhirOutline {
			body[i] = pt{ax + p.x, ay + p.y}
		}
		c.shadePoly(body, menhirShade(int(ax)))
	})
	menhirTufts(c, ax, ay)
	if state != StoneDormant {
		carvePrompt(c, int(ax)-6, int(ay)-promptLift, state == StoneLit)
	}
}

func menhirShade(mid int) shader {
	return func(x, y int) color.NRGBA {
		n := noise(x, y, 83)
		switch {
		case x >= mid+5:
			return greyR[0]
		case x >= mid:
			if n > 0.85 {
				return greyR[2]
			}
			return greyR[1]
		case n > 0.9:
			return greyR[3]
		case n < 0.1:
			return greyR[1]
		case y > int(SpeakStoneAY)-6 && n < 0.35:
			return greenR[1]
		}
		return greyR[2]
	}
}

func menhirTufts(c *canvas, ax, ay float64) {
	for i := 0; i < 7; i++ {
		h := hash(i, 9, 419)
		x := int(ax) - 10 + int(h%21)
		y := int(ay) + 1 + int(h/21%4)
		c.dot(x, y, []color.NRGBA{grassR[2], grassR[3], meadowR[2]}[h%3])
		c.dot(x, y-1, grassR[2])
	}
}

func carvePrompt(c *canvas, x, y int, lit bool) {
	edge, core := promptDim, promptLit
	if lit {
		edge, core = promptLit, promptCore
	}
	for _, dx := range []int{0, 1} {
		c.line(x+dx, y, x+3+dx, y+3, edge)
		c.line(x+3+dx, y+3, x+dx, y+6, edge)
	}
	c.line(x+1, y+1, x+3, y+3, core)
	c.rect(x+5, y+6, 5, 2, edge)
	c.rect(x+6, y+6, 3, 1, core)
}
