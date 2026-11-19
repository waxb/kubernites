package art

import "image/color"

const (
	sailPlain = iota
	sailStripes
	sailPatched
	sailCross
	sailBand
)

type sailStyle struct {
	cloth, mark ramp
	pattern     int
}

type shipLook struct {
	wood, deck, trim, flag ramp
	sail                   sailStyle
	shields, lantern       bool
	crew                   int
	seed                   int
}

var (
	creamCloth = ramp{rgb(170, 152, 120), rgb(212, 196, 160), rgb(236, 224, 192), rgb(252, 246, 224)}
	whiteCloth = ramp{rgb(150, 156, 164), rgb(200, 204, 208), rgb(232, 234, 232), rgb(252, 252, 248)}
	tanCloth   = ramp{rgb(140, 110, 74), rgb(184, 154, 108), rgb(214, 188, 138), rgb(238, 218, 174)}
	rustCloth  = ramp{rgb(116, 56, 48), rgb(160, 86, 70), rgb(192, 118, 94), rgb(222, 160, 130)}
	patchCloth = ramp{rgb(120, 96, 66), rgb(160, 132, 92), rgb(190, 160, 114), rgb(214, 190, 146)}
)

var shipWoods = []ramp{
	woodR,
	{rgb(54, 34, 28), rgb(86, 56, 40), rgb(120, 82, 54), rgb(160, 116, 76)},
	{rgb(64, 58, 56), rgb(98, 90, 82), rgb(136, 124, 108), rgb(174, 162, 140)},
	{rgb(90, 42, 30), rgb(138, 72, 44), rgb(180, 106, 62), rgb(220, 154, 96)},
}

var shipDecks = []ramp{
	{rgb(120, 84, 52), rgb(168, 124, 78), rgb(206, 164, 108), rgb(232, 200, 146)},
	{rgb(104, 76, 50), rgb(150, 112, 74), rgb(186, 148, 100), rgb(214, 184, 136)},
	{rgb(112, 104, 92), rgb(156, 146, 128), rgb(190, 180, 158), rgb(218, 210, 188)},
	{rgb(128, 82, 50), rgb(176, 122, 76), rgb(212, 160, 104), rgb(236, 196, 140)},
}

var shipTrims = []ramp{
	redR, blueR, greenR, goldR, tealR,
	{rgb(30, 26, 32), rgb(46, 40, 48), rgb(66, 58, 66), rgb(96, 86, 92)},
}

var shipFlags = []ramp{redR, blueR, goldR, greenR, purpleR, {rgb(170, 170, 176), rgb(214, 214, 214), rgb(244, 244, 240), white}}

var sailStyles = []sailStyle{
	{cloth: creamCloth, pattern: sailPlain},
	{cloth: whiteCloth, mark: redR, pattern: sailCross},
	{cloth: tanCloth, pattern: sailPlain},
	{cloth: rustCloth, pattern: sailPlain},
	{cloth: creamCloth, mark: blueR, pattern: sailStripes},
	{cloth: creamCloth, mark: patchCloth, pattern: sailPatched},
	{cloth: whiteCloth, mark: blueR, pattern: sailBand},
	{cloth: tanCloth, mark: redR, pattern: sailStripes},
}

var crewTunics = []color.NRGBA{redR[2], blueR[2], cream, greenR[2], goldR[1]}

func lookFor(class string, variant int) shipLook {
	k := classIndex(class)
	v := ((variant % ShipVariants) + ShipVariants) % ShipVariants
	wood := (v + k) % len(shipWoods)
	return shipLook{
		wood:    shipWoods[wood],
		deck:    shipDecks[wood],
		trim:    shipTrims[(v*3+k)%len(shipTrims)],
		flag:    shipFlags[(v*5+k+2)%len(shipFlags)],
		sail:    sailStyles[(v*5+k*3)%len(sailStyles)],
		shields: k <= 1 && (v+k)%3 == 0,
		lantern: (v+k)%2 == 0 || k >= 2,
		crew:    1 + (v*7+k)%3,
		seed:    v*31 + k*7,
	}
}

func classIndex(class string) int {
	for i, c := range ShipClasses {
		if c == class {
			return i
		}
	}
	return 0
}

func (s sailStyle) color(sp, t float64, idx int) color.NRGBA {
	k := min(max(idx+1, 1), 3)
	base, mark := s.cloth[k], s.mark[k]
	switch s.pattern {
	case sailStripes:
		if int(sp*6)%2 == 1 {
			return mark
		}
	case sailPatched:
		if sp > 0.56 && sp < 0.8 && t > 0.3 && t < 0.58 {
			return mark
		}
	case sailCross:
		if (absf(sp-0.5) < 0.09 && t > 0.2 && t < 0.85) || (absf(t-0.45) < 0.09 && sp > 0.25 && sp < 0.75) {
			return mark
		}
	case sailBand:
		if t > 0.38 && t < 0.62 {
			return mark
		}
	}
	return base
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
