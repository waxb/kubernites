package art

import (
	"image/color"
	"slices"
)

const (
	SpeakStoneW     = 26
	SpeakStoneH     = 48
	SpeakStoneAX    = 13
	SpeakStoneAY    = 40
	SpeakStoneTall  = int(slabTip)
	SpeakStoneHalfW = 11
	SpeakStoneFoot  = 7
	StallW          = stallW
	StallH          = stallH
	StallAX         = stallAX
	StallAY         = stallAY
	slabX0          = 0.30
	slabX1          = 0.70
	slabY0          = 0.34
	slabY1          = 0.66
	slabBase        = 3.0
	slabTop         = 28.0
	slabTip         = 36.0
	daisLo          = 0.16
	daisHi          = 0.84
)

var runeGlyphs = [][]string{
	{"x..", "xx.", "x.x", "x.."},
	{".x.", "xxx", ".x.", "x.x"},
	{"x.x", ".x.", "x.x", "..."},
	{"xx.", "x.x", "xx.", "x.."},
}

type runeSpot struct {
	left   bool
	u, z   float64
	glyph  int
	bright bool
}

var stoneRunes = []runeSpot{
	{left: true, u: 0.5, z: 23, glyph: 0, bright: true},
	{left: true, u: 0.5, z: 17, glyph: 1},
	{left: true, u: 0.5, z: 11, glyph: 2, bright: true},
	{left: false, u: 0.5, z: 20, glyph: 3},
	{left: false, u: 0.5, z: 13, glyph: 1, bright: true},
}

const (
	StoneObelisk = "obelisk"
	StoneMenhir  = "menhir"
)

var SpeakStoneLooks = []string{StoneObelisk, StoneMenhir}

const (
	StoneDormant = "dormant"
	StoneAwake   = "awake"
	StoneLit     = "lit"
	obeliskGlow  = 17.0
)

var SpeakStoneStates = []string{StoneDormant, StoneAwake, StoneLit}

var stonePainters = map[string]func(c *canvas, state string){StoneObelisk: paintSpeakStone, StoneMenhir: paintMenhir}

func SpeakStoneLook(ageTier int) string {
	if ageTier >= AgeTiers-1 {
		return StoneObelisk
	}
	return StoneMenhir
}

func SpeakStoneName(look, state string) string {
	if _, ok := stonePainters[look]; !ok {
		look = StoneObelisk
	}
	if !slices.Contains(SpeakStoneStates, state) {
		state = StoneDormant
	}
	return "stone/speaking/" + look + "/" + state
}

func SpeakStoneGlowLift(look string) float64 {
	if look == StoneMenhir {
		return promptGlow
	}
	return obeliskGlow
}

func addSpeakStone(b *atlasBuilder) {
	for _, look := range SpeakStoneLooks {
		for _, state := range SpeakStoneStates {
			paint, st := stonePainters[look], state
			b.add(SpeakStoneName(look, state), SpeakStoneW, SpeakStoneH, SpeakStoneAX, SpeakStoneAY, func(c *canvas) { paint(c, st) })
		}
	}
}

func speakStoneFrame(c *canvas) iso {
	return isoFrame(c, 1, SpeakStoneAX, SpeakStoneAY+TileHeight/2)
}

func paintSpeakStone(c *canvas, state string) {
	inked(c, func(c *canvas) {
		s := speakStoneFrame(c)
		s.stoneBox(daisLo, daisLo, daisHi, daisHi, 0, slabBase, stoneR, flat(stoneR[3]), 61)
		daisMoss(s)
		s.box(slabX0, slabY0, slabX1, slabY1, slabBase, slabTop, faces{top: flat(slateR[3]), left: roughStone(slateR, 2, 67), right: roughStone(slateR, 1, 71)})
		apex := s.p((slabX0+slabX1)/2+0.02, (slabY0+slabY1)/2-0.02, slabTip)
		s.c.shadePoly([]pt{s.p(slabX0, slabY1, slabTop), s.p(slabX1, slabY1, slabTop), apex}, roughStone(slateR, 3, 73))
		s.c.shadePoly([]pt{s.p(slabX1, slabY0, slabTop), s.p(slabX1, slabY1, slabTop), apex}, roughStone(slateR, 2, 79))
	})
	if state == StoneDormant {
		return
	}
	s := speakStoneFrame(c)
	for _, r := range stoneRunes {
		carveRune(s, r, state == StoneLit)
	}
}

func roughStone(r ramp, tone, seed int) shader {
	return func(x, y int) color.NRGBA {
		switch n := noise(x, y, seed); {
		case n > 0.9:
			return r[min(tone+1, 3)]
		case n < 0.12:
			return r[max(tone-1, 0)]
		}
		return r[tone]
	}
}

func daisMoss(s iso) {
	for i := 0; i < 6; i++ {
		h := hash(i, 5, 307)
		gx := daisLo + 0.06 + float64(h%60)/100
		gy := daisLo + 0.06 + float64(h/60%60)/100
		if gx > slabX0-0.04 && gx < slabX1+0.04 && gy > slabY0-0.04 && gy < slabY1+0.04 {
			continue
		}
		p := s.p(gx, gy, slabBase)
		s.c.dot(int(p.x), int(p.y), []color.NRGBA{greenR[1], greenR[2], meadowR[2]}[h%3])
	}
}

func carveRune(s iso, r runeSpot, lit bool) {
	at := s.p(r.u*(slabX1-slabX0)+slabX0, slabY1, r.z)
	if !r.left {
		at = s.p(slabX1, r.u*(slabY1-slabY0)+slabY0, r.z)
	}
	cut, glow := slateR[3], slateR[3]
	if r.bright {
		glow = tealR[2]
	}
	if lit {
		cut, glow = tealR[2], tealR[3]
		if r.bright {
			glow = glassR[3]
		}
	}
	x0, y0 := int(at.x)-1, int(at.y)-2
	for dy, row := range runeGlyphs[r.glyph%len(runeGlyphs)] {
		for dx, ch := range row {
			if ch != 'x' {
				continue
			}
			col := cut
			if dy == 0 || dx == 1 {
				col = glow
			}
			s.c.dot(x0+dx, y0+dy, col)
		}
	}
}
