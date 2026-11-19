package view

type Territory struct {
	Keep   string  `json:"keep"`
	Label  string  `json:"label"`
	Center Point   `json:"center"`
	Radius int     `json:"radius"`
	Tiles  int     `json:"tiles"`
	Border []Point `json:"border"`
}

type LandCache struct {
	Sig      uint64               `json:"-"`
	Owner    []int16              `json:"-"`
	Surveys  map[SurveyKey]Survey `json:"-"`
	Spots    map[string]KeepSpot  `json:"-"`
	ForumAge string               `json:"-"`
}

func WithinReach(dx, dy, radius int) bool {
	return dx*dx+dy*dy <= radius*radius+radius
}

func KeepCenter(s Structure) Point {
	return Point{X: s.X + s.W/2, Y: s.Y + s.H/2}
}

func CenterOf(x, y, w, h int) Point {
	return Point{X: x + w/2, Y: y + h/2}
}

func KeepRadius(s Structure) int {
	switch {
	case s.Kind != StructKeep:
		return 0
	case s.Territory > 0:
		return s.Territory
	case s.Variant == TowerVariant:
		return TowerTerritory
	}
	if kt, ok := KeepTypeOf(s.Variant); ok {
		return kt.Territory
	}
	return 0
}

func (s Site) regionOf(i int) uint8 {
	if len(s.Region) == len(s.Tiles) {
		return s.Region[i]
	}
	if IsLand(s.Tiles[i]) {
		return 1
	}
	return 0
}

func (s Site) EachInReach(c Point, radius int, fn func(i, d2 int)) {
	ci, ok := s.Index(c.X, c.Y)
	if !ok || radius <= 0 {
		return
	}
	home := s.regionOf(ci)
	if home == 0 {
		return
	}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if !WithinReach(dx, dy, radius) {
				continue
			}
			if i, ok := s.Index(c.X+dx, c.Y+dy); ok && s.regionOf(i) == home && s.Tiles[i] != TileForum {
				fn(i, dx*dx+dy*dy)
			}
		}
	}
}

func (s Site) OwnerGrid() []int16 {
	owner := make([]int16, len(s.Tiles))
	best := make([]int32, len(s.Tiles))
	for k, st := range s.Structures {
		s.EachInReach(KeepCenter(st), KeepRadius(st), func(i, d2 int) {
			if owner[i] == 0 || int32(d2) < best[i] {
				owner[i], best[i] = int16(k+1), int32(d2)
			}
		})
	}
	return owner
}

func (s Site) KeepLand() []int16 {
	return s.OwnerGrid()
}

func (s Site) TerritoryAt(x, y int) (Structure, bool) {
	i, ok := s.Index(x, y)
	if !ok || s.Tiles[i] == TileForum {
		return Structure{}, false
	}
	found, best, out := false, 0, Structure{}
	for _, st := range s.Structures {
		c, radius := KeepCenter(st), KeepRadius(st)
		dx, dy := x-c.X, y-c.Y
		ci, inside := s.Index(c.X, c.Y)
		if radius == 0 || !inside || !WithinReach(dx, dy, radius) || s.regionOf(ci) != s.regionOf(i) || s.regionOf(i) == 0 {
			continue
		}
		if d2 := dx*dx + dy*dy; !found || d2 < best {
			found, best, out = true, d2, st
		}
	}
	return out, found
}

func (s Site) Territories() []Territory {
	owner := s.OwnerGrid()
	slot := make([]int, len(s.Structures))
	out := []Territory{}
	for k, st := range s.Structures {
		slot[k] = -1
		if radius := KeepRadius(st); radius > 0 {
			slot[k] = len(out)
			out = append(out, Territory{Keep: st.ID, Label: st.Label, Center: KeepCenter(st), Radius: radius, Border: []Point{}})
		}
	}
	for i, o := range owner {
		if o == 0 {
			continue
		}
		t := &out[slot[o-1]]
		t.Tiles++
		if s.OnBorder(owner, i) {
			t.Border = append(t.Border, Point{X: i % s.Width, Y: i / s.Width})
		}
	}
	return out
}

func (s Site) OnBorder(owner []int16, i int) bool {
	x, y := i%s.Width, i/s.Width
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		j, ok := s.Index(x+d[0], y+d[1])
		if !ok || owner[j] != owner[i] && s.Tiles[j] != TileForum {
			return true
		}
	}
	return false
}

func (s Site) TerritoryTiles(x, y, w, h, radius int) []Point {
	out := []Point{}
	s.EachInReach(CenterOf(x, y, w, h), radius, func(i, _ int) {
		out = append(out, Point{X: i % s.Width, Y: i / s.Width})
	})
	return out
}

func (c *LandCache) OwnerAt(s Site, x, y int) (Structure, bool) {
	i, ok := s.Index(x, y)
	if !ok || i >= len(c.Owner) || c.Owner[i] == 0 || int(c.Owner[i]) > len(s.Structures) {
		return Structure{}, false
	}
	return s.Structures[c.Owner[i]-1], true
}
