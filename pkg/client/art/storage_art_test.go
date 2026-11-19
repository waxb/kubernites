package art

import (
	"image/color"
	"testing"
)

type spriteSpec struct {
	name           string
	w, h, ax, ay   int
	atlasIsIcons   bool
	skipPaletteCap bool
}

func storageSprites() []spriteSpec {
	out := []spriteSpec{}
	for _, st := range WardenPostStates {
		out = append(out, spriteSpec{name: WardenPostName(st), w: WardenPostW, h: WardenPostH, ax: WardenPostAX, ay: WardenPostAY})
	}
	for _, n := range []string{StrandedTagName(), DeedTagName()} {
		out = append(out, spriteSpec{name: n, w: MarkTagW, h: MarkTagH, ax: MarkTagAX, ay: MarkTagAY})
	}
	for _, st := range VaultStates {
		for l := 0; l < VaultShelfLevels; l++ {
			out = append(out, spriteSpec{name: VaultName(st, l), w: VaultW, h: VaultH, ax: VaultAX, ay: VaultAY})
		}
	}
	for b := 0; b < VaultDetachBanners; b++ {
		out = append(out, spriteSpec{name: VaultDetachName(b), w: VaultW, h: VaultH, ax: VaultAX, ay: VaultAY})
	}
	for _, sealed := range []bool{true, false} {
		out = append(out, spriteSpec{name: DeedScrollName(sealed), w: DeedScrollW, h: DeedScrollH, ax: DeedScrollW / 2, ay: DeedScrollH / 2})
	}
	for _, red := range []bool{false, true} {
		for p := 0; p < ChainPhases; p++ {
			out = append(out, spriteSpec{name: ChainLinkName(red, p), w: ChainLinkSize, h: ChainLinkSize, ax: ChainLinkSize / 2, ay: ChainLinkSize / 2})
		}
	}
	out = append(out, spriteSpec{name: MountDotName(), w: MountDotSize, h: MountDotSize, ax: MountDotSize / 2, ay: MountDotSize / 2})
	for v := 0; v < LoosePageVariants; v++ {
		out = append(out, spriteSpec{name: LoosePagesName(v), w: LoosePagesW, h: LoosePagesH, ax: LoosePagesAX, ay: LoosePagesAY})
	}
	for _, id := range StoragePageIcons {
		out = append(out, spriteSpec{name: PageIconName(id), w: PageIconSize, h: PageIconSize, ax: PageIconSize / 2, ay: PageIconSize / 2, atlasIsIcons: true})
	}
	return out
}

func specAtlas(s Set, sp spriteSpec) Atlas {
	if sp.atlasIsIcons {
		return s.Icons
	}
	return s.Buildings
}

func TestStorageSpritesExistWithTheirSizesAndAnchors(t *testing.T) {
	s := built()
	for _, sp := range storageSprites() {
		r, ok := specAtlas(s, sp).Regions[sp.name]
		if !ok {
			t.Fatalf("%s is missing", sp.name)
		}
		if r.W != sp.w || r.H != sp.h || r.AnchorX != sp.ax || r.AnchorY != sp.ay {
			t.Errorf("%s: got %dx%d anchor %d,%d, want %dx%d anchor %d,%d", sp.name, r.W, r.H, r.AnchorX, r.AnchorY, sp.w, sp.h, sp.ax, sp.ay)
		}
	}
}

func TestStorageSpritesStayInThePalette(t *testing.T) {
	s := built()
	for _, sp := range storageSprites() {
		a := specAtlas(s, sp)
		r := a.Regions[sp.name]
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				if col := regionAt(a, r, x, y); col.A == 255 && paletteDistance(col) > 64 {
					t.Fatalf("%s: colour %v at %d,%d is %.0f away from the palette", sp.name, col, x, y, paletteDistance(col))
				}
			}
		}
	}
}

func pixelsApart(a Atlas, p, q Region) int {
	n := 0
	for y := 0; y < min(p.H, q.H); y++ {
		for x := 0; x < min(p.W, q.W); x++ {
			if regionAt(a, p, x, y) != regionAt(a, q, x, y) {
				n++
			}
		}
	}
	return n
}

