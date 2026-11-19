package art

import (
	"image"
	"image/color"
	"math"
	"sort"
)

const (
	LivingTileW     = 32
	LivingTileH     = 16
	LivingRise      = 7
	WheatLevels     = 5
	BushFruits      = 4
	CairnStages     = 4
	RubbleStages    = 5
	BoardSlots      = 6
	LivestockFrames = 4
)

type Piece struct {
	Img    *image.NRGBA
	AX, AY int
}

var (
	wheatRamps = [WheatLevels][3]color.NRGBA{
		{rgb(168, 138, 84), rgb(196, 164, 100), rgb(214, 186, 120)},
		{rgb(70, 124, 54), rgb(104, 160, 66), rgb(144, 190, 84)},
		{rgb(92, 146, 56), rgb(132, 182, 70), rgb(176, 208, 92)},
		{rgb(136, 160, 56), rgb(190, 190, 70), rgb(226, 208, 96)},
		{rgb(170, 116, 34), rgb(222, 164, 48), rgb(250, 214, 96)},
	}
	soilRamp = ramp{rgb(84, 58, 36), rgb(112, 78, 46), rgb(132, 92, 54), rgb(156, 112, 66)}
)

func piece(c *canvas, ax, ay int) Piece { return Piece{Img: c.img, AX: ax, AY: ay} }

func tileShare(x, y int) float64 {
	cx, cy := float64(LivingTileW-1)/2, float64(LivingTileH-1)/2
	return math.Abs(float64(x)-cx)/(cx+.5) + math.Abs(float64(y)-cy)/(cy+.5)
}

func PaintWheat(level, variant int) Piece {
	level = min(max(level, 0), WheatLevels-1)
	c := newCanvas(LivingTileW, LivingTileH+LivingRise)
	for y := 0; y < LivingTileH; y++ {
		for x := 0; x < LivingTileW; x++ {
			d := tileShare(x, y)
			if d > 1 {
				continue
			}
			col := soilRamp[1]
			if (x+2*y+variant)%4 == 0 {
				col = soilRamp[0]
			}
			if d > .88 {
				col = mix(col, soilRamp[0], .5)
			}
			c.dot(x, y+LivingRise, col)
		}
	}
	for y := 2; y < LivingTileH-2; y += 3 {
		for x := 3; x < LivingTileW-3; x += 4 {
			px := x + (y/3%2)*2 + variant%2
			if tileShare(px, y) > .78 {
				continue
			}
			wheatStalks(c, level, px, y+LivingRise, int(hash(px, y, 71+variant)%3))
		}
	}
	return piece(c, LivingTileW/2, LivingTileH/2+LivingRise)
}

func wheatStalks(c *canvas, level, x, y, jitter int) {
	r := wheatRamps[level]
	if level == 0 {
		c.dot(x, y-1, r[1])
		c.dot(x+1, y, r[0])
		return
	}
	height := []int{0, 2, 3, 4, 5}[level]
	for i := 0; i < height; i++ {
		c.dot(x, y-i, r[(i*2/height)%2])
		if level >= 2 && i > 0 && i < height-1 && jitter == 0 {
			c.dot(x+1, y-i, r[0])
		}
	}
	c.dot(x, y-height, r[2])
	if level >= 3 {
		c.dot(x+1, y-height+1, r[2])
		c.dot(x-1, y-height+1, r[1])
	}
	if level == 4 {
		c.dot(x, y-height-1, r[2])
	}
}

var (
	berryRed    = rgb(206, 52, 66)
	berryDark   = rgb(124, 28, 52)
	twigBrown   = rgb(108, 78, 50)
	bushWither  = rgb(122, 128, 62)
	bushWither2 = rgb(150, 150, 78)
)

