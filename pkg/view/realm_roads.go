package view

const WearMax = 20000

const WearTrodden = 400

const WearDirt = 2500

const WearCobbled = 8000

const WearKeepPerMil = 983

const fnvPrime = 1099511628211

type RouteKind uint8

type Route struct {
	Kind  RouteKind `json:"-"`
	Node  string    `json:"-"`
	Haven int       `json:"-"`
	Path  []int32   `json:"-"`
}

type RoadCache struct {
	Sig      uint64             `json:"-"`
	Routes   []Route            `json:"-"`
	Flows    []float64          `json:"-"`
	Front    map[string]float64 `json:"-"`
	Ledger   map[string]float64 `json:"-"`
	Builds   int                `json:"-"`
	Hand     HandFlow           `json:"-"`
	HandTick float64            `json:"-"`
}

func RoadLevel(w uint16) int {
	switch {
	case w >= WearCobbled:
		return 3
	case w >= WearDirt:
		return 2
	case w >= WearTrodden:
		return 1
	}
	return 0
}

func Mix(h, v uint64) uint64 {
	return (h ^ v) * fnvPrime
}

type RouteEnd struct {
	Kind  RouteKind `json:"-"`
	Node  string    `json:"-"`
	S     Structure `json:"-"`
	Haven int       `json:"-"`
	Tile  int       `json:"-"`
	Exact bool      `json:"-"`
}

func FirstStructure(site Site, kind string) (Structure, bool) {
	for _, s := range site.Structures {
		if s.Kind == kind {
			return s, true
		}
	}
	return Structure{}, false
}

func (g PathGrid) RoutesFrom(hub Structure, ends []RouteEnd) []Route {
	from, ok := g.Door(hub)
	if !ok || len(ends) == 0 {
		return nil
	}
	doors := make([]int, 0, len(ends))
	for _, e := range ends {
		doors = append(doors, g.endTile(e))
	}
	parent := g.Search(from, doors)
	out := []Route{}
	for i, e := range ends {
		if path := TracePath(parent, from, doors[i]); path != nil {
			out = append(out, Route{Kind: e.Kind, Node: e.Node, Haven: e.Haven, Path: path})
		}
	}
	return out
}

func (s *Site) EnsureWear() {
	if len(s.Wear) != len(s.Tiles) {
		s.Wear = make([]uint16, len(s.Tiles))
	}
}

func (s *Site) Tread(path []int32, step int) {
	if step <= 0 {
		return
	}
	for _, i := range path {
		switch s.Tiles[i] {
		case TileWater, TileShallows, TileRoad, TilePlaza, TileForum:
			continue
		}
		s.Wear[i] = uint16(min(int(s.Wear[i])+step, WearMax))
	}
}

func (s *Site) FadeWear() {
	for i, w := range s.Wear {
		if w > 0 {
			s.Wear[i] = uint16(uint32(w) * WearKeepPerMil / 1000)
		}
	}
}

func (s *Site) ClearUnderfoot() {
	if len(s.Wear) != len(s.Tiles) {
		return
	}
	for _, h := range s.Havens {
		yards := h.yardRects()
		for _, r := range [3]Rect{h.QuayRect(), yards[0], yards[1]} {
			s.clearWear(r)
		}
	}
	for _, st := range s.Structures {
		s.clearWear(Footprint(st))
	}
}

func (s *Site) clearWear(fp Rect) {
	for y := fp.Y0; y <= fp.Y1; y++ {
		for x := fp.X0; x <= fp.X1; x++ {
			if i, ok := s.Index(x, y); ok {
				s.Wear[i] = 0
			}
		}
	}
}
