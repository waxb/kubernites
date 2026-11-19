package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestGhostKeepPreviewsItsSizeAndClass(t *testing.T) {
	built := grassSite(true)
	s := GhostStructure(built, Ghost{Kind: "great-keep", W: 3, H: 3, Class: "compute"})
	if s.Class != "compute" || s.Object.Name != gfx.KeepPreviewNode("great") {
		t.Fatalf("keep ghost %+v", s)
	}
	names := gfx.PreviewNames(s, 2)
	if names[0] != gfx.KeepPreview("great", "compute", 2).Name("ready") {
		t.Fatalf("the ghost draws %v", names)
	}
	tower := GhostStructure(grassSite(false), Ghost{Kind: view.StructKeep, W: 3, H: 3})
	if tower.Object.Name != art.KeepPreviewNode("tower") || gfx.PreviewNames(tower, 0)[0] != gfx.KeepPreview("tower", "", 0).Name("ready") {
		t.Fatalf("tower ghost %+v", tower)
	}
	if codex := GhostStructure(built, Ghost{Kind: "codex", Class: "memory"}); codex.Class != "" || gfx.PreviewNames(codex, 3)[0] != "building/codex/ready/3/0" {
		t.Fatalf("codex ghost %+v", codex)
	}
}

func TestGhostDrawsTheAgedLook(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	lib := gfx.Load(art.Build())
	r := &Renderer{Lib: lib}
	site := grassSite(true)
	cam := render.NewCamera(480, 60).CenterOn(4, 4, 960, 540)
	for _, g := range []Ghost{{Kind: "codex", X: 5, Y: 5, W: 2, H: 2, Valid: true}, {Kind: "great-keep", X: 4, Y: 4, W: 3, H: 3, Class: "memory"}} {
		r.drawGhost(dst, cam, Scene{Site: site, Age: 2}, g)
	}
	if !lib.Has(gfx.KeepPreview("great", "memory", 2).Name("ready")) {
		t.Fatal("the keep ghost was painted from its procedural look")
	}
	if len(lib.Flags(gfx.KeepPreview("great", "memory", 2).Name("ready"))) == 0 {
		t.Fatal("the keep ghost has no flags to wave")
	}
}
