package art

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func mockText(dst *image.NRGBA, text string, x, y int, col color.NRGBA) {
	for i, r := range strings.ToUpper(text) {
		g, ok := mockGlyphs[r]
		if !ok {
			continue
		}
		for row, bits := range g {
			for c := 0; c < 5; c++ {
				if bits&(1<<uint(4-c)) != 0 {
					dst.SetNRGBA(x+i*6+c, y+row, col)
				}
			}
		}
	}
}

func blitRegion(dst *image.NRGBA, a Atlas, name string, x, y int) {
	blitOver(dst, sheetCell{a: a, r: a.Regions[name]}, x, y)
}

func blitScaled(dst *image.NRGBA, a Atlas, src, to image.Rectangle) {
	for y := to.Min.Y; y < to.Max.Y; y++ {
		for x := to.Min.X; x < to.Max.X; x++ {
			sx := src.Min.X + (x-to.Min.X)*src.Dx()/max(to.Dx(), 1)
			sy := src.Min.Y + (y-to.Min.Y)*src.Dy()/max(to.Dy(), 1)
			col := a.Image.NRGBAAt(sx, sy)
			if col.A == 0 {
				continue
			}
			m := mix(dst.NRGBAAt(x, y), col, float64(col.A)/255)
			m.A = 255
			dst.SetNRGBA(x, y, m)
		}
	}
}

func nineSlice(dst *image.NRGBA, a Atlas, name string, box image.Rectangle) {
	r := a.Regions[name]
	s := r.Slice
	sx := []int{r.X, r.X + s, r.X + r.W - s, r.X + r.W}
	sy := []int{r.Y, r.Y + s, r.Y + r.H - s, r.Y + r.H}
	dx := []int{box.Min.X, box.Min.X + s, box.Max.X - s, box.Max.X}
	dy := []int{box.Min.Y, box.Min.Y + s, box.Max.Y - s, box.Max.Y}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			blitScaled(dst, a, image.Rect(sx[i], sy[j], sx[i+1], sy[j+1]), image.Rect(dx[i], dy[j], dx[i+1], dy[j+1]))
		}
	}
}

type mockPage struct {
	id, line1, line2, state string
}

var mockRows = [][]mockPage{
	{{"depots", "Depots", "Service", "learned"}, {"carters", "Carters", "Ingress", "researching"}, {"gateways", "Gateways", "Gateway API", "locked"}},
	{{"territory", "Territory", "Nodes", "learned"}, {"ranks", "Rank", "PriorityClass", "available"}, {"rebalancer", "Rebalancer", "Descheduler", "locked"}},
	{{"crews", "Crews", "Deployment", "learned"}, {"war-horns", "War Horns", "HPA", "learned"}, {"budgets", "Budgets", "PDB", "available"}},
	{{"keep-floor", "Keep Floor", "Ephemeral", "available"}, {"palimpsest", "Palimpsest", "etcd defrag", "locked"}, {"squid", "Cache", "Squid", "locked"}},
}

func roadState(a, b string) string {
	switch {
	case a == "learned" && b == "learned":
		return "learned"
	case a == "learned":
		return "open"
	}
	return "locked"
}

func drawMockPlaque(dst *image.NRGBA, s Set, p mockPage, x, y int) {
	nineSlice(dst, s.UI, uiName(PlaqueName(p.state)), image.Rect(x, y, x+132, y+PlaqueH))
	blitRegion(dst, s.Icons, KMapMedallionName(p.state), x+5, y+5)
	blitRegion(dst, s.Icons, PageIconName(p.id), x+8, y+8)
	col := PlaqueText[p.state]
	mockText(dst, p.line1, x+40, y+9, col)
	mockText(dst, p.line2, x+40, y+22, mix(col, rgb(74, 137, 156), 0.35))
}

func mockKnowledgeMap() *image.NRGBA {
	s := built()
	w, h := 600, 320
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 12; y < h-12; y += KMapParchment {
		for x := 12; x < w-12; x += KMapParchment {
			blitRegion(img, s.Icons, KMapParchmentName(), x, y)
		}
	}
	nineSlice(img, s.UI, uiName("kmap-frame"), image.Rect(0, 0, w, h))
	for i, d := range [][3]int{{0, 520, 20}, {1, 470, 250}, {2, 250, 262}, {3, 380, 270}, {4, 540, 200}, {3, 90, 284}} {
		blitRegion(img, s.Icons, KMapDecorName(KMapDecor[d[0]]), d[1], d[2]+i%2)
	}
	for row, pages := range mockRows {
		y := 24 + row*70
		blitRegion(img, s.Icons, KMapBannerName(KMapBranches[row]), 22, y-6)
		for i, p := range pages {
			x := 64 + i*160
			if i > 0 {
				for rx := x - 28 - 16; rx < x+4; rx += KMapCell {
					blitRegion(img, s.Icons, KMapRoadName(roadState(pages[i-1].state, p.state), "ew"), rx, y+12)
				}
			}
			drawMockPlaque(img, s, p, x, y)
		}
	}
	blitRegion(img, s.Icons, KMapRoadName("open", "ns"), 120, 64)
	blitRegion(img, s.Icons, KMapRoadName("open", "ns"), 120, 80)
	return img
}

func TestWriteKnowledgeMapSheets(t *testing.T) {
	dir := sheetDir(t)
	s := built()
	img := mockKnowledgeMap()
	writeSheet(t, dir, "kmap-mock-1x.png", img)
	writeSheet(t, dir, "kmap-mock-2x.png", upscale(img, 2))
	pages := [][]sheetCell{}
	for i := 0; i < len(PageIcons); i += 11 {
		pages = append(pages, cellsOf(s.Icons, prefixed("page/", PageIcons[i:min(i+11, len(PageIcons))]), false))
	}
	writeScaled(t, dir, "kmap-pages", pages, func(x, y int) color.NRGBA { return parchmentAt(x, y) }, 1, 3)
	parts := [][]sheetCell{}
	for _, st := range KMapRoadStates {
		parts = append(parts, cellsOf(s.Icons, prefixed("kmap/road/"+st+"/", KMapRoadMasks), false))
	}
	parts = append(parts, cellsOf(s.Icons, prefixed("kmap/medallion/", KMapStates), false))
	parts = append(parts, cellsOf(s.Icons, prefixed("kmap/banner/", KMapBranches), false))
	parts = append(parts, cellsOf(s.UI, prefixed("ui/plaque-", KMapStates), false))
	parts = append(parts, cellsOf(s.Icons, prefixed("icon/", FreightIcons), false))
	writeScaled(t, dir, "kmap-parts", parts, func(x, y int) color.NRGBA { return parchmentAt(x, y) }, 3)
}
