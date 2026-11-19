package art

import (
	"image"
	"image/color"
	"testing"
)

func mosaic() *Land {
	n := 30
	l := NewLand(n, n)
	bands := []Ground{GroundGrass, GroundMeadow, GroundForest, GroundRock, GroundSand, GroundPlaza, GroundRoad}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := y*n + x
			l.Ground[i] = bands[(x/4)%len(bands)]
			if x < 2 || y < 2 || x >= n-2 || y >= n-2 {
				l.Ground[i] = GroundShallows
			}
			if y >= 10 && y < 18 && x >= 2 && x < 26 && l.Ground[i] != GroundPlaza {
				l.Height[i] = uint8(1 + (y-10)/3)
			}
		}
	}
	for x := 3; x < 20; x++ {
		l.Road[22*n+x] = uint8(1 + x%3)
	}
	l.Road[25*n+6], l.Road[26*n+7] = 2, 2
	l.Road[24*n+22] = 1
	l.Road[20*n+20], l.Road[19*n+21] = 3, 3
	for i := range l.Power {
		l.Power[i] = uint8(i % 4)
	}
	l.Prepare()
	return l
}

func TestMosaicPaintsEveryGround(t *testing.T) {
	l := mosaic()
	img := l.Render(3)
	want := map[Ground]func(c color.NRGBA) bool{
		GroundGrass:  greenish,
		GroundForest: func(c color.NRGBA) bool { return c.A == 255 },
		GroundRock:   func(c color.NRGBA) bool { return c.A == 255 },
		GroundSand:   func(c color.NRGBA) bool { return c.A == 255 && c.R > c.B },
		GroundPlaza:  func(c color.NRGBA) bool { return c.A == 255 && c.R > 100 },
	}
	for g, ok := range want {
		x := 0
		for bands := []Ground{GroundGrass, GroundMeadow, GroundForest, GroundRock, GroundSand, GroundPlaza, GroundRoad}; bands[x/4] != g; x += 4 {
		}
		if c := at(img, float64(x)+1.5, 5, 0); !ok(c) {
			t.Errorf("ground %d at %d looks like %v", g, x, c)
		}
	}
	lifted := 0
	for x := 3.0; x < 25; x++ {
		if c := at(img, x, 16, 3*ElevationStep); c.A == 255 {
			lifted++
		}
	}
	if lifted < 15 {
		t.Errorf("third terrace missing: %d", lifted)
	}
}

func TestStampBlendsSprites(t *testing.T) {
	a := built().Terrain
	dst := image.NewNRGBA(image.Rect(-20, -40, 20, 10))
	Stamp(dst, a, "decor/tree-oak/0", image.Pt(0, 0), false)
	Stamp(dst, a, "decor/tree-oak/0", image.Pt(3, 0), true)
	Stamp(dst, a, "missing", image.Pt(0, 0), false)
	opaque := 0
	for i := 3; i < len(dst.Pix); i += 4 {
		if dst.Pix[i] > 0 {
			opaque++
		}
	}
	if opaque < 200 {
		t.Errorf("stamped %d pixels", opaque)
	}
	if got := over(color.NRGBA{10, 10, 10, 255}, color.NRGBA{250, 250, 250, 128}); got.R < 100 || got.R > 160 {
		t.Errorf("over = %v", got)
	}
}

func TestLandEdgesAndBareFields(t *testing.T) {
	l := &Land{W: 4, H: 4, Ground: make([]Ground, 16), Height: make([]uint8, 16), Road: make([]uint8, 16)}
	if l.powerField(1.5, 1.5) != 0 || l.depthField(1.5, 1.5) != seaReach || l.roadAt(-1, 2) != 0 || l.heightAt(9, 9) != 0 {
		t.Error("bare land fields")
	}
	b := newGroundBuf(image.Rect(0, 0, 8, 8))
	if b.matAt(-99, -99) != voidGround || b.lvlAt(-99, 0) != -1 {
		t.Error("buffer edges")
	}
	wild := flatLand(12, GroundForest)
	for i := range wild.Wild {
		wild.Wild[i] = true
		wild.Height[i] = uint8(min(i%12, 11-i%12) / 3)
	}
	wild.Prepare()
	if img := wild.Chunk(0, 0); img.Rect.Empty() {
		t.Error("wild hills paint nothing")
	}
}
