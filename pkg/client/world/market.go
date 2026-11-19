package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	marketReach   = 4
	MarketGrowth  = 2400.0
	marketLayers  = 2
	marketSpread  = 1000
	marketSecond  = .45
	marketHeartR2 = 2
)

var marketNames = func() [art.MarketGrowths]string {
	var out [art.MarketGrowths]string
	for i := range out {
		out[i] = art.MarketGrowthName(i)
	}
	return out
}()

type MarketTuft struct {
	At      view.Point
	Variant int
}

func MarketGrown(realmTime float64) float64 {
	return min(max(realmTime/MarketGrowth, 0), 1)
}

func marketTuft(x, y, layer int) (float64, int) {
	h := hash2(x*7+layer*131, y*13+layer*17)
	threshold := float64(h%marketSpread) / marketSpread
	if layer > 0 {
		threshold = marketSecond + threshold*(1-marketSecond)
	}
	return threshold, int(h>>10) % art.MarketGrowths
}

func MarketTufts(site view.Site, grown float64) []MarketTuft {
	out := []MarketTuft{}
	p := site.Plaza
	for y := p.Y - marketReach; y <= p.Y+marketReach; y++ {
		for x := p.X - marketReach; x <= p.X+marketReach; x++ {
			if site.Tile(x, y) != view.TilePlaza || (x-p.X)*(x-p.X)+(y-p.Y)*(y-p.Y) <= marketHeartR2 {
				continue
			}
			for layer := 0; layer < marketLayers; layer++ {
				if t, v := marketTuft(x, y, layer); grown > t {
					out = append(out, MarketTuft{At: view.Point{X: x, Y: y}, Variant: v})
				}
			}
		}
	}
	return out
}

func (r *Renderer) drawMarketGround(dst *ebiten.Image, cam render.Camera, sc Scene) {
	grown := MarketGrown(sc.RealmTime)
	if grown <= 0 {
		return
	}
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	zoom := cam.Scale()
	for _, t := range MarketTufts(sc.Site, grown) {
		if t.At.X < x0 || t.At.X > x1 || t.At.Y < y0 || t.At.Y > y1 {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{float64(t.At.X), float64(t.At.Y)})
		r.Lib.Draw(dst, marketNames[t.Variant], sx, sy, zoom, render.SpriteOptions{})
	}
}
