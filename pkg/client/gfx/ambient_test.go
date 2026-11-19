package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestAmbientNamesResolveInTheLibrary(t *testing.T) {
	lib := Load(art.Build())
	names := []string{
		WispName(0), WispName(11), CartName(-1, "se", 5), CartName(9, "sw", -1),
		AmbientName("lamp", 0), AmbientName("lamp", 1), AmbientName("ward", -3), AmbientName("idle", 0),
		AmbientName("bundle", 0), AmbientName("scroll", 0), AmbientName("crate", 0), AmbientName("chimney", 0),
	}
	for i := 0; i < ScrollPiles; i++ {
		names = append(names, AmbientName("scrolls", i))
	}
	for i := 0; i < StallLooks; i++ {
		names = append(names, AmbientName("stall", i))
	}
	for _, role := range FolkRoles {
		for look := -1; look <= FolkLooks; look++ {
			for _, action := range []string{"idle", "walk", "work", "carry"} {
				names = append(names, FolkName(role, look, action, "se", look+7), FolkName(role, look, action, "sw", -look))
			}
		}
	}
	if missing := Missing(lib, names); len(missing) > 0 {
		t.Fatalf("missing ambient sprites: %v", missing)
	}
}

func TestFolkNamesWrapLooksAndFrames(t *testing.T) {
	if FolkName("farmer", 0, "walk", "se", 4) != FolkName("farmer", FolkLooks, "walk", "se", 0) {
		t.Error("looks and frames wrap")
	}
	if FolkName("child", 0, "idle", "sw", 1) != "walker/child/idle/sw/1" {
		t.Errorf("plain look = %s", FolkName("child", 0, "idle", "sw", 1))
	}
	if CartName(1, "se", 2) != "ambient/cart/1/se/2" || AmbientName("lamp", 1) != "ambient/lamp/1" {
		t.Error("ambient name patterns")
	}
}
