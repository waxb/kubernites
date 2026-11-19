package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	Mebi      = 1 << 20
	pileSteps = art.RequestPileUnits - 1
)

var (
	cpuPileSteps = [pileSteps]int64{100, 250, 500}
	memPileSteps = [pileSteps]int64{64 * Mebi, 128 * Mebi, 256 * Mebi}
	woodPileAt   = [2]float64{-12, -6}
	stonePileAt  = [2]float64{-3, -2}
	woodPiles    = requestPileNames(art.PileWood)
	stonePiles   = requestPileNames(art.PileStone)
)

func requestPileNames(kind string) [art.RequestPileUnits + 1]string {
	var out [art.RequestPileUnits + 1]string
	for n := 1; n <= art.RequestPileUnits; n++ {
		out[n] = art.RequestPileName(kind, n)
	}
	return out
}

func PileUnits(v int64, steps [pileSteps]int64) int {
	if v <= 0 {
		return 0
	}
	n := 1
	for _, s := range steps {
		if v > s {
			n++
		}
	}
	return n
}

func HutPiles(p model.PodView) (int, int) {
	if p.QoS == model.QoSBestEffort || p.State == model.HearthBanished {
		return 0, 0
	}
	cpu, mem := p.RequestSum()
	return PileUnits(cpu, cpuPileSteps), PileUnits(mem, memPileSteps)
}

type PileSprite struct {
	Name   string
	DX, DY float64
}

func PileSprites(p model.PodView, zoom float64) ([2]PileSprite, int) {
	var out [2]PileSprite
	if zoom < familiarZoom {
		return out, 0
	}
	n := 0
	wood, stone := HutPiles(p)
	if wood > 0 {
		out[n] = PileSprite{Name: woodPiles[wood], DX: woodPileAt[0] * zoom, DY: woodPileAt[1] * zoom}
		n++
	}
	if stone > 0 {
		out[n] = PileSprite{Name: stonePiles[stone], DX: stonePileAt[0] * zoom, DY: stonePileAt[1] * zoom}
		n++
	}
	return out, n
}

func (r *Renderer) drawRequestPiles(dst *ebiten.Image, p model.PodView, sx, sy, zoom float64) {
	piles, n := PileSprites(p, zoom)
	for _, s := range piles[:n] {
		r.Lib.Draw(dst, s.Name, sx+s.DX, sy+s.DY, zoom, render.SpriteOptions{})
	}
}
