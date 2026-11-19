package art

import (
	"image/color"
	"math"
	"strings"
	"testing"
)

func pixelAt(a Atlas, r Region, x, y int) color.NRGBA { return a.Image.NRGBAAt(r.X+x, r.Y+y) }

func TestKnowledgeMapContractSizes(t *testing.T) {
	s := built()
	cases := []struct {
		names        []string
		a            Atlas
		w, h, ax, ay int
	}{
		{prefixed("page/", PageIcons), s.Icons, PageIconSize, PageIconSize, PageIconSize / 2, PageIconSize / 2},
		{prefixed("kmap/medallion/", KMapStates), s.Icons, MedallionSize, MedallionSize, 0, 0},
		{prefixed("kmap/banner/", KMapBranches), s.Icons, BannerW + 2*kmapPad, BannerH + 2*kmapPad, kmapPad, kmapPad},
		{[]string{KMapParchmentName()}, s.Icons, KMapParchment, KMapParchment, 0, 0},
	}
	for _, st := range KMapRoadStates {
		cases = append(cases, struct {
			names        []string
			a            Atlas
			w, h, ax, ay int
		}{prefixed("kmap/road/"+st+"/", KMapRoadMasks), s.Icons, KMapCell, KMapCell, 0, 0})
	}
	for _, c := range cases {
		for _, n := range c.names {
			r, ok := c.a.Regions[n]
			if !ok || r.W != c.w || r.H != c.h || r.AnchorX != c.ax || r.AnchorY != c.ay {
				t.Errorf("%s: %+v (present %v)", n, r, ok)
			}
		}
	}
	for _, st := range KMapStates {
		r := s.UI.Regions[uiName(PlaqueName(st))]
		if r.W != PlaqueW || r.H != PlaqueH || r.Slice != PlaqueSlice {
			t.Errorf("plaque %s: %+v", st, r)
		}
	}
	for _, d := range KMapDecor {
		if _, ok := s.Icons.Regions[KMapDecorName(d)]; !ok {
			t.Errorf("decor %s missing", d)
		}
	}
}

func roadRows(a Atlas, r Region, x int) string {
	var b strings.Builder
	for y := 0; y < r.H; y++ {
		if pixelAt(a, r, x, y).A > 0 {
			b.WriteByte('#')
		} else {
			b.WriteByte('.')
		}
	}
	return b.String()
}

func roadCols(a Atlas, r Region, y int) string {
	var b strings.Builder
	for x := 0; x < r.W; x++ {
		if pixelAt(a, r, x, y).A > 0 {
			b.WriteByte('#')
		} else {
			b.WriteByte('.')
		}
	}
	return b.String()
}

func roadSides(a Atlas, r Region) map[byte]string {
	return map[byte]string{'e': roadRows(a, r, KMapCell-1), 'w': roadRows(a, r, 0), 'n': roadCols(a, r, 0), 's': roadCols(a, r, KMapCell-1)}
}

func TestRoadTilesMeetOnlyWhereTheyConnect(t *testing.T) {
	a := built().Icons
	empty := strings.Repeat(".", KMapCell)
	for _, st := range KMapRoadStates {
		ref := roadSides(a, a.Regions[KMapRoadName(st, "ew")])
		for side, v := range roadSides(a, a.Regions[KMapRoadName(st, "ns")]) {
			if side == 'n' || side == 's' {
				ref[side] = v
			}
		}
		for _, m := range KMapRoadMasks {
			got := roadSides(a, a.Regions[KMapRoadName(st, m)])
			for _, side := range []byte("nesw") {
				want := empty
				if strings.IndexByte(m, side) >= 0 {
					want = ref[side]
				}
				if got[side] != want {
					t.Errorf("%s/%s side %c: %s, want %s", st, m, side, got[side], want)
				}
			}
		}
		if ref['w'] == empty || ref['n'] == empty {
			t.Errorf("%s road leaves no mark on its edges", st)
		}
	}
}

func TestRoadTilesRepeatSeamlessly(t *testing.T) {
	a := built().Icons
	for _, st := range KMapRoadStates {
		ew, ns := a.Regions[KMapRoadName(st, "ew")], a.Regions[KMapRoadName(st, "ns")]
		for y := 0; y < KMapCell; y++ {
			for x := 0; x < KMapCell; x++ {
				if x < KMapCell/2 && pixelAt(a, ew, x, y) != pixelAt(a, ew, x+KMapCell/2, y) {
					t.Fatalf("%s ew is not periodic at %d,%d", st, x, y)
				}
				if y < KMapCell/2 && pixelAt(a, ns, x, y) != pixelAt(a, ns, x, y+KMapCell/2) {
					t.Fatalf("%s ns is not periodic at %d,%d", st, x, y)
				}
			}
		}
	}
}