func PaintBush(fruit, variant int) Piece {
	fruit = min(max(fruit, 0), BushFruits-1)
	const w, h = 15, 12
	c := newCanvas(w, h)
	c.ellipse(7.5, 10.5, 6, 1.7, withAlpha(ink, 70))
	bare := fruit == 0
	leaf := greenR
	if bare {
		leaf = ramp{rgb(96, 100, 52), bushWither, bushWither2, rgb(176, 170, 92)}
	}
	for _, b := range [][3]float64{{7.5, 6.5, 5}, {4.5, 7.5, 3.4}, {10.5, 7.5, 3.4}, {7.5, 4.5, 3.4}} {
		c.disc(b[0], b[1], b[2], leaf[1])
	}
	c.outline(ink)
	for y := 1; y < h-3; y++ {
		for x := 2; x < w-2; x++ {
			if c.at(x, y) != leaf[1] {
				continue
			}
			switch n := hash(x, y, 13+variant) % 7; {
			case n == 0:
				c.dot(x, y, leaf[3])
			case n < 3 && y < 6:
				c.dot(x, y, leaf[2])
			case n == 3 && y > 7:
				c.dot(x, y, leaf[0])
			}
		}
	}
	if bare {
		for _, p := range [][2]int{{5, 6}, {7, 4}, {9, 7}, {6, 8}, {10, 5}} {
			c.dot(p[0], p[1], twigBrown)
		}
		return piece(c, w/2, h-2)
	}
	spots := [][2]int{{5, 6}, {9, 5}, {7, 8}, {10, 8}, {4, 8}, {7, 4}}
	if variant%2 == 1 {
		spots = [][2]int{{6, 5}, {10, 6}, {8, 8}, {5, 8}, {11, 8}, {8, 3}}
	}
	for i := 0; i < fruit*2; i++ {
		s := spots[i%len(spots)]
		c.dot(s[0], s[1], berryRed)
		if i%2 == 0 {
			c.dot(s[0], s[1]+1, berryDark)
		}
	}
	return piece(c, w/2, h-2)
}

func PaintCairn(stage, variant int) Piece {
	stage = min(max(stage, 0), CairnStages-1)
	const w, h = 15, 15
	c := newCanvas(w, h)
	c.ellipse(7.5, 12.4, 6.5, 2, withAlpha(ink, 80))
	tiers := [][][3]float64{
		{{4.2, 10.6, 2.9}, {7.6, 11, 3.1}, {11, 10.6, 2.8}},
		{{5.8, 7.6, 2.5}, {9.6, 7.8, 2.5}},
		{{7.5, 5, 2.1}},
		{{7.8, 2.8, 1.5}},
	}
	for ti, tier := range tiers {
		for si, s := range tier {
			shade := stoneR[(ti+si)%3+1]
			c.disc(s[0], s[1], s[2], shade)
			c.dot(int(s[0]-s[2]/2), int(s[1]-s[2]/2), stoneR[3])
			c.dot(int(s[0]+s[2]/2), int(s[1]+s[2]/2), stoneR[0])
		}
	}
	c.outline(ink)
	moss := []color.NRGBA{greenR[1], greenR[2], greenR[0]}
	cover := []float64{0, .2, .48, .78}[stage]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			col := c.at(x, y)
			if col.A == 0 || col == ink || noise(x, y, 31+variant) > cover*(.4+float64(y)/float64(h)) {
				continue
			}
			c.dot(x, y, moss[hash(x, y, 5)%3])
		}
	}
	if stage >= 2 {
		tuft(c, 2, 12, grassR, hash(3, 10, variant))
		tuft(c, 13, 12, grassR, hash(12, 10, variant+1))
	}
	if stage == 3 {
		c.dot(8, 1, rgb(244, 232, 120))
		c.dot(4, 7, white)
		c.dot(11, 8, rgb(240, 150, 170))
	}
	return piece(c, w/2, h-3)
}

