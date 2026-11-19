package world

import (
	"fmt"
	"math"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func forumProps(site view.Site, kerb kerbPlan, tiles []forumTile, replaced map[Vec]bool) []forumProp {
	free := freeRim(site, tiles)
	out := termini(kerb)
	out = append(out, statues(free)...)
	taken := map[Point]bool{}
	for _, p := range out {
		taken[Point{int(math.Round(p.at.X)), int(math.Round(p.at.Y))}] = true
	}
	if f, ok := fountainSpot(site, free, taken); ok {
		out = append(out, forumProp{kind: PropFountain, at: Vec{float64(f.X), float64(f.Y)}, id: "forum/prop/fountain", from: tierUAT, upTo: tierProd})
		taken[f] = true
	}
	if r, ok := rostraSpot(site, free, taken); ok {
		out = append(out, forumProp{kind: PropRostra, at: Vec{float64(r.X), float64(r.Y)}, axis: rostraAxis(site.Forum.Side), id: "forum/prop/rostra", from: tierProd, upTo: tierProd})
	}
	out = append(out, archProps(site, kerb, replaced)...)
	openPortico(out, tiles, replaced)
	return append(out, colonnades(kerb, replaced)...)
}

func openPortico(props []forumProp, tiles []forumTile, replaced map[Vec]bool) {
	rims := make(map[Point][4]bool, len(tiles))
	for _, t := range tiles {
		rims[t.at] = t.rim
	}
	for _, p := range props {
		if p.kind != PropStatue && p.kind != PropFountain && p.kind != PropRostra {
			continue
		}
		at := TileOf(p.at)
		rim := rims[at]
		if rim[2] {
			replaced[Vec{float64(at.X) - .5, float64(at.Y)}], replaced[Vec{float64(at.X) - .5, float64(at.Y + 1)}] = true, true
		}
		if rim[3] {
			replaced[Vec{float64(at.X), float64(at.Y) - .5}], replaced[Vec{float64(at.X + 1), float64(at.Y) - .5}] = true, true
		}
	}
}

func freeRim(site view.Site, tiles []forumTile) []forumTile {
	out := []forumTile{}
	for _, t := range tiles {
		x, y := t.at.X, t.at.Y
		if !(t.rim[0] || t.rim[1] || t.rim[2] || t.rim[3]) || Occupied(site, x, y) || besideCodex(site, x, y) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func besideCodex(site view.Site, x, y int) bool {
	codex, ok := FirstOfKind(site, view.StructCodex, false)
	return ok && nearFootprint(codex, x, y)
}

func termini(kerb kerbPlan) []forumProp {
	out := []forumProp{}
	for i, p := range kerb.posts {
		if p.corner || i%terminusEvery == 0 {
			out = append(out, forumProp{kind: PropTerminus, at: p.at, v: p.v % art.ForumTermini, id: fmt.Sprintf("forum/prop/terminus/%d", i), from: 0, upTo: 0})
		}
	}
	return out
}

func statues(free []forumTile) []forumProp {
	out := []forumProp{}
	for _, t := range free {
		if !(t.rim[0] || t.rim[2]) || !(t.rim[1] || t.rim[3]) {
			continue
		}
		id := fmt.Sprintf("forum/prop/statue/%d/%d", t.at.X, t.at.Y)
		out = append(out, forumProp{kind: PropStatue, at: Vec{float64(t.at.X), float64(t.at.Y)}, v: int(t.h % art.ForumStatues), id: id, from: tierUAT, upTo: tierProd})
	}
	return out
}

func fountainSpot(site view.Site, free []forumTile, taken map[Point]bool) (Point, bool) {
	heart, _ := forumHeart(site)
	best, found, bestScore := Point{}, false, math.Inf(1)
	for _, t := range free {
		if taken[t.at] || !(t.rim[2] || t.rim[3]) {
			continue
		}
		if s := Dist(Vec{float64(t.at.X), float64(t.at.Y)}, heart); s < bestScore {
			best, found, bestScore = t.at, true, s
		}
	}
	return best, found
}

func rostraSpot(site view.Site, free []forumTile, taken map[Point]bool) (Point, bool) {
	f := site.Forum
	codex := Vec{float64(f.Codex.X) + float64(f.Size.X-1)/2, float64(f.Codex.Y) + float64(f.Size.Y-1)/2}
	best, found, bestScore := Point{}, false, math.Inf(1)
	for _, t := range free {
		back := Point{t.at.X - f.Side.X, t.at.Y - f.Side.Y}
		if taken[t.at] || site.OnForum(back.X, back.Y) {
			continue
		}
		d := Dist(Vec{float64(t.at.X), float64(t.at.Y)}, codex)
		if d < 2 {
			continue
		}
		if d < bestScore {
			best, found, bestScore = t.at, true, d
		}
	}
	return best, found
}

func rostraAxis(side view.Point) string {
	if side.X != 0 {
		return "y"
	}
	return "x"
}

func edgeAxis(dx int) string {
	if dx != 0 {
		return "y"
	}
	return "x"
}

func colonnades(kerb kerbPlan, replaced map[Vec]bool) []forumProp {
	out := []forumProp{}
	for i, k := range kerb.pieces {
		if (k.dx >= 0 && k.dy >= 0) || replaced[k.mid] {
			continue
		}
		replaced[k.mid] = true
		out = append(out, forumProp{kind: PropColonnade, at: k.mid, axis: edgeAxis(k.dx), v: k.v % art.ForumColonnades, id: fmt.Sprintf("forum/prop/colonnade/%d", i), from: tierProd, upTo: tierProd})
	}
	return out
}

func archProps(site view.Site, kerb kerbPlan, replaced map[Vec]bool) []forumProp {
	side := site.Forum.Side
	heart, _ := forumHeart(site)
	far := map[Vec]kerbPiece{}
	for _, k := range kerb.pieces {
		if k.dx == side.X && k.dy == side.Y {
			far[k.mid] = k
		}
	}
	lateral := Vec{float64(side.Y), float64(side.X)}
	best, found, bestScore := Vec{}, false, math.Inf(1)
	for _, k := range kerb.pieces {
		next := Vec{k.mid.X + lateral.X, k.mid.Y + lateral.Y}
		if _, ok := far[k.mid]; !ok {
			continue
		}
		if _, ok := far[next]; !ok {
			continue
		}
		v := Vec{(k.mid.X + next.X) / 2, (k.mid.Y + next.Y) / 2}
		if s := math.Abs((v.X-heart.X)*lateral.X + (v.Y-heart.Y)*lateral.Y); s < bestScore {
			best, found, bestScore = v, true, s
		}
	}
	if !found {
		return nil
	}
	for _, m := range []Vec{{best.X - lateral.X/2, best.Y - lateral.Y/2}, {best.X + lateral.X/2, best.Y + lateral.Y/2}} {
		replaced[m] = true
	}
	return []forumProp{{kind: PropArch, at: best, axis: edgeAxis(side.X), id: "forum/prop/arch", from: tierProd, upTo: tierProd}}
}
