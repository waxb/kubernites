package view

import (
	"strings"
)

const PlaceForumCorner = "forum-corner"

const ForumTitle = "The Forum"

const ForumTitleK8s = "No Node"

const ForumLesson = "etcd keeps the realm's record; what belongs to no Node stands at its door."

const ForumLessonK8s = "etcd keeps the cluster's record; objects on no Node (Services, HPAs, Pending Pods) stand beside it."

type Forum struct {
	Codex Point `json:"codex"`
	Size  Point `json:"size"`
	Side  Point `json:"side"`
	Rings int   `json:"rings"`
	Lo    Point `json:"lo"`
	Hi    Point `json:"hi"`
}

func (f Forum) CodexRect() Rect {
	return Rect{X0: f.Codex.X, Y0: f.Codex.Y, X1: f.Codex.X + f.Size.X - 1, Y1: f.Codex.Y + f.Size.Y - 1}
}

func (f Forum) Box() Rect {
	return Rect{X0: f.Lo.X, Y0: f.Lo.Y, X1: f.Hi.X, Y1: f.Hi.Y}
}

func (s Site) ForumSig(h uint64) uint64 {
	if s.Forum == nil {
		return Mix(h, 0)
	}
	f := s.Forum
	for _, v := range []int{f.Codex.X, f.Codex.Y, f.Side.X, f.Side.Y, f.Rings, f.Lo.X, f.Lo.Y, f.Hi.X, f.Hi.Y} {
		h = Mix(h, uint64(v))
	}
	return Mix(h, 1)
}

func (s Site) OnForum(x, y int) bool {
	i, ok := s.Index(x, y)
	return ok && s.Tiles[i] == TileForum
}

func (s Site) ForumBox() (Point, Point, bool) {
	if s.Forum == nil || s.Forum.Hi.X < s.Forum.Lo.X || s.Forum.Hi.Y < s.Forum.Lo.Y {
		return Point{}, Point{}, false
	}
	return s.Forum.Lo, s.Forum.Hi, true
}

func (s Site) ForumCentre() (Point, bool) {
	lo, hi, ok := s.ForumBox()
	return Point{X: (lo.X + hi.X) / 2, Y: (lo.Y + hi.Y) / 2}, ok
}

func (s Site) CoversForum(fp Rect) bool {
	return s.AnyTile(fp, s.OnForum)
}

func (s Site) TouchesForum(fp Rect) bool {
	return s.CoversForum(Rect{X0: fp.X0 - 1, Y0: fp.Y0 - 1, X1: fp.X1 + 1, Y1: fp.Y1 + 1})
}

func (s Site) ReachesForum(c Point, radius int) bool {
	ci, ok := s.Index(c.X, c.Y)
	if !ok || radius <= 0 {
		return false
	}
	home := s.regionOf(ci)
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			i, in := s.Index(c.X+dx, c.Y+dy)
			if in && WithinReach(dx, dy, radius) && s.regionOf(i) == home && s.Tiles[i] == TileForum {
				return true
			}
		}
	}
	return false
}

func HeraldDef() BuildDef {
	return BuildDef{Kind: StructHerald, W: 1, H: 2, Placement: PlaceForum}
}

func (s Site) AnyTile(fp Rect, test func(x, y int) bool) bool {
	return s.CountTiles(fp, test) > 0
}

func (s Site) CountTiles(fp Rect, test func(x, y int) bool) int {
	n := 0
	for y := fp.Y0; y <= fp.Y1; y++ {
		for x := fp.X0; x <= fp.X1; x++ {
			if test(x, y) {
				n++
			}
		}
	}
	return n
}

func (s Site) ForumSide(x, y int) bool {
	return !s.OnForum(x, y) && (s.OnForum(x-1, y) || s.OnForum(x+1, y) || s.OnForum(x, y-1) || s.OnForum(x, y+1))
}

func ForumDef(s Structure) (BuildDef, bool) {
	switch {
	case s.Kind == StructHerald:
		return HeraldDef(), true
	case s.Kind == StructPlaque:
		return PlaqueDef(), true
	case s.Kind == StructWarHorn:
		return HornDef(), true
	case IsDepotStructure(s):
		return DepotDef(strings.TrimPrefix(s.Kind, DepotPrefix)), true
	}
	def, ok := FindBuildDef(s.Kind)
	return def, ok && def.Placement == PlaceForum
}

func ForumStanding(s Structure) bool {
	_, ok := ForumDef(s)
	return ok
}