func TestVaultShelvesFillInStepsAndStatesLookApart(t *testing.T) {
	a := built().Buildings
	empty := a.Regions[VaultName("bound", 0)]
	last := 0
	for l := 1; l < VaultShelfLevels; l++ {
		d := pixelsApart(a, empty, a.Regions[VaultName("bound", l)])
		if d <= last {
			t.Fatalf("shelf level %d changes %d pixels, not more than level %d's %d", l, d, l-1, last)
		}
		last = d
	}
	for i, st := range VaultStates {
		for _, other := range VaultStates[i+1:] {
			if d := pixelsApart(a, a.Regions[VaultName(st, 1)], a.Regions[VaultName(other, 1)]); d < 6 {
				t.Errorf("vault %s and %s differ in only %d pixels", st, other, d)
			}
		}
	}
}

func countNear(a Atlas, r Region, want color.NRGBA, within float64) int {
	n := 0
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if col := regionAt(a, r, x, y); col.A == 255 && colorDist(col, want) <= within {
				n++
			}
		}
	}
	return n
}

func TestTheUnboundDeedHasNoSealAndTheBoundVaultHasOne(t *testing.T) {
	a := built().Buildings
	if sealed, bare := countNear(a, a.Regions[DeedScrollName(true)], waxRed, 30), countNear(a, a.Regions[DeedScrollName(false)], waxRed, 30); sealed < 3 || bare != 0 {
		t.Fatalf("wax on the sealed scroll %d, on the unbound one %d", sealed, bare)
	}
	if bound, open := countNear(a, a.Regions[VaultName("bound", 0)], waxRed, 30), countNear(a, a.Regions[VaultName("available", 0)], waxRed, 30); bound < 3 || open != 0 {
		t.Fatalf("wax on the bound vault %d, on the available one %d", bound, open)
	}
}

func TestTheRedChainIsRedAndTheTagsFollowTheCauseTagColours(t *testing.T) {
	a := built().Buildings
	for p := 0; p < ChainPhases; p++ {
		if n, r := countNear(a, a.Regions[ChainLinkName(false, p)], redR[2], 40), countNear(a, a.Regions[ChainLinkName(true, p)], redR[2], 40); n != 0 || r < 3 {
			t.Fatalf("phase %d: red pixels normal %d, red %d", p, n, r)
		}
	}
	stranded := a.Regions[StrandedTagName()]
	if countNear(a, stranded, tagRedEdge, 1) < 10 || countNear(a, stranded, tagRedFill, 1) < 10 {
		t.Fatal("the stranded tag does not wear the red cause-tag colours")
	}
	if regionAt(a, stranded, MarkTagAX, MarkTagAY).A == 0 {
		t.Fatal("the stranded tag's tail tip is not at its anchor")
	}
	if countNear(a, a.Regions[DeedTagName()], tagGoldEdge, 1) < 10 {
		t.Fatal("the deed tag does not wear the gold strain-tag edge")
	}
}

func TestWardenPostStatesLookApart(t *testing.T) {
	a := built().Buildings
	run, start, absent := a.Regions[WardenPostName("running")], a.Regions[WardenPostName("starting")], a.Regions[WardenPostName("absent")]
	if opaquePixels(a, absent) >= opaquePixels(a, start) || opaquePixels(a, start) >= opaquePixels(a, run) {
		t.Fatalf("opaque pixels: absent %d, starting %d, running %d", opaquePixels(a, absent), opaquePixels(a, start), opaquePixels(a, run))
	}
	if countNear(a, run, lampLit, 20) == 0 || countNear(a, start, lampLit, 20) != 0 {
		t.Fatal("only a running post lights its lantern")
	}
}

func TestLoosePageVariantsDiffer(t *testing.T) {
	a := built().Buildings
	for v := 1; v < LoosePageVariants; v++ {
		if d := pixelsApart(a, a.Regions[LoosePagesName(v-1)], a.Regions[LoosePagesName(v)]); d < 10 {
			t.Fatalf("page variants %d and %d differ in %d pixels", v-1, v, d)
		}
	}
}
