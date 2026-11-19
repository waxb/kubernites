package world

import (
	"hash/fnv"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	flagCycle     = 1.1
	gustPeriod    = 45.0
	gustSpread    = 80.0
	gustStrength  = 0.6
	flagPaceSwing = 0.25
)

func flagSeed(s view.Structure, i int) (float64, float64) {
	h := fnv.New32a()
	h.Write([]byte(strconv.Itoa(s.X) + "," + strconv.Itoa(s.Y) + "," + strconv.Itoa(i)))
	v := h.Sum32()
	phase := float64(v%1000) / 1000
	pace := 1 - flagPaceSwing + 2*flagPaceSwing*float64((v/1000)%1000)/1000
	return phase, pace
}

func FlagFrame(t float64, s view.Structure, i int) int {
	phase, pace := flagSeed(s, i)
	gust := gustStrength * math.Sin(2*math.Pi*(t/gustPeriod-float64(s.X+s.Y)/gustSpread))
	turn := phase + t*pace/flagCycle + gust
	frac := turn - math.Floor(turn)
	return int(frac*art.FlagFrames) % art.FlagFrames
}

func (r *Renderer) drawFlags(dst *ebiten.Image, name string, s view.Structure, sx, sy, zoom, t float64, alpha float32) {
	for i, m := range r.Lib.Flags(name) {
		gfx.DrawFlag(dst, m, FlagFrame(t, s, i), sx, sy, zoom, render.SpriteOptions{Alpha: alpha})
	}
}