func PaintRubble(stage, variant int) Piece {
	stage = min(max(stage, 0), RubbleStages-1)
	const w, h = 60, 34
	c := newCanvas(w, h)
	c.ellipse(30, 22, 26, 9, withAlpha(ink, 70))
	type chunk struct{ x, y, bw, bh, col int }
	chunks := []chunk{}
	for i := 0; i < 30; i++ {
		x := 8 + int(hash(i, 1, 90+variant)%44)
		y := 14 + int(hash(i, 2, 90+variant)%12)
		pile := 1 - math.Abs(float64(x)-30)/24
		rise := int(pile*pile*8) + int(hash(i, 7, 90)%2)
		chunks = append(chunks, chunk{x: x, y: y - rise, bw: 3 + int(hash(i, 3, 90)%5), bh: 2 + int(hash(i, 4, 90)%3), col: i})
	}
	sort.Slice(chunks, func(a, b int) bool { return chunks[a].y < chunks[b].y })
	for _, k := range chunks {
		col := stoneR[1+k.col%3]
		if k.col%5 == 4 {
			col = woodR[1+k.col%2]
		}
		c.rect(k.x-k.bw/2, k.y, k.bw, k.bh, col)
		c.rect(k.x-k.bw/2, k.y, k.bw, 1, mix(col, white, .3))
		c.rect(k.x-k.bw/2, k.y+k.bh-1, k.bw, 1, darken(col, .3))
	}
	for y := 3; y < 14; y++ {
		c.dot(14, y, stoneR[2])
		c.dot(15, y, stoneR[1])
		c.dot(16, y, stoneR[0])
	}
	c.rect(14, 3, 3, 1, stoneR[3])
	c.line(36, 6, 46, 15, woodR[2])
	c.line(37, 6, 47, 15, woodR[1])
	c.outline(ink)
	cover := []float64{0, .2, .45, .72, .94}[stage]
	green := []color.NRGBA{grassR[1], grassR[2], grassR[0], meadowR[2]}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			col := c.at(x, y)
			if col.A == 0 || col == ink || col.A < 255 || noise(x, y, 77+variant) > cover*(.4+float64(y)/float64(h)) {
				continue
			}
			c.dot(x, y, green[hash(x, y, 9)%4])
		}
	}
	for i := 0; i < stage*4; i++ {
		x, y := 9+int(hash(i, 21, 5)%42), 14+int(hash(i, 22, 5)%9)
		tuft(c, x, y, grassR, hash(x, y, 4))
	}
	if stage == RubbleStages-1 {
		c.dot(22, 15, rgb(244, 232, 120))
		c.dot(40, 17, rgb(240, 150, 170))
		c.dot(30, 13, white)
	}
	return piece(c, w/2, h-10)
}

func PaintLantern(lit bool) Piece {
	const w, h = 9, 20
	c := newCanvas(w, h)
	c.ellipse(4.5, 18, 3.5, 1.2, withAlpha(ink, 80))
	c.rect(4, 7, 1, 11, woodR[1])
	c.rect(3, 17, 3, 1, stoneR[1])
	c.line(4, 7, 7, 5, woodR[2])
	c.rect(7, 5, 1, 2, inkSft)
	glass := glassR[1]
	core := rgb(120, 110, 100)
	if lit {
		glass, core = rgb(255, 214, 110), rgb(255, 244, 190)
	}
	c.rect(6, 7, 3, 4, glass)
	c.dot(7, 8, core)
	c.dot(7, 9, core)
	c.rect(6, 6, 3, 1, woodR[0])
	c.rect(6, 11, 3, 1, woodR[0])
	c.outline(withAlpha(ink, 220))
	return piece(c, 4, h-2)
}

const GardenKinds = 3

var (
	fenceR   = ramp{rgb(120, 92, 62), rgb(168, 134, 92), rgb(210, 180, 128), rgb(240, 222, 176)}
	tomatoes = rgb(214, 62, 52)
	sunHead  = rgb(250, 208, 70)
)

func PaintTidyGarden(kind int) Piece {
	kind = ((kind % GardenKinds) + GardenKinds) % GardenKinds
	const w, h = 28, 22
	c := newCanvas(w, h)
	cx, cy := 14.0, 13.5
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Abs(float64(x)-cx)/13 + math.Abs(float64(y)-cy)/6.5
			if d <= 1 {
				col := soilRamp[1]
				if (x+2*y)%4 == 0 {
					col = soilRamp[0]
				}
				c.dot(x, y, col)
			}
		}
	}
	for row := 0; row < 3; row++ {
		for k := 0; k < 4; k++ {
			x := 7 + k*4 + row%2*2
			y := 11 + row*2 - k
			if math.Abs(float64(x)-cx)/11.5+math.Abs(float64(y)-cy)/5.5 > 1 {
				continue
			}
			plantRow(c, kind, x, y, row+k)
		}
	}
	posts := [][2]int{{1, 14}, {14, 20}, {26, 14}, {14, 8}}
	for i := 0; i < 4; i++ {
		a, b := posts[i], posts[(i+1)%4]
		if i < 2 {
			c.line(a[0], a[1]-3, b[0], b[1]-3, fenceR[2])
			c.line(a[0], a[1]-1, b[0], b[1]-1, fenceR[1])
		}
	}
	for i, p := range posts {
		if i == 3 {
			continue
		}
		c.rect(p[0]-1, p[1]-5, 2, 6, fenceR[2])
		c.dot(p[0]-1, p[1]-5, fenceR[3])
	}
	return piece(c, w/2, 14)
}

