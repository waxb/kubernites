package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func TestStorageTargetsPointAtTheirVaultsAndHuts(t *testing.T) {
	site := view.Site{Structures: []view.Structure{{ID: "vault-1", Kind: view.StructVault, X: 20, Y: 20, W: 2, H: 2, Object: view.ObjectRef{Kind: "PersistentVolume", Name: "vault-1"}}}}
	st := view.GameState{}
	st.Storage.Deeds = []view.DeedView{{Name: "books-archive-0", Phase: "Bound", Volume: "vault-1"}, {Name: "books-archive-1", Phase: "Pending"}}
	st.StatefulSets = []view.StatefulSetView{{Name: "archive", Pods: []view.StatefulPodView{{Name: "archive-0", Claim: "books-archive-0"}, {Name: "archive-1", Claim: "books-archive-1"}}}}
	home := func(name string) (world.Vec, bool) {
		return map[string]world.Vec{"archive-0": {X: 7, Y: 8}, "archive-1": {X: 9, Y: 4}}[name], name == "archive-0" || name == "archive-1"
	}
	cases := map[string]world.Vec{
		"pv:vault-1": {X: 21.5, Y: 21.5}, "deed:books-archive-0": {X: 21.5, Y: 21.5}, "deed:books-archive-1": {X: 9.5, Y: 4.5}, "sts:archive": {X: 7.5, Y: 8.5},
	}
	for target, want := range cases {
		if !hud.MapTarget(target) {
			t.Errorf("%s is not a map target", target)
		}
		if got, ok := StorageTarget(target, st, site, home); !ok || got != want {
			t.Errorf("%s = %v %v, want %v", target, got, ok, want)
		}
	}
	for _, target := range []string{"pv:none", "deed:none", "sts:none", "pod:archive-0"} {
		if _, ok := StorageTarget(target, st, site, home); ok {
			t.Errorf("%s resolved", target)
		}
	}
}

func TestTheSoulvaultTargetIsNotAVault(t *testing.T) {
	if hud.MapTarget("vault:porter:1.1") {
		t.Fatal("vault: is the Soulvault's target, not a place on the map")
	}
}
