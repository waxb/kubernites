package art

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type sheetCell struct {
	a      Atlas
	r      Region
	anchor bool
	layers []string
	under  []string
}

func sheetDir(t *testing.T) string {
	dir := os.Getenv("ART_SHEETS")
	if dir == "" {
		t.Skip("set ART_SHEETS=<dir> to write contact sheets")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}
	return dir
}

func seaBackground(x, y int) color.NRGBA { return seaColor(x, y, 0) }

func grassBackground(x, y int) color.NRGBA {
	if hash(x, y, 5)%9 == 0 {
		return grassR[2]
	}
	return grassR[1]
}

func drawSheet(rows [][]sheetCell, k int, bg func(x, y int) color.NRGBA) *image.NRGBA {
	gap := 4
	w, h := gap, gap
	for _, row := range rows {
		rw, rh := gap, 0
		for _, c := range row {
			rw += c.r.W + gap
			rh = max(rh, c.r.H)
		}
		w, h = max(w, rw), h+rh+gap
	}
	base := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			base.SetNRGBA(x, y, bg(x, y))
		}
	}
	y := gap
	for _, row := range rows {
		x, rh := gap, 0
		for _, c := range row {
			blitOver(base, c, x, y)
			x += c.r.W + gap
			rh = max(rh, c.r.H)
		}
		y += rh + gap
	}
	return upscale(base, k)
}

func blitOver(dst *image.NRGBA, c sheetCell, x, y int) {
	for _, name := range c.under {
		r := c.a.Regions[name]
		blitOver(dst, sheetCell{a: c.a, r: r}, x+c.r.AnchorX-r.AnchorX, y+c.r.AnchorY-r.AnchorY)
	}
	for yy := 0; yy < c.r.H; yy++ {
		for xx := 0; xx < c.r.W; xx++ {
			src := c.a.Image.NRGBAAt(c.r.X+xx, c.r.Y+yy)
			if src.A == 0 {
				continue
			}
			under := dst.NRGBAAt(x+xx, y+yy)
			m := mix(under, src, float64(src.A)/255)
			m.A = 255
			dst.SetNRGBA(x+xx, y+yy, m)
		}
	}
	for _, name := range c.layers {
		r := c.a.Regions[name]
		blitOver(dst, sheetCell{a: c.a, r: r}, x+c.r.AnchorX-r.AnchorX, y+c.r.AnchorY-r.AnchorY)
	}
	if c.anchor {
		dst.SetNRGBA(x+c.r.AnchorX, y+c.r.AnchorY, color.NRGBA{255, 0, 255, 255})
	}
}

func upscale(src *image.NRGBA, k int) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := 0; y < b.Dy()*k; y++ {
		for x := 0; x < b.Dx()*k; x++ {
			out.SetNRGBA(x, y, src.NRGBAAt(x/k, y/k))
		}
	}
	return out
}

func writeSheet(t *testing.T, dir, name string, img image.Image) {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
}

func cellsOf(a Atlas, names []string, anchor bool) []sheetCell {
	out := make([]sheetCell, 0, len(names))
	for _, n := range names {
		if r, ok := a.Regions[n]; ok {
			out = append(out, sheetCell{a: a, r: r, anchor: anchor})
		}
	}
	return out
}

func sandBackground(x, y int) color.NRGBA {
	if hash(x, y, 9)%7 == 0 {
		return sandR[1]
	}
	return sandR[2]
}

func writeScaled(t *testing.T, dir, stem string, rows [][]sheetCell, bg func(x, y int) color.NRGBA, scales ...int) {
	for _, k := range scales {
		writeSheet(t, dir, fmt.Sprintf("%s-%dx.png", stem, k), drawSheet(rows, k, bg))
	}
}

func TestWriteShipSheets(t *testing.T) {
	dir := sheetDir(t)
	a := built().Effects
	rows := [][]sheetCell{}
	for _, class := range ShipClasses {
		names := []string{}
		for v := 0; v < ShipVariants; v++ {
			names = append(names, shipName(class, v, "se", 0))
		}
		rows = append(rows, cellsOf(a, names, false))
		names = names[:0]
		for _, d := range ShipDirs {
			names = append(names, shipName(class, 1, d, 0))
		}
		rows = append(rows, cellsOf(a, names, false))
	}
	writeScaled(t, dir, "ships", rows, seaBackground, 1, 2)
	for _, class := range ShipClasses {
		frames := [][]sheetCell{}
		for _, d := range ShipDirs {
			names := []string{}
			for f := 0; f < ShipFrames; f++ {
				names = append(names, shipName(class, 3, d, f))
			}
			frames = append(frames, cellsOf(a, names, true))
		}
		writeScaled(t, dir, "frames-"+class, frames, seaBackground, 3)
	}
}

func TestWriteHarborSheets(t *testing.T) {
	dir := sheetDir(t)
	s := built()
	pieces := [][]sheetCell{}
	for _, group := range [][]string{{"pier-x", "pier-y", "quay"}, {"bollard", "crates", "sign", "crane"}, {"warehouse", "lighthouse"}} {
		names := []string{}
		for _, k := range group {
			for v := 0; v < HavenVariants(k); v++ {
				names = append(names, havenName(k, v))
			}
		}
		pieces = append(pieces, cellsOf(s.Buildings, names, false))
	}
	writeScaled(t, dir, "haven", pieces, sandBackground, 1, 2)
	fx := [][]sheetCell{}
	for _, k := range []string{"wake", "request"} {
		names := []string{}
		for f := 0; f < EffectFrames; f++ {
			names = append(names, effectName(k, f))
		}
		fx = append(fx, cellsOf(s.Effects, names, true))
	}
	writeScaled(t, dir, "fx", fx, seaBackground, 2, 4)
	people := [][]sheetCell{}
	for _, pose := range [][2]string{{"idle", "se"}, {"walk", "sw"}} {
		row := []sheetCell{}
		for v := 0; v < WalkerLooks; v++ {
			row = append(row, sheetCell{a: s.Walkers, r: pilgrimPose(s.Walkers, v, pose[0], pose[1])})
		}
		people = append(people, row)
	}
	writeScaled(t, dir, "pilgrims", people, sandBackground, 2, 4)
}

