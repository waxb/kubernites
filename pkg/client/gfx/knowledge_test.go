package gfx

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestKnowledgeMapNameHelpers(t *testing.T) {
	checks := [][2]string{
		{BorderStoneName(4), "border/stone/1"},
		{BorderFlagName(-1), "border/flag/1"},
		{GroundTintName("poor"), "ground/tint/poor"},
		{GroundTintName("?"), "ground/tint/own"},
		{PageIconName("muster"), "page/muster"},
		{PageIconName("nope"), "page/blank"},
		{PlaqueName("learned"), "plaque-learned"},
		{PlaqueName(""), "plaque-locked"},
		{KMapMedallionName("researching"), "kmap/medallion/researching"},
		{KMapRoadName("open", true, false, true, false), "kmap/road/open/ns"},
		{KMapRoadName("gold", false, true, false, true), "kmap/road/locked/ew"},
		{KMapRoadName("learned", true, true, true, true), "kmap/road/learned/nesw"},
		{KMapRoadName("open", false, false, false, false), ""},
		{KMapBannerName("upkeep"), "kmap/banner/upkeep"},
		{KMapBannerName("magic"), ""},
		{KMapDecorName("compass"), "kmap/deco/compass"},
		{KMapParchmentName(), "kmap/parchment"},
		{KMapState("available"), "available"},
		{KMapRoadState("learned"), "learned"},
	}
	for _, c := range checks {
		if c[0] != c[1] {
			t.Errorf("got %q, want %q", c[0], c[1])
		}
	}
	if PlaqueTextColor("locked") == PlaqueTextColor("learned") {
		t.Error("locked plaques use the dim text colour")
	}
}

func TestKnowledgeMapNamesResolveInTheLibrary(t *testing.T) {
	lib := Load(art.Build())
	names := []string{KMapParchmentName()}
	for _, st := range art.KMapStates {
		names = append(names, KMapMedallionName(st), "ui/"+PlaqueName(st))
	}
	for _, st := range art.KMapRoadStates {
		for m := 1; m < 16; m++ {
			names = append(names, KMapRoadName(st, m&1 != 0, m&2 != 0, m&4 != 0, m&8 != 0))
		}
	}
	for _, b := range art.KMapBranches {
		names = append(names, KMapBannerName(b))
	}
	for _, d := range art.KMapDecor {
		names = append(names, KMapDecorName(d))
	}
	for _, id := range append(append([]string{}, art.PageIcons...), "missing-page") {
		names = append(names, PageIconName(id))
	}
	for v := -1; v < 4; v++ {
		names = append(names, BorderStoneName(v), BorderFlagName(v))
	}
	for _, k := range append(append([]string{}, art.GroundTints...), "x") {
		names = append(names, GroundTintName(k))
	}
	names = append(names, "icon/wood", "icon/stone", "decor/outcrop/2", "decor/quarry/0", "ui/kmap-frame", "ui/kmap-ribbon")
	if missing := Missing(lib, names); len(missing) > 0 {
		t.Fatalf("%d knowledge-map sprites missing, first %s", len(missing), missing[0])
	}
	if w, h := lib.Size(PageIconName("ranks")); w != PageIconSize || h != PageIconSize {
		t.Errorf("page icon is %dx%d", w, h)
	}
}

func TestPlaquesNineSliceAtTwoLinesOfText(t *testing.T) {
	lib := Load(art.Build())
	dst := ebiten.NewImage(200, 60)
	for _, st := range art.KMapStates {
		if !lib.NineSlice(dst, PlaqueName(st), ui.Rect{X: 4, Y: 4, W: 160, H: PlaqueHeight}) {
			t.Errorf("plaque %s does not nine-slice at 160x%d", st, PlaqueHeight)
		}
	}
	if !lib.NineSlice(dst, "kmap-frame", ui.Rect{W: 200, H: 60}) {
		t.Error("the map frame does not nine-slice")
	}
	if PlaqueHeight < 2*ui.LineHeight+8 {
		t.Fatalf("a %dpx plaque cannot hold two %dpx lines", PlaqueHeight, ui.LineHeight)
	}
}
