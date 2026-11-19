package art

import (
	"image"
	"testing"
)

func art2Sprites() []spriteSpec {
	out := []spriteSpec{}
	hut := hearthSize()
	for _, q := range HearthQoS {
		for age := 0; age < AgeTiers; age++ {
			for _, open := range []bool{false, true} {
				out = append(out, spriteSpec{name: ShutterName(q, age, open), w: hut.w, h: hut.h, ax: hut.ax, ay: hut.ay})
			}
			for f := 0; f < BellFrames; f++ {
				out = append(out, spriteSpec{name: BellName(q, age, f), w: hut.w, h: hut.h, ax: hut.ax, ay: hut.ay})
			}
		}
	}
	for f := 0; f < HourglassFrames; f++ {
		out = append(out, spriteSpec{name: HourglassName(f), w: HourglassW, h: HourglassH, ax: HourglassW / 2, ay: HourglassH - 1})
	}
	for n := 1; n <= KnockMax; n++ {
		out = append(out, spriteSpec{name: KnockName(n), w: MarkTagW, h: MarkTagH, ax: MarkTagAX, ay: MarkTagAY})
	}
	out = append(out, spriteSpec{name: FrostName(), w: MarkTagW, h: MarkTagH, ax: MarkTagAX, ay: MarkTagAY})
	for n := 1; n <= RestartBadgeMax; n++ {
		out = append(out, spriteSpec{name: RestartBadgeName(n), w: RestartBadgeW, h: RestartBadgeH, ax: RestartBadgeW / 2, ay: RestartBadgeH - 1})
	}
	for step := 0; step <= len(BackoffWaits); step++ {
		out = append(out, spriteSpec{name: BackoffRingName(step), w: BackoffRingW, h: BackoffRingH, ax: BackoffRingAX, ay: BackoffRingAY})
	}
	for f := 0; f < IncidentFrames; f++ {
		out = append(out, spriteSpec{name: IncidentMarkerName(f), w: IncidentMarkerW, h: IncidentMarkerH, ax: IncidentMarkerW / 2, ay: IncidentMarkerH - 1})
	}
	for _, k := range ClueKinds {
		for _, lit := range []bool{false, true} {
			out = append(out, spriteSpec{name: ClueIconName(k, lit), w: IconSize, h: IconSize, ax: 8, ay: 8, atlasIsIcons: true})
		}
	}
	for _, k := range TimelineKinds {
		out = append(out, spriteSpec{name: TimelineIconName(k), w: IconSize, h: IconSize, ax: 8, ay: 8, atlasIsIcons: true})
	}
	for _, k := range TreasureKinds {
		for _, open := range []bool{false, true} {
			out = append(out, spriteSpec{name: TreasureChestName(k, open), w: ChestW, h: ChestH, ax: ChestW / 2, ay: ChestH - 2})
		}
	}
	return out
}

func TestArt2SpritesExistWithTheirSizesAndStayInThePalette(t *testing.T) {
	s := built()
	for _, sp := range art2Sprites() {
		a := specAtlas(s, sp)
		r, ok := a.Regions[sp.name]
		if !ok {
			t.Fatalf("%s is missing", sp.name)
		}
		if r.W != sp.w || r.H != sp.h || r.AnchorX != sp.ax || r.AnchorY != sp.ay {
			t.Fatalf("%s: got %dx%d anchor %d,%d, want %dx%d anchor %d,%d", sp.name, r.W, r.H, r.AnchorX, r.AnchorY, sp.w, sp.h, sp.ax, sp.ay)
		}
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				if col := regionAt(a, r, x, y); col.A == 255 && paletteDistance(col) > 64 {
					t.Fatalf("%s: colour %v at %d,%d is off the palette", sp.name, col, x, y)
				}
			}
		}
	}
	if _, ok := s.Walkers.Regions[walkerName("watchman", "walk", "se", 0)]; !ok {
		t.Fatal("the watchman has no walk cycle")
	}
}

