package art

import (
	"image"
	"strings"
	"testing"
)

const (
	oldPersonHeight  = 21.4
	oldWandererWidth = 10
)

var smallCast = list("lamplighter", "baker", "scribe", "granary-keeper", "archivist", "courier", "tallyman", "oracle",
	"chancellor", "codex-keeper", "quartermaster", "steward", "herald", "wayfinder", "augur", "reeve", "pilgrim",
	"builder", "wanderer", "helmbearer", "clerk")

var oldBeastHeights = map[string]float64{"imp": 15.2, "sheep": 13.4, "raven": 9.7, "hound": 14.5}

func solidBounds(a Atlas, r Region) image.Rectangle {
	x0, y0, x1, y1 := r.W, r.H, -1, -1
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 200 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	return image.Rect(x0, y0, x1+1, y1+1)
}

func averageHeight(a Atlas, role string) float64 {
	sum, n := 0, 0
	for name, r := range a.Regions {
		if strings.HasPrefix(name, "walker/"+role+"/") {
			sum += solidBounds(a, r).Dy()
			n++
		}
	}
	return float64(sum) / float64(max(n, 1))
}

func TestPeopleAreFifteenPercentSmaller(t *testing.T) {
	a := built().Walkers
	total := 0.0
	for _, role := range smallCast {
		r := a.Regions[walkerName(role, "idle", "se", 0)]
		b := solidBounds(a, r)
		if b.Max.Y-1 != r.AnchorY {
			t.Errorf("%s: feet end on row %d, anchor is %d", role, b.Max.Y-1, r.AnchorY)
		}
		if b.Dy() > 21 {
			t.Errorf("%s: idle figure is %d px tall", role, b.Dy())
		}
		total += averageHeight(a, role)
	}
	if avg := total / float64(len(smallCast)); avg > oldPersonHeight*0.88 || avg < oldPersonHeight*0.78 {
		t.Errorf("people average %.1f px, want about 15%% under %.1f", avg, oldPersonHeight)
	}
}

func TestBodiesAreNarrower(t *testing.T) {
	a := built().Walkers
	for f := 0; f < WalkerFrames; f++ {
		if w := solidBounds(a, a.Regions[walkerName("wanderer", "idle", "se", f)]).Dx(); w >= oldWandererWidth {
			t.Errorf("wanderer idle frame %d is %d px wide, was %d", f, w, oldWandererWidth)
		}
	}
}

func TestBeastRolesShrinkWithThePeople(t *testing.T) {
	a := built().Walkers
	for role, old := range oldBeastHeights {
		if h := averageHeight(a, role); h > old*0.92 {
			t.Errorf("%s averages %.1f px, was %.1f", role, h, old)
		}
	}
}

func TestWriteWalkerSheets(t *testing.T) {
	dir := sheetDir(t)
	a := built().Walkers
	rows := [][]sheetCell{}
	for _, action := range list("idle", "walk") {
		names := []string{}
		for _, role := range smallCast {
			names = append(names, walkerName(role, action, "se", 1))
		}
		rows = append(rows, cellsOf(a, names, true))
	}
	beasts := []string{}
	for _, role := range append(list("imp", "sheep", "raven", "hound"), CritterRoles...) {
		beasts = append(beasts, walkerName(role, "idle", "se", 0))
	}
	rows = append(rows, cellsOf(a, beasts, true))
	frames := []string{}
	for _, action := range WalkerActions {
		for f := 0; f < WalkerFrames; f++ {
			frames = append(frames, walkerName("lamplighter", action, "se", f))
		}
	}
	rows = append(rows, cellsOf(a, frames, true))
	looks := []string{}
	for v := 1; v < WalkerLooks; v++ {
		looks = append(looks, walkerLookName("lamplighter", 1, v, "walk", "sw", 1), walkerLookName("pilgrim", 1, v, "walk", "se", 1))
	}
	rows = append(rows, cellsOf(a, append(looks, walkerLookName("lamplighter", 2, 0, "idle", "se", 0), walkerLookName("lamplighter", 3, 0, "idle", "se", 0)), true))
	writeScaled(t, dir, "walkers-after", rows, grassBackground, 1, 2, 4)
}
