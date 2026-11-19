package world

import (
	"sort"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const MaxMarks = 10

func markable(mood string) bool {
	return mood == MoodRest || mood == MoodSick || mood == MoodStrained
}

func MarkNearest(cam render.Camera, site view.Site, figs []Figure) []Figure {
	type near struct {
		i int
		d float64
	}
	cands := make([]near, 0, len(figs))
	cx, cy := float64(ViewWidth)/2, float64(ViewHeight)/2
	for i := range figs {
		if !markable(figs[i].Mood) {
			continue
		}
		sx, sy := ScreenPointOf(cam, site, figs[i].Pose.Pos)
		cands = append(cands, near{i: i, d: (sx-cx)*(sx-cx) + (sy-cy)*(sy-cy)})
	}
	sort.SliceStable(cands, func(a, b int) bool { return cands[a].d < cands[b].d })
	for k := range min(len(cands), MaxMarks) {
		figs[cands[k].i].mark = true
	}
	return figs
}