func silhouette(a Atlas, r Region) image.Rectangle {
	out := image.Rectangle{Min: image.Pt(r.W, r.H)}
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if regionAt(a, r, x, y).A > 0 {
				out.Min.X, out.Min.Y = min(out.Min.X, x), min(out.Min.Y, y)
				out.Max.X, out.Max.Y = max(out.Max.X, x+1), max(out.Max.Y, y+1)
			}
		}
	}
	return out
}

func TestShuttersAndProbesReadDifferently(t *testing.T) {
	a := built().Buildings
	for _, q := range HearthQoS {
		shut, open := a.Regions[ShutterName(q, 1, false)], a.Regions[ShutterName(q, 1, true)]
		if pixelsApart(a, shut, open) < 10 || countNear(a, open, lampLit, 30) == 0 {
			t.Errorf("%s: shut and open shutters do not read apart", q)
		}
		if pixelsApart(a, a.Regions[BellName(q, 1, 0)], a.Regions[BellName(q, 1, 1)]) == 0 {
			t.Errorf("%s: the bell never swings", q)
		}
	}
	for n := 2; n <= KnockMax; n++ {
		if opaquePixels(a, a.Regions[KnockName(n)]) <= opaquePixels(a, a.Regions[KnockName(n-1)]) {
			t.Fatalf("knock %d does not show more than knock %d", n, n-1)
		}
	}
	if pixelsApart(a, a.Regions[HourglassName(0)], a.Regions[HourglassName(HourglassFrames-1)]) == 0 {
		t.Fatal("the hourglass sand never runs")
	}
}

func TestIncidentMarksGrowWithTheirStory(t *testing.T) {
	a := built().Buildings
	if RestartBadgeName(0) != RestartBadgeName(1) || RestartBadgeName(150) != RestartBadgeName(99) {
		t.Fatal("restart badge numbers clamp to 1..99")
	}
	if pixelsApart(a, a.Regions[RestartBadgeName(7)], a.Regions[RestartBadgeName(8)]) == 0 || pixelsApart(a, a.Regions[RestartBadgeName(12)], a.Regions[RestartBadgeName(21)]) == 0 {
		t.Fatal("different restart counts look the same")
	}
	last := -1
	for step := 0; step <= len(BackoffWaits); step++ {
		lit := countNear(a, a.Regions[BackoffRingName(step)], backoffLit, 1)
		if lit <= last && step > 0 {
			t.Fatalf("back-off step %d lights %d pixels, not more than %d", step, lit, last)
		}
		last = lit
	}
	arcs := backoffArcs()
	for i := 1; i < len(arcs); i++ {
		if arcs[i].sweep <= arcs[i-1].sweep {
			t.Fatalf("back-off segment %d is not longer than %d", i, i-1)
		}
	}
	for f := 1; f < IncidentFrames; f++ {
		if pixelsApart(a, a.Regions[IncidentMarkerName(f-1)], a.Regions[IncidentMarkerName(f)]) == 0 {
			t.Fatalf("incident frames %d and %d are the same", f-1, f)
		}
	}
}

func TestCluesDimUntilFoundAndChestsOpenWithAGleam(t *testing.T) {
	s := built()
	for _, k := range ClueKinds {
		dim, lit := s.Icons.Regions[ClueIconName(k, false)], s.Icons.Regions[ClueIconName(k, true)]
		if meanOf(s.Icons, dim).R >= meanOf(s.Icons, lit).R && meanOf(s.Icons, dim).G >= meanOf(s.Icons, lit).G {
			t.Errorf("clue %s is not dimmer until found", k)
		}
	}
	b := s.Buildings
	for _, k := range TreasureKinds {
		shut, open := b.Regions[TreasureChestName(k, false)], b.Regions[TreasureChestName(k, true)]
		if countNear(b, open, goldR[3], 20) <= countNear(b, shut, goldR[3], 20) {
			t.Errorf("the %s chest opens without a gleam", k)
		}
	}
	if pixelsApart(b, b.Regions[TreasureChestName("ledger", false)], b.Regions[TreasureChestName("sealed", false)]) < 6 {
		t.Fatal("chests of different kinds carry the same mark")
	}
}
