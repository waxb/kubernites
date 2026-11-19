package art

import (
	"image/color"
	"strings"
	"testing"
)

var (
	contractStates = list("ready", "kindling", "guttering", "ghost", "banished", "pending")
	contractQoS    = list("vagrant", "freeman", "landed")
	caesarIcons    = list("decree", "rating-reliability", "rating-efficiency", "rating-resilience", "rating-favor",
		"overlay-midden", "overlay-labor", "overlay-unhoused", "rank-high", "rank-normal", "rank-low", "council")
	councilFaces = list("advisor-chief", "advisor-quartermaster", "advisor-wayfinder", "advisor-treasurer", "steward", "augur")
)

func caesarContract() map[string][]string {
	buildings := expand("hearth", list("1", "2", "3"), contractStates, contractQoS)
	buildings = append(buildings, expand("midden", seq(4), seq(2))...)
	buildings = append(buildings, expand("codexbloat", seq(4))...)
	return map[string][]string{
		"buildings": buildings,
		"effects":   expand("fx", list("alarm"), seq(6)),
		"icons":     expand("icon", caesarIcons),
		"portraits": expand("portrait", councilFaces),
	}
}

func TestCaesarContractNamesExist(t *testing.T) {
	all := atlases(built())
	for atlas, names := range caesarContract() {
		for _, n := range names {
			if _, ok := all[atlas].Regions[n]; !ok {
				t.Errorf("%s: missing %s", atlas, n)
			}
		}
	}
}

func TestCaesarLevelCounts(t *testing.T) {
	if MiddenLevels != 4 || MiddenVariants != 2 || CodexBloatLevels != 4 || EffectFrameCount("alarm") != 6 {
		t.Fatalf("levels %d variants %d bloat %d alarm %d", MiddenLevels, MiddenVariants, CodexBloatLevels, EffectFrameCount("alarm"))
	}
	if len(HearthQoS) != 3 {
		t.Fatalf("qos classes = %v", HearthQoS)
	}
	s := built()
	for _, n := range []string{middenName(4, 0), middenName(0, 2), codexBloatName(4), effectName("alarm", 6)} {
		if _, ok := s.Buildings.Regions[n]; ok {
			t.Errorf("%s should not exist", n)
		}
		if _, ok := s.Effects.Regions[n]; ok {
			t.Errorf("%s should not exist", n)
		}
	}
}

func TestQoSHutsKeepTheOldHutGeometry(t *testing.T) {
	a := built().Buildings
	for tier := 1; tier <= 3; tier++ {
		for _, st := range HearthStates {
			old := a.Regions[hearthName(tier, st)]
			for _, q := range HearthQoS {
				r := a.Regions[hearthQoSName(tier, st, q)]
				if r.W != old.W || r.H != old.H || r.AnchorX != old.AnchorX || r.AnchorY != old.AnchorY {
					t.Errorf("%s: %+v differs from the old hut %+v", hearthQoSName(tier, st, q), r, old)
				}
			}
		}
	}
}

func insideTile(x, y, ax, ay, margin int) bool {
	cy := ay - TileHeight/2
	return y <= ay+1 && abs(x-ax)*TileHeight+abs(y-cy)*TileWidth <= (TileWidth/2+margin)*TileHeight
}

func TestQoSHutsStandOnTheirOwnTile(t *testing.T) {
	a := built().Buildings
	for tier := 1; tier <= 3; tier++ {
		for _, st := range HearthStates {
			for _, q := range HearthQoS {
				r := a.Regions[hearthQoSName(tier, st, q)]
				for y := r.AnchorY - TileHeight/2; y < r.H; y++ {
					for x := 0; x < r.W; x++ {
						if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 && !insideTile(x, y, r.AnchorX, r.AnchorY, 2) {
							t.Fatalf("%s: pixel %d,%d spills off the tile", hearthQoSName(tier, st, q), x, y)
						}
					}
				}
			}
		}
	}
}