func plantRow(c *canvas, kind, x, y, n int) {
	leaf := greenR
	switch kind {
	case 0:
		c.disc(float64(x), float64(y-1), 1.9, purpleR[2])
		c.dot(x, y-2, purpleR[3])
		c.dot(x-1, y-1, purpleR[1])
		c.dot(x-2, y, leaf[2])
		c.dot(x+2, y, leaf[2])
	case 1:
		c.rect(x, y-3, 1, 3, leaf[1])
		c.dot(x-1, y-2, leaf[2])
		c.dot(x+1, y-3, leaf[2])
		c.dot(x, y-2, tomatoes)
		if n%2 == 0 {
			c.dot(x+1, y-1, tomatoes)
		}
	default:
		c.rect(x, y-4, 1, 4, leaf[1])
		c.dot(x-1, y-2, leaf[2])
		c.rect(x-1, y-6, 3, 2, sunHead)
		c.dot(x, y-5, woodR[1])
	}
}

func PaintBoard(parchments, variant int) Piece {
	n := min(max(parchments, 0), BoardSlots)
	const w, h = 24, 30
	c := newCanvas(w, h)
	c.ellipse(12, 27, 9, 2.4, withAlpha(ink, 80))
	c.rect(3, 12, 2, 15, woodR[1])
	c.rect(19, 12, 2, 15, woodR[1])
	c.rect(3, 12, 1, 15, woodR[2])
	c.rect(19, 12, 1, 15, woodR[2])
	c.rect(1, 4, 22, 14, woodR[2])
	c.rect(1, 4, 22, 1, woodR[3])
	c.rect(1, 17, 22, 1, woodR[0])
	c.rect(2, 5, 20, 12, woodR[1])
	c.rect(2, 5, 20, 1, woodR[2])
	for x := 6; x < 21; x += 6 {
		c.rect(x, 5, 1, 12, woodR[0])
	}
	c.line(0, 3, 12, 0, redR[1])
	c.line(24, 3, 12, 0, redR[1])
	c.rect(1, 3, 22, 1, redR[2])
	c.outline(ink)
	slots := [BoardSlots][2]int{{3, 6}, {9, 6}, {15, 6}, {3, 12}, {9, 11}, {15, 12}}
	for i := 0; i < n; i++ {
		s := slots[i]
		sheet := parchmentRamp[i%2]
		c.rect(s[0], s[1], 5, 5, sheet)
		c.rect(s[0], s[1], 5, 1, parchmentRamp[2])
		c.dot(s[0]+1, s[1]+2, inkSft)
		c.dot(s[0]+3, s[1]+3, inkSft)
		c.dot(s[0]+2, s[1], redR[2])
	}
	return piece(c, w/2, h-3)
}

var parchmentRamp = [3]color.NRGBA{rgb(232, 211, 164), rgb(214, 190, 142), rgb(250, 238, 200)}

func PaintScaffold() Piece {
	const w, h = 30, 36
	c := newCanvas(w, h)
	cx, base := 15, 33
	corners := [4][2]int{{3, base - 6}, {cx, base}, {w - 4, base - 6}, {cx, base - 12}}
	rise := 26
	for _, p := range corners {
		for y := p[1] - rise; y <= p[1]; y++ {
			c.dot(p[0], y, woodR[2])
			c.dot(p[0]+1, y, woodR[1])
		}
	}
	for _, lvl := range []int{rise, rise / 2} {
		for i := 0; i < 4; i++ {
			a, b := corners[i], corners[(i+1)%4]
			if i >= 2 {
				continue
			}
			c.line(a[0], a[1]-lvl, b[0], b[1]-lvl, woodR[3])
			c.line(a[0], a[1]-lvl+1, b[0], b[1]-lvl+1, woodR[1])
		}
	}
	c.line(corners[0][0], corners[0][1], corners[1][0], corners[1][1]-rise/2, woodR[2])
	c.line(corners[2][0], corners[2][1], corners[1][0], corners[1][1]-rise/2, woodR[2])
	plank := []pt{{3, float64(base - 6 - rise/2)}, {float64(cx), float64(base - rise/2 - 12)}, {float64(w - 4), float64(base - 6 - rise/2)}, {float64(cx), float64(base - rise/2)}}
	for i := range plank {
		a, b := plank[i], plank[(i+1)%4]
		c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[3])
	}
	c.outline(withAlpha(ink, 150))
	return piece(c, w/2, base)
}

