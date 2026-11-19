package art

import (
	"image"
	"sort"
	"testing"
)

type portPlace struct {
	name   string
	gx, gy float64
	lift   float64
	flat   bool
}

const (
	portCellW = 250
	portCellH = 190
)

func portScene(level, age int) []portPlace {
	out := []portPlace{}
	for j := 0; j < 3; j++ {
		for i := -2; i <= 2; i++ {
			out = append(out, portPlace{name: PortLook{Kind: "deck", Level: level, Age: age, Variant: (i + j + 9) % 3}.Name(), gx: -float64(j) + 0.5, gy: float64(i) + 0.5, flat: true})
		}
	}
	for k := 1; k <= 2+level; k++ {
		out = append(out, portPlace{name: PortLook{Kind: "pier-x", Level: level, Age: age, Variant: k % 3}.Name(), gx: float64(k) + 0.5, gy: 0.5, flat: true})
	}
	deck := func(kind string, gx, gy float64) portPlace {
		return portPlace{name: PortLook{Kind: kind, Level: level, Age: age}.Name(), gx: gx, gy: gy, lift: HavenDeck}
	}
	out = append(out, deck("sign", 0.5, 2.5), deck("house", -1.5, 2.5), deck("bollard", 2.5+float64(level), 0.1), deck("bollard", 2.5+float64(level), 0.9))
	if level >= 2 {
		out = append(out, deck("crane", -0.5, -1))
	}
	if level >= 3 {
		w := PortLook{Kind: "warehouse", Level: level, Age: age, Variant: 1}.Name()
		out = append(out, portPlace{name: w, gx: -3, gy: -1}, portPlace{name: w, gx: -3, gy: 2})
	}
	if level >= 4 {
		out = append(out, deck("lighthouse", 1, -1))
	}
	return out
}

func portCell(level, age int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, portCellW, portCellH))
	for y := 0; y < portCellH; y++ {
		for x := 0; x < portCellW; x++ {
			img.SetNRGBA(x, y, seaColor(x, y, 0))
		}
	}
	shore := newCanvas(portCellW, portCellH)
	s := iso{c: shore, ox: 120, oy: 110}
	s.box(-6, -4, 0.02, 5, -1, 0, faces{top: flat(sandR[2]), left: flat(sandR[1]), right: flat(sandR[0])})
	blitCanvas(img, shore, 0, 0)
	places := portScene(level, age)
	sort.SliceStable(places, func(i, j int) bool {
		if places[i].flat != places[j].flat {
			return places[i].flat
		}
		return places[i].gx+places[i].gy < places[j].gx+places[j].gy
	})
	for _, p := range places {
		sprite, r, _ := PortSpriteByName(p.name)
		at := s.p(p.gx, p.gy, p.lift)
		blitCanvas(img, &canvas{img: sprite, w: r.W, h: r.H}, int(at.x)-r.AnchorX, int(at.y)-r.AnchorY)
	}
	return img
}

func blitCanvas(dst *image.NRGBA, c *canvas, ox, oy int) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			src := c.img.NRGBAAt(x, y)
			if src.A == 0 || !image.Pt(ox+x, oy+y).In(dst.Bounds()) {
				continue
			}
			m := mix(dst.NRGBAAt(ox+x, oy+y), src, float64(src.A)/255)
			m.A = 255
			dst.SetNRGBA(ox+x, oy+y, m)
		}
	}
}

func TestWritePortSheet(t *testing.T) {
	dir := sheetDir(t)
	sheet := image.NewNRGBA(image.Rect(0, 0, portCellW*AgeTiers, portCellH*PortLevels))
	for level := 1; level <= PortLevels; level++ {
		for age := 0; age < AgeTiers; age++ {
			blitCanvas(sheet, &canvas{img: portCell(level, age), w: portCellW, h: portCellH}, age*portCellW, (level-1)*portCellH)
		}
	}
	writeSheet(t, dir, "ports-1x.png", sheet)
	writeSheet(t, dir, "ports-2x.png", upscale(sheet, 2))
	ferries := [][]sheetCell{}
	for age := 0; age < AgeTiers; age++ {
		row := []sheetCell{}
		for _, dir := range ShipDirs {
			for _, skiff := range []bool{false, true} {
				img, r := FerrySprite(FerryLook{Skiff: skiff, Age: age, Dir: dir, Frame: 1})
				r.X, r.Y = 0, 0
				row = append(row, sheetCell{a: Atlas{Image: img}, r: r})
			}
		}
		ferries = append(ferries, row)
	}
	writeScaled(t, dir, "ferries", ferries, seaBackground, 1, 3)
}
