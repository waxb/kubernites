package gfx

import (
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestPreviewNamesShowTheAgeWithTheStablePreviewLook(t *testing.T) {
	codex := view.Structure{ID: "codex-4", Kind: view.StructCodex, State: view.StateReady}
	if got := PreviewNames(codex, 2); got[0] != "building/codex/ready/2/0" || got[len(got)-1] != "building/codex/ready" {
		t.Fatalf("codex preview %v", got)
	}
	depot := view.Structure{Kind: view.DepotStructure("crate"), W: 1, H: 2, Variant: view.FaceNW, State: view.StateReady}
	if got := PreviewNames(depot, 3); got[0] != "building/stall-crate-nw/ready/3/0" || got[len(got)-1] != "building/stall-crate-nw/ready" {
		t.Fatalf("depot preview %v", got)
	}
	ghost := view.Structure{Kind: view.StructGuildhall, W: 1, H: 2, State: view.StateReady}
	if got := PreviewNames(ghost, 1); got[0] != "building/stall-guildhall-se/ready/1/0" {
		t.Fatalf("a Guildhall with no face previews %v", got)
	}
	keep := view.Structure{Kind: view.StructKeep, Variant: "great", Class: "compute", State: view.StateReady, Object: view.ObjectRef{Name: KeepPreviewNode("great")}}
	got := PreviewNames(keep, 1)
	if got[0] != KeepPreview("great", "compute", 1).Name("ready") || got[1] != "building/keep-great/ready/1/0" || got[len(got)-1] != "building/keep-great/ready" {
		t.Fatalf("keep preview %v", got)
	}
}

func TestBuildIconsFollowTheAge(t *testing.T) {
	cases := map[string]string{"codex": "buildicon/codex/2", "guildhall": "buildicon/stall-guildhall/2", "archive": "buildicon/stall-archive/2", "depot-sealed": "buildicon/stall-seal/2", "mystery": "portrait/mystery"}
	for kind, want := range cases {
		if got := BuildIcon(kind, 2); got != want {
			t.Errorf("%s: %s, want %s", kind, got, want)
		}
	}
	lib := Load(art.Set{})
	icon := KeepPreview("citadel", "rich", 1).BuildIconName()
	if _, r, ok := lib.Lookup(icon); !ok || r.W != art.BuildIconW || r.H != art.BuildIconH {
		t.Fatalf("keep icon %s: %+v %v", icon, r, ok)
	}
	if !strings.HasPrefix(icon, BuildIconPrefix) {
		t.Fatalf("icon %s", icon)
	}
}

func TestBuildPortraitFollowsTheAge(t *testing.T) {
	cases := map[string]string{"codex": "portrait/codex/2", "depot-parcel": "portrait/stall-parcel/2", "herald": "portrait/stall-herald/2", "mystery": "portrait/mystery"}
	for kind, want := range cases {
		if got := BuildPortrait(kind, 2); got != want {
			t.Errorf("%s: %s, want %s", kind, got, want)
		}
	}
	if got := BuildPortrait("codex", 0); got != "portrait/codex" {
		t.Errorf("the Dev portrait keeps its plain name, got %s", got)
	}
}

func TestKeepPortraitsArePaintedFromTheKeepItself(t *testing.T) {
	keep := keepFixture(view.StateReady)
	name := StructurePortrait(keep, 2)
	if name != KeepLookOf(keep, 2).PortraitName() || !strings.HasPrefix(name, "portrait/keep-great-memory/2/") {
		t.Fatalf("keep card portrait %s", name)
	}
	lib := Load(art.Set{})
	if _, r, ok := lib.Lookup(name); !ok || r.W != art.PortraitSize || r.H != art.PortraitSize {
		t.Fatalf("the keep portrait was not painted: %+v %v", r, ok)
	}
	if lib.Has("portrait/keep-great-memory/9/zz") || lib.Has("portrait/keep-moat-memory/1/00000001") {
		t.Fatal("a malformed keep portrait resolved")
	}
	preview := KeepPreview("tower", "", 3)
	if preview.Size != "tower" || preview.Class != "balanced" || preview.Seed != art.NameSeed(testTowerNode) || preview.Age != 3 {
		t.Fatalf("the first Keep previews the Lone Tower itself: %+v", preview)
	}
}

func TestKeepPortraitsFallBackWhileTheyPaint(t *testing.T) {
	if alt, ok := KeepFallback("portrait/keep-citadel-rich/2/0000beef"); !ok || alt != "portrait/keep-citadel/2" {
		t.Fatalf("fallback %q %v", alt, ok)
	}
	if alt, ok := KeepFallback("portrait/keep-stone-rich/0/0000beef"); !ok || alt != "portrait/keep-stone" {
		t.Fatalf("Dev fallback %q %v", alt, ok)
	}
	if alt, ok := KeepFallback(KeepPreview("great", "memory", 3).BuildIconName()); !ok || alt != "buildicon/keep-great/3" {
		t.Fatalf("icon fallback %q %v", alt, ok)
	}
	if _, ok := KeepFallback("portrait/codex/2"); ok {
		t.Fatal("only keep portraits and icons fall back")
	}
	lib := &Library{index: map[string]entry{"portrait/keep-stone/2": {sheet: ebiten.NewImage(4, 4), region: art.Region{W: 4, H: 4}}}, lazy: newLazySprites()}
	lib.SetAsync(true)
	dst := ebiten.NewImage(32, 32)
	name := KeepPreview("stone", "memory", 2).PortraitName()
	if !lib.Portrait(dst, strings.TrimPrefix(name, "portrait/"), ui.Rect{W: 24, H: 24}) {
		t.Fatal("a keep portrait still painting shows the canonical keep of its Age")
	}
	if lib.Portrait(dst, "codex/2", ui.Rect{W: 24, H: 24}) {
		t.Fatal("a missing portrait draws nothing")
	}
	deadline := time.Now().Add(20 * time.Second)
	for !lib.Has(name) {
		if time.Now().After(deadline) {
			t.Fatal("the keep portrait was never painted")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
