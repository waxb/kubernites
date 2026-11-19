package model

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestCauseTitlesNameEveryCalamity(t *testing.T) {
	cases := map[string]string{
		view.TagOOMKilled: "BOX TOO BIG", view.TagThrottled: "HUNGRY", view.TagEvicted: "BANISHED",
		view.TagCrashLoop: "GUTTERING", view.TagImagePull: "NEVER ARRIVED", view.TagUnhoused: "UNHOUSED",
		view.CalamityNoDepot: "NO DEPOT", view.CalamityNoCrew: "NO CREW", view.CalamitySpoiling: "SPOILING",
		"cold-feet": "COLD FEET",
	}
	for kind, want := range cases {
		if got := CauseTitle(kind); got != want {
			t.Errorf("CauseTitle(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestKubernetesModeTitlesACrashLoopByItsRealName(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(func(s string) string { return s })
	for kind, want := range map[string]string{view.TagCrashLoop: "CRASHLOOP", view.TagEvicted: "EVICTED", view.TagThrottled: "THROTTLED", view.TagImagePull: "IMAGE PULL"} {
		if got := CauseTitle(kind); got != want {
			t.Errorf("CauseTitle(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestClassColorsTellTheLandApart(t *testing.T) {
	seen := map[[4]uint8]string{}
	for _, class := range []string{"balanced", "compute", "memory", "rich"} {
		c := ClassColor(class)
		key := [4]uint8{c.R, c.G, c.B, c.A}
		if other, dup := seen[key]; dup {
			t.Errorf("%s and %s share a colour", class, other)
		}
		seen[key] = class
	}
	if ClassColor("odd") != ClassColor("") {
		t.Error("unknown classes fall back to text")
	}
}
