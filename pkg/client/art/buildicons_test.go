package art

import "testing"

func regionPixels(a Atlas, r Region) int {
	n := 0
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 {
				n++
			}
		}
	}
	return n
}

func TestEveryBuildingHasAnIconForEveryAge(t *testing.T) {
	set := built()
	for _, kind := range BuildingKinds {
		for age := 0; age < AgeTiers; age++ {
			r, ok := set.Portraits.Regions[BuildIconName(kind, age)]
			if !ok || r.W != BuildIconW || r.H != BuildIconH || r.AnchorY != BuildIconH-buildIconFoot {
				t.Fatalf("%s age %d: icon %+v %v", kind, age, r, ok)
			}
			if regionPixels(set.Portraits, r) < BuildIconW*BuildIconH/5 {
				t.Errorf("%s age %d: the icon is nearly empty", kind, age)
			}
		}
	}
	if BuildIconName("codex", 0) != "buildicon/codex" || BuildIconName("codex", 7) != "buildicon/codex/3" {
		t.Fatal("icon names follow the aged portrait names")
	}
}

func TestKeepIconsRoundTripAndPaint(t *testing.T) {
	k := KeepLookOf("stone", "memory", "stone-keep-4", 1)
	back, ok := ParseKeepIconName(k.BuildIconName())
	if !ok || back != k {
		t.Fatalf("%s parsed to %+v %v", k.BuildIconName(), back, ok)
	}
	for _, bad := range []string{"buildicon/keep-stone/1", "buildicon/codex/1", k.PortraitName(), "buildicon/keep-stone-memory/1"} {
		if _, ok := ParseKeepIconName(bad); ok {
			t.Errorf("%s parsed", bad)
		}
	}
	img, r := KeepBuildIcon(k)
	if r.W != BuildIconW || img.Rect.Dy() != BuildIconH || keepPixels(img) < BuildIconW*BuildIconH/4 {
		t.Fatalf("keep icon %+v with %d pixels", r, keepPixels(img))
	}
	if edges := touchedEdges(img, Region{W: r.W, H: r.H}); edges != "" {
		t.Fatalf("the keep icon touches %s", edges)
	}
}

func TestShotMemoPaintsEachBuildingOnce(t *testing.T) {
	m := newShotMemo()
	a, b := m.get("codex", 2), m.get("codex", 2)
	if a != b || a == m.get("codex", 1) {
		t.Fatal("the memo shares one painting per kind and age")
	}
}
