package art

import (
	"image"
	"testing"
)

func shipRegion(class string, v int, dir string, f int) Region {
	return built().Effects.Regions[shipName(class, v, dir, f)]
}

func contentBounds(a Atlas, r Region) image.Rectangle {
	x0, y0, x1, y1 := r.W, r.H, -1, -1
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	return image.Rect(x0, y0, x1+1, y1+1)
}

func forEachShip(fn func(class string, v int, dir string, f int)) {
	for _, class := range ShipClasses {
		for v := 0; v < ShipVariants; v++ {
			for _, d := range ShipDirs {
				for f := 0; f < ShipFrames; f++ {
					fn(class, v, d, f)
				}
			}
		}
	}
}

func TestShipSpritesFitTheirCanvas(t *testing.T) {
	a := built().Effects
	forEachShip(func(class string, v int, dir string, f int) {
		r := shipRegion(class, v, dir, f)
		b := contentBounds(a, r)
		if b.Empty() || b.Min.X < 1 || b.Min.Y < 1 || b.Max.X > r.W-1 || b.Max.Y > r.H-1 {
			t.Errorf("%s: content %v touches or leaves the %dx%d canvas", shipName(class, v, dir, f), b, r.W, r.H)
		}
	})
	for _, class := range ShipClasses {
		sp := shipSpecs[class]
		union := image.Rectangle{}
		for v := 0; v < ShipVariants; v++ {
			for _, d := range ShipDirs {
				for f := 0; f < ShipFrames; f++ {
					union = union.Union(contentBounds(a, shipRegion(class, v, d, f)))
				}
			}
		}
		if union.Min.X > 2 || union.Min.Y > 2 || union.Max.X < sp.w-2 || union.Max.Y < sp.h-2 {
			t.Errorf("%s: canvas %dx%d is loose around %v", class, sp.w, sp.h, union)
		}
	}
}

func TestShipAnchorSitsOnTheHullAtTheWaterline(t *testing.T) {
	a := built().Effects
	forEachShip(func(class string, v int, dir string, f int) {
		r := shipRegion(class, v, dir, f)
		if a.Image.NRGBAAt(r.X+r.AnchorX, r.Y+r.AnchorY-1).A != 255 {
			t.Errorf("%s: no hull just above the anchor %d,%d", shipName(class, v, dir, f), r.AnchorX, r.AnchorY)
		}
		below := int(shipSpecs[class].beam*8) + 4
		if r.AnchorY+below < r.H && a.Image.NRGBAAt(r.X+r.AnchorX, r.Y+r.AnchorY+below).A != 0 {
			t.Errorf("%s: pixels far below the waterline centre", shipName(class, v, dir, f))
		}
	})
}

func TestShipClassesGrowStrictly(t *testing.T) {
	a := built().Effects
	prevW, prevH, prevN := 0, 0, 0
	for _, class := range ShipClasses {
		r := shipRegion(class, 0, "se", 0)
		n := opaquePixels(a, r)
		if r.W <= prevW || r.H <= prevH || n <= prevN {
			t.Errorf("%s (%dx%d, %d px) is not larger than the class before", class, r.W, r.H, n)
		}
		prevW, prevH, prevN = r.W, r.H, n
	}
}

func TestShipVariantsDirectionsAndFramesDiffer(t *testing.T) {
	a := built().Effects
	for _, class := range ShipClasses {
		for v := 0; v < ShipVariants; v++ {
			for w := v + 1; w < ShipVariants; w++ {
				if samePixels(a, shipRegion(class, v, "se", 0), shipRegion(class, w, "se", 0)) {
					t.Errorf("%s: variants %d and %d are identical", class, v, w)
				}
			}
			for f := 1; f < ShipFrames; f++ {
				if samePixels(a, shipRegion(class, v, "sw", f-1), shipRegion(class, v, "sw", f)) {
					t.Errorf("%s/%d: frames %d and %d are identical", class, v, f-1, f)
				}
			}
			for i, d := range ShipDirs {
				for _, e := range ShipDirs[i+1:] {
					if samePixels(a, shipRegion(class, v, d, 0), shipRegion(class, v, e, 0)) {
						t.Errorf("%s/%d: %s and %s are identical", class, v, d, e)
					}
				}
			}
		}
	}
}