func differingShare(a Atlas, x, y Region) float64 {
	diff, union := 0, 0
	for yy := 0; yy < min(x.H, y.H); yy++ {
		for xx := 0; xx < min(x.W, y.W); xx++ {
			p, q := a.Image.NRGBAAt(x.X+xx, x.Y+yy), a.Image.NRGBAAt(y.X+xx, y.Y+yy)
			if p.A == 0 && q.A == 0 {
				continue
			}
			union++
			if p != q {
				diff++
			}
		}
	}
	return float64(diff) / float64(max(union, 1))
}

func TestQoSStylesDiffer(t *testing.T) {
	a := built().Buildings
	for tier := 1; tier <= 3; tier++ {
		for _, st := range HearthStates {
			for i, p := range HearthQoS {
				for _, q := range HearthQoS[i+1:] {
					if d := differingShare(a, a.Regions[hearthQoSName(tier, st, p)], a.Regions[hearthQoSName(tier, st, q)]); d < 0.3 {
						t.Errorf("tier %d %s: %s and %s differ in only %.0f%% of pixels", tier, st, p, q, d*100)
					}
				}
			}
		}
	}
}

func accentCount(a Atlas, r Region, tier int) int {
	want := map[color.NRGBA]bool{}
	for _, rr := range []ramp{tierRamp(tier), tentRamp(tier)} {
		for _, c := range rr {
			want[c] = true
		}
	}
	n := 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if want[a.Image.NRGBAAt(x, y)] {
				n++
			}
		}
	}
	return n
}

func TestQoSTiersStayDistinct(t *testing.T) {
	a := built().Buildings
	for _, q := range HearthQoS {
		for _, st := range HearthStates {
			for tier := 1; tier <= 3; tier++ {
				r := a.Regions[hearthQoSName(tier, st, q)]
				for other := tier + 1; other <= 3; other++ {
					if samePixels(a, r, a.Regions[hearthQoSName(other, st, q)]) {
						t.Errorf("%s %s: tiers %d and %d are identical", q, st, tier, other)
					}
				}
				if st == "ghost" || st == "banished" {
					continue
				}
				own := accentCount(a, r, tier)
				for other := 1; other <= 3; other++ {
					if other != tier && accentCount(a, r, other) >= own {
						t.Errorf("%s %s tier %d: tier %d colours (%d) rival its own (%d)", q, st, tier, other, accentCount(a, r, other), own)
					}
				}
				if own < 6 {
					t.Errorf("%s %s tier %d: only %d accent pixels", q, st, tier, own)
				}
			}
		}
	}
}

func caesarWorldSprites() []string {
	out := []string{}
	for atlas, names := range caesarContract() {
		if atlas != "portraits" {
			out = append(out, names...)
		}
	}
	return out
}

func TestCaesarSpritesAreNotOpaqueRectangles(t *testing.T) {
	s := built()
	for _, n := range caesarWorldSprites() {
		a := findAtlas(s, n)
		r := a.Regions[n]
		if share := float64(opaquePixels(a, r)) / float64(r.W*r.H); share > 0.85 {
			t.Errorf("%s: %.0f%% opaque", n, share*100)
		}
		for _, p := range [][2]int{{0, 0}, {r.W - 1, 0}, {0, r.H - 1}, {r.W - 1, r.H - 1}} {
			if a.Image.NRGBAAt(r.X+p[0], r.Y+p[1]).A != 0 {
				t.Errorf("%s: corner %v is opaque", n, p)
			}
		}
	}
}

func lowestOpaqueRow(a Atlas, r Region) int {
	for y := r.H - 1; y >= 0; y-- {
		for x := 0; x < r.W; x++ {
			if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 {
				return y
			}
		}
	}
	return -1
}

func TestMiddenGrowsWithItsLevelAndSitsOnATile(t *testing.T) {
	a := built().Buildings
	for v := 0; v < MiddenVariants; v++ {
		prev := 0
		for l := 0; l < MiddenLevels; l++ {
			r := a.Regions[middenName(l, v)]
			if r.W != middenSize.w || r.H != middenSize.h || r.AnchorX != r.W/2 || r.AnchorY != middenSize.ay {
				t.Errorf("%s: region %+v", middenName(l, v), r)
			}
			if low := lowestOpaqueRow(a, r); low > r.AnchorY+2 {
				t.Errorf("%s: content reaches row %d below the anchor %d", middenName(l, v), low, r.AnchorY)
			}
			n := opaquePixels(a, r)
			if n <= prev*11/10 {
				t.Errorf("%s: %d opaque pixels, level before had %d", middenName(l, v), n, prev)
			}
			prev = n
		}
	}
	for l := 0; l < MiddenLevels; l++ {
		if differingShare(a, a.Regions[middenName(l, 0)], a.Regions[middenName(l, 1)]) < 0.2 {
			t.Errorf("midden level %d: variants barely differ", l)
		}
	}
}

