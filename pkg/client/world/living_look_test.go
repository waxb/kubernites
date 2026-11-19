package world

import (
	"image"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func keepView(alloc, requested, used int64) view.KeepView {
	return view.KeepView{
		Allocatable: view.ResourcePair{CPU: alloc}, Requested: view.ResourcePair{CPU: requested}, Used: view.ResourcePair{CPU: used},
	}
}

func TestWheatFollowsTheKeepsFreeFoodAndRipensInTheMiddle(t *testing.T) {
	idle, busy, full := keepView(4000, 400, 200), keepView(4000, 3000, 2800), keepView(4000, 3900, 4000)
	if got := WheatLevel(FoodFree(idle), 1); got != art.WheatLevels-1 {
		t.Errorf("an idle Keep's middle wheat is level %d, want the ripest", got)
	}
	if WheatLevel(FoodFree(busy), 1) >= WheatLevel(FoodFree(idle), 1) {
		t.Error("a busier Keep grows poorer wheat")
	}
	if got := WheatLevel(FoodFree(full), 0); got != 0 {
		t.Errorf("a Keep with no free food has bare furrows, got level %d", got)
	}
	for _, free := range []float64{.2, .5, .8, 1} {
		if WheatLevel(free, 1) < WheatLevel(free, 0) {
			t.Errorf("free %.1f: the edge of a patch is riper than its middle", free)
		}
	}
	shades := map[int]bool{}
	for free := 0.0; free <= 1; free += .1 {
		for mid := 0.0; mid <= 1; mid += .25 {
			shades[WheatLevel(free, mid)] = true
		}
	}
	if len(shades) < art.WheatLevels {
		t.Errorf("wheat shows %d shades, want %d", len(shades), art.WheatLevels)
	}
}

func TestAHungryKeepsBushesAreBareAndRegrowWhenFoodReturns(t *testing.T) {
	starved := keepView(4000, 3800, 1000)
	if !Hungry(starved, 0, false) {
		t.Fatal("requests near the allocatable food leave the Keep hungry")
	}
	if !Hungry(keepView(4000, 1000, 500), 1, false) {
		t.Error("a throttled worker means hunger")
	}
	if !Hungry(keepView(4000, 2900, 500), 0, true) {
		t.Error("pods Pending for lack of food hunger a nearly full Keep")
	}
	if Hungry(keepView(4000, 800, 500), 0, true) {
		t.Error("a Keep with food to spare is not hungry because another Keep is full")
	}
	if Hungry(keepView(0, 0, 0), 3, true) {
		t.Error("a Keep without a reading is never called hungry")
	}
	if BerryFruit(BerryTarget(true, .9)) != 0 {
		t.Error("a hungry Keep's bushes are picked bare")
	}
	if BerryFruit(BerryTarget(false, .6)) != art.BushFruits-1 {
		t.Error("a fed Keep's bushes hang full")
	}
	level := BerryTarget(false, .6)
	for range 20 {
		level = EaseBerries(level, 0, 1)
	}
	if level != 0 {
		t.Errorf("the bushes are bare after a while, level %v", level)
	}
	if one := EaseBerries(level, 3, 1); one <= 0 || one >= 1 {
		t.Errorf("fruit returns slowly, got %v after one second", one)
	}
}

func TestIvyCreepsWithAKeepsQuietAgeAndSeasonsStartBare(t *testing.T) {
	if IvyLevel(0) != 0 || IvyLevel(IvyMossAfter-1) != 0 {
		t.Error("a new Keep is bare")
	}
	prev, steps := 0, map[int]bool{}
	for age := 0.0; age <= IvyFullAfter*1.5; age += 20 {
		level := IvyLevel(age)
		if level < prev {
			t.Fatalf("ivy recedes at age %v: %d after %d", age, level, prev)
		}
		prev = level
		steps[level] = true
	}
	if prev != IvySteps || len(steps) != IvySteps+1 {
		t.Errorf("ivy ends at %d with %d steps", prev, len(steps))
	}
}

func artSolid(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+3] = 255
	}
	return img
}

func opaqueMask(w, h int) gfx.Mask {
	img := artSolid(w, h)
	return gfx.MaskOf(img, art.Region{W: w, H: h})
}

func TestIvyOverlayClimbsFromTheWallBaseAndStaysOnTheBuilding(t *testing.T) {
	mask := opaqueMask(40, 60)
	count := func(level int) int {
		n, img := 0, IvyOverlay(mask, 20, level)
		for y := 0; y < 60; y++ {
			for x := 0; x < 40; x++ {
				if img.NRGBAAt(x, y).A > 0 {
					n++
					if !mask.At(x, y) {
						t.Fatalf("ivy at %d,%d is off the building", x, y)
					}
				}
			}
		}
		return n
	}
	if count(0) != 0 {
		t.Error("a freshly rebuilt Keep carries no ivy")
	}
	if !(count(1) > 0 && count(1) < count(3) && count(3) < count(IvySteps)) {
		t.Errorf("ivy spreads with age: %d %d %d", count(1), count(3), count(IvySteps))
	}
	img := IvyOverlay(mask, 20, IvySteps)
	for x := 0; x < 40; x++ {
		if img.NRGBAAt(x, 5).A > 0 {
			t.Fatalf("the roof line at column %d is overgrown", x)
		}
	}
}

