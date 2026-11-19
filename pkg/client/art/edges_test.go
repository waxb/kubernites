package art

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"testing"
)

var edgeBleeds = map[string]string{
	"sea/":            "sea tiles tile the water edge to edge",
	"ground/tint/":    "ground tints fill a whole tile diamond",
	"ui/":             "nine-slice panels and bars stretch edge to edge",
	"portrait/":       "portraits are framed cards that fill their square",
	"icon/":           "HUD glyphs fill their 16 pixel cell",
	"page/":           "Grimoire page glyphs fill their cell",
	"appicon":         "the app icon is a full square",
	"kmap/parchment":  "the parchment tiles the knowledge map",
	"kmap/road/":      "road pieces join their neighbours at the cell edges",
	"kmap/medallion/": "medallion discs fill their cell",
}

func bleedsByDesign(name string) bool {
	for prefix := range edgeBleeds {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func touchedEdges(img *image.NRGBA, r Region) string {
	sides := []struct {
		tag   string
		n     int
		pixel func(i int) (int, int)
	}{
		{"top", r.W, func(i int) (int, int) { return r.X + i, r.Y }},
		{"bottom", r.W, func(i int) (int, int) { return r.X + i, r.Y + r.H - 1 }},
		{"left", r.H, func(i int) (int, int) { return r.X, r.Y + i }},
		{"right", r.H, func(i int) (int, int) { return r.X + r.W - 1, r.Y + i }},
	}
	out := []string{}
	for _, s := range sides {
		for i := 0; i < s.n; i++ {
			if x, y := s.pixel(i); img.NRGBAAt(x, y).A > 0 {
				out = append(out, s.tag)
				break
			}
		}
	}
	return strings.Join(out, "+")
}

func atlasOffenders(set Set) []string {
	out := []string{}
	for _, a := range []Atlas{set.Terrain, set.Buildings, set.Walkers, set.Effects, set.Icons, set.Portraits, set.UI} {
		for name, r := range a.Regions {
			if bleedsByDesign(name) {
				continue
			}
			if edges := touchedEdges(a.Image, r); edges != "" {
				out = append(out, name+" ("+edges+")")
			}
		}
	}
	sort.Strings(out)
	return out
}

func keepOffenders() []string {
	out := []string{}
	for i, size := range KeepSizes {
		for j, class := range KeepClasses {
			for age := 0; age < AgeTiers; age++ {
				k := KeepLookOf(size, class, fmt.Sprintf("edge-%d-%d-%d", i, j, age), age)
				for _, state := range BuildingStates {
					img, r := KeepSprite(k, state)
					if edges := touchedEdges(img, r); edges != "" {
						out = append(out, k.Name(state)+" ("+edges+")")
					}
				}
			}
		}
	}
	return out
}

func TestNoSpriteTouchesItsCanvasEdge(t *testing.T) {
	offenders := append(atlasOffenders(built()), keepOffenders()...)
	if len(offenders) > 0 {
		t.Fatalf("%d sprites touch their canvas edge (clipped art):\n%s", len(offenders), strings.Join(offenders, "\n"))
	}
}

func TestTouchedEdgesNamesEverySide(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	r := Region{X: 2, Y: 2, W: 4, H: 4}
	if got := touchedEdges(img, r); got != "" {
		t.Fatalf("an empty sprite touches nothing, got %q", got)
	}
	img.Pix[img.PixOffset(3, 2)+3] = 255
	img.Pix[img.PixOffset(5, 4)+3] = 255
	if got := touchedEdges(img, r); got != "top+right" {
		t.Fatalf("got %q, want top+right", got)
	}
	img.Pix[img.PixOffset(2, 5)+3] = 255
	if got := touchedEdges(img, r); got != "top+bottom+left+right" {
		t.Fatalf("got %q", got)
	}
	if !bleedsByDesign("sea/3") || bleedsByDesign("decor/tree-oak/0") || bleedsByDesign("ambient/stall/0") {
		t.Fatal("only tiles, cells and frames bleed by design")
	}
}
