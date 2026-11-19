package view

const ForumBand = 4

const ForumPad = 3

const ForumDepth = 5

const ForumPlanRing = 3

const forumAway = 2

const ForumFront = 12

var ForumSides = [4]Point{{X: 1}, {Y: 1}, {X: -1}, {Y: -1}}

type forumGround struct {
	site    Site
	centre  Point
	radius  int
	region  uint8
	level   uint8
	area    Rect
	blocked []bool
}

type forumChoice struct {
	Side  Point `json:"-"`
	Base  Rect  `json:"-"`
	plan  Rect
	Laid  bool `json:"-"`
	whole bool
	Away  int `json:"-"`
	Score int `json:"-"`
}

type ForumPreview struct {
	Side Point
	Base []Point
	Plan []Point
}

func (s Site) loneTower() (Structure, bool) {
	for _, st := range s.Structures {
		if st.Kind == StructKeep && st.Variant == TowerVariant {
			return st, true
		}
	}
	return Structure{}, false
}

func ForumRect(codex Rect, side Point, ring int) Rect {
	pad, depth := ForumPad+ring, ForumDepth+ring
	switch {
	case side.X > 0:
		return Rect{X0: codex.X1 + 1, Y0: codex.Y0 - pad, X1: codex.X1 + depth, Y1: codex.Y1 + pad}
	case side.X < 0:
		return Rect{X0: codex.X0 - depth, Y0: codex.Y0 - pad, X1: codex.X0 - 1, Y1: codex.Y1 + pad}
	case side.Y > 0:
		return Rect{X0: codex.X0 - pad, Y0: codex.Y1 + 1, X1: codex.X1 + pad, Y1: codex.Y1 + depth}
	}
	return Rect{X0: codex.X0 - pad, Y0: codex.Y0 - depth, X1: codex.X1 + pad, Y1: codex.Y0 - 1}
}

func (s Site) ForumGroundFor(codex, area Rect) (forumGround, bool) {
	tower, ok := s.loneTower()
	if !ok {
		return forumGround{}, false
	}
	c := KeepCenter(tower)
	ci, in := s.Index(c.X, c.Y)
	if !in {
		return forumGround{}, false
	}
	g := forumGround{site: s, centre: c, radius: KeepRadius(tower) + ForumBand, region: s.regionOf(ci), level: s.ElevationAt(codex.X0, codex.Y0), area: area}
	g.blocked = make([]bool, (area.X1-area.X0+1)*(area.Y1-area.Y0+1))
	for _, st := range s.Structures {
		g.fence(st, tower.ID)
	}
	stone, _ := KeepTier(StructKeep, false)
	for _, sp := range s.Spots {
		if sp.Isle == HomeIslandID {
			g.claim(Point{X: sp.X + 1, Y: sp.Y + 1}, stone.Territory)
		}
	}
	return g, true
}

func (g forumGround) fence(st Structure, tower string) {
	if r := KeepRadius(st); r > 0 && st.ID != tower {
		g.claim(KeepCenter(st), r)
	}
	if ForumStanding(st) {
		return
	}
	fp := Footprint(st)
	for y := max(fp.Y0, g.area.Y0); y <= min(fp.Y1, g.area.Y1); y++ {
		for x := max(fp.X0, g.area.X0); x <= min(fp.X1, g.area.X1); x++ {
			g.blocked[g.slot(x, y)] = true
		}
	}
}

func (g forumGround) claim(c Point, radius int) {
	for y := max(c.Y-radius, g.area.Y0); y <= min(c.Y+radius, g.area.Y1); y++ {
		for x := max(c.X-radius, g.area.X0); x <= min(c.X+radius, g.area.X1); x++ {
			if WithinReach(x-c.X, y-c.Y, radius) {
				g.blocked[g.slot(x, y)] = true
			}
		}
	}
}

func (g forumGround) slot(x, y int) int {
	return (y-g.area.Y0)*(g.area.X1-g.area.X0+1) + (x - g.area.X0)
}

func (g forumGround) layable(x, y int) bool {
	i, ok := g.site.Index(x, y)
	if !ok || !g.area.Contains(x, y) || g.blocked[g.slot(x, y)] {
		return false
	}
	if !WithinReach(x-g.centre.X, y-g.centre.Y, g.radius) || g.site.regionOf(i) != g.region || g.site.ElevationAt(x, y) != g.level {
		return false
	}
	switch g.site.Tiles[i] {
	case TileGrass, TileMeadow, TileSand, TileForum:
	default:
		return false
	}
	_, harbor := g.site.HavenOver(Rect{X0: x, Y0: y, X1: x, Y1: y})
	return !harbor
}

func (g forumGround) fits(area Rect) bool {
	for y := area.Y0; y <= area.Y1; y++ {
		for x := area.X0; x <= area.X1; x++ {
			if !g.layable(x, y) {
				return false
			}
		}
	}
	return true
}