func mirrored(a Atlas, x, y Region) bool {
	for yy := 0; yy < x.H; yy++ {
		for xx := 0; xx < x.W; xx++ {
			if a.Image.NRGBAAt(x.X+xx, x.Y+yy) != a.Image.NRGBAAt(y.X+y.W-1-xx, y.Y+yy) {
				return false
			}
		}
	}
	return true
}

func TestShipHeadingsAreLitNotMirrored(t *testing.T) {
	a := built().Effects
	for _, class := range ShipClasses {
		se, sw := shipRegion(class, 0, "se", 0), shipRegion(class, 0, "sw", 0)
		if mirrored(a, se, sw) {
			t.Errorf("%s: sw is a plain mirror of se, so the light flips", class)
		}
		if hullLuma(a, se) <= hullLuma(a, sw) {
			t.Errorf("%s: the se hull shows its lit side, so it must be brighter than sw", class)
		}
	}
}

func hullLuma(a Atlas, r Region) float64 {
	sum, n := 0.0, 0
	for y := r.AnchorY - 3; y < r.AnchorY+4; y++ {
		for x := 0; x < r.W; x++ {
			if c := a.Image.NRGBAAt(r.X+x, r.Y+y); c.A == 255 && c != ink {
				sum += float64(grey(c).R)
				n++
			}
		}
	}
	return sum / float64(max(n, 1))
}

func TestNoSpriteIsAnOpaqueRectangle(t *testing.T) {
	s := built()
	check := func(a Atlas, name string) {
		r := a.Regions[name]
		for _, p := range [][2]int{{0, 0}, {r.W - 1, 0}, {0, r.H - 1}, {r.W - 1, r.H - 1}} {
			if a.Image.NRGBAAt(r.X+p[0], r.Y+p[1]).A == 255 {
				t.Errorf("%s: corner %v is opaque", name, p)
			}
		}
		if opaquePixels(a, r) >= r.W*r.H*9/10 {
			t.Errorf("%s: nearly the whole canvas is painted", name)
		}
	}
	forEachShip(func(class string, v int, dir string, f int) { check(s.Effects, shipName(class, v, dir, f)) })
	for _, k := range HavenKinds {
		for v := 0; v < HavenVariants(k); v++ {
			check(s.Buildings, havenName(k, v))
		}
	}
	for f := 0; f < EffectFrames; f++ {
		check(s.Effects, effectName("wake", f))
		check(s.Effects, effectName("request", f))
	}
}

func TestShipLooksAreDeterministicAndVaried(t *testing.T) {
	for _, class := range ShipClasses {
		seen := map[int]bool{}
		for v := 0; v < ShipVariants; v++ {
			if lookFor(class, v) != lookFor(class, v+ShipVariants) {
				t.Errorf("%s: variant %d does not wrap", class, v)
			}
			seen[lookFor(class, v).sail.pattern*100+int(lookFor(class, v).sail.cloth[2].R)] = true
		}
		if len(seen) < 6 {
			t.Errorf("%s: only %d sail looks", class, len(seen))
		}
	}
	if classIndex("nope") != 0 || lookFor("cog", -1) != lookFor("cog", 7) {
		t.Error("unknown class or negative variant")
	}
}

func BenchmarkShipAtlas(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ab := &atlasBuilder{}
		addShips(ab)
		ab.build()
	}
}

func BenchmarkHarborAtlas(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ab := &atlasBuilder{}
		addHaven(ab)
		for f := 0; f < EffectFrames; f++ {
			ab.add(effectName("wake", f), WakeWidth, WakeHeight, WakeWidth/2, WakeHeight/2, func(c *canvas) { paintWake(c, 0) })
		}
		ab.build()
	}
}