func luminance(c color.NRGBA) float64 {
	ch := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

func contrast(a, b color.NRGBA) float64 {
	la, lb := luminance(a), luminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

func TestPlaquesStretchCleanlyAndStayLegible(t *testing.T) {
	a := built().UI
	for _, st := range KMapStates {
		r := a.Regions[uiName(PlaqueName(st))]
		fill := pixelAt(a, r, r.W/2, r.H/2)
		for y := r.Slice; y < r.H-r.Slice; y++ {
			for x := r.Slice; x < r.W-r.Slice; x++ {
				if pixelAt(a, r, x, y) != fill {
					t.Fatalf("%s: the centre is not flat at %d,%d", st, x, y)
				}
			}
		}
		for y := 0; y < r.H; y++ {
			for x := r.Slice; x < r.W-r.Slice; x++ {
				if (y < r.Slice || y >= r.H-r.Slice) && pixelAt(a, r, x, y) != pixelAt(a, r, r.Slice, y) {
					t.Fatalf("%s: the top or bottom edge varies at %d,%d", st, x, y)
				}
			}
		}
		if c := contrast(PlaqueText[st], fill); c < 4.5 {
			t.Errorf("%s: text contrast %.1f", st, c)
		}
	}
	for i, x := range KMapStates {
		for _, y := range KMapStates[i+1:] {
			if samePixels(a, a.Regions[uiName(PlaqueName(x))], a.Regions[uiName(PlaqueName(y))]) {
				t.Errorf("plaques %s and %s are identical", x, y)
			}
		}
	}
}

func TestMapFrameIsOpenInTheMiddle(t *testing.T) {
	a := built().UI
	r := a.Regions[uiName("kmap-frame")]
	if r.Slice != KMapFrameSlice || pixelAt(a, r, r.W/2, r.H/2).A != 0 || pixelAt(a, r, 2, r.H/2).A == 0 {
		t.Fatalf("frame %+v", r)
	}
	for x := r.Slice; x < r.W-r.Slice; x++ {
		for y := 0; y < r.Slice; y++ {
			if pixelAt(a, r, x, y) != pixelAt(a, r, r.Slice, y) {
				t.Fatalf("the frame's top edge varies at %d,%d", x, y)
			}
		}
	}
}

func TestParchmentTilesWrap(t *testing.T) {
	a := built().Icons
	r := a.Regions[KMapParchmentName()]
	seam, inner := 0.0, 0.0
	for y := 0; y < r.H; y++ {
		seam += math.Abs(float64(pixelAt(a, r, r.W-1, y).R) - float64(pixelAt(a, r, 0, y).R))
		inner += math.Abs(float64(pixelAt(a, r, 31, y).R) - float64(pixelAt(a, r, 32, y).R))
		if pixelAt(a, r, 5, y).A != 255 {
			t.Fatal("parchment has holes")
		}
	}
	if seam > inner*2+float64(r.H) {
		t.Errorf("the wrap seam differs by %.0f, the inside by %.0f", seam, inner)
	}
}

func TestPageIconsAreDistinct(t *testing.T) {
	a := built().Icons
	if len(PageIcons) < 24 {
		t.Fatalf("only %d page icons", len(PageIcons))
	}
	for i, x := range PageIcons {
		r := a.Regions[PageIconName(x)]
		if share := float64(opaquePixels(a, r)) / float64(r.W*r.H); share < 0.2 {
			t.Errorf("page %s fills only %.0f%%", x, share*100)
		}
		for _, y := range PageIcons[i+1:] {
			if d := differingShare(a, r, a.Regions[PageIconName(y)]); d < 0.3 {
				t.Errorf("pages %s and %s differ in only %.0f%%", x, y, d*100)
			}
		}
	}
	for i, x := range KMapBranches {
		for _, y := range KMapBranches[i+1:] {
			if differingShare(a, a.Regions[KMapBannerName(x)], a.Regions[KMapBannerName(y)]) < 0.3 {
				t.Errorf("banners %s and %s look alike", x, y)
			}
		}
	}
	if KMapRoadMask(true, true, true, true) != "nesw" || KMapRoadMask(false, true, false, true) != "ew" || KMapRoadMask(false, false, false, false) != "" {
		t.Error("road masks")
	}
}
