package art

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
)

const (
	FerryFrames = 4
	ferryDeckIn = 0.2
	bowLampAt   = 0.86
	bowLampUp   = 8.5
)

type FerryLook struct {
	Skiff bool
	Age   int
	Dir   string
	Frame int
}

type FerrySlot struct{ DX, DY float64 }

var ferrySpec = shipSpec{
	w: 44, h: 34, ax: 22, ay: 22,
	length: 1.4, beam: 0.66, freeboard: 3, sheer: 1.5, rise: 0.5, bulwark: 1, stripe: 1,
	bowPow: 3.2, sternPinch: 0.15,
}

var skiffSpec = shipSpec{
	w: 30, h: 26, ax: 15, ay: 17,
	length: 0.86, beam: 0.42, freeboard: 2.5, sheer: 2, rise: 1, bulwark: 1,
	bowPow: 1.6, sternPinch: 0.5,
}

var ferryPoles = [FerryFrames]float64{0, 0.3, 0.6, 0.3}

func (f FerryLook) kind() string {
	if f.Skiff {
		return "skiff"
	}
	return "ferry"
}

func (f FerryLook) Name() string {
	return fmt.Sprintf("%s%s/%d/%s/%d", PortPrefix, f.kind(), clampAge(f.Age), f.Dir, ((f.Frame%FerryFrames)+FerryFrames)%FerryFrames)
}

func ParseFerryName(name string) (FerryLook, bool) {
	rest, ok := strings.CutPrefix(name, PortPrefix)
	parts := strings.Split(rest, "/")
	if !ok || len(parts) != 4 || (parts[0] != "ferry" && parts[0] != "skiff") {
		return FerryLook{}, false
	}
	age, err := strconv.Atoi(parts[1])
	frame, err2 := strconv.Atoi(parts[3])
	if _, dir := headingAxes[parts[2]]; err != nil || err2 != nil || !dir || age != clampAge(age) || frame < 0 || frame >= FerryFrames {
		return FerryLook{}, false
	}
	return FerryLook{Skiff: parts[0] == "skiff", Age: age, Dir: parts[2], Frame: frame}, true
}

func (f FerryLook) spec() shipSpec {
	if f.Skiff {
		return skiffSpec
	}
	return ferrySpec
}

func ferryLookOf(age int) shipLook {
	switch clampAge(age) {
	case ageDark:
		return shipLook{wood: driftR, deck: driftR, trim: ropeR, flag: redR, seed: 3}
	case ageFeudal:
		return shipLook{wood: woodR, deck: shipDecks[0], trim: redR, flag: redR, seed: 5}
	case ageCastle:
		return shipLook{wood: tarR, deck: shipDecks[2], trim: blueR, flag: blueR, lantern: true, seed: 7}
	}
	return shipLook{wood: limedR, deck: shipDecks[3], trim: goldR, flag: blueR, lantern: true, seed: 11}
}

func FerrySprite(f FerryLook) (*image.NRGBA, Region) {
	sp := f.spec()
	c := newCanvas(sp.w, sp.h)
	s := shipScene{v: newHullView(c, sp.ax, sp.ay, f.Dir, shipBob[f.Frame%len(shipBob)]), sp: sp, lk: ferryLookOf(f.Age), frame: f.Frame}
	s.hull()
	if !f.Skiff {
		s.ferryman(ferryPoles[f.Frame%FerryFrames])
		if s.lk.lantern {
			s.bowLamp()
		}
	}
	c.outline(ink)
	s.waterline()
	return c.img, Region{W: sp.w, H: sp.h, AnchorX: sp.ax, AnchorY: sp.ay}
}

func (s shipScene) deckAt(t, w float64) pt {
	return s.v.p(t*s.sp.half(), s.sp.halfBeam(t)*w, s.sp.rail(t)-s.sp.bulwark)
}

func (s shipScene) ferryman(phase float64) {
	t := -0.74
	p := s.deckAt(t, 0)
	x, y := int(math.Floor(p.x)), int(math.Floor(p.y))
	c := s.v.c
	c.rect(x-1, y-6, 3, 5, blueR[1])
	c.rect(x-1, y-6, 1, 5, blueR[2])
	c.rect(x-1, y-9, 3, 3, skinR[2])
	c.rect(x-2, y-10, 5, 1, sandR[2])
	c.rect(x-1, y-11, 3, 1, sandR[2])
	near := s.nearSign()
	foot := s.v.p((t-0.25-0.35*phase)*s.sp.half(), near*s.sp.beam*0.75, -s.v.lift)
	hand := pt{float64(x) + 1, float64(y) - 7}
	top := pt{hand.x + (hand.x-foot.x)*0.35, hand.y + (hand.y-foot.y)*0.35}
	c.line(int(top.x), int(top.y), int(foot.x), int(foot.y), woodR[2])
	c.dot(x+1, y-5, skinR[1])
}

func FerryLamp(f FerryLook) (float64, float64, bool) {
	sp := f.spec()
	s := shipScene{v: newHullView(newCanvas(1, 1), sp.ax, sp.ay, f.Dir, shipBob[f.Frame%len(shipBob)]), sp: sp, lk: ferryLookOf(f.Age)}
	if f.Skiff || !s.lk.lantern {
		return 0, 0, false
	}
	p := s.deckAt(bowLampAt, 0)
	return math.Floor(p.x) + 0.5 - s.v.ox, math.Floor(p.y) - bowLampUp - s.v.oy, true
}

func (s shipScene) bowLamp() {
	p := s.deckAt(bowLampAt, 0)
	x, y := int(math.Floor(p.x)), int(math.Floor(p.y))
	c := s.v.c
	c.rect(x, y-7, 1, 7, s.lk.wood[0])
	c.rect(x-1, y-10, 3, 3, lampLit)
	c.dot(x, y-11, ink)
}

func FerrySlots(f FerryLook, n int) []FerrySlot {
	sp := f.spec()
	s := shipScene{v: newHullView(newCanvas(1, 1), sp.ax, sp.ay, f.Dir, shipBob[f.Frame%len(shipBob)]), sp: sp}
	spots := [][2]float64{{0.45, 0.2}, {0.15, -0.25}, {-0.15, 0.25}, {0.3, -0.3}, {-0.35, -0.1}, {0.6, -0.05}}
	if f.Skiff {
		spots = [][2]float64{{0, 0}}
	}
	out := make([]FerrySlot, 0, min(n, len(spots)))
	for _, sp := range spots[:min(n, len(spots))] {
		p := s.deckAt(sp[0], sp[1]*(1-ferryDeckIn))
		out = append(out, FerrySlot{DX: p.x - float64(s.v.ox), DY: p.y - float64(s.v.oy)})
	}
	return out
}

func FerryCapacity() int { return 6 }
