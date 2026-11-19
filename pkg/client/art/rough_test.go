package art

import (
	"image/color"
	"testing"
)

func testFrame(age, variant int) iso {
	c := newCanvas(100, 112)
	c.enableTags()
	fr := isoFrame(c, 3, 50, 110).with(newLook("chancery", age, variant))
	fr.gnd = newCanvas(100, 112)
	return fr
}

func opaqueCount(c *canvas) int {
	n := 0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.at(x, y).A > 0 {
				n++
			}
		}
	}
	return n
}

func TestTagsFollowThePen(t *testing.T) {
	c := newCanvas(4, 4)
	c.dot(1, 1, ink)
	if c.tagAt(1, 1) != matNone || c.liftAt(1, 1) != 255 {
		t.Fatal("an untagged canvas reports no tags")
	}
	c.enableTags()
	c.pen, c.penZ = matRoof, 7
	c.dot(2, 2, ink)
	if c.tagAt(2, 2) != matRoof || c.liftAt(2, 2) != 7 {
		t.Fatalf("tag %d lift %d", c.tagAt(2, 2), c.liftAt(2, 2))
	}
	c.erase(2, 2)
	if c.at(2, 2).A != 0 || c.tagAt(2, 2) != matNone || c.tagAt(9, 9) != matNone {
		t.Fatal("erase clears the pixel and its tag")
	}
	c.pen, c.penZ = matStone, 9
	c.dot(1, 2, ink)
	c.copyPixel(3, 3, 1, 2, white)
	if c.tagAt(3, 3) != matStone || c.liftAt(3, 3) != 9 || c.at(3, 3) != white {
		t.Fatal("copyPixel carries the tag and the lift of its source")
	}
	c.copyPixel(9, 9, 1, 2, white)
}

func TestBlockTagsWallsWithTheirHeight(t *testing.T) {
	s := testFrame(ageCastle, 0)
	s.block(1, 1, 2, 2, 0, 20, s.kit().wall, s.kit().wall, matStone)
	low, high := 255, 0
	for y := 0; y < s.c.h; y++ {
		for x := 0; x < s.c.w; x++ {
			if s.c.tagAt(x, y) == matStone {
				low, high = min(low, s.c.liftAt(x, y)), max(high, s.c.liftAt(x, y))
			}
		}
	}
	if low > 1 || high < 18 {
		t.Fatalf("wall lift spans %d..%d, want about 0..20", low, high)
	}
	if s.c.pen != matNone {
		t.Fatal("the pen resets after a block")
	}
}

func TestBreakupChipsAndBumpsButKeepsTheMass(t *testing.T) {
	s := testFrame(ageDark, 1)
	s.hall(hallSpec{x0: 0.4, y0: 0.6, x1: 2.4, y1: 2.2, h: 24, rise: 16})
	before := opaqueCount(s.c)
	src := snapshot(s.c)
	breakup(s.c, s.lk)
	changed := 0
	for y := 0; y < s.c.h; y++ {
		for x := 0; x < s.c.w; x++ {
			if src.NRGBAAt(x, y) != s.c.at(x, y) {
				changed++
			}
		}
	}
	after := opaqueCount(s.c)
	if changed == 0 {
		t.Fatal("a first-tier hall must lose its straight edges")
	}
	if diff := after - before; diff > before/20 || -diff > before/20 {
		t.Fatalf("breakup changed the mass from %d to %d pixels", before, after)
	}
}

func TestWeatherDarkensTheFootAndFadesTheRidge(t *testing.T) {
	s := testFrame(ageFeudal, 0)
	s.hall(hallSpec{x0: 0.4, y0: 0.6, x1: 2.4, y1: 2.2, h: 28, rise: 16})
	src := snapshot(s.c)
	weather(s.c, s.lk)
	foot, ridge := 0, 0
	for y := 0; y < s.c.h; y++ {
		for x := 0; x < s.c.w; x++ {
			was, now := src.NRGBAAt(x, y), s.c.at(x, y)
			if was == now {
				continue
			}
			switch s.c.tagAt(x, y) {
			case matWall, matStone:
				foot++
			case matRoof:
				ridge++
			}
		}
	}
	if foot == 0 || ridge == 0 {
		t.Fatalf("weathering touched %d wall and %d roof pixels", foot, ridge)
	}
}

func TestContactShadowAndTuftsGroundTheBuilding(t *testing.T) {
	s := testFrame(ageDark, 0)
	s.apron(3, 0.05)
	s.hall(hallSpec{x0: 0.6, y0: 0.6, x1: 2.2, y1: 2.2, h: 20, rise: 12})
	out := newCanvas(s.c.w, s.c.h)
	finishRough(out, s.gnd, s.c, s.lk)
	shaded, tufts := 0, 0
	for x := 0; x < s.c.w; x++ {
		yb, ok := baseRow(s.c, x)
		if !ok {
			continue
		}
		if g := s.gnd.at(x, yb+1); g.A > 0 && g.A < 255 || g.A == 255 && g.R < 120 {
			shaded++
		}
		if col := out.at(x, yb); col != ink && col.A > 0 {
			tufts++
		}
	}
	if shaded == 0 || tufts == 0 {
		t.Fatalf("contact shadow on %d columns, %d tufts", shaded, tufts)
	}
}

func meanColour(c *canvas) [3]float64 {
	var sum [3]float64
	n := 0.0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if col := c.at(x, y); col.A == 255 {
				sum[0], sum[1], sum[2], n = sum[0]+float64(col.R), sum[1]+float64(col.G), sum[2]+float64(col.B), n+1
			}
		}
	}
	return [3]float64{sum[0] / n, sum[1] / n, sum[2] / n}
}

func meanNRGBA(m [3]float64) color.NRGBA {
	return color.NRGBA{uint8(m[0]), uint8(m[1]), uint8(m[2]), 255}
}

func TestApronStylesDifferByAge(t *testing.T) {
	means := [AgeTiers][3]float64{}
	for age := 0; age < AgeTiers; age++ {
		s := testFrame(age, 0)
		s.apron(3, 0.05)
		if opaqueCount(s.gnd) < 1000 {
			t.Fatalf("age %d apron covers only %d pixels", age, opaqueCount(s.gnd))
		}
		means[age] = meanColour(s.gnd)
	}
	for a := 0; a < AgeTiers; a++ {
		for b := a + 1; b < AgeTiers; b++ {
			if colorDist(meanNRGBA(means[a]), meanNRGBA(means[b])) < 8 {
				t.Errorf("ages %d and %d have the same apron: %v %v", a, b, means[a], means[b])
			}
		}
	}
}

func TestGroundLayerFallsBackToTheCanvas(t *testing.T) {
	c := newCanvas(10, 10)
	s := isoFrame(c, 1, 5, 9)
	if s.groundLayer() != c {
		t.Fatal("without a ground layer the apron paints on the body")
	}
}
