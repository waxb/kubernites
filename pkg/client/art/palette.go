package art

import "image/color"

type ramp [4]color.NRGBA

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{r, g, b, 255} }

var (
	clear  = color.NRGBA{}
	ink    = rgb(33, 27, 34)
	inkSft = rgb(58, 47, 54)
	cream  = rgb(255, 238, 190)
	white  = rgb(245, 244, 236)
	shadow = color.NRGBA{20, 16, 24, 90}

	waterR   = ramp{rgb(30, 78, 112), rgb(44, 110, 140), rgb(70, 150, 168), rgb(170, 222, 220)}
	shallowR = ramp{rgb(58, 132, 150), rgb(78, 160, 168), rgb(108, 188, 184), rgb(196, 236, 226)}
	sandR    = ramp{rgb(176, 138, 84), rgb(206, 172, 108), rgb(228, 200, 138), rgb(246, 226, 172)}
	grassR   = ramp{rgb(52, 98, 54), rgb(78, 130, 64), rgb(108, 158, 72), rgb(152, 188, 92)}
	meadowR  = ramp{rgb(84, 128, 58), rgb(112, 156, 66), rgb(142, 180, 82), rgb(188, 206, 110)}
	forestR  = ramp{rgb(36, 70, 44), rgb(50, 92, 52), rgb(68, 114, 58), rgb(98, 138, 66)}
	rockR    = ramp{rgb(84, 80, 82), rgb(118, 112, 106), rgb(152, 146, 134), rgb(196, 188, 168)}
	dirtR    = ramp{rgb(112, 74, 50), rgb(150, 104, 68), rgb(186, 138, 88), rgb(214, 172, 116)}
	plazaR   = ramp{rgb(120, 110, 100), rgb(160, 150, 134), rgb(188, 178, 156), rgb(218, 208, 182)}
	stoneR   = ramp{rgb(92, 88, 90), rgb(130, 124, 118), rgb(168, 160, 146), rgb(206, 196, 172)}
	woodR    = ramp{rgb(78, 44, 32), rgb(118, 70, 42), rgb(160, 102, 58), rgb(202, 142, 82)}
	plasterR = ramp{rgb(176, 150, 118), rgb(214, 190, 150), rgb(236, 216, 176), rgb(252, 240, 206)}
	redR     = ramp{rgb(98, 34, 36), rgb(150, 52, 46), rgb(190, 78, 58), rgb(228, 124, 86)}
	blueR    = ramp{rgb(38, 56, 96), rgb(56, 88, 138), rgb(82, 126, 172), rgb(132, 176, 204)}
	purpleR  = ramp{rgb(62, 40, 80), rgb(98, 60, 118), rgb(134, 88, 154), rgb(180, 132, 194)}
	greenR   = ramp{rgb(40, 76, 50), rgb(58, 110, 62), rgb(86, 146, 74), rgb(136, 182, 96)}
	goldR    = ramp{rgb(140, 84, 28), rgb(206, 148, 42), rgb(240, 196, 70), rgb(255, 234, 140)}
	tealR    = ramp{rgb(26, 76, 80), rgb(38, 110, 108), rgb(62, 150, 138), rgb(122, 196, 170)}
	orangeR  = ramp{rgb(130, 60, 24), rgb(196, 98, 34), rgb(236, 142, 50), rgb(252, 196, 104)}
	greyR    = ramp{rgb(70, 68, 74), rgb(104, 102, 106), rgb(144, 140, 138), rgb(190, 186, 176)}
	skinR    = ramp{rgb(138, 82, 58), rgb(196, 134, 92), rgb(232, 178, 126), rgb(250, 212, 160)}
	fireR    = ramp{rgb(190, 50, 30), rgb(240, 110, 40), rgb(255, 176, 60), rgb(255, 238, 150)}
	voidR    = ramp{rgb(56, 24, 86), rgb(104, 50, 150), rgb(160, 96, 214), rgb(220, 180, 255)}
	lampLit  = rgb(255, 214, 110)
	glassR   = ramp{rgb(40, 52, 70), rgb(70, 96, 120), rgb(120, 160, 180), rgb(200, 230, 236)}
)

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func withAlpha(c color.NRGBA, a uint8) color.NRGBA {
	return color.NRGBA{c.R, c.G, c.B, a}
}

func darken(c color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{uint8(float64(c.R) * (1 - t)), uint8(float64(c.G) * (1 - t)), uint8(float64(c.B) * (1 - t)), c.A}
}

func grey(c color.NRGBA) color.NRGBA {
	l := uint8((uint16(c.R)*3 + uint16(c.G)*5 + uint16(c.B)*2) / 10)
	return color.NRGBA{l, l, l, c.A}
}

func dusk(c color.NRGBA) color.NRGBA {
	g := grey(c)
	m := mix(c, g, 0.55)
	return color.NRGBA{uint8(float64(m.R) * 0.62), uint8(float64(m.G) * 0.64), uint8(float64(m.B) * 0.78), c.A}
}
