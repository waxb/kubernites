package art

import (
	"image"
	"testing"
)

func boxContract() []string {
	names := []string{}
	for size := 0; size < BoxSizes; size++ {
		names = append(names, BoxSealName(size), BoxCrownName(size))
		for l := 0; l < FreshLevels; l++ {
			names = append(names, BoxFreshName(size, l))
		}
		for _, k := range BoxKinds {
			for v := 0; v < BoxVariants; v++ {
				names = append(names, BoxName(size, k, v), BoxSpoiledName(size, k, v))
			}
		}
	}
	for _, k := range BoxKinds {
		for l := 0; l < BoxPileLevels; l++ {
			names = append(names, BoxPileName(k, l))
		}
	}
	return names
}

func TestBoxContractNamesExist(t *testing.T) {
	a := built().Buildings
	for _, n := range boxContract() {
		if _, ok := a.Regions[n]; !ok {
			t.Errorf("missing %s", n)
		}
	}
	for _, n := range []string{BoxName(BoxSizes, "parcel", 0), BoxName(0, "parcel", BoxVariants), BoxFreshName(0, FreshLevels), BoxPileName("crate", BoxPileLevels)} {
		if _, ok := a.Regions[n]; ok {
			t.Errorf("%s should not exist", n)
		}
	}
	if len(BoxSizeNames) != BoxSizes || BoxSizeNames[3] != "huge" {
		t.Errorf("size names = %v", BoxSizeNames)
	}
}

func TestBoxOverlaysShareTheBoxCanvasAndAnchor(t *testing.T) {
	a := built().Buildings
	for size := 0; size < BoxSizes; size++ {
		base := a.Regions[BoxName(size, "parcel", 0)]
		sz := boxCanvas(size)
		if base.W != sz.w || base.H != sz.h || base.AnchorX != sz.ax || base.AnchorY != sz.ay || base.AnchorX != base.W/2 {
			t.Errorf("size %d: base region %+v, canvas %+v", size, base, sz)
		}
		names := []string{BoxSealName(size), BoxCrownName(size), BoxFreshName(size, 2)}
		for _, k := range BoxKinds {
			names = append(names, BoxName(size, k, 3), BoxSpoiledName(size, k, 1))
		}
		for _, n := range names {
			r := a.Regions[n]
			if r.W != base.W || r.H != base.H || r.AnchorX != base.AnchorX || r.AnchorY != base.AnchorY {
				t.Errorf("%s: region %+v does not align with %+v", n, r, base)
			}
		}
	}
}

func TestBoxesStandOnTheirAnchorAndGrow(t *testing.T) {
	a := built().Buildings
	prev := 0
	for size := 0; size < BoxSizes; size++ {
		d := worldBoxes[size]
		for _, k := range BoxKinds {
			r := a.Regions[BoxName(size, k, 0)]
			b := solidBounds(a, r)
			if low := b.Max.Y - 1; low < r.AnchorY || low > r.AnchorY+int(d.hh())+1 {
				t.Errorf("%s: lowest solid row %d, anchor %d", BoxName(size, k, 0), low, r.AnchorY)
			}
			if cx := (b.Min.X + b.Max.X) / 2; abs(cx-r.AnchorX) > 1 {
				t.Errorf("%s: centred at %d, anchor %d", BoxName(size, k, 0), cx, r.AnchorX)
			}
			if top := r.AnchorY - b.Min.Y; top < BoxLift(size) || top > BoxLift(size)+int(d.hh())+2 {
				t.Errorf("%s: rises %d px, lift %d", BoxName(size, k, 0), top, BoxLift(size))
			}
		}
		n := opaquePixels(a, a.Regions[BoxName(size, "crate", 0)])
		if n <= prev || (size > 0 && BoxLift(size) <= BoxLift(size-1)) {
			t.Errorf("size %d does not grow: %d pixels after %d", size, n, prev)
		}
		prev = n
	}
	if BoxLift(-5) != BoxLift(0) || BoxLift(99) != BoxLift(3) {
		t.Error("BoxLift clamps")
	}
}

func TestBoxVariantsAndKindsDiffer(t *testing.T) {
	a := built().Buildings
	for size := 0; size < BoxSizes; size++ {
		for _, k := range BoxKinds {
			for v := 0; v < BoxVariants; v++ {
				for w := v + 1; w < BoxVariants; w++ {
					if samePixels(a, a.Regions[BoxName(size, k, v)], a.Regions[BoxName(size, k, w)]) {
						t.Errorf("size %d %s: variants %d and %d are identical", size, k, v, w)
					}
				}
			}
		}
		for i, k := range BoxKinds {
			for _, other := range BoxKinds[i+1:] {
				if d := differingShare(a, a.Regions[BoxName(size, k, 0)], a.Regions[BoxName(size, other, 0)]); d < 0.35 {
					t.Errorf("size %d: %s and %s differ in only %.0f%%", size, k, other, d*100)
				}
			}
		}
	}
}

