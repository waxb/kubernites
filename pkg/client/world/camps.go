package world

import (
	"sort"
	"sync"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	maxCamps = 8
	campGap  = 1.5
)

var campJitter = []Vec{{0, 0}, {.3, -.3}, {-.3, .3}, {.3, .3}, {-.3, -.3}}

var campMemo struct {
	sync.Mutex
	sig   uint64
	spots []Vec
}

func campSpots(site view.Site) []Vec {
	sig := StructureSig(site) ^ forumSig(site) ^ uint64(site.Width)
	campMemo.Lock()
	defer campMemo.Unlock()
	if campMemo.spots != nil && campMemo.sig == sig {
		return campMemo.spots
	}
	campMemo.sig, campMemo.spots = sig, freeForum(site, maxCamps)
	return campMemo.spots
}

func forumHeart(site view.Site) (Vec, bool) {
	lo, hi, ok := site.ForumBox()
	return Vec{float64(lo.X+hi.X) / 2, float64(lo.Y+hi.Y) / 2}, ok
}

func freeForum(site view.Site, n int) []Vec {
	heart, ok := forumHeart(site)
	w, inWindow := forumWindowOf(site)
	if !ok || !inWindow {
		return []Vec{}
	}
	open, edge := []Vec{}, []Vec{}
	props := forumPropTiles(site)
	w.each(func(x, y int) {
		switch {
		case !site.OnForum(x, y) || Occupied(site, x, y) || props[Point{x, y}]:
		case NearAnyStructure(site, x, y):
			edge = append(edge, Vec{float64(x), float64(y)})
		default:
			open = append(open, Vec{float64(x), float64(y)})
		}
	})
	for _, tier := range [][]Vec{open, edge} {
		sort.SliceStable(tier, func(i, j int) bool { return Dist(tier[i], heart) < Dist(tier[j], heart) })
	}
	spaced := []Vec{}
	for _, v := range append(open, edge...) {
		if len(spaced) < n && clearOfAll(v, spaced, campGap) {
			spaced = append(spaced, v)
		}
	}
	return spaced
}
