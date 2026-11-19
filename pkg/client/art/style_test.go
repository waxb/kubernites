package art

import (
	"image/color"
	"math"
	"testing"
)

func TestLookIsDeterministicAndVariantsShiftTheHue(t *testing.T) {
	a, b := newLook("codex", 2, 1), newLook("codex", 2, 1)
	if a != b {
		t.Fatal("the same kind, age and variant must give the same look")
	}
	base := newLook("codex", 2, 0)
	if base.id != tealR {
		t.Fatalf("variant 0 keeps the identity ramp, got %v", base.id)
	}
	for v := 1; v < BuildingVariants; v++ {
		l := newLook("codex", 2, v)
		if l.id == tealR || l.seed == base.seed {
			t.Errorf("variant %d must shift the roof hue and the seed", v)
		}
		for i := range l.id {
			if d := colorDist(l.id[i], tealR[i]); d > 60 {
				t.Errorf("variant %d strays %.0f from the identity hue", v, d)
			}
		}
	}
	if newLook("codex", 9, 0).age != AgeTiers-1 || newLook("codex", -3, 0).age != 0 {
		t.Fatal("ages clamp to the four tiers")
	}
}

func TestLookRollsAreStable(t *testing.T) {
	l := newLook("herald", 1, 2)
	for k := 0; k < 50; k++ {
		if r := l.roll(k); r < 0 || r >= 1 || r != l.roll(k) {
			t.Fatalf("roll %d = %v", k, r)
		}
		if p := l.pick(k, 5); p < 0 || p >= 5 {
			t.Fatalf("pick %d = %d", k, p)
		}
		if j := l.jit(k, 0.2); j < -0.2 || j > 0.2 {
			t.Fatalf("jit %d = %v", k, j)
		}
	}
	if !l.at(1) || l.at(2) {
		t.Fatal("at compares the age")
	}
	if l.pick(3, 0) != 0 {
		t.Fatal("pick of nothing is zero")
	}
}

func TestLookNames(t *testing.T) {
	cases := []struct{ got, want string }{
		{BuildingLookName("codex", "ready", 0, 0), "building/codex/ready"},
		{BuildingLookName("codex", "dark", 2, 0), "building/codex/dark/2/0"},
		{BuildingLookName(StallArtKind("stall-crate", "se"), "ready", 0, 2), "building/stall-crate-se/ready/0/2"},
		{HearthLookName(2, "ready", "landed", 0, 0), "hearth/2/ready/landed"},
		{HearthLookName(3, "ghost", "vagrant", 1, 2), "hearth/3/ghost/vagrant/1/2"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %s, want %s", c.got, c.want)
		}
	}
	if LookVariant("codex-4") != LookVariant("codex-4") || LookVariant("x") >= BuildingVariants {
		t.Fatal("LookVariant must be stable and bounded")
	}
}

func TestEveryKindHasAHueAndKin(t *testing.T) {
	for _, k := range BuildingKinds {
		r := kindRamp(k)
		kin := hueKin(r)
		if kin[0] == r && kin[1] == greyR {
			t.Errorf("%s has no hue kin", k)
		}
	}
	if kindRamp("nonsense") != redR || hueKin(greyR)[1] != greyR {
		t.Fatal("unknown kinds fall back")
	}
}

func colorDist(a, b color.NRGBA) float64 {
	dr, dg, db := float64(a.R)-float64(b.R), float64(a.G)-float64(b.G), float64(a.B)-float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}
