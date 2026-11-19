package art

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func flatLand(n int, g Ground) *Land {
	l := NewLand(n, n)
	for i := range l.Ground {
		l.Ground[i] = g
	}
	return l
}

func at(img *image.NRGBA, gx, gy, lift float64) color.NRGBA {
	p := Project(gx, gy, lift)
	return img.NRGBAAt(p.X, p.Y)
}

func greenish(c color.NRGBA) bool { return c.A == 255 && c.G > c.R && c.G > c.B }

func TestLoneIslandIsRounded(t *testing.T) {
	l := flatLand(9, GroundWater)
	l.Ground[4*9+4] = GroundGrass
	l.Prepare()
	img := l.Chunk(0, 0)
	if c := at(img, 4, 4, 0); !greenish(c) {
		t.Fatalf("island centre = %v", c)
	}
	for _, corner := range [][2]float64{{3.56, 3.56}, {4.44, 4.44}, {4.44, 3.56}, {3.56, 4.44}} {
		if c := at(img, corner[0], corner[1], 0); greenish(c) {
			t.Errorf("diamond corner %v is still grass: the coast is not rounded", corner)
		}
	}
}

func TestTerraceShowsSoilFace(t *testing.T) {
	l := flatLand(12, GroundGrass)
	for y := 4; y <= 7; y++ {
		for x := 4; x <= 7; x++ {
			l.Height[y*12+x] = 1
		}
	}
	l.Prepare()
	img := l.Chunk(0, 0)
	edge := Project(7.5, 7.5, 0)
	soil := 0
	for y := edge.Y - ElevationStep; y <= edge.Y; y++ {
		if c := img.NRGBAAt(edge.X, y); c.R > c.G {
			soil++
		}
	}
	if soil < 2 {
		t.Fatalf("no soil face under the plateau front (found %d)", soil)
	}
	if c := at(img, 5.5, 5.5, ElevationStep); !greenish(c) {
		t.Errorf("plateau top not lifted: %v", c)
	}
}

func TestRoadsJoinNeighbours(t *testing.T) {
	l := flatLand(10, GroundGrass)
	for x := 2; x <= 7; x++ {
		l.Road[5*10+x] = 2
	}
	l.Prepare()
	img := l.Chunk(0, 0)
	for _, gx := range []float64{2.5, 4.5, 6.5} {
		if c := at(img, gx, 5, 0); c.R <= c.G {
			t.Errorf("road broken at %v: %v", gx, c)
		}
	}
	if c := at(img, 4.5, 5.7, 0); !greenish(c) {
		t.Errorf("road spills onto the verge: %v", c)
	}
	single := flatLand(6, GroundGrass)
	single.Road[3*6+3] = 3
	single.Ground[3*6+4] = GroundPlaza
	single.Prepare()
	if len(single.segs) != 1 {
		t.Errorf("a road next to the plaza joins it: %d segments", len(single.segs))
	}
}

func TestChunksTileWithoutGaps(t *testing.T) {
	l := flatLand(40, GroundGrass)
	l.Prepare()
	img := l.Render(0)
	for y := 3.0; y < 37; y += 0.7 {
		for x := 3.0; x < 37; x += 0.7 {
			if c := at(img, x, y, 0); c.A != 255 || c.B > c.G {
				t.Fatalf("gap or water at %v,%v: %v", x, y, c)
			}
		}
	}
}

func meanBlue(img *image.NRGBA) float64 {
	sum, n := 0.0, 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] == 255 {
			sum += float64(img.Pix[i+2]) / float64(int(img.Pix[i+1])+1)
			n++
		}
	}
	return sum / float64(max(n, 1))
}

func TestWildIslandsLookCooler(t *testing.T) {
	owned, wild := flatLand(16, GroundGrass), flatLand(16, GroundGrass)
	for i := range wild.Wild {
		wild.Wild[i] = true
	}
	owned.Prepare()
	wild.Prepare()
	if meanBlue(wild.Chunk(0, 0)) <= meanBlue(owned.Chunk(0, 0))*1.05 {
		t.Error("unowned land is not cooler")
	}
}

func veins(l *Land) int {
	img := l.Chunk(0, 0)
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] == 255 && img.Pix[i+2] > img.Pix[i+1]+20 {
			n++
		}
	}
	return n
}

func TestLeyVeinsFollowPower(t *testing.T) {
	quiet, strong := flatLand(16, GroundGrass), flatLand(16, GroundGrass)
	for i := range strong.Power {
		strong.Power[i] = 3
	}
	quiet.Prepare()
	strong.Prepare()
	if veins(quiet) != 0 || veins(strong) < 20 {
		t.Errorf("veins quiet %d strong %d", veins(quiet), veins(strong))
	}
}

func TestOpenSeaChunksAreEmptyAndCoastsFoam(t *testing.T) {
	sea := flatLand(20, GroundWater)
	sea.Prepare()
	if !sea.Empty(0, 0) {
		t.Fatal("open sea chunk should be empty")
	}
	l := flatLand(20, GroundWater)
	for y := 6; y < 14; y++ {
		for x := 6; x < 14; x++ {
			l.Ground[y*20+x] = GroundSand
		}
	}
	l.Prepare()
	if l.Empty(0, 0) {
		t.Fatal("chunk with an island is not empty")
	}
	img := l.Chunk(0, 0)
	foam := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 200 && img.Pix[i+1] > 200 && img.Pix[i+2] > 200 && img.Pix[i+3] > 150 {
			foam++
		}
	}
	if foam < 120 {
		t.Errorf("little foam along the coast: %d", foam)
	}
	if a, b := l.Chunk(0, 0), l.Chunk(0, 0); !bytes.Equal(a.Pix, b.Pix) {
		t.Error("chunk painting is not deterministic")
	}
}

func TestSeaFramesTile(t *testing.T) {
	for f := 0; f < SeaFrames; f++ {
		if SeaName(f+SeaFrames) != SeaName(f) {
			t.Fatal("sea names wrap")
		}
	}
	a := built().Terrain
	r := a.Regions[SeaName(0)]
	if r.W != SeaWidth || r.H != SeaHeight {
		t.Fatalf("sea frame %+v", r)
	}
	if seaColor(0, 5, 2) != seaColor(SeaWidth, 5, 2) && seaWave(0, 5, 2) != seaWave(SeaWidth, 5, 2) {
		t.Error("sea waves are not periodic")
	}
}
