package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

var previewLib = gfx.Load(art.Build())

func castleFrame(t *testing.T) Frame {
	t.Helper()
	f := keepsFrame(t)
	f.State.Age = view.AgeCastle
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, Class: "compute"}, {KeepType: "great-keep", Class: "memory"}}
	return f
}

func buildIcon(t *testing.T, h *HUD, f Frame, id string) string {
	t.Helper()
	for _, b := range h.BuildButtons(f) {
		if b.ID == id {
			return b.Icon
		}
	}
	t.Fatalf("no %s button", id)
	return ""
}

func TestBuildButtonsShowTheBuildingOfTheAge(t *testing.T) {
	h := New(previewLib, Backend{})
	f := castleFrame(t)
	if got := buildIcon(t, h, f, "build:archive"); got != "buildicon/stall-archive/2" {
		t.Fatalf("the archive button shows %s at the third tier", got)
	}
	if got := buildIcon(t, h, keepsFrame(t), "build:archive"); got != "buildicon/stall-archive" {
		t.Fatalf("the archive button shows %s at the first tier", got)
	}
	stone := buildIcon(t, h, f, "build:keep")
	if stone != gfx.KeepPreview("stone", "compute", 2).BuildIconName() || !strings.HasPrefix(stone, "buildicon/keep-stone-compute/2/") {
		t.Fatalf("the Stone Keep button previews %s", stone)
	}
	if great := buildIcon(t, h, f, "build:great-keep"); !strings.HasPrefix(great, "buildicon/keep-great-memory/2/") {
		t.Fatalf("the Great Keep button previews %s", great)
	}
	for _, name := range []string{stone, "buildicon/stall-archive/2"} {
		if !previewLib.Has(name) {
			t.Errorf("%s does not resolve", name)
		}
	}
}

func TestBuildPageDrawsTheAgedPreviews(t *testing.T) {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h := New(previewLib, Backend{})
	f := castleFrame(t)
	h.PageID = PageBuild
	h.Draw(dst, f)
	if !previewLib.Has(gfx.KeepPreview("great", "memory", 2).BuildIconName()) {
		t.Fatal("drawing the build page painted the Great Keep preview")
	}
}

func TestKeepCardShowsItsOwnKeep(t *testing.T) {
	f := castleFrame(t)
	keep := view.Structure{ID: "keep-9", Kind: view.StructKeep, Variant: "great", Class: "rich", State: view.StateReady, Label: "Great Keep 2",
		Object: view.ObjectRef{Kind: "Node", Name: "great-keep-2"}}
	c := StructureCard(keep, f)
	if want := gfx.KeepLookOf(keep, 2).PortraitName(); c.Portrait != want {
		t.Fatalf("keep card portrait %s, want %s", c.Portrait, want)
	}
}