var (
	henBody  = rgb(238, 232, 214)
	henWing  = rgb(190, 150, 104)
	henComb  = rgb(212, 56, 52)
	henBeak  = rgb(240, 190, 60)
	goatCoat = rgb(224, 218, 202)
	goatShad = rgb(164, 156, 142)
	goatDark = rgb(96, 82, 70)
	goatHorn = rgb(214, 196, 150)
)

func PaintChicken(frame int) Piece {
	const w, h = 9, 9
	c := newCanvas(w, h)
	c.ellipse(4.5, 7.6, 3.5, 1, withAlpha(ink, 80))
	frame = ((frame % LivestockFrames) + LivestockFrames) % LivestockFrames
	peck := frame == 1 || frame == 3
	bob := 0
	if frame == 2 {
		bob = -1
	}
	c.ellipse(4, 5+float64(bob), 2.8, 2.2, henBody)
	c.rect(2, 5+bob, 3, 1, henWing)
	c.dot(1, 4+bob, henBody)
	c.dot(0, 3+bob, henBody)
	if peck {
		c.rect(6, 5, 2, 1, henBody)
		c.dot(7, 5, henComb)
		c.dot(8, 6, henBeak)
	} else {
		c.rect(6, 2+bob, 2, 3, henBody)
		c.dot(6, 1+bob, henComb)
		c.dot(8, 3+bob, henBeak)
		c.dot(7, 2+bob, ink)
	}
	c.dot(3, 7, henBeak)
	c.dot(5-frame%2, 7, henBeak)
	c.outline(withAlpha(ink, 200))
	return piece(c, w/2, h-1)
}

func PaintGoat(frame int) Piece {
	const w, h = 15, 13
	c := newCanvas(w, h)
	c.ellipse(7.5, 11, 6, 1.5, withAlpha(ink, 80))
	frame = ((frame % LivestockFrames) + LivestockFrames) % LivestockFrames
	graze := frame == 1 || frame == 3
	c.rect(3, 5, 8, 4, goatCoat)
	c.rect(3, 8, 8, 1, goatShad)
	c.rect(4, 4, 6, 1, goatCoat)
	for _, x := range []int{4, 6, 9, 10} {
		step := 0
		if frame%2 == 1 && x%2 == 0 {
			step = -1
		}
		c.rect(x, 9, 1, 2+step, goatDark)
	}
	if graze {
		c.rect(10, 6, 2, 2, goatCoat)
		c.rect(12, 8, 2, 2, goatCoat)
		c.dot(13, 10, goatDark)
		c.dot(11, 5, goatHorn)
	} else {
		c.rect(10, 3, 2, 3, goatCoat)
		c.rect(11, 5, 3, 2, goatCoat)
		c.dot(13, 7, goatDark)
		c.dot(11, 1, goatHorn)
		c.dot(12, 2, goatHorn)
		c.dot(12, 4, ink)
		c.dot(11, 7, goatShad)
	}
	c.dot(2, 4, goatCoat)
	c.dot(2, 5, goatShad)
	c.outline(withAlpha(ink, 210))
	return piece(c, w/2, h-3)
}

func PaintGull(variant int) Piece {
	const w, h = 9, 8
	c := newCanvas(w, h)
	c.ellipse(4.5, 6.5, 3.5, 1, withAlpha(ink, 70))
	c.ellipse(4, 4, 3, 2, white)
	c.rect(1, 3, 4, 2, greyR[3])
	c.rect(5, 1, 2, 3, white)
	c.dot(7, 2, henBeak)
	c.dot(6, 2, ink)
	c.dot(3, 6, henBeak)
	c.dot(5, 6, henBeak)
	c.outline(withAlpha(ink, 190))
	return piece(c, w/2, h-1)
}

func PaintWornTile(variant int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, LivingTileW, LivingTileH))
	for y := 0; y < LivingTileH; y++ {
		for x := 0; x < LivingTileW; x++ {
			d := tileShare(x, y)
			if d > 1 {
				continue
			}
			edge := 1 - math.Max(d-.5, 0)*1.6
			if noise(x, y, 41+variant) > edge+.15 {
				continue
			}
			col := dirtR[1]
			switch hash(x, y, 8+variant) % 9 {
			case 0:
				col = dirtR[2]
			case 1:
				col = dirtR[0]
			case 2:
				col = dirtR[3]
			}
			img.SetNRGBA(x, y, col)
		}
	}
	return img
}
