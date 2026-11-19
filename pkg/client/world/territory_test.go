package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func landSite() view.Site {
	const n = 40
	tiles := make([]view.TileKind, n*n)
	for i := range tiles {
		tiles[i] = view.TileGrass
	}
	for x := 0; x < n; x++ {
		tiles[x], tiles[(n-1)*n+x] = view.TileWater, view.TileWater
	}
	for y := 0; y < n; y++ {
		tiles[y*n], tiles[y*n+n-1] = view.TileWater, view.TileWater
	}
	for x := 5; x < 12; x++ {
		tiles[20*n+x] = view.TileRoad
	}
	return view.Site{
		Width: n, Height: n, Tiles: tiles, Elevation: make([]uint8, n*n),
		Structures: []view.Structure{
			{ID: "tower", Kind: view.StructKeep, Variant: "tower", X: 8, Y: 8, W: 3, H: 3, Territory: 6, State: view.StateReady, Label: "Lone Tower",
				Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}},
			{ID: "stone", Kind: view.StructKeep, Variant: "stone", X: 22, Y: 22, W: 3, H: 3, Territory: 4, State: view.StateReady, Label: "Stone Keep 2",
				Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-2"}},
			{ID: "codex", Kind: view.StructCodex, X: 10, Y: 13, W: 2, H: 2, State: view.StateReady},
		},
	}
}

func chebyshev(a, b view.Point) int {
	return max(Abs(a.X-b.X), Abs(a.Y-b.Y))
}

func TestLandOwnsWhatTheRealmsTerritoriesOwn(t *testing.T) {
	site := landSite()
	l := BordersOf(site)
	if len(l.Areas) != 2 || l.Sig != BorderSig(site) {
		t.Fatalf("areas = %d", len(l.Areas))
	}
	counts := make([]int, len(l.Areas))
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			keep, held := site.TerritoryAt(x, y)
			i, owned := l.At(x, y)
			if held != owned || held && l.Areas[i].Keep != keep.ID {
				t.Fatalf("tile %d,%d: realm %v %s, land %v %d", x, y, held, keep.ID, owned, i)
			}
			if owned {
				counts[i]++
			}
		}
	}
	for i, a := range l.Areas {
		if counts[i] != a.Tiles {
			t.Errorf("%s holds %d tiles, the realm says %d", a.Keep, counts[i], a.Tiles)
		}
	}
	if _, ok := l.At(-1, 3); ok {
		t.Error("off the map is nobody's land")
	}
	if i, ok := l.AreaOf("stone"); !ok || l.Areas[i].Label != "Stone Keep 2" {
		t.Error("area lookup by keep id")
	}
	if _, ok := l.AreaOf("nope"); ok {
		t.Error("unknown keep has no area")
	}
}

func TestLandSigFollowsTheStructures(t *testing.T) {
	site := landSite()
	before := BorderSig(site)
	site.Structures = append(site.Structures, view.Structure{ID: "k9", Kind: view.StructKeep, Variant: "stone", X: 30, Y: 8, W: 3, H: 3, Territory: 4})
	if BorderSig(site) == before {
		t.Error("a new Keep changes the land")
	}
	moved := landSite()
	moved.Structures[1].X++
	if BorderSig(moved) == before {
		t.Error("a moved Keep changes the land")
	}
}

func TestBorderMarksRingEveryKeepWithSpacedStonesAndCornerFlags(t *testing.T) {
	site := landSite()
	l := BordersOf(site)
	border := map[view.Point]bool{}
	for _, a := range l.Areas {
		for _, p := range a.Border {
			border[p] = true
		}
	}
	flags := 0
	stones := []view.Point{}
	for _, m := range l.Marks {
		p := view.Point{X: m.X, Y: m.Y}
		if !border[p] {
			t.Errorf("mark %v is not on a border", p)
		}
		if m.Flag {
			flags++
			continue
		}
		for _, s := range stones {
			if chebyshev(s, p) <= 1 {
				t.Errorf("stones %v and %v touch", s, p)
			}
		}
		stones = append(stones, p)
		if m.Variant < 0 || m.Variant > 2 {
			t.Errorf("variant %d", m.Variant)
		}
	}
	if flags != 2*4 {
		t.Errorf("flags = %d, want 4 corners per Keep", flags)
	}
	for p := range border {
		if !FreeTile(site, p.X, p.Y) {
			continue
		}
		near := false
		for _, m := range l.Marks {
			near = near || chebyshev(p, view.Point{X: m.X, Y: m.Y}) <= 2
		}
		if !near {
			t.Errorf("border tile %v has no stone within 2 tiles", p)
		}
	}
}

