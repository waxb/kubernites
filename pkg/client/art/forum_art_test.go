package art

import (
	"image"
	"strings"
	"testing"
)

func forumNames() []string {
	out := []string{}
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < ForumPaveVariants; v++ {
			out = append(out, ForumPaveName(age, v))
		}
	}
	for age := ageCastle; age < AgeTiers; age++ {
		for _, side := range ForumSides {
			out = append(out, ForumRimName(age, side))
		}
		out = append(out, ForumBandName(age, "x"), ForumBandName(age, "y"), ForumRoseName(age, 0, 0), ForumRoseName(age, -1, 1), ForumFountainName(age))
	}
	for age := ageFeudal; age < AgeTiers; age++ {
		out = append(out, ForumKerbName("x", age, 0), ForumKerbName("y", age, 1), ForumPostName(age, 0), ForumPostName(age, 1))
	}
	for _, axis := range []string{"x", "y"} {
		out = append(out, ForumColonnadeName(axis, 0), ForumColonnadeName(axis, 1), ForumRostraName(axis), ForumArchName(axis))
	}
	for v := 0; v < ForumStatues; v++ {
		out = append(out, ForumStatueName(v), ForumTerminusName(v))
	}
	out = append(out, ForumPlinthName(ageCastle))
	for v := 0; v < MarketGrowths; v++ {
		out = append(out, MarketGrowthName(v))
	}
	return out
}

func opaqueIn(img *image.NRGBA, r Region) int {
	n := 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if img.NRGBAAt(x, y).A > 0 {
				n++
			}
		}
	}
	return n
}

func forumEdgeTouch(img *image.NRGBA, r Region) bool {
	for i := 0; i < r.W; i++ {
		if img.NRGBAAt(r.X+i, r.Y).A > 0 || img.NRGBAAt(r.X+i, r.Y+r.H-1).A > 0 {
			return true
		}
	}
	for i := 0; i < r.H; i++ {
		if img.NRGBAAt(r.X, r.Y+i).A > 0 || img.NRGBAAt(r.X+r.W-1, r.Y+i).A > 0 {
			return true
		}
	}
	return false
}

func TestEveryForumSpriteIsDrawnInsideItsCanvas(t *testing.T) {
	a := buildBuildings()
	for _, name := range forumNames() {
		r, ok := a.Regions[name]
		if !ok {
			t.Fatalf("no sprite %q", name)
		}
		if opaqueIn(a.Image, r) == 0 {
			t.Fatalf("%q is blank", name)
		}
		if forumEdgeTouch(a.Image, r) {
			t.Errorf("%q touches its canvas edge", name)
		}
	}
}

func TestTheForumPavementClosesUpWithTheAges(t *testing.T) {
	a := buildBuildings()
	cover := [AgeTiers]int{}
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < ForumPaveVariants; v++ {
			cover[age] += opaqueIn(a.Image, a.Regions[ForumPaveName(age, v)])
		}
	}
	if cover[0]*4 > cover[1]*3 || cover[1] > cover[2]+ForumPaveVariants*4 || cover[2] > cover[3]+ForumPaveVariants*4 {
		t.Fatalf("pavement coverage by age %v: Dev must leave grass between its stones, the later Ages close it", cover)
	}
	if strings.Count(ForumRoseName(9, 1, -1), "/") != 4 || ForumPaveName(-3, 0) != "forum/pave/0/0" {
		t.Fatal("Forum names do not clamp the Age")
	}
}

func TestDevStonesAndWildflowersVaryBySeed(t *testing.T) {
	seen := map[[2]int]bool{}
	for v := 0; v < ForumPaveVariants; v++ {
		stones, flowers := 0, 0
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				col := devPave(float64(x)/32, float64(y)/32, x, y, v)
				switch {
				case col == plazaR[1] || col == plazaR[2] || col == plazaR[3] || col == plazaR[0]:
					stones++
				case col.A > 0 && col != grassR[0] && col != grassR[3] && col != mossR[2]:
					flowers++
				}
			}
		}
		if stones == 0 || stones > 32*32*3/4 {
			t.Fatalf("variant %d lays %d stone pixels", v, stones)
		}
		seen[[2]int{stones, flowers}] = true
	}
	if len(seen) < ForumPaveVariants/2 {
		t.Fatalf("Dev variants barely differ: %v", seen)
	}
}