func regionSaturation(a Atlas, r Region) float64 {
	sum, n := 0.0, 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			c := a.Image.NRGBAAt(x, y)
			if c.A < 200 {
				continue
			}
			hi, lo := max(c.R, c.G, c.B), min(c.R, c.G, c.B)
			sum += float64(hi - lo)
			n++
		}
	}
	return sum / float64(max(n, 1))
}

func TestSpoiledBoxesAreDullerAndMouldy(t *testing.T) {
	a := built().Buildings
	for size := 0; size < BoxSizes; size++ {
		for _, k := range BoxKinds {
			fresh, rotten := a.Regions[BoxName(size, k, 2)], a.Regions[BoxSpoiledName(size, k, 2)]
			if regionSaturation(a, rotten) >= regionSaturation(a, fresh)*0.8 {
				t.Errorf("size %d %s: spoiled saturation %.1f vs %.1f", size, k, regionSaturation(a, rotten), regionSaturation(a, fresh))
			}
			if opaquePixels(a, rotten) != opaquePixels(a, fresh) {
				t.Errorf("size %d %s: the spoiled box changes its silhouette", size, k)
			}
		}
	}
}

func colourCount(a Atlas, r Region, want func(r, g, b uint8) bool) int {
	n := 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if c := a.Image.NRGBAAt(x, y); c.A == 255 && want(c.R, c.G, c.B) {
				n++
			}
		}
	}
	return n
}

func TestFreshnessRunsFromGreenToRed(t *testing.T) {
	a := built().Buildings
	for size := 0; size < BoxSizes; size++ {
		for l := 0; l < FreshLevels; l++ {
			col := freshCols[l]
			r := a.Regions[BoxFreshName(size, l)]
			if n := colourCount(a, r, func(r, g, b uint8) bool { return r == col.R && g == col.G && b == col.B }); n < 1 {
				t.Errorf("fresh %d/%d shows no %v sand", size, l, col)
			}
			if l > 0 && samePixels(a, r, a.Regions[BoxFreshName(size, l-1)]) {
				t.Errorf("fresh %d: levels %d and %d are identical", size, l-1, l)
			}
		}
	}
	if freshCols[0].G <= freshCols[0].R || freshCols[3].R <= freshCols[3].G {
		t.Error("fresh must be green and stale red")
	}
}

func TestMarksAreSmallAndSitOnTheBox(t *testing.T) {
	a := built().Buildings
	for size := 0; size < BoxSizes; size++ {
		box := solidBounds(a, a.Regions[BoxName(size, "crate", 0)]).Inset(-3)
		for _, n := range []string{BoxSealName(size), BoxCrownName(size), BoxFreshName(size, 0)} {
			r := a.Regions[n]
			if b := contentBounds(a, r); !b.In(box) {
				t.Errorf("%s: mark %v strays from the box %v", n, b, box)
			}
			if share := float64(opaquePixels(a, r)) / float64(opaquePixels(a, a.Regions[BoxName(size, "crate", 0)])); share > 0.75 {
				t.Errorf("%s covers %.0f%% of the box", n, share*100)
			}
		}
		if redPixels(a, a.Regions[BoxSealName(size)]) < 3 {
			t.Errorf("seal %d is not red wax", size)
		}
	}
}

func TestBoxPilesGrowOnATile(t *testing.T) {
	a := built().Buildings
	for _, k := range BoxKinds {
		prev := 0
		for l := 0; l < BoxPileLevels; l++ {
			r := a.Regions[BoxPileName(k, l)]
			if r.W != boxPileW || r.H != boxPileH || r.AnchorX != boxPileAX || r.AnchorY != boxPileAY {
				t.Errorf("%s: region %+v", BoxPileName(k, l), r)
			}
			b := contentBounds(a, r)
			if b.Min.X < 0 || b.Min.Y < 1 || b.Max.Y > r.H-1 {
				t.Errorf("%s: content %v touches the canvas edge", BoxPileName(k, l), b)
			}
			n := opaquePixels(a, r)
			if n <= prev*12/10 {
				t.Errorf("%s: %d pixels, level before %d", BoxPileName(k, l), n, prev)
			}
			prev = n
		}
	}
}

func TestSpoilFliesBuzz(t *testing.T) {
	a := built().Effects
	for f := 0; f < SpoilFrames; f++ {
		r := a.Regions[effectName("spoil", f)]
		if r.AnchorX != 8 || r.AnchorY != 15 || opaquePixels(a, r) < 6 {
			t.Errorf("spoil %d: %+v", f, r)
		}
		if samePixels(a, r, a.Regions[effectName("spoil", (f+1)%SpoilFrames)]) {
			t.Errorf("spoil frames %d and %d are identical", f, f+1)
		}
	}
}