func TestBorderMarksKeepOffRoadsAndBuildings(t *testing.T) {
	site := landSite()
	for _, m := range BordersOf(site).Marks {
		if !FreeTile(site, m.X, m.Y) {
			t.Errorf("mark on a road or a footprint at %d,%d", m.X, m.Y)
		}
	}
	if FreeTile(site, 6, 20) || FreeTile(site, 10, 13) || !FreeTile(site, 3, 3) {
		t.Error("road and footprint tiles are not free")
	}
}

func TestCornersPickTheFourScreenExtremes(t *testing.T) {
	a := view.Territory{Center: view.Point{X: 10, Y: 10}, Border: []view.Point{
		{X: 7, Y: 7}, {X: 13, Y: 13}, {X: 7, Y: 13}, {X: 13, Y: 7}, {X: 10, Y: 6}, {X: 8, Y: 7},
	}}
	got := Corners(a)
	want := map[view.Point]bool{{X: 7, Y: 7}: true, {X: 13, Y: 13}: true, {X: 7, Y: 13}: true, {X: 13, Y: 7}: true}
	if len(got) != 4 {
		t.Fatalf("corners = %v", got)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected corner %v", p)
		}
	}
	if len(Corners(view.Territory{Border: []view.Point{{X: 1, Y: 1}}})) != 0 {
		t.Error("a tiny land has no corners")
	}
}

func TestTintsCoverAreasAndDiscs(t *testing.T) {
	l := BordersOf(landSite())
	own := AreaTints(l, []int{1}, TintOwn)
	if len(own) != l.Areas[1].Tiles {
		t.Fatalf("own tints = %d, tiles = %d", len(own), l.Areas[1].Tiles)
	}
	for _, tn := range own {
		if i, ok := l.At(tn.X, tn.Y); !ok || i != 1 || tn.Kind != TintOwn {
			t.Fatalf("tint %+v outside its land", tn)
		}
	}
	if len(AreaTints(l, nil, TintOwn)) != 0 {
		t.Error("no areas, no tint")
	}
	disc := DiscTints([]view.Point{{X: 1, Y: 2}, {X: 3, Y: 4}}, TintPoor)
	if len(disc) != 2 || disc[1] != (Tint{X: 3, Y: 4, Kind: TintPoor}) {
		t.Errorf("disc = %+v", disc)
	}
	merged := MergeTints(own, DiscTints([]view.Point{{X: own[0].X, Y: own[0].Y}}, TintGood))
	if len(merged) != len(own) {
		t.Fatalf("merge keeps one tint per tile: %d", len(merged))
	}
	found := false
	for _, tn := range merged {
		if tn.X == own[0].X && tn.Y == own[0].Y {
			found = tn.Kind == TintGood
		}
	}
	if !found {
		t.Error("a later tint wins its tile")
	}
}

func TestMarkNamesWaveOnlyWithMotion(t *testing.T) {
	flag := BorderMark{X: 3, Y: 4, Flag: true}
	if MarkName(flag, 0, true) != MarkName(flag, 5.3, true) {
		t.Error("reduced motion holds the flag still")
	}
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		seen[MarkName(flag, float64(i)*.17, false)] = true
	}
	if len(seen) != 2 {
		t.Errorf("flag frames = %v", seen)
	}
	if MarkName(BorderMark{Variant: 2}, 0, false) != "border/stone/2" {
		t.Error("stone name")
	}
}

