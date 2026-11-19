package art

import (
	"image"
	"testing"
)

func portOpaque(img *image.NRGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] >= 128 {
			n++
		}
	}
	return n
}

func portSame(a, b *image.NRGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

func allPortLooks() []PortLook {
	out := []PortLook{}
	for _, kind := range PortKinds {
		for level := 1; level <= PortLevels; level++ {
			for age := 0; age < AgeTiers; age++ {
				for v := 0; v < PortVariants(kind); v++ {
					out = append(out, PortLook{Kind: kind, Level: level, Age: age, Variant: v})
				}
			}
		}
	}
	return out
}

func TestEveryPortPiecePaintsAndNamesRoundTrip(t *testing.T) {
	for _, l := range allPortLooks() {
		img, r := PortSprite(l)
		if r.W != img.Bounds().Dx() || r.H != img.Bounds().Dy() || portOpaque(img) == 0 {
			t.Fatalf("%s paints %dx%d with %d opaque pixels", l.Name(), r.W, r.H, portOpaque(img))
		}
		back, ok := ParsePortName(l.Name())
		if !ok || back != l {
			t.Fatalf("%s parses back as %+v (%v)", l.Name(), back, ok)
		}
		if _, _, ok := PortSpriteByName(l.Name()); !ok {
			t.Fatalf("%s is not paintable by name", l.Name())
		}
	}
}

func TestPortNamesRejectStrangers(t *testing.T) {
	for _, name := range []string{"port/deck/0/0/0", "port/deck/5/0/0", "port/deck/1/4/0", "port/deck/1/0/3", "port/moat/1/0/0", "port/deck/1/0", "haven/quay/0", "port/ferry/0/up/0", "port/ferry/0/ne/9"} {
		if _, ok := ParsePortName(name); ok {
			t.Errorf("%s parsed as a port piece", name)
		}
		if _, _, ok := PortSpriteByName(name); ok {
			t.Errorf("%s painted", name)
		}
	}
}

func TestPortHouseAndPierGrowWithLevel(t *testing.T) {
	for _, kind := range []string{"house", "pier-x", "pier-y"} {
		for age := 0; age < AgeTiers; age++ {
			prev := 0
			for level := 1; level <= PortLevels; level++ {
				img, _ := PortSprite(PortLook{Kind: kind, Level: level, Age: age})
				n := portOpaque(img)
				if n <= prev {
					t.Errorf("%s at level %d, age %d holds %d pixels, no more than level %d (%d)", kind, level, age, n, level-1, prev)
				}
				prev = n
			}
		}
	}
}

func TestPortPiecesChangeWithTheStage(t *testing.T) {
	for _, kind := range PortKinds {
		for level := 1; level <= PortLevels; level++ {
			for age := 1; age < AgeTiers; age++ {
				a, _ := PortSprite(PortLook{Kind: kind, Level: level, Age: age - 1})
				b, _ := PortSprite(PortLook{Kind: kind, Level: level, Age: age})
				if portSame(a, b) {
					t.Errorf("%s at level %d looks the same in stages %d and %d", kind, level, age-1, age)
				}
			}
		}
	}
}

func TestStonePortsLookDifferentFromTimberOnes(t *testing.T) {
	for age := 0; age < AgeTiers; age++ {
		wood, _ := PortSprite(PortLook{Kind: "deck", Level: 2, Age: age})
		stone, _ := PortSprite(PortLook{Kind: "deck", Level: 3, Age: age})
		if portSame(wood, stone) {
			t.Errorf("the stone quay at age %d paints like the timber pier", age)
		}
	}
}

func TestFerriesPaintEveryHeadingAndRowAlong(t *testing.T) {
	for _, skiff := range []bool{false, true} {
		for age := 0; age < AgeTiers; age++ {
			for _, dir := range ShipDirs {
				first, _ := FerrySprite(FerryLook{Skiff: skiff, Age: age, Dir: dir})
				moved := false
				for f := 0; f < FerryFrames; f++ {
					l := FerryLook{Skiff: skiff, Age: age, Dir: dir, Frame: f}
					img, r := FerrySprite(l)
					if portOpaque(img) == 0 {
						t.Fatalf("%s is empty", l.Name())
					}
					if back, ok := ParseFerryName(l.Name()); !ok || back != l {
						t.Fatalf("%s parses back as %+v", l.Name(), back)
					}
					moved = moved || !portSame(first, img)
					for _, s := range FerrySlots(l, FerryCapacity()) {
						if x, y := float64(r.AnchorX)+s.DX, float64(r.AnchorY)+s.DY; x < 0 || y < 0 || x >= float64(r.W) || y >= float64(r.H) {
							t.Fatalf("%s seats a passenger outside the hull at %.1f,%.1f", l.Name(), x, y)
						}
					}
				}
				if !moved {
					t.Errorf("the %v ferry at age %d heading %s never moves", skiff, age, dir)
				}
			}
		}
	}
}

func TestFerrySeatsAsManyAsAsked(t *testing.T) {
	l := FerryLook{Age: 1, Dir: "se"}
	for n := 0; n <= FerryCapacity(); n++ {
		if got := len(FerrySlots(l, n)); got != n {
			t.Fatalf("asked for %d seats, got %d", n, got)
		}
	}
	if got := len(FerrySlots(FerryLook{Skiff: true, Dir: "se"}, 3)); got != 1 {
		t.Fatalf("a skiff seats %d", got)
	}
}