func pilgrimPose(a Atlas, v int, action, dir string) Region {
	if v == 0 {
		return a.Regions[walkerName("pilgrim", action, dir, 1)]
	}
	return a.Regions[walkerLookName("pilgrim", 1, v, action, dir, 1)]
}

type placedSprite struct {
	a      Atlas
	name   string
	gx, gy float64
	lift   float64
	dy     float64
}

func findAtlas(s Set, name string) Atlas {
	for _, a := range atlases(s) {
		if _, ok := a.Regions[name]; ok {
			return a
		}
	}
	return Atlas{}
}

func sceneCanvas(w, h int, land func(gx, gy int) bool, ox, oy float64) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := (float64(x) + 0.5 - ox) / 16
			t := (float64(y) + 0.5 - oy) / 8
			gx, gy := (t+d)/2, (t-d)/2
			col := seaColor(x, y, 0)
			if land(int(math.Floor(gx)), int(math.Floor(gy))) {
				col = sandR[2]
				if hash(x, y, 3)%7 == 0 {
					col = sandR[1]
				}
			}
			img.SetNRGBA(x, y, col)
		}
	}
	return img
}

func drawPlaced(img *image.NRGBA, s Set, items []placedSprite, ox, oy float64) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].gx+items[i].gy < items[j].gx+items[j].gy })
	for _, it := range items {
		a := findAtlas(s, it.name)
		r, ok := a.Regions[it.name]
		if !ok {
			continue
		}
		sx := ox + (it.gx-it.gy)*16
		sy := oy + (it.gx+it.gy)*8 - it.lift + it.dy
		blitOver(img, sheetCell{a: a, r: r}, int(math.Round(sx))-r.AnchorX, int(math.Round(sy))-r.AnchorY)
	}
}

func harborItems(frame int) []placedSprite {
	s := built()
	items := []placedSprite{}
	add := func(a Atlas, name string, gx, gy, lift, dy float64) {
		items = append(items, placedSprite{a: a, name: name, gx: gx, gy: gy, lift: lift, dy: dy})
	}
	for i := 0; i < 5; i++ {
		add(s.Buildings, havenName("quay", i%3), float64(i)+0.5, 4.5, 0, 0)
		add(s.Buildings, havenName("quay", (i+1)%3), float64(i)+0.5, 3.5, 0, 0)
	}
	for i := 0; i < 4; i++ {
		add(s.Buildings, havenName("pier-y", i%3), 2.5, 5.5+float64(i), 0, 0)
	}
	add(s.Buildings, havenName("warehouse", 0), 2, 2, 0, 0)
	add(s.Buildings, havenName("warehouse", 1), 5, 2, 0, 0)
	add(s.Buildings, havenName("lighthouse", 0), 7, 3, 0, 0)
	add(s.Buildings, havenName("crane", 0), 1, 4.02, 0, 0)
	add(s.Buildings, havenName("crates", 0), 3.3, 3.4, HavenDeck, 0)
	add(s.Buildings, havenName("crates", 1), 4.4, 3.6, HavenDeck, 0)
	add(s.Buildings, havenName("bollard", 0), 2.85, 7.5, HavenDeck, 0)
	add(s.Buildings, havenName("sign", 0), 4.8, 4.6, HavenDeck, 0)
	for i, p := range [][2]float64{{0.8, 4.2}, {1.5, 4.5}, {2.2, 4.3}, {3.6, 4.6}, {4.2, 4.2}} {
		add(s.Walkers, walkerLookName("pilgrim", 1, i+1, "idle", []string{"se", "sw"}[i%2], frame%4), p[0], p[1], HavenDeck, 0)
		add(s.Effects, effectName("request", frame%6), p[0], p[1], HavenDeck, -21)
	}
	add(s.Effects, effectName("wake", frame%6), 4.3, 8.1, 0, 0)
	add(s.Effects, shipName("caravel", 2, "sw", frame%4), 4.3, 8.1, 0, 0)
	add(s.Effects, effectName("wake", (frame+3)%6), 0.7, 9.5, 0, 0)
	add(s.Effects, shipName("cog", 5, "ne", frame%4), 0.7, 9.5, 0, 0)
	add(s.Effects, shipName("galleon", 6, "se", frame%4), 7.5, 7.2, 0, 0)
	add(s.Effects, shipName("carrack", 3, "nw", frame%4), 5.2, 12.8, 0, 0)
	return items
}

func TestWriteHarborScene(t *testing.T) {
	dir := sheetDir(t)
	w, h := 360, 250
	ox, oy := 150.0, 40.0
	land := func(gx, gy int) bool { return gy <= 4 && gx >= -2 && gx <= 9 && gy >= -3 }
	for f := 0; f < 2; f++ {
		img := sceneCanvas(w, h, land, ox, oy)
		drawPlaced(img, built(), harborItems(f), ox, oy)
		writeSheet(t, dir, fmt.Sprintf("harbor-%d-1x.png", f), img)
		writeSheet(t, dir, fmt.Sprintf("harbor-%d-2x.png", f), upscale(img, 2))
	}
}