func TestRendererDrawsBordersAndTints(t *testing.T) {
	site := landSite()
	l := BordersOf(site)
	r := NewRenderer(nil)
	cam := render.NewCamera(480, 60).CenterOn(15, 15, 960, 480)
	sc := Scene{Site: site, Borders: &l, Tints: AreaTints(l, []int{0, 1}, TintGood)}
	items := r.BorderItems(cam, sc)
	if len(items) == 0 || len(items) > len(l.Marks) {
		t.Fatalf("border items = %d of %d", len(items), len(l.Marks))
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, it := range items {
		it.Draw(dst)
	}
	r.DrawTints(dst, cam, sc)
	r.Draw(dst, cam, sc)
	if len(r.BorderItems(cam, Scene{Site: site})) != 0 {
		t.Error("no land, no stones")
	}
}

func TestPreviewMarksRingTheWouldBeLand(t *testing.T) {
	site := landSite()
	tiles := site.TerritoryTiles(30, 8, 3, 3, 4)
	ring := RingOf(tiles)
	if len(ring) == 0 || len(ring) >= len(tiles) {
		t.Fatalf("ring %d of %d tiles", len(ring), len(tiles))
	}
	marks := PreviewMarks(site, tiles, view.Point{X: 31, Y: 9})
	flags := 0
	for _, m := range marks {
		if m.Flag {
			flags++
		}
	}
	if len(marks) < 8 || flags != 4 {
		t.Errorf("preview marks = %d, flags = %d", len(marks), flags)
	}
	if PreviewMarks(site, nil, view.Point{}) != nil {
		t.Error("no land, no preview")
	}
	l := BordersOf(site)
	r := NewRenderer(nil)
	cam := render.NewCamera(480, 60).CenterOn(30, 10, 960, 480)
	sc := Scene{Site: site, Borders: &l, Preview: marks, Spots: []SpotMark{{X: 30, Y: 8, W: 3, H: 3, Class: "compute"}}}
	if got := len(r.BorderItems(cam, sc)); got <= len(marks)/2 {
		t.Errorf("preview stones join the scene: %d", got)
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.DrawTints(dst, cam, sc)
	r.DrawSpots(dst, cam.ZoomAt(.5, 480, 240), sc)
}

func TestBorderSigFollowsCharters(t *testing.T) {
	site := landSite()
	site.Islands = []view.Island{{ID: "home", Owned: true}, {ID: "wild"}}
	before := BorderSig(site)
	site.Islands = []view.Island{{ID: "home", Owned: true}, {ID: "wild", Owned: true}}
	if BorderSig(site) == before {
		t.Error("chartering an isle changes where one may build")
	}
}

func TestAPlotLabelNamesItsMachine(t *testing.T) {
	plain := SpotMark{Class: "compute"}
	named := SpotMark{Class: "compute", Machine: "compute-xlarge: 8 Hands, 4 GiB Lore"}
	if got := SpotLabel(plain); got != "compute" {
		t.Errorf("a plot without a survey reads %q, want its class", got)
	}
	if got := SpotLabel(named); got != named.Machine {
		t.Errorf("a surveyed plot reads %q, want its machine", got)
	}
	best := named
	best.Note = "best for the porters"
	if got := BestLines(best, 2); len(got) != 2 || got[0] != best.Note || got[1] != best.Machine {
		t.Errorf("the best plot reads %q up close, want its note over its machine", got)
	}
	if got := BestLines(best, .5); len(got) != 1 || got[0] != "compute: best for the porters" {
		t.Errorf("the best plot reads %q from afar, want one line", got)
	}
}

func TestAPlotLabelStandsOnAPlateClearOfThePanels(t *testing.T) {
	lines := []string{"best for the porters", "compute-xlarge: 8 Hands, 4 GiB Lore"}
	open, ok := SpotPlate(lines, 300, 200, nil)
	if !ok || open.H != 12+ui.LineHeight || open.X > 300 || open.X+open.W < 300 {
		t.Fatalf("a two-line plate at 300,200 = %+v %v", open, ok)
	}
	panel := ui.Rect{X: open.X - 10, Y: open.Y - 10, W: 60, H: 60}
	moved, ok := SpotPlate(lines, 300, 200, []ui.Rect{panel})
	if !ok || Overlaps(moved, panel) || moved.X > 300 || moved.X+moved.W < 300 {
		t.Fatalf("the plate under a panel settled at %+v %v", moved, ok)
	}
}
