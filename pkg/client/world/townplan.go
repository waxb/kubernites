package world

import "github.com/waxb/kubernites/pkg/view"

const (
	spurPasses  = 3
	blobPasses  = 3
	blobCrowd   = 4
	spurNeeds   = 2
	shoulderMax = RoadDirt
)

var around = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

func joins(site view.Site, plan []uint8, x, y int) bool {
	if x < 0 || y < 0 || x >= site.Width || y >= site.Height {
		return false
	}
	if plan[y*site.Width+x] > RoadNone {
		return true
	}
	switch site.Tile(x, y) {
	case view.TilePlaza, view.TileForum:
		return true
	}
	return false
}

func TownPlan(site view.Site) []uint8 {
	w := site.Width
	plan := make([]uint8, w*site.Height)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < w; x++ {
			plan[y*w+x] = RoadLevel(site, x, y)
		}
	}
	plan = dropShoulders(site, plan)
	for range blobPasses {
		plan = thinBlobs(site, plan)
	}
	for range spurPasses {
		plan = dropSpurs(site, plan)
	}
	return plan
}

func hasHigher(site view.Site, plan []uint8, x, y int, lv uint8) bool {
	for _, d := range around {
		nx, ny := x+d[0], y+d[1]
		if nx >= 0 && ny >= 0 && nx < site.Width && ny < site.Height && plan[ny*site.Width+nx] > lv {
			return true
		}
	}
	return false
}

func leadsAway(site view.Site, plan []uint8, x, y int, lv uint8) bool {
	for _, d := range around {
		nx, ny := x+d[0], y+d[1]
		if nx < 0 || ny < 0 || nx >= site.Width || ny >= site.Height {
			continue
		}
		o := plan[ny*site.Width+nx]
		if o > RoadNone && o <= lv && !hasHigher(site, plan, nx, ny, o) {
			return true
		}
	}
	return false
}

func dropShoulders(site view.Site, plan []uint8) []uint8 {
	w := site.Width
	out := append([]uint8(nil), plan...)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < w; x++ {
			lv := plan[y*w+x]
			if lv == RoadNone || lv > shoulderMax || !hasHigher(site, plan, x, y, lv) {
				continue
			}
			if !leadsAway(site, plan, x, y, lv) {
				out[y*w+x] = RoadNone
			}
		}
	}
	return out
}

func dropSpurs(site view.Site, plan []uint8) []uint8 {
	w := site.Width
	out := append([]uint8(nil), plan...)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < w; x++ {
			if plan[y*w+x] != RoadTrodden {
				continue
			}
			n := 0
			for _, d := range around {
				if joins(site, plan, x+d[0], y+d[1]) {
					n++
				}
			}
			if n < spurNeeds {
				out[y*w+x] = RoadNone
			}
		}
	}
	return out
}

func thinBlobs(site view.Site, plan []uint8) []uint8 {
	w := site.Width
	out := append([]uint8(nil), plan...)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < w; x++ {
			if plan[y*w+x] != RoadTrodden {
				continue
			}
			n := 0
			for _, d := range around {
				nx, ny := x+d[0], y+d[1]
				if nx >= 0 && ny >= 0 && nx < w && ny < site.Height && plan[ny*w+nx] > RoadNone {
					n++
				}
			}
			if n >= blobCrowd {
				out[y*w+x] = RoadNone
			}
		}
	}
	return out
}
