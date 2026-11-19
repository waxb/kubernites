package gfx

import (
	"reflect"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestHearthQoSNamesFallBackToTheOldHuts(t *testing.T) {
	cases := []struct {
		tier         int
		state, qos   string
		name         string
		chain        []string
		unknownClass bool
	}{
		{2, "ready", "Guaranteed", "hearth/2/ready/landed", []string{"hearth/2/ready/landed", "hearth/2/ready"}, false},
		{7, "guttering", "BestEffort", "hearth/3/guttering/vagrant",
			[]string{"hearth/3/guttering/vagrant", "hearth/3/guttering", "hearth/3/ready/vagrant", "hearth/3/ready"}, false},
		{0, "pending", "freeman", "hearth/1/pending/freeman",
			[]string{"hearth/1/pending/freeman", "hearth/1/pending", "hearth/1/ready/freeman", "hearth/1/ready"}, false},
		{2, "ghost", "", "hearth/2/ghost", []string{"hearth/2/ghost", "hearth/2/ready"}, true},
		{1, "ready", "Mystery", "hearth/1/ready", []string{"hearth/1/ready"}, true},
	}
	for _, c := range cases {
		if got := HearthQoSName(c.tier, c.state, c.qos); got != c.name {
			t.Errorf("HearthQoSName(%d,%s,%s) = %s, want %s", c.tier, c.state, c.qos, got, c.name)
		}
		if got := HearthQoSNames(c.tier, c.state, c.qos); !reflect.DeepEqual(got, c.chain) {
			t.Errorf("HearthQoSNames(%d,%s,%s) = %v, want %v", c.tier, c.state, c.qos, got, c.chain)
		}
		if (QoSArt(c.qos) == "") != c.unknownClass {
			t.Errorf("QoSArt(%s) = %q", c.qos, QoSArt(c.qos))
		}
	}
	if QoSArt("Burstable") != "freeman" {
		t.Fatal("burstable is the freeman cottage")
	}
}

func TestCaesarNameHelpersClampAndWrap(t *testing.T) {
	checks := [][2]string{
		{MiddenName(9, 3), "midden/3/1"},
		{MiddenName(-1, -1), "midden/0/1"},
		{CodexBloatName(8), "codexbloat/3"},
		{CodexBloatName(-2), "codexbloat/0"},
		{AlarmName(7), "fx/alarm/1"},
		{CouncilPortrait("x"), "advisor-chief"},
		{CouncilPortrait("stewards"), "steward"},
		{RankIcon("high"), "rank-high"},
		{RankIcon("low"), "rank-low"},
		{RankIcon(""), "rank-normal"},
		{OverlayIcon("codex"), "codex"},
		{RatingIcon("favor"), "rating-favor"},
	}
	for _, c := range checks {
		if c[0] != c[1] {
			t.Errorf("got %s, want %s", c[0], c[1])
		}
	}
	if OverlayIcon("power") != "" || RatingIcon("x") != "" {
		t.Fatal("unknown modes have no icon")
	}
}

func TestCaesarLevelsFollowTheThresholds(t *testing.T) {
	for _, c := range []struct {
		used float64
		flag bool
		m, b int
	}{
		{0, false, 0, 0}, {0.5, false, 1, 1}, {0.72, false, 1, 2}, {0.8, false, 2, 2}, {0.95, false, 3, 3}, {0.1, true, 3, 3},
	} {
		if got := MiddenLevel(c.used, c.flag); got != c.m {
			t.Errorf("MiddenLevel(%v,%v) = %d, want %d", c.used, c.flag, got, c.m)
		}
		if got := CodexBloatLevel(c.used, c.flag); got != c.b {
			t.Errorf("CodexBloatLevel(%v,%v) = %d, want %d", c.used, c.flag, got, c.b)
		}
	}
}

func TestCaesarNamesResolveInTheLibrary(t *testing.T) {
	lib := Load(art.Build())
	names := []string{}
	for tier := 1; tier <= 3; tier++ {
		for _, st := range art.HearthStates {
			for _, q := range []string{"BestEffort", "Burstable", "Guaranteed"} {
				names = append(names, HearthQoSNames(tier, st, q)...)
			}
		}
	}
	for l := 0; l < MiddenLevels; l++ {
		for v := 0; v < MiddenVariants; v++ {
			names = append(names, MiddenName(l, v))
		}
	}
	for l := 0; l < CodexBloatLevels; l++ {
		names = append(names, CodexBloatName(l))
	}
	for f := 0; f < 6; f++ {
		names = append(names, AlarmName(f))
	}
	for _, s := range CouncilSeats {
		names = append(names, "portrait/"+CouncilPortrait(s))
	}
	for _, i := range []string{OverlayIcon("midden"), OverlayIcon("codex"), OverlayIcon("labor"), OverlayIcon("unhoused"),
		RatingIcon("reliability"), RatingIcon("efficiency"), RatingIcon("resilience"), RatingIcon("favor"),
		RankIcon("high"), RankIcon("normal"), RankIcon("low"), "decree", "council"} {
		names = append(names, "icon/"+i)
	}
	if missing := Missing(lib, names); len(missing) > 0 {
		t.Fatalf("%d caesar sprites missing, first %s", len(missing), missing[0])
	}
	if MiddenLevels != art.MiddenLevels || MiddenVariants != art.MiddenVariants || CodexBloatLevels != art.CodexBloatLevels {
		t.Fatal("gfx and art disagree on the midden or codex levels")
	}
}
