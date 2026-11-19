package hud

import "testing"

func TestTheBannerClothGrowsWithItsTitle(t *testing.T) {
	if w := bannerWidthFor(50); w != bannerMinW {
		t.Errorf("a short title gets the snug banner, got %d", w)
	}
	if w := bannerWidthFor(bannerBaseW - bannerMargin); w != bannerBaseW {
		t.Errorf("a title that fills the classic banner keeps it, got %d", w)
	}
	if w := bannerWidthFor(bannerBaseW - bannerMargin + 1); w != bannerBaseW+bannerTileW {
		t.Errorf("a title one pixel too wide adds one tile, got %d", w)
	}
	if w := bannerWidthFor(5000); w != bannerMaxW {
		t.Errorf("the banner stops growing at %d, got %d", bannerMaxW, w)
	}
	for _, w := range []int{bannerMinW, bannerBaseW, bannerBaseW + 2*bannerTileW} {
		parts, covered := bannerParts(w), 0
		for i, p := range parts {
			if p.dx != covered {
				t.Fatalf("width %d: part %d starts at %d, want %d", w, i, p.dx, covered)
			}
			covered += p.w
		}
		if covered != w {
			t.Errorf("width %d: parts cover %d", w, covered)
		}
	}
}
