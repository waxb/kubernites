package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	hintAbove = 12
	hintGap   = 12
	hintTop   = 36
)

var (
	plankLight = color.RGBA{176, 128, 78, 255}
	plankDark  = color.RGBA{128, 88, 52, 255}
	plankEdge  = color.RGBA{72, 48, 30, 255}
	sailCloth  = color.RGBA{236, 226, 198, 255}
	foamWhite  = color.RGBA{236, 244, 246, 255}
)

func drawGangplank(dst *ebiten.Image, sx, sy, px, py, zoom float64) {
	w := float32(max(2*zoom, 1))
	vector.StrokeLine(dst, float32(sx), float32(sy-2*zoom)+w/2, float32(px), float32(py)+w/2, w+1, plankEdge, false)
	vector.StrokeLine(dst, float32(sx), float32(sy-2*zoom), float32(px), float32(py), w, plankLight, false)
}

func isoQuad(dst *ebiten.Image, sx, sy, zoom, along, across float64, pier string, c color.RGBA) {
	ax, ay := render.TileWidth/2*zoom, render.TileHeight/2*zoom
	u, v := [2]float64{ax, ay}, [2]float64{-ax, ay}
	if pier == "pier-y" {
		u, v = v, u
	}
	var path vector.Path
	for i, k := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		x := sx + (u[0]*k[0]*along+v[0]*k[1]*across)/2
		y := sy + (u[1]*k[0]*along+v[1]*k[1]*across)/2
		if i == 0 {
			path.MoveTo(float32(x), float32(y))
			continue
		}
		path.LineTo(float32(x), float32(y))
	}
	path.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, &path, &vector.FillOptions{}, op)
}

func block(dst *ebiten.Image, sx, sy, zoom, w, h float64, c color.RGBA) {
	vector.FillRect(dst, float32(sx-w/2*zoom)-1, float32(sy-h*zoom)-1, float32(w*zoom)+2, float32(h*zoom)+2, plankEdge, false)
	vector.FillRect(dst, float32(sx-w/2*zoom), float32(sy-h*zoom), float32(w*zoom), float32(h*zoom), c, false)
}

var pieceBlocks = map[string]struct {
	w, h float64
	c    color.RGBA
}{
	"bollard": {3, 4, plankEdge}, "crates": {9, 7, plankDark}, "sign": {7, 12, plankLight}, "crane": {5, 26, plankDark},
	"warehouse": {22, 18, color.RGBA{150, 104, 70, 255}}, "lighthouse": {7, 30, color.RGBA{232, 226, 214, 255}},
}

func drawPieceFallback(dst *ebiten.Image, kind string, variant int, sx, sy, zoom float64) {
	switch kind {
	case "quay":
		c := plankLight
		if variant%2 == 1 {
			c = plankDark
		}
		isoQuad(dst, sx, sy, zoom, 1, 1, "pier-x", plankEdge)
		isoQuad(dst, sx, sy, zoom, .9, .9, "pier-x", c)
	case "pier-x", "pier-y":
		isoQuad(dst, sx, sy+zoom, zoom, 1.02, .66, kind, plankEdge)
		isoQuad(dst, sx, sy, zoom, 1, .6, kind, plankLight)
	default:
		if b, ok := pieceBlocks[kind]; ok {
			block(dst, sx, sy, zoom, b.w, b.h, b.c)
		}
	}
}

func drawShipFallback(dst *ebiten.Image, class string, variant int, pose ShipPose, sx, sy, zoom float64) {
	l := shipLength(class) * 22 * zoom
	axis := [2]float64{1, .5}
	if pose.Dir == "sw" || pose.Dir == "ne" {
		axis = [2]float64{-1, .5}
	}
	n := math.Hypot(axis[0], axis[1])
	ux, uy := axis[0]/n*l/2, axis[1]/n*l/2
	alpha := uint8(255 * min(max(pose.Alpha, 0), 1))
	hull := ui.Tint(color.RGBA{uint8(96 + variant*9), uint8(62 + variant*4), 36, 255}, alpha)
	var path vector.Path
	path.MoveTo(float32(sx+ux), float32(sy+uy-3*zoom))
	path.LineTo(float32(sx-ux), float32(sy-uy-3*zoom))
	path.LineTo(float32(sx-ux*.8), float32(sy-uy*.8+2*zoom))
	path.LineTo(float32(sx+ux*.8), float32(sy+uy*.8+2*zoom))
	path.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(hull)
	vector.FillPath(dst, &path, &vector.FillOptions{}, op)
	mast := l * .9
	vector.StrokeLine(dst, float32(sx), float32(sy-2*zoom), float32(sx), float32(sy-mast), float32(max(zoom, 1)), ui.Tint(plankEdge, alpha), false)
	vector.FillRect(dst, float32(sx-l*.28), float32(sy-mast*.92), float32(l*.56), float32(mast*.55), ui.Tint(sailCloth, alpha), false)
}

func drawWakeFallback(dst *ebiten.Image, sx, sy, zoom, t float64, alpha float32) {
	c := ui.Tint(foamWhite, uint8(200*alpha))
	for i := 0; i < 2; i++ {
		d := float32((1 + math.Mod(t*2+float64(i)*.5, 1)) * 4 * zoom)
		vector.StrokeLine(dst, float32(sx)-d, float32(sy)+float32(i)*2, float32(sx)+d, float32(sy)+float32(i)*2, float32(max(zoom, 1)), c, false)
	}
}

func (r *Renderer) drawHint(dst *ebiten.Image, sx, sy, zoom float64, sc Scene) {
	p := 1.0
	if !sc.Reduce {
		p = .5 + .5*math.Sin(sc.RealTime()*2*math.Pi*1.2)
	}
	for i, grow := range []float64{.34, .34 + .18*p} {
		a := uint8(230 - float64(i)*140*p)
		isoRing(dst, sx, sy, zoom, grow, ui.Tint(ui.Palette.GoldBright, a))
	}
	label := HintText
	box, clear := r.settleLabel(labelBox(label, sx, max(sy-(beaconHeight(zoom)+hintAbove)*zoom-hintGap, hintTop)), int(sx), sc.Panels)
	if !clear {
		return
	}
	ui.Fill(dst, box.Offset(1, 1), ui.Palette.Shadow)
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, box, ui.Palette.Gold)
	var font ui.Font
	font.Draw(dst, label, box.X+4, box.Y+3, 1, ui.Palette.GoldBright)
}

const HintText = "Click the parcels: each pays gold"

func isoRing(dst *ebiten.Image, sx, sy, zoom, size float64, c color.RGBA) {
	hw, hh := float32(render.TileWidth*size*zoom), float32(render.TileHeight*size*zoom)
	x, y := float32(sx), float32(sy)
	w := float32(max(zoom, 1))
	vector.StrokeLine(dst, x-hw, y, x, y-hh, w, c, false)
	vector.StrokeLine(dst, x, y-hh, x+hw, y, w, c, false)
	vector.StrokeLine(dst, x+hw, y, x, y+hh, w, c, false)
	vector.StrokeLine(dst, x, y+hh, x-hw, y, w, c, false)
}