func TestCarryOverlaysSitOnTheHead(t *testing.T) {
	a := built().Walkers
	head := figureTop + CarryAnchorY - (WalkerHeight - 2)
	for size := 0; size < BoxSizes; size++ {
		for _, k := range BoxKinds {
			for f := 0; f < WalkerFrames; f++ {
				r := a.Regions[CarryName(size, k, "se", f)]
				if r.W != CarryWidth || r.H != CarryHeight || r.AnchorX != CarryAnchorX || r.AnchorY != CarryAnchorY {
					t.Fatalf("%s: region %+v", CarryName(size, k, "se", f), r)
				}
				bob := poseFor("carry", f).bob
				b := solidBounds(a, r)
				if low := b.Max.Y - 1; low > head+bob+2 || low < head+bob-1 {
					t.Errorf("%s: box bottom at %d, head top at %d", CarryName(size, k, "se", f), low, head+bob)
				}
				if b.Min.Y < 0 || abs((b.Min.X+b.Max.X)/2-CarryAnchorX) > 1 {
					t.Errorf("%s: box %v is off the head", CarryName(size, k, "se", f), b)
				}
			}
		}
	}
}

func TestCarrySouthWestMirrorsSouthEast(t *testing.T) {
	a := built().Walkers
	for size := 0; size < BoxSizes; size++ {
		se, sw := a.Regions[CarryName(size, "ledger", "se", 1)], a.Regions[CarryName(size, "ledger", "sw", 1)]
		for y := 0; y < se.H; y++ {
			for x := 0; x < se.W; x++ {
				if a.Image.NRGBAAt(se.X+x, se.Y+y) != a.Image.NRGBAAt(sw.X+se.W-1-x, sw.Y+y) {
					t.Fatalf("carry %d: sw is not a mirror of se at %d,%d", size, x, y)
				}
			}
		}
	}
}

func worldFreightSprites() []string {
	out := boxContract()
	for size := 0; size < BoxSizes; size++ {
		out = append(out, CarryName(size, "crate", "se", 0))
	}
	for f := 0; f < BorderFlags; f++ {
		out = append(out, BorderFlagName(f))
	}
	for v := 0; v < BorderStones; v++ {
		out = append(out, BorderStoneName(v))
	}
	for _, k := range append(stallContract(), HelperKinds...) {
		for _, st := range BuildingStates {
			out = append(out, buildingName(k, st))
		}
	}
	return out
}

func TestFreightSpritesAreNotOpaqueRectangles(t *testing.T) {
	s := built()
	for _, n := range worldFreightSprites() {
		a := findAtlas(s, n)
		r := a.Regions[n]
		if share := float64(opaquePixels(a, r)) / float64(r.W*r.H); share > 0.8 || share == 0 {
			t.Errorf("%s: %.0f%% opaque", n, share*100)
		}
		for _, p := range []image.Point{{0, 0}, {r.W - 1, 0}, {0, r.H - 1}, {r.W - 1, r.H - 1}} {
			if a.Image.NRGBAAt(r.X+p.X, r.Y+p.Y).A != 0 {
				t.Errorf("%s: corner %v is opaque", n, p)
			}
		}
	}
}

func BenchmarkFreightSprites(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ab := &atlasBuilder{}
		addFreight(ab)
		addBorders(ab)
		addCarry(ab)
		addTradeHolds(ab)
		ab.build()
	}
}

func TestParcelsAreLightWoodenBoxesAndPaperIsKept(t *testing.T) {
	a := built().Buildings
	for size := 0; size < BoxSizes; size++ {
		light, dark := 0.0, 0.0
		for v := 0; v < BoxVariants; v++ {
			parcel := a.Regions[BoxName(size, "parcel", v)]
			light += luma(a, parcel)
			dark += luma(a, a.Regions[BoxName(size, "crate", v)])
			paper, ok := a.Regions[PaperBoxName(size, v)]
			if !ok || paper.W != parcel.W || paper.H != parcel.H || paper.AnchorX != parcel.AnchorX || paper.AnchorY != parcel.AnchorY {
				t.Fatalf("paper box %d/%d: %+v", size, v, paper)
			}
			if differingShare(a, paper, parcel) < 0.3 {
				t.Errorf("size %d v%d: the wooden parcel still looks like paper", size, v)
			}
		}
		if light <= dark {
			t.Errorf("size %d: parcels (%.0f) are not lighter than crates (%.0f)", size, light/BoxVariants, dark/BoxVariants)
		}
		if n := colourCount(a, a.Regions[BoxName(size, "parcel", 0)], func(r, g, b uint8) bool {
			return (r == battenR[2].R && g == battenR[2].G && b == battenR[2].B) || (r == battenR[3].R && g == battenR[3].G && b == battenR[3].B)
		}); n < 2 {
			t.Errorf("size %d: no X batten on the parcel", size)
		}
	}
}
