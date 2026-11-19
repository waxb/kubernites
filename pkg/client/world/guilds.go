package world

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ScribeRole = "clerk"
	ledgerBarW = 24
	ledgerBarH = 3
	ledgerWarn = 0.9
)

type Ledger struct {
	Open  bool
	Share float64
}

func (r *Renderer) ScribeItems(cam render.Camera, sc Scene, p model.PodView) []Drawable {
	f, ok := r.scribeFigure(sc, p)
	if !ok {
		return nil
	}
	return []Drawable{r.FigureDrawable(cam, sc, f)}
}

func (r *Renderer) scribeFigure(sc Scene, p model.PodView) (Figure, bool) {
	home, ok := r.Homes[p.Name]
	if !ok {
		return Figure{}, false
	}
	archive, hasArchive := doorOf(sc.Site, view.StructArchive)
	start := Vec{home.X + .35, home.Y + .35}
	source, hasSource := r.RecordSource(sc, model.Hash(p.Name))
	open := hasArchive && hasSource && sc.Ledger.Open
	if !open {
		source, archive = start, start
	}
	rate, working, throttled := Workload(sc, p, open)
	f := r.paced(sc, p, start, source, archive, rate, working, throttled)
	f.home = home
	return f, true
}

func (r *Renderer) RecordSource(sc Scene, seed uint32) (Vec, bool) {
	names := []string{}
	for _, p := range sc.Pods {
		if !p.System && p.Sprite != ScribeRole && p.State == model.HearthReady {
			if _, ok := r.Homes[p.Name]; ok {
				names = append(names, p.Name)
			}
		}
	}
	if len(names) == 0 {
		return Vec{}, false
	}
	sort.Strings(names)
	home := r.Homes[names[int(seed)%len(names)]]
	return Vec{home.X + .6, home.Y + .9}, true
}

func (r *Renderer) DrawStructureBars(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if !sc.Ledger.Open {
		return
	}
	for _, it := range r.placeStructures(cam, sc) {
		if it.S.Kind != view.StructArchive || it.S.State != view.StateReady {
			continue
		}
		z := cam.Scale()
		top := it.sy - 60*z
		if head, ok := r.Lib.Headroom(it.names[0]); ok {
			top = it.sy - float64(head)*z
		}
		w, h := int(ledgerBarW*max(z, 1)), int(ledgerBarH*max(z, 1))
		bar := ui.Rect{X: int(it.sx) - w/2, Y: int(top) - h - 3, W: w, H: h}
		drawLedgerBar(dst, r, bar, sc.Ledger.Share)
	}
}

func drawLedgerBar(dst *ebiten.Image, r *Renderer, bar ui.Rect, share float64) {
	fill := ui.Palette.Green
	if share >= ledgerWarn {
		fill = ui.Palette.RedText
	}
	if !r.Lib.NineSlice(dst, "bar-mini", bar) {
		ui.Fill(dst, bar, ui.Palette.ConsoleEdge)
		ui.Stroke(dst, bar, ui.Palette.Ink)
	}
	inner := bar.Inset(1)
	inner.W = int(float64(inner.W) * min(max(share, 0), 1))
	if inner.W > 0 {
		ui.Fill(dst, inner, fill)
	}
}
