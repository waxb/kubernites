package main

import (
	"image"
	"image/draw"
	"math"

	"github.com/waxb/kubernites/pkg/client/art"
)

const sceneTiles = 32

type sprite struct {
	atlas art.Atlas
	name  string
	gx    float64
	gy    float64
	flip  bool
}

func sceneGround(x, y int) art.Ground {
	d := math.Hypot(float64(x)-15.5, float64(y)-15.5) + math.Sin(float64(x)*0.7)*0.8 + math.Cos(float64(y)*0.5)*0.8
	switch {
	case d > 15:
		return art.GroundWater
	case d > 13.6:
		return art.GroundShallows
	case d > 12.2:
		return art.GroundSand
	case x >= 12 && x <= 15 && y >= 17 && y <= 19:
		return art.GroundPlaza
	case x > 19 && y < 12:
		return art.GroundForest
	case x < 9 && y < 14:
		return art.GroundRock
	case math.Hypot(float64(x)-19, float64(y)-22) < 4.5 || math.Hypot(float64(x)-11, float64(y)-12) < 3:
		return art.GroundMeadow
	}
	return art.GroundGrass
}

func sceneHeight(x, y int) uint8 {
	if g := sceneGround(x, y); g == art.GroundWater || g == art.GroundShallows || g == art.GroundSand || g == art.GroundPlaza {
		return 0
	}
	return uint8(max(0, 3-int(math.Round(math.Hypot(float64(x)-22, float64(y)-10)/2.2))))
}

func sceneRoad(x, y int) uint8 {
	switch {
	case y == 18 && x > 5 && x < 12:
		return 3
	case x == 14 && y > 19 && y < 26:
		return 2
	case x+y == 30 && x > 15 && x < 24:
		return 1
	}
	return 0
}

func sceneLand() *art.Land {
	l := art.NewLand(sceneTiles, sceneTiles)
	for y := 0; y < sceneTiles; y++ {
		for x := 0; x < sceneTiles; x++ {
			i := y*sceneTiles + x
			l.Ground[i] = sceneGround(x, y)
			l.Height[i] = sceneHeight(x, y)
			if l.Ground[i] != art.GroundWater && l.Ground[i] != art.GroundShallows {
				l.Road[i] = sceneRoad(x, y)
			}
			l.Wild[i] = x+y > 44
			if math.Hypot(float64(x)-8, float64(y)-24) < 4 {
				l.Power[i] = uint8(3 - int(math.Hypot(float64(x)-8, float64(y)-24)/1.6))
			}
		}
	}
	l.Prepare()
	return l
}

func renderScene(set art.Set, age int) *image.NRGBA {
	l := sceneLand()
	img := l.Render(0)
	for _, s := range sortSprites(sceneSprites(set, age)) {
		lift := float64(l.Height[int(s.gy)*sceneTiles+int(s.gx)]) * art.ElevationStep
		art.Stamp(img, s.atlas, s.name, art.Project(s.gx, s.gy, lift), s.flip)
	}
	out := image.NewNRGBA(image.Rect(0, 0, img.Rect.Dx(), img.Rect.Dy()))
	draw.Draw(out, out.Rect, img, img.Rect.Min, draw.Src)
	return out
}

func sceneSprites(set art.Set, age int) []sprite {
	b, t, wk, fx := set.Buildings, set.Terrain, set.Walkers, set.Effects
	return []sprite{
		{b, art.BuildingLookName("keep-great", "ready", age, 0), 18.5, 14.5, false},
		{b, art.BuildingLookName("keep-citadel", "ready", age, 0), 9.5, 25.5, false},
		{b, art.BuildingLookName(art.StallArtKind("stall-guildhall", "sw"), "ready", age, 0), 15.5, 16.5, false},
		{b, art.BuildingLookName("treasury", "ready", age, 2), 16.5, 19.5, false},
		{b, art.HearthLookName(2, "ready", "freeman", age, 1), 20.5, 17.5, false},
		{b, art.HearthLookName(1, "ready", "vagrant", age, 2), 18.5, 21.5, false},
		{t, "decor/tree-oak/0", 21.6, 7.4, false},
		{t, "decor/tree-pine/1", 23, 6.2, false},
		{t, "decor/tree-birch/0", 25.3, 8.2, false},
		{t, "decor/tree-maple/0", 22.4, 9.2, false},
		{t, "decor/tree-spruce/0", 24.1, 7.1, false},
		{t, "decor/tree-old/1", 24.2, 10.6, false},
		{t, "decor/tree-maple/1", 20.2, 6.4, true},
		{t, "decor/bush/1", 19.4, 10.3, false},
		{t, "decor/boulder/0", 4, 6, false},
		{t, "decor/rock/1", 6, 9, false},
		{t, "decor/tree-pine/0", 7.2, 7.4, false},
		{t, "decor/reeds/0", 3.4, 16.2, false},
		{t, "decor/tree-willow/0", 5.2, 20.4, false},
		{t, "decor/ley-stone/1", 7.2, 22.1, false},
		{t, "decor/log/1", 12.4, 6.6, false},
		{wk, "walker/lamplighter/walk/se/1", 8.2, 18.1, false},
		{wk, "walker/pilgrim/walk/sw/2", 14.1, 22.4, false},
		{wk, "walker/deer/work/se/1", 11.2, 8.4, false},
		{wk, "walker/stag/idle/sw/0", 12.3, 9.1, false},
		{wk, "walker/rabbit/idle/sw/0", 13.4, 10.2, false},
		{wk, "walker/sheep/work/se/0", 20.6, 23.5, false},
		{wk, "walker/sheep/idle/sw/1", 21.4, 24.1, false},
		{wk, "walker/fox/walk/sw/2", 19.2, 12.8, false},
		{wk, "walker/hound/walk/se/1", 8.9, 18.6, false},
		{wk, "walker/raven/walk/se/2", 13.6, 21.9, false},
		{wk, "walker/clerk/walk/sw/1", 16.8, 21.2, false},
		{fx, "fx/charter/1", 26.5, 22.5, false},
		{fx, "fx/ley/2", 7.2, 22.1, false},
	}
}

func sortSprites(in []sprite) []sprite {
	out := append([]sprite(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].gx+out[j].gy < out[j-1].gx+out[j-1].gy; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func itoa(v int) string {
	if v < 10 {
		return string(rune('0' + v))
	}
	return string(rune('0'+v/10)) + string(rune('0'+v%10))
}