func TestCairnsAndRubbleGrowOverWithTime(t *testing.T) {
	if CairnStage(0) != 0 || CairnStage(CairnStageSpan*10) != art.CairnStages-1 {
		t.Error("a cairn starts bare and ends overgrown")
	}
	if RubbleStage(0) != 0 || RubbleStage(RubbleStageSpan*10) != art.RubbleStages-1 {
		t.Error("rubble starts bare and ends grassed over")
	}
	for age := 0.0; age < 2000; age += 5 {
		if CairnStage(age+5) < CairnStage(age) || RubbleStage(age+5) < RubbleStage(age) {
			t.Fatalf("growth runs backwards at %v", age)
		}
	}
}

func TestOnlyHousingBoundCrewsTendGardens(t *testing.T) {
	if HasBinding(model.CrewChain{}) {
		t.Error("an unbound crew has no garden")
	}
	if !HasBinding(model.CrewChain{Resources: model.Resources{MemLimit: 128}}) {
		t.Error("a Housing Binding is a memory limit")
	}
	crews := []model.CrewChain{
		{Name: "bound", Resources: model.Resources{MemLimit: 64}, Foremen: []model.Foreman{{Pods: []string{"b1", "b2"}}}},
		{Name: "loose", Foremen: []model.Foreman{{Pods: []string{"l1"}}}},
	}
	got := boundPods(crews)
	if !got["b1"] || !got["b2"] || got["l1"] {
		t.Errorf("bound pods = %v", got)
	}
}

func TestLanternsOneLitPerReadyWorker(t *testing.T) {
	for _, c := range []struct{ desired, ready, total, lit int }{{3, 3, 3, 3}, {3, 1, 3, 1}, {4, 0, 4, 0}, {20, 20, maxLanterns, maxLanterns}, {2, 5, 5, 5}} {
		if total, lit := LanternsLit(c.desired, c.ready); total != c.total || lit != c.lit {
			t.Errorf("desired %d ready %d: %d lit of %d, want %d of %d", c.desired, c.ready, lit, total, c.lit, c.total)
		}
	}
}

func TestWashingHangsOnlyWhenACrewStandsAtItsFullCount(t *testing.T) {
	stand := model.CrewChain{Name: "porters", Desired: 4, Ready: 4}
	if !LaundryHangs(stand) {
		t.Error("a crew at its full count airs its washing")
	}
	if LaundryHangs(model.CrewChain{Desired: 4, Ready: 3}) {
		t.Error("a short crew hangs no washing")
	}
	if LaundryHangs(model.CrewChain{Desired: 1, Ready: 1}) {
		t.Error("one hut has nobody to share a line with")
	}
	rolling := stand
	rolling.Foremen = []model.Foreman{{Desired: 2, Newest: true}, {Desired: 2}}
	if LaundryHangs(rolling) {
		t.Error("washing comes in during a rollout")
	}
}

func TestScaffoldingStandsOnlyOverHutsStillToBeRebuilt(t *testing.T) {
	crew := model.CrewChain{Name: "porters", Foremen: []model.Foreman{
		{Name: "old", Desired: 2, Current: 2, Pods: []string{"p1", "p2"}},
		{Name: "new", Desired: 2, Current: 2, Newest: true, Pods: []string{"p3", "p4"}},
	}}
	got := ScaffoldedPods([]model.CrewChain{crew})
	if !got["p1"] || !got["p2"] || got["p3"] || got["p4"] {
		t.Errorf("scaffolds = %v", got)
	}
	crew.Foremen[0].Desired, crew.Foremen[0].Current = 0, 0
	if got := ScaffoldedPods([]model.CrewChain{crew}); len(got) != 0 {
		t.Errorf("the rollout is done, yet scaffolds stand: %v", got)
	}
}

func TestLivestockNeedsFiveHutsAndBoatsNeedTheFirstTier(t *testing.T) {
	if HoldsLivestock(LivestockHuts-1) || !HoldsLivestock(LivestockHuts) {
		t.Error("five huts make a farmyard")
	}
	if BoatsLaunch(view.Haven{Open: true, Tier: 0}) || !BoatsLaunch(view.Haven{Open: true, Tier: 1}) || BoatsLaunch(view.Haven{Tier: 2}) {
		t.Error("a boat waits for an open harbor at its first tier")
	}
}

func TestTheBoardGainsAParchmentPerCrewOrDepot(t *testing.T) {
	if BoardParchments(0, 0) != 0 || BoardParchments(2, 3) != 5 {
		t.Error("a parchment for each declared crew and depot")
	}
}

func TestWornGrassTurnsToDirtOnlyAlongBusyRoutes(t *testing.T) {
	if WornLevel(0) != 0 || WornLevel(wornFloor) != 0 {
		t.Error("a single step leaves no mark")
	}
	if WornLevel(wornFull) != 1 || WornLevel(wornFull*3) != 1 {
		t.Error("a busy route is fully worn")
	}
	if !(WornLevel(wornFloor+1) < WornLevel(wornFull-1)) {
		t.Error("wear deepens with footfall")
	}
}

func TestCrewHuesAreStablePerCrew(t *testing.T) {
	if CrewHue("porters") != CrewHue("porters") {
		t.Fatal("a crew keeps its colour")
	}
	seen := map[[3]uint8]bool{}
	for _, n := range []string{"porters", "clerks", "haulers", "scribes", "smiths", "cooks", "wardens", "bakers"} {
		c := CrewHue(n)
		seen[[3]uint8{c.R, c.G, c.B}] = true
	}
	if len(seen) < 4 {
		t.Errorf("eight crews share only %d colours", len(seen))
	}
}
