package art

import (
	"fmt"
	"image"
	"testing"
)

func regionsOf(a Atlas, names []string) []sheetCell {
	return cellsOf(a, names, false)
}

func TestWriteCaesarSheets(t *testing.T) {
	dir := sheetDir(t)
	s := built()
	huts := [][]sheetCell{}
	for _, st := range HearthStates {
		names := []string{}
		for _, q := range HearthQoS {
			for tier := 1; tier <= 3; tier++ {
				names = append(names, hearthQoSName(tier, st, q))
			}
		}
		huts = append(huts, regionsOf(s.Buildings, names))
	}
	writeScaled(t, dir, "caesar-huts", huts, grassBackground, 1, 2)
	midden := [][]sheetCell{}
	for v := 0; v < MiddenVariants; v++ {
		names := []string{}
		for l := 0; l < MiddenLevels; l++ {
			names = append(names, middenName(l, v))
		}
		midden = append(midden, cellsOf(s.Buildings, names, true))
	}
	writeScaled(t, dir, "caesar-midden", midden, grassBackground, 1, 2)
	icons := [][]sheetCell{regionsOf(s.Icons, prefixed("icon/", caesarIcons)), regionsOf(s.Portraits, prefixed("portrait/", councilFaces))}
	alarm := []string{}
	for f := 0; f < EffectFrameCount("alarm"); f++ {
		alarm = append(alarm, effectName("alarm", f))
	}
	icons = append(icons, cellsOf(s.Effects, alarm, true))
	writeScaled(t, dir, "caesar-icons", icons, grassBackground, 1, 2, 4)
}

func prefixed(prefix string, names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = prefix + n
	}
	return out
}

func grassScene(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, grassBackground(x, y))
		}
	}
	return img
}

func headroom(a Atlas, name string) float64 {
	r := a.Regions[name]
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 {
				return float64(r.AnchorY - y)
			}
		}
	}
	return 0
}

func caesarItems(level int) []placedSprite {
	s := built()
	items := []placedSprite{}
	add := func(a Atlas, name string, gx, gy, dy float64) {
		items = append(items, placedSprite{a: a, name: name, gx: gx, gy: gy, dy: dy})
	}
	add(s.Buildings, buildingName("keep-stone", "ready"), 3, 3, 0)
	add(s.Buildings, middenName(level, level%2), 4, 3, 0)
	add(s.Buildings, buildingName("codex", "ready"), 8, 2, 0)
	add(s.Buildings, codexBloatName(level), 8, 2, 0)
	if level == 3 {
		add(s.Effects, effectName("alarm", 1), 8, 2, -headroom(s.Buildings, buildingName("codex", "ready"))-2)
	}
	for i, q := range HearthQoS {
		for tier := 1; tier <= 3; tier++ {
			add(s.Buildings, hearthQoSName(tier, "ready", q), float64(1+tier*2), float64(5+i*2), 0)
		}
	}
	return items
}

func TestWriteCaesarScene(t *testing.T) {
	dir := sheetDir(t)
	for level := 0; level < MiddenLevels; level++ {
		w, h := 330, 260
		ox, oy := 150.0, 70.0
		img := grassScene(w, h)
		drawPlaced(img, built(), caesarItems(level), ox, oy)
		writeSheet(t, dir, fmt.Sprintf("caesar-scene-%d-1x.png", level), img)
		writeSheet(t, dir, fmt.Sprintf("caesar-scene-%d-2x.png", level), upscale(img, 2))
	}
}
