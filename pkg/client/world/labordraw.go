package world

import (
	"cmp"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	scarSpread = .22
	scarDepth  = -.28
	laborReach = 2
	laborTall  = 48
)

var scarKinds = map[string][3]string{
	RoleCutter: {"", "stump", "sapling"},
	RoleMason:  {"", "quarry", "outcrop"},
}

func (r *Renderer) laborItems(cam render.Camera, sc Scene) []Drawable {
	l := r.laborOf()
	l.clearHidden()
	if r.Land == nil || r.Land.land == nil {
		return nil
	}
	l.sync(sc.Site, r.Land.habitat(sc.Site), r.Land.version)
	out := []Drawable{}
	budget := laborBudget
	for _, ls := range l.sites {
		sh := l.shiftsFor(sc.Site, ls, sc.Time, sc.Metrics.Nodes[ls.node])
		if !landInView(cam, ls) {
			continue
		}
		out = r.tradeItems(out, cam, sc, ls, cutterTrade, sh[0], ls.trees, &budget)
		out = r.tradeItems(out, cam, sc, ls, masonTrade, sh[1], ls.rocks, &budget)
	}
	return out
}

func landInView(cam render.Camera, ls laborSite) bool {
	zoom := cam.Scale()
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range []Point{{ls.lo.X - laborReach, ls.lo.Y - laborReach}, {ls.hi.X + laborReach, ls.lo.Y - laborReach},
		{ls.lo.X - laborReach, ls.hi.Y + laborReach}, {ls.hi.X + laborReach, ls.hi.Y + laborReach}} {
		sx, sy := cam.GridToScreenElevated(float64(c.X), float64(c.Y), 0)
		x0, y0, x1, y1 = min(x0, sx), min(y0, sy), max(x1, sx), max(y1, sy)
	}
	m := cullMargin * max(zoom, 1)
	return x1 >= -m && x0 <= ViewWidth+m && y1 >= -m && y0-laborTall*zoom <= ViewHeight+2*m
}

func (r *Renderer) tradeItems(out []Drawable, cam render.Camera, sc Scene, ls laborSite, tr trade, sh shift, tiles []int, budget *int) []Drawable {
	if len(tiles) == 0 {
		return out
	}
	for _, sc2 := range r.laborOf().scars(sc.Time, ls, tr, sh, tiles) {
		out = r.scarItem(out, cam, sc, tr, sc2)
	}
	if !drawFiguresWithoutPods {
		return out
	}
	for slot := range maxLaborSlots {
		for _, j := range []int{tr.jobIndex(sc.Time, ls.seed, slot) - 1, tr.jobIndex(sc.Time, ls.seed, slot)} {
			jb := tr.jobAt(ls.seed, slot, j, len(tiles))
			if *budget <= 0 || !sh.works(slot, jb.start) {
				continue
			}
			pose, ok := tr.pose(sc.Time, jb, ls.door, tileCenter(tiles[jb.target], sc.Site.Width))
			if !ok {
				continue
			}
			if sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos); onScreen(sx, sy, cam.Scale()) {
				*budget--
				out = append(out, r.walkerDrawable(cam, sc, tr.role, pose, nil, false))
			}
		}
	}
	return out
}

func tileCenter(i, width int) Vec {
	return Vec{float64(i%width) + .5, float64(i/width) + .5}
}

func (l *labor) scars(t float64, ls laborSite, tr trade, sh shift, tiles []int) []scar {
	out := l.scarBuf[:0]
	for slot := range maxLaborSlots {
		for j := tr.jobIndex(t-tr.window(), ls.seed, slot); j <= tr.jobIndex(t, ls.seed, slot); j++ {
			jb := tr.jobAt(ls.seed, slot, j, len(tiles))
			if !sh.works(slot, jb.start) {
				continue
			}
			tile := tiles[jb.target]
			at := tr.doneAt(jb, ls.door, tileCenter(tile, l.width))
			if stage := tr.stage(t, at); stage != ScarNone {
				out = append(out, scar{tile: tile, at: at, stage: stage})
			}
		}
	}
	l.scarBuf = youngestScars(out)
	return l.scarBuf
}

func youngestScars(all []scar) []scar {
	slices.SortFunc(all, func(a, b scar) int {
		if a.tile != b.tile {
			return a.tile - b.tile
		}
		return cmp.Compare(b.at, a.at)
	})
	out := all[:0]
	for i, s := range all {
		if i == 0 || s.tile != all[i-1].tile {
			out = append(out, s)
		}
	}
	return out
}

func (r *Renderer) scarItem(out []Drawable, cam render.Camera, sc Scene, tr trade, s scar) []Drawable {
	l := r.laborOf()
	l.hideTile(s.tile)
	x, y := s.tile%sc.Site.Width, s.tile/sc.Site.Width
	h := tileHash(x, y)
	pos := Vec{float64(x) + .5 + jitterAt(h, 4, scarSpread), float64(y) + .5 + jitterAt(h, 14, scarSpread)}
	sx, sy := cam.GridToScreenElevated(pos.X, pos.Y, float64(sc.Site.ElevationAt(x, y)))
	zoom := cam.Scale()
	if !onScreen(sx, sy, zoom) {
		return out
	}
	name := gfx.DecorName(scarKinds[tr.role][s.stage], int(h>>20))
	opts := HillLight(sc.Site.ElevationAt(x, y))
	opts.Flip = h&1 == 1
	return append(out, Drawable{Depth: render.DepthKey(pos.X, pos.Y) + scarDepth, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, opts)
	}})
}

func (l *labor) hideTile(i int) {
	if i < 0 || i >= len(l.hide) || l.hide[i] {
		return
	}
	l.hide[i] = true
	l.hidden = append(l.hidden, i)
}

func (l *labor) clearHidden() {
	for _, i := range l.hidden {
		if i < len(l.hide) {
			l.hide[i] = false
		}
	}
	l.hidden = l.hidden[:0]
}

func (l *labor) hiddenAt(i int32) bool {
	return l != nil && i >= 0 && int(i) < len(l.hide) && l.hide[i]
}
