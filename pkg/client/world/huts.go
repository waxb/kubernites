package world

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	rankGap    = 3
	rankHeight = 3
	QoSUp      = "qosup"
	QoSDown    = "qosdown"
	qosRise    = 10.0
)

var (
	rankHigh   = ui.Palette.GoldBright
	rankNormal = ui.Palette.Cream
	rankLow    = color.RGBA{150, 150, 140, 255}
)

type hutKey struct {
	tier    int
	state   string
	look    string
	age     int
	variant int
}

const carryLift = 18.0

func HutNames(tier int, state, qos string) []string {
	return gfx.HearthQoSNames(tier, state, qos)
}

func (r *Renderer) headLift(name string, zoom float64) float64 {
	if head, ok := r.Lib.Headroom(name); ok && name != "" {
		return float64(head-1) * zoom
	}
	return carryLift * zoom
}

func (r *Renderer) hutName(p model.PodView) string {
	age, _ := r.agesOf().phase(p.Name)
	key := hutKey{tier: p.Tier, state: p.State, look: model.QoSLook(p.QoS), age: age, variant: gfx.LookVariant(p.Name)}
	if name, ok := r.huts[key]; ok {
		return name
	}
	if r.huts == nil {
		r.huts = map[hutKey]string{}
	}
	names := gfx.HearthLookNames(p.Tier, p.State, p.QoS, age, key.variant)
	chosen := names[len(names)-1]
	for _, n := range names {
		if r.Lib.Has(n) {
			chosen = n
			break
		}
	}
	r.huts[key] = chosen
	return chosen
}

func RankChevrons(rank string) (int, bool, color.RGBA) {
	switch rank {
	case model.RankHigh:
		return 2, true, rankHigh
	case model.RankNormal:
		return 1, true, rankNormal
	case model.RankLow:
		return 1, false, rankLow
	}
	return 0, false, color.RGBA{}
}

func (r *Renderer) drawRankMark(dst *ebiten.Image, name, rank string, sx, sy, zoom float64) {
	n, up, c := RankChevrons(rank)
	if n == 0 {
		return
	}
	top := sy - 20*zoom
	if head, ok := r.Lib.Headroom(name); ok {
		top = sy - float64(head)*zoom
	}
	px := max(int(zoom), 1)
	for i := range n {
		drawChevron(dst, int(sx), int(top)-rankGap*px-i*(rankHeight+1)*px, px, up, c)
	}
}

func drawChevron(dst *ebiten.Image, cx, bottom, px int, up bool, c color.RGBA) {
	for row := range rankHeight {
		spread := row
		if !up {
			spread = rankHeight - 1 - row
		}
		y := bottom - (rankHeight-row)*px
		ui.Fill(dst, ui.Rect{X: cx - (spread+1)*px, Y: y, W: px, H: px}, c)
		ui.Fill(dst, ui.Rect{X: cx + spread*px, Y: y, W: px, H: px}, c)
	}
}

func QoSShiftKind(from, to string) (string, bool) {
	a, b := model.QoSRank(from), model.QoSRank(to)
	switch {
	case a < 0 || b < 0 || a == b:
		return "", false
	case b > a:
		return QoSUp, true
	}
	return QoSDown, true
}
