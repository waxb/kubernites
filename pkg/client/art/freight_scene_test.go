package art

import (
	"image"
	"math"
)

type sceneAdder func(a Atlas, name string, gx, gy, lift, dy float64)

const (
	freightOX = 120.0
	freightOY = 118.0
)

func freightCanvas() *image.NRGBA {
	img := sceneCanvas(480, 320, func(gx, gy int) bool { return gy <= 5 && gx >= -3 && gx <= 14 && gy >= -6 }, freightOX, freightOY)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			d := (float64(x) + 0.5 - freightOX) / 16
			t := (float64(y) + 0.5 - freightOY) / 8
			gx, gy := math.Floor((t+d)/2), math.Floor((t-d)/2)
			if gy < 2 && gy >= -6 && gx >= -3 && gx <= 14 {
				img.SetNRGBA(x, y, grassBackground(x, y))
			}
		}
	}
	return img
}

func freightGround() []placedSprite {
	s := built()
	items := []placedSprite{}
	for gx := -2; gx <= 13; gx++ {
		for gy := -5; gy <= 1; gy++ {
			tint := "own"
			if gx >= 12 {
				tint = []string{"good", "poor"}[min((gy+5)/4, 1)]
			}
			items = append(items, placedSprite{a: s.Terrain, name: GroundTintName(tint), gx: float64(gx) + 0.5, gy: float64(gy) + 0.5})
		}
	}
	return items
}

func freightItems(frame int) []placedSprite {
	s := built()
	items := []placedSprite{}
	add := func(a Atlas, name string, gx, gy, lift, dy float64) {
		items = append(items, placedSprite{a: a, name: name, gx: gx, gy: gy, lift: lift, dy: dy})
	}
	for i := 0; i < 9; i++ {
		for j := 4; j <= 5; j++ {
			add(s.Buildings, havenName("quay", (i+j)%3), float64(i)+0.5, float64(j)+0.5, 0, 0)
		}
	}
	for i, k := range BoxKinds {
		add(s.Buildings, BoxPileName(k, []int{1, 2, 0, 3}[i]), float64(1+i*2)+0.5, 4.5, HavenDeck, 0)
	}
	for i, k := range []string{"stall-parcel", "stall-crate", "stall-ledger"} {
		add(s.Buildings, buildingName(StallArtKind(k, "sw"), "ready"), float64(3+i*4), 0, 0, 0)
	}
	add(s.Buildings, BoxPileName("parcel", 1), 2.5, 0.8, 0, 0)
	add(s.Buildings, BoxPileName("crate", 2), 6.5, 0.8, 0, 0)
	quayBoxes(add, s, frame)
	freightWalkers(add, s, frame)
	borderRow(add, s, frame)
	add(s.Effects, shipName("caravel", 2, "sw", frame%4), 4.3, 8.1, 0, 0)
	return items
}

func quayBoxes(add sceneAdder, s Set, frame int) {
	for size := 0; size < BoxSizes; size++ {
		add(s.Buildings, BoxName(size, BoxKinds[size], size), float64(size)*1.1+0.6, 5.55, HavenDeck, 0)
	}
	marked := [][]string{{BoxSealName(1)}, {BoxCrownName(1), BoxFreshName(1, 1)}, {BoxFreshName(1, 3)}}
	for i, layers := range marked {
		gx := 5.0 + float64(i)*0.8
		add(s.Buildings, BoxName(1, BoxKinds[i], 1), gx, 5.55, HavenDeck, 0)
		for _, l := range layers {
			add(s.Buildings, l, gx, 5.55, HavenDeck, 0)
		}
	}
	add(s.Buildings, BoxSpoiledName(1, "crate", 0), 7.6, 5.55, HavenDeck, 0)
	add(s.Effects, effectName("spoil", frame), 7.6, 5.55, HavenDeck, -float64(BoxLift(1)))
}

func freightWalkers(add sceneAdder, s Set, frame int) {
	walkers := []struct {
		role      string
		v, size   int
		kind, dir string
		gx, gy    float64
	}{
		{"porter", 0, -1, "", "sw", 1.2, 2.4}, {"porter", 3, 0, "parcel", "se", 2.4, 2.9},
		{"hauler", 0, -1, "", "se", 4.2, 2.3}, {"hauler", 2, 2, "crate", "sw", 5.3, 2.9},
		{"clerk", 1, -1, "", "se", 6.4, 2.4}, {"courier", 0, -1, "", "sw", 7.4, 2.9},
		{"sealwright", 0, -1, "", "se", 8.4, 2.3}, {"porter", 5, 1, "parcel", "sw", 9.4, 2.8},
	}
	for i, w := range walkers {
		f := (frame + i) % WalkerFrames
		if w.size < 0 {
			name := walkerName(w.role, "carry", w.dir, f)
			if w.v > 0 {
				name = walkerLookName(w.role, 1, w.v, "carry", w.dir, f)
			}
			add(s.Walkers, name, w.gx, w.gy, 0, 0)
			continue
		}
		add(s.Walkers, tradeHoldName(w.role, 1, w.v, w.dir, f), w.gx, w.gy, 0, 0)
		add(s.Walkers, CarryName(w.size, w.kind, w.dir, f), w.gx, w.gy, 0, 0)
	}
}

func borderRow(add sceneAdder, s Set, frame int) {
	for i := 0; i < 6; i++ {
		gy := -5.0 + float64(i)*1.3
		if i%5 == 0 {
			add(s.Buildings, BorderFlagName(frame%BorderFlags), -2, gy, 0, 0)
			continue
		}
		add(s.Buildings, BorderStoneName(i%BorderStones), -2, gy, 0, 0)
	}
	for _, p := range [][3]float64{{12.6, -4.2, 0}, {13.4, -2.4, 1}, {12.9, 0.6, 2}} {
		add(s.Terrain, decorName([]string{"outcrop", "quarry", "outcrop"}[int(p[2])], int(p[2])), p[0], p[1], 0, 0)
	}
}
