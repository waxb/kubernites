package art

import (
	"math"
	"testing"
)

func silhouetteWidth(c *canvas) int {
	lo, hi := c.w, -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.at(x, y).A > 0 {
				lo, hi = min(lo, x), max(hi, x)
			}
		}
	}
	return hi - lo + 1
}

func TestLatheProfilesStartWideAndCloseAtTheTop(t *testing.T) {
	for name, p := range map[string]func(float64) float64{"cone": coneProfile, "dome": domeProfile, "bell": bellProfile, "onion": onionProfile, "flare": flareProfile, "straight": straight} {
		if p(0) <= 0 {
			t.Errorf("%s has no base", name)
		}
		if name != "straight" && p(1) > 0.05 {
			t.Errorf("%s does not close at the top: %v", name, p(1))
		}
	}
	if onionProfile(0.4) <= onionProfile(0) {
		t.Error("an onion bulges above its base")
	}
}

func TestRoundTowerIsAsWideAsItsRadius(t *testing.T) {
	for _, shape := range []int{shapeRound, shapeOct, shapeSquare} {
		s := testFrame(ageCastle, 0)
		s.turn(lathe{cx: 1.5, cy: 1.5, h: 30, r: 0.5, shape: shape}, toneMat(stoneR), matStone)
		want := 0.5 * 16 * math.Sqrt2 * 2
		if shape == shapeSquare {
			want = 0.5 * 64
		}
		if w := float64(silhouetteWidth(s.c)); math.Abs(w-want) > 3 {
			t.Errorf("shape %d is %v px wide, want about %v", shape, w, want)
		}
	}
}

func TestRoofFormsPaintTaggedRoofs(t *testing.T) {
	draws := map[string]func(s iso){
		"bell":  func(s iso) { s.roofBell(1.5, 1.5, 0.5, 20, 16, shapeRound, s.roofMat()) },
		"onion": func(s iso) { s.roofOnion(1.5, 1.5, 0.4, 20, 18, s.domeMat()) },
		"hip":   func(s iso) { s.roofHip(roofBox{0.5, 0.8, 2.5, 2.0, 20, 12, 0.1}, s.roofMat()) },
		"hipY":  func(s iso) { s.roofHip(roofBox{0.8, 0.5, 2.0, 2.5, 20, 12, 0.1}, s.roofMat()) },
		"shedY": func(s iso) { s.roofShed(0.5, 0.5, 1.5, 1.5, 10, 16, false, s.roofMat()) },
	}
	for name, draw := range draws {
		s := testFrame(ageFeudal, 1)
		draw(s)
		roof := 0
		for i, tag := range s.c.tags {
			if tag == matRoof && s.c.img.Pix[i*4+3] > 0 {
				roof++
			}
		}
		if roof < 60 {
			t.Errorf("%s paints only %d roof pixels", name, roof)
		}
	}
}

func TestTowersAndTurretsAcrossTheAges(t *testing.T) {
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < BuildingVariants; v++ {
			s := testFrame(age, v)
			tw := towerSpec{cx: 1.5, cy: 1.5, r: 0.4, h: 40, roofH: 14, crown: age == ageCastle}
			top := s.tower(tw)
			s.turret(0.8, 0.8, 0.16, 20, 20)
			s.towerWindow(tw, 80, 20)
			s.towerWindow(tw, 10, 24)
			s.towerWindow(tw, 200, 24)
			s.towerDoor(tw, 100, 12)
			s.towerDoor(tw, 0, 12)
			if top <= 0 || opaqueCount(s.c) < 400 {
				t.Errorf("age %d variant %d: tower top %v, %d pixels", age, v, top, opaqueCount(s.c))
			}
		}
	}
}

func TestAnnexesAndPropsAcrossTheAges(t *testing.T) {
	for age := 0; age < AgeTiers; age++ {
		s := testFrame(age, 2)
		s.hall(hallSpec{x0: 0.4, y0: 0.6, x1: 2.2, y1: 2.0, h: 24, rise: 14, alongY: true})
		s.leanTo(2.2, 0.6, 2.7, 1.6, 10, 16, false)
		s.porch(2.0, 0.8, 1.4, 0.3, 12)
		s.arcade(sideR(2.2), 0.7, 1.9, 0, 12, 3)
		s.dormer(1.2, 1.9, 26, 0.3, 6)
		s.chimney(0.6, 0.8, 18, 32)
		s.banner(sideR(2.2), 1.2, 20, 0.14, 8)
		for k := propBarrel; k < propKinds; k++ {
			s.prop(k, 0.3+float64(k)*0.25, 2.6)
		}
		s.window(sideR(2.2), 1.0, 12)
		s.doorway(sideR(2.2), 1.4, 0.3, 12)
		if opaqueCount(s.c) < 2000 {
			t.Errorf("age %d paints only %d pixels", age, opaqueCount(s.c))
		}
	}
}
