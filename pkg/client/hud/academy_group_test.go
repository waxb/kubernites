package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

func mixedAcademy() []view.SoulstoneEntry {
	return []view.SoulstoneEntry{
		{Image: "porter:1.0", Cost: 0, Owned: true},
		{Image: "porter:1.1", Cost: 150, Owned: true},
		{Image: "clerk:1.0", Cost: 200, Visible: true},
		{Image: "raven:1.0", Cost: 300, Visible: true},
		{Image: "porter:2.0", Cost: 3000, Visible: true},
		{Image: "clerk:1.1", Cost: 5000, Visible: true},
		{Image: "porter:3.0", Cost: 9000, Visible: true},
		{Image: "clerk:2.0", Cost: 27000, Visible: true},
		{Image: "porter:latest", Cost: 27000, Visible: true},
		{Image: "hauler:1.0", Cost: 30000},
	}
}

func TestTheAcademyGroupsEachProfessionInItsProgression(t *testing.T) {
	f := Frame{}
	f.State.Soulstones = mixedAcademy()
	got := []string{}
	for _, s := range academyRows(f) {
		got = append(got, s.Image)
	}
	want := []string{"porter:1.0", "porter:1.1", "porter:2.0", "porter:3.0", "porter:latest", "clerk:1.0", "clerk:1.1", "clerk:2.0", "raven:1.0"}
	if !slices.Equal(got, want) {
		t.Fatalf("academy rows = %v, want %v", got, want)
	}
	seen := map[string]bool{}
	last := ""
	for _, img := range got {
		p, _ := soulstone.ProfessionOf(img)
		if p != last && seen[p] {
			t.Fatalf("profession %s is split in %v", p, got)
		}
		seen[p], last = true, p
	}
}

func TestTheAcademyScrollsToTheClassTheGuideNames(t *testing.T) {
	f := Frame{}
	for i := range academyVisible() + 2 {
		f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "porter:" + string(rune('a'+i)), Cost: float64(i), Visible: true})
	}
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "clerk:2.0", Cost: 1, Visible: true})
	f.State.Gold = 1e12
	f.Guide.Button = "vault:clerk:2.0"
	h := New(nil, Backend{})
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:2.0"); !ok {
		t.Fatalf("the guided class is not on screen in the registry")
	}
}
