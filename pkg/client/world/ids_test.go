package world

import (
	"hash/fnv"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func structureSigByHasher(site view.Site) uint64 {
	h := fnv.New64a()
	for _, s := range site.Structures {
		h.Write([]byte(s.Kind))
		h.Write([]byte{byte(s.X), byte(s.X >> 8), byte(s.Y), byte(s.Y >> 8), byte(s.W), byte(s.H), byte(s.Plot), byte(s.Territory)})
	}
	for _, hv := range site.Havens {
		h.Write([]byte(hv.ID))
		h.Write([]byte{byte(hv.Quay.X), byte(hv.Quay.Y), byte(hv.Tier), byte(hv.Pier), boolByte(hv.Open)})
	}
	for _, is := range site.Islands {
		h.Write([]byte(is.ID))
		h.Write([]byte{boolByte(is.Owned)})
	}
	return h.Sum64()
}

func TestStructureSigIsTheFNVOfTheSitesStructuresHavensAndIslands(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	odd := lifeSite()
	odd.Structures = append(odd.Structures, view.Structure{Kind: "kéép\x00\xff", X: -3, Y: 70000, W: 300, H: -1, Plot: 257, Territory: -9})
	odd.Havens = []view.Haven{{ID: "h1", Quay: view.Point{X: 512, Y: -1}, Tier: 200, Pier: 3, Open: true}, {ID: ""}}
	odd.Islands = []view.Island{{ID: "home", Owned: true}, {ID: "far"}}
	for i, site := range []view.Site{{}, lifeSite(), roadlessSite(), sc.Site, odd} {
		if got, want := StructureSig(site), structureSigByHasher(site); got != want {
			t.Errorf("site %d: StructureSig %x, the FNV-1a of the same bytes %x", i, got, want)
		}
	}
}

func TestOwnedByMatchesThePodsReplicaSetToTheCrewsForemenByTheirJoinedIDs(t *testing.T) {
	crews := []model.CrewChain{
		{Foremen: []model.Foreman{{Namespace: "market", Name: "shop-7d9"}, {Namespace: "market", Name: "shop-5c4"}}},
		{Foremen: []model.Foreman{{Namespace: "a/b", Name: "c"}}},
		{Foremen: []model.Foreman{{Namespace: "a", Name: "b/c"}}},
		{Foremen: []model.Foreman{{Namespace: "", Name: ""}}},
		{},
	}
	pods := []model.PodView{
		{Namespace: "market", Owner: ownerPrefix + "shop-7d9"},
		{Namespace: "market", Owner: ownerPrefix + "shop-5c4"},
		{Namespace: "market", Owner: ownerPrefix + "shop"},
		{Namespace: "market", Owner: "StatefulSet/shop-7d9"},
		{Namespace: "other", Owner: ownerPrefix + "shop-7d9"},
		{Namespace: "a", Owner: ownerPrefix + "b/c"},
		{Namespace: "a/b", Owner: ownerPrefix + "c"},
		{Namespace: "", Owner: ownerPrefix},
		{Namespace: "", Owner: ""},
		{Namespace: "market", Owner: ownerPrefix + "shop-7d9x"},
	}
	for _, c := range crews {
		for _, p := range pods {
			want := false
			for _, f := range c.Foremen {
				want = want || f.ID() == OwnerRS(p)
			}
			if got := ownedBy(c, p); got != want {
				t.Errorf("crew %+v, Pod %s %s: ownedBy %v, the joined IDs say %v", c.Foremen, p.Namespace, p.Owner, got, want)
			}
		}
	}
}
