package hud

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	riseTileW   = 24
	riseTileH   = 12
	riseReach   = 6
	riseKeeps   = 8
	riseHuts    = 48
	riseZoom    = .6
	riseSeaTop  = 44
	riseSeaH    = 168
	riseCaption = 220
)

const (
	LiveReadingUsage = "usage"
	LiveDrawing      = "drawing"
)

type LiveProgress struct {
	Step       string
	List       string
	Count      int
	Done       int
	Total      int
	Nodes      int
	Namespaces int
	Pods       int
}

var (
	riseSea   = color.RGBA{38, 74, 112, 255}
	riseSand  = color.RGBA{214, 190, 132, 255}
	riseGrass = color.RGBA{96, 150, 70, 255}
	riseHill  = color.RGBA{78, 128, 58, 255}
)

func (p LiveProgress) Caption() []string {
	if p.Total == 0 {
		return []string{"Asking the cluster for its version...", ""}
	}
	reading := fmt.Sprintf("Reading %s: %d so far", p.List, p.Count)
	switch {
	case p.Step == LiveReadingUsage:
		reading = "Every list is read: reading usage from metrics-server"
	case p.Step == LiveDrawing:
		reading = "Every list is read: drawing the realm"
	case p.Done >= p.Total:
		reading = "Every list is read"
	}
	return []string{reading, fmt.Sprintf("%d of %d lists   %d nodes   %d namespaces   %d pods", min(p.Done, p.Total), p.Total, p.Nodes, p.Namespaces, p.Pods)}
}

func (p LiveProgress) FailTitle() string {
	if p.Step == LiveDrawing {
		return "THE REALM COULD NOT BE DRAWN"
	}
	return "THE CLUSTER DID NOT ANSWER"
}

type riseTile struct {
	x, y int
	d    float64
}

var riseOrder = func() []riseTile {
	var out []riseTile
	for y := -riseReach; y <= riseReach; y++ {
		for x := -riseReach; x <= riseReach; x++ {
			wobble := .8 * math.Sin(float64(x)*1.3+float64(y)*.7) * math.Cos(float64(y)*1.1-float64(x)*.4)
			if d := math.Hypot(float64(x), float64(y)) + wobble; d <= riseReach {
				out = append(out, riseTile{x, y, d})
			}
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].d < out[j-1].d; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}()

func riseAt(x, y int) (float32, float32) {
	cx, cy := float32(LiveBox.X+LiveBox.W/2), float32(LiveBox.Y+riseSeaTop+riseSeaH/2)
	return cx + float32(x-y)*riseTileW/2, cy + float32(x+y)*riseTileH/2
}

func riseDiamond(dst *ebiten.Image, x, y int, clr color.RGBA) {
	sx, sy := riseAt(x, y)
	var path vector.Path
	path.MoveTo(sx, sy-riseTileH/2)
	path.LineTo(sx+riseTileW/2, sy)
	path.LineTo(sx, sy+riseTileH/2)
	path.LineTo(sx-riseTileW/2, sy)
	path.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &path, &vector.FillOptions{}, op)
}

func (p *LivePicker) drawOpening(dst *ebiten.Image) {
	title := "READING THE CLUSTER"
	if p.survey.drawAt != 0 {
		title = "DRAWING THE REALM"
	}
	p.title(dst, title, ui.Palette.GoldBright)
	sea := liveSea()
	if p.survey.drawAt == 0 {
		ui.Fill(dst, sea, surveyTable)
	} else {
		ui.Fill(dst, sea, riseSea)
		land := int(math.Round(p.survey.risen() * float64(len(riseOrder))))
		for i, t := range riseOrder[:land] {
			clr := riseGrass
			switch {
			case t.d > riseReach-1.2:
				clr = riseSand
			case i%7 == 3:
				clr = riseHill
			}
			riseDiamond(dst, t.x, t.y, clr)
		}
		p.drawRisen(dst, land)
	}
	if p.survey.rolled() < 1 {
		p.drawSurvey(dst)
	}
	y := LiveBox.Y + riseCaption
	for i, line := range p.Progress.Caption() {
		clr := ui.Palette.Text
		if i == 1 {
			clr = ui.Palette.TextDim
		}
		liveFont.DrawCentered(dst, line, ui.Rect{X: LiveBox.X, Y: y, W: LiveBox.W, H: 12}, 1, clr)
		y += ui.LineHeight + 2
	}
	liveFont.DrawCentered(dst, "Nothing is written.", ui.Rect{X: LiveBox.X, Y: y + 2, W: LiveBox.W, H: 12}, 1, ui.Palette.TextDim)
}

func (p *LivePicker) drawRisen(dst *ebiten.Image, land int) {
	if p.Lib == nil || land == 0 {
		return
	}
	inner := 0
	for inner < land && riseOrder[inner].d <= riseReach-2 {
		inner++
	}
	keeps := min(p.Progress.Nodes, riseKeeps)
	spots := make([][2]int, 0, keeps)
	for k := 0; k < keeps && inner > 0; k++ {
		a := 2*math.Pi*float64(k)/float64(max(keeps, 1)) + .4
		r := 0.0
		if keeps > 1 {
			r = 2.6
		}
		spots = append(spots, [2]int{int(math.Round(r * math.Cos(a))), int(math.Round(r * math.Sin(a)))})
	}
	risen := map[[2]int]bool{}
	for _, t := range riseOrder[:land] {
		if t.d <= riseReach-1.2 {
			risen[[2]int{t.x, t.y}] = true
		}
	}
	huts := min(p.Progress.Pods, riseHuts)
	for h := 0; h < huts && len(spots) > 0; h++ {
		k := spots[h%len(spots)]
		ring := 1.4 + float64(h/len(spots)/6)*.8
		a := float64(h/len(spots)%6)*math.Pi/3 + float64(h%len(spots))*.5
		at := [2]int{k[0] + int(math.Round(ring*math.Cos(a))), k[1] + int(math.Round(ring*math.Sin(a)))}
		if !risen[at] {
			continue
		}
		sx, sy := riseAt(at[0], at[1])
		p.Lib.DrawFirst(dst, gfx.HearthLookNames(1, "ready", "Burstable", 0, h), float64(sx), float64(sy)+riseTileH/2, riseZoom, render.SpriteOptions{})
	}
	keep := gfx.PreviewNames(view.Structure{Kind: view.StructKeep, Variant: "stone", State: view.StateReady}, 0)
	for _, k := range spots {
		sx, sy := riseAt(k[0], k[1])
		p.Lib.DrawFirst(dst, keep, float64(sx), float64(sy)+riseTileH, riseZoom, render.SpriteOptions{})
	}
}
