package world

import (
	"fmt"
	"math"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	PropTerminus  = "terminus"
	PropStatue    = "statue"
	PropFountain  = "fountain"
	PropRostra    = "rostra"
	PropArch      = "arch"
	PropColonnade = "colonnade"
	bandEvery     = 4
	bandAt        = 2
	terminusEvery = 5
	tierUAT       = 2
	tierProd      = 3
)

var forumNormals = [4]Point{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

type forumTile struct {
	at     Point
	id     string
	h      uint32
	rim    [4]bool
	bands  []string
	rose   Point
	inRose bool
}

type forumProp struct {
	kind string
	at   Vec
	axis string
	v    int
	id   string
	from int
	upTo int
}

type forumDecor struct {
	sig   uint64
	tiles []forumTile
	props []forumProp
	kerb  kerbPlan
	edges map[Vec]bool
	names [art.AgeTiers][][]string
}

func planForumDecor(site view.Site) forumDecor {
	d := forumDecor{sig: forumSig(site), kerb: PlanKerb(site), edges: map[Vec]bool{}}
	w, ok := forumWindowOf(site)
	if !ok || site.Forum == nil {
		return d
	}
	lo, _, _ := site.ForumBox()
	heart, _ := forumHeart(site)
	centre := view.Point{X: int(math.Round(heart.X)), Y: int(math.Round(heart.Y))}
	rose := roseFits(site, centre)
	w.each(func(x, y int) {
		if site.OnForum(x, y) {
			d.tiles = append(d.tiles, planTile(site, x, y, lo, centre, rose))
		}
	})
	d.props = forumProps(site, d.kerb, d.tiles, d.edges)
	return d
}

func planTile(site view.Site, x, y int, lo, centre view.Point, rose bool) forumTile {
	t := forumTile{at: Point{x, y}, id: fmt.Sprintf("forum/%d/%d", x, y), h: hash2(x, y)}
	for i, n := range forumNormals {
		t.rim[i] = !site.OnForum(x+n.X, y+n.Y)
	}
	if (x-lo.X)%bandEvery == bandAt {
		t.bands = append(t.bands, "y")
	}
	if (y-lo.Y)%bandEvery == bandAt {
		t.bands = append(t.bands, "x")
	}
	if dx, dy := x-centre.X, y-centre.Y; rose && Abs(dx) <= art.ForumRoseReach && Abs(dy) <= art.ForumRoseReach {
		t.rose, t.inRose = Point{dx, dy}, true
	}
	return t
}

func forumPropTiles(site view.Site) map[Point]bool {
	d := planForumDecor(site)
	out := map[Point]bool{}
	for _, p := range d.props {
		if p.kind == PropStatue || p.kind == PropFountain || p.kind == PropRostra {
			out[TileOf(p.at)] = true
		}
	}
	return out
}

func roseFits(site view.Site, c view.Point) bool {
	for dy := -art.ForumRoseReach; dy <= art.ForumRoseReach; dy++ {
		for dx := -art.ForumRoseReach; dx <= art.ForumRoseReach; dx++ {
			if !site.OnForum(c.X+dx, c.Y+dy) {
				return false
			}
		}
	}
	return true
}

func paveVariant(t forumTile, age int) int {
	even := (t.at.X+t.at.Y)%2 == 0
	switch {
	case age == tierUAT && even:
		return 0
	case age == tierUAT:
		return 1 + ((t.at.X%2)+2)%2
	case age == tierProd && even:
		return 1
	case age == tierProd:
		return 2
	}
	return int(t.h % art.ForumPaveVariants)
}

func (d *forumDecor) tileNames(i, age int) []string {
	if d.names[age] == nil {
		d.names[age] = make([][]string, len(d.tiles))
	}
	if n := d.names[age][i]; n != nil {
		return n
	}
	t := d.tiles[i]
	out := []string{art.ForumPaveName(age, paveVariant(t, age))}
	if age >= tierUAT {
		for s, rim := range t.rim {
			if rim {
				out = append(out, art.ForumRimName(age, art.ForumSides[s]))
			}
		}
		if t.inRose {
			out = append(out, art.ForumRoseName(age, t.rose.X, t.rose.Y))
		}
	}
	if age == tierProd {
		for _, b := range t.bands {
			out = append(out, art.ForumBandName(age, b))
		}
	}
	d.names[age][i] = out
	return out
}