func TestCodexBloatSharesTheCodexAnchor(t *testing.T) {
	a := built().Buildings
	codex := a.Regions[buildingName("codex", "ready")]
	prev := 0
	for l := 0; l < CodexBloatLevels; l++ {
		r := a.Regions[codexBloatName(l)]
		if r.AnchorX != r.W/2 || r.W < codex.W || r.AnchorY < codex.AnchorY || r.H-r.AnchorY > 8 {
			t.Errorf("%s: region %+v does not line up with the codex %+v", codexBloatName(l), r, codex)
		}
		n := opaquePixels(a, r)
		if n <= prev*13/10 {
			t.Errorf("%s: %d opaque pixels, level before had %d", codexBloatName(l), n, prev)
		}
		prev = n
	}
}

func TestAlarmPulses(t *testing.T) {
	a := built().Effects
	for f := 0; f < EffectFrameCount("alarm"); f++ {
		r := a.Regions[effectName("alarm", f)]
		if r.AnchorX != r.W/2 || r.AnchorY != r.H-1-effectPads["alarm"] {
			t.Errorf("alarm %d: anchor %d,%d is not bottom-centre of %dx%d", f, r.AnchorX, r.AnchorY, r.W, r.H)
		}
		next := a.Regions[effectName("alarm", (f+1)%EffectFrameCount("alarm"))]
		if samePixels(a, r, next) {
			t.Errorf("alarm frames %d and %d are identical", f, f+1)
		}
		if reds := redPixels(a, r); reds < 30 {
			t.Errorf("alarm %d: only %d red pixels", f, reds)
		}
	}
}

func redPixels(a Atlas, r Region) int {
	n := 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			c := a.Image.NRGBAAt(x, y)
			if c.A > 0 && int(c.R) > int(c.G)+60 && int(c.R) > int(c.B)+60 {
				n++
			}
		}
	}
	return n
}

func TestCaesarIconsAreDistinct(t *testing.T) {
	a := built().Icons
	names := append(append([]string{}, caesarIcons...), IconNames...)
	for i, x := range names {
		for _, y := range names[i+1:] {
			if x != y && samePixels(a, a.Regions[iconName(x)], a.Regions[iconName(y)]) {
				t.Errorf("icons %s and %s are identical", x, y)
			}
		}
	}
}

func TestCouncilPortraitsAreDistinctBusts(t *testing.T) {
	a := built().Portraits
	for i, x := range councilFaces {
		for _, y := range councilFaces[i+1:] {
			if differingShare(a, a.Regions[portraitName(x)], a.Regions[portraitName(y)]) < 0.1 {
				t.Errorf("portraits %s and %s barely differ", x, y)
			}
		}
	}
	for _, k := range []string{"quartermaster", "wayfinder"} {
		if samePixels(a, a.Regions[portraitName("advisor-"+k)], a.Regions[portraitName(k)]) {
			t.Errorf("advisor-%s is the building portrait", k)
		}
	}
}

func TestPortraitNamesAreUnique(t *testing.T) {
	seen := map[string]int{}
	for _, n := range portraitEntries() {
		seen[n]++
	}
	for n, c := range seen {
		if c > 1 {
			t.Errorf("portrait %s is painted %d times", n, c)
		}
	}
	if !strings.HasPrefix(portraitName("x"), "portrait/") {
		t.Fatal("portrait prefix")
	}
}

func BenchmarkCaesarSprites(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ab := &atlasBuilder{}
		addQoSHearths(ab)
		addMidden(ab)
		addCodexBloat(ab)
		ab.build()
	}
}
