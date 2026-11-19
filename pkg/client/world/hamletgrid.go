package world

import (
	"github.com/waxb/kubernites/pkg/view"
)

const (
	sigOffset = 14695981039346656037
	sigPrime  = 1099511628211
)

func sigMix(h uint64, v int) uint64 {
	return (h ^ uint64(v)) * sigPrime
}

func sigText(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h = (h ^ uint64(s[i])) * sigPrime
	}
	return sigMix(h, len(s))
}

func hutSig(site view.Site) uint64 {
	h := sigMix(sigMix(sigMix(sigOffset, site.Width), site.Height), len(site.Tiles))
	for _, s := range site.Structures {
		h = sigText(sigText(sigText(h, s.ID), s.Kind), s.Variant)
		h = sigMix(sigMix(sigMix(sigMix(sigMix(sigMix(h, s.X), s.Y), s.W), s.H), s.Territory), s.Plot)
		h = sigText(h, s.Object.Name)
	}
	for _, hv := range site.Havens {
		h = sigText(h, hv.ID)
		h = sigMix(sigMix(sigMix(sigMix(sigMix(h, hv.Quay.X), hv.Quay.Y), int(hv.Tier)), hv.Pier), boolInt(hv.Open))
	}
	for _, is := range site.Islands {
		h = sigMix(sigText(h, is.ID), boolInt(is.Owned))
	}
	return h
}

func (h *Hamlet) sync(site view.Site) {
	sig := hutSig(site)
	if sig == h.sig && h.hard != nil {
		return
	}
	h.sig, h.w, h.h = sig, site.Width, site.Height
	h.Builds++
	areas := site.Territories()
	h.owner = ownerOf(site, areas)
	h.areas, h.reach = make(map[string]int16, len(areas)), make(map[string]int, len(areas))
	for i, a := range areas {
		h.areas[a.Keep], h.reach[a.Keep] = int16(i+1), a.Radius
	}
	h.hard = hardGrid(site, h.owner)
	h.soft, h.lists, h.aims, h.Nears = nil, map[int16][]int{}, map[aimKey]float64{}, map[nearKey]nearSpot{}
	clear(h.Homeless)
	h.revalidate(site)
}

func (h *Hamlet) revalidate(site view.Site) {
	h.occ = make([]uint8, h.w*h.h)
	for name, s := range h.SpotList {
		keep, ok := KeepForNode(site, s.Node)
		i, in := h.index(s.At.X, s.At.Y)
		if !ok || !in || !h.hard[i] || h.owner[i] != h.areas[keep.ID] {
			delete(h.SpotList, name)
			delete(h.Homes, name)
			continue
		}
		h.occ[i] = occHut
	}
	for name, s := range h.SpotList {
		if s.Sown && h.hardAt(s.Garden) && h.occAt(s.Garden) == occFree {
			h.mark(s.Garden, occGarden)
			continue
		}
		s.Sown = false
		h.SpotList[name] = s
	}
	kept := h.gone[:0]
	for _, g := range h.gone {
		if h.hardAt(g.spot.At) {
			g.spot.Sown = false
			h.mark(g.spot.At, occHut)
			kept = append(kept, g)
		}
	}
	h.gone = kept
}

func hardGrid(site view.Site, owner []int16) []bool {
	blocked := BlockedGrid(site)
	out := make([]bool, len(owner))
	for i := range out {
		if owner[i] == 0 || blocked[i] {
			continue
		}
		x, y := i%site.Width, i/site.Width
		out[i] = Homeable(site.Tile(x, y)) && !InAnyShadow(site, x, y)
	}
	return out
}

func BlockedGrid(site view.Site) []bool {
	out := make([]bool, site.Width*site.Height)
	block := func(x0, y0, x1, y1 int) {
		for y := max(y0, 0); y <= min(y1, site.Height-1); y++ {
			for x := max(x0, 0); x <= min(x1, site.Width-1); x++ {
				out[y*site.Width+x] = true
			}
		}
	}
	for _, s := range site.Structures {
		lo, hi := Footprint(s)
		block(lo.X-1, lo.Y-1, hi.X+1, hi.Y+1)
	}
	for _, hv := range SiteHavens(site) {
		p := PlanHaven(site, hv)
		for _, d := range p.Deck {
			block(d.X-1, d.Y-1, d.X+1, d.Y+1)
		}
		for _, v := range p.Pier {
			t := TileOf(v)
			block(t.X-1, t.Y-1, t.X+1, t.Y+1)
		}
		for _, y := range hv.Yards() {
			block(y.X-1, y.Y-1, y.X+yardSize, y.Y+yardSize)
		}
	}
	for _, l := range PlanLanes(site) {
		for _, t := range append(l.Home.Claims(), l.Away.Claims()...) {
			block(t.X-1, t.Y-1, t.X+1, t.Y+1)
		}
	}
	return out
}

func (h *Hamlet) softGrid(site view.Site) []bool {
	if h.soft != nil {
		return h.soft
	}
	soft := make([]bool, len(h.hard))
	for i, ok := range h.hard {
		x, y := i%h.w, i/h.w
		soft[i] = ok && RoadLevel(site, x, y) == RoadNone && (h.cover == nil || !h.cover(x, y))
	}
	for _, k := range Keeps(site) {
		if at, ok := DoorHut(site, k); ok {
			if i, in := h.index(at.X, at.Y); in {
				soft[i] = false
			}
		}
		if i, in := h.index(GateTile(k).X, GateTile(k).Y); in {
			soft[i] = false
		}
	}
	h.soft = soft
	return soft
}

func (h *Hamlet) list(site view.Site, area int16) []int {
	if l, ok := h.lists[area]; ok {
		return l
	}
	soft := h.softGrid(site)
	l := []int{}
	for i, o := range h.owner {
		if o == area && area != 0 && soft[i] {
			l = append(l, i)
		}
	}
	h.lists[area] = l
	return l
}

func (h *Hamlet) index(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= h.w || y >= h.h {
		return 0, false
	}
	return y*h.w + x, true
}

func (h *Hamlet) occAt(p Point) uint8 {
	if i, ok := h.index(p.X, p.Y); ok && i < len(h.occ) {
		return h.occ[i]
	}
	return occFree
}

func (h *Hamlet) hardAt(p Point) bool {
	i, ok := h.index(p.X, p.Y)
	return ok && i < len(h.hard) && h.hard[i]
}

func (h *Hamlet) mark(p Point, v uint8) {
	if i, ok := h.index(p.X, p.Y); ok && i < len(h.occ) {
		h.occ[i] = v
	}
}
