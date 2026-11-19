package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func storageSite() view.Site {
	site := landSite()
	site.Region = make([]uint8, site.Width*site.Height)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			site.Region[y*site.Width+x] = 1
			if x >= 20 {
				site.Region[y*site.Width+x] = 2
			}
		}
	}
	site.Islands = []view.Island{{ID: "home", Zone: "isle-home", Owned: true}, {ID: "morrow", Zone: "isle-morrow", Owned: true}}
	site.Structures = append(site.Structures,
		view.Structure{ID: "vault-1", Kind: view.StructVault, X: 4, Y: 14, W: 2, H: 2, State: view.StateReady, Object: view.ObjectRef{Kind: "PersistentVolume", Name: "vault-1"}},
		view.Structure{ID: "vault-2", Kind: view.StructVault, X: 30, Y: 30, W: 2, H: 2, State: view.StateReady, Object: view.ObjectRef{Kind: "PersistentVolume", Name: "vault-2"}},
	)
	return site
}

func storageScene() Scene {
	return Scene{
		Site: storageSite(),
		Pods: []model.PodView{
			{Namespace: "default", Name: "archive-0", Node: "lone-tower", Image: "archivist:1.0", State: model.HearthReady, Ready: true, Claims: []string{"books-archive-0"}},
			{Namespace: "default", Name: "archive-1", Image: "archivist:1.0", State: model.HearthPending, Claims: []string{"books-archive-1"}},
			{Namespace: "default", Name: "couriers-a", Node: "stone-keep-2", Image: "courier:1.0", State: model.HearthReady},
			{Namespace: "default", Name: "couriers-b", Node: "stone-keep-2", Image: "courier:1.0", State: model.HearthReady},
			{Namespace: "default", Name: "porters-c", Node: "stone-keep-2", Image: "porter:1.0", State: model.HearthReady},
			{Namespace: "default", Name: "couriers-d", Node: "lone-tower", Image: "courier:1.0", State: model.HearthReady},
		},
		Keeps: []view.KeepView{
			{Name: "lone-tower", Daemons: []view.DaemonOnKeep{{Name: "post-wardens-x", Ready: true}}},
			{Name: "stone-keep-2", Stranded: 12},
		},
		Storage: view.StorageState{
			Storage: view.StorageView{
				Vaults: []view.VaultView{
					{Name: "vault-1", Isle: "home", Shelves: 2, Phase: "Bound", Claim: "default/books-archive-0"},
					{Name: "vault-2", Isle: "morrow", Shelves: 2, Phase: "Bound", Claim: "default/books-archive-1"},
				},
				Deeds: []view.DeedView{
					{Name: "books-archive-0", Shelves: 2, Phase: "Bound", Volume: "vault-1"},
					{Name: "books-archive-1", Shelves: 2, Phase: "Bound", Volume: "vault-2"},
				},
			},
			Archive: &view.ArchiveView{SettleIn: 150, Books: 2},
		},
	}
}

func storageHomes() map[string]Vec {
	return map[string]Vec{"archive-0": {12, 12}, "archive-1": {14, 18}, "couriers-a": {26, 22}, "couriers-b": {27, 23}, "porters-c": {26, 26}, "couriers-d": {9, 13}}
}

func TestEveryKeepShowsItsWardenPostInItsState(t *testing.T) {
	sc := storageScene()
	sc.Keeps = append(sc.Keeps, view.KeepView{Name: "far-keep", Daemons: []view.DaemonOnKeep{{Name: "w", Ready: false}}})
	sc.Site.Structures = append(sc.Site.Structures, view.Structure{ID: "far", Kind: view.StructKeep, X: 30, Y: 10, W: 3, H: 3, Object: view.ObjectRef{Kind: "Node", Name: "far-keep"}})
	got := map[string]string{}
	for _, m := range WardenPosts(sc) {
		got[m.Keep] = m.State
		k, _ := KeepForNode(sc.Site, m.Keep)
		if Covers(k, int(m.At.X), int(m.At.Y)) {
			t.Errorf("the post of %s stands inside the Keep at %v", m.Keep, m.At)
		}
	}
	want := map[string]string{"lone-tower": "running", "stone-keep-2": "absent", "far-keep": "starting"}
	for k, st := range want {
		if got[k] != st {
			t.Errorf("post on %s = %q, want %q (all %v)", k, got[k], st, got)
		}
	}
	sc.Pods = nil
	for _, m := range WardenPosts(sc) {
		if m.State == "absent" {
			t.Fatalf("no courier came, yet %s shows an empty post", m.Keep)
		}
	}
}

func TestStrandedTagsHangOverTheCourierHutsOfAWardenlessKeep(t *testing.T) {
	got := StrandedHuts(storageScene())
	if strings.Join(got, ",") != "couriers-a,couriers-b" {
		t.Fatalf("stranded huts = %v", got)
	}
}

func TestVaultLookFollowsItsPhaseAndTheArchiveFill(t *testing.T) {
	a := &view.ArchiveView{SettleIn: view.ArchiveSettleSeconds, Books: 1}
	cases := []struct {
		v    view.VaultView
		a    *view.ArchiveView
		want string
	}{
		{view.VaultView{Phase: "Available"}, a, art.VaultName("available", 0)},
		{view.VaultView{Phase: "Released"}, a, art.VaultName("released", 0)},
		{view.VaultView{Phase: "Terminating", Claim: "default/x"}, nil, art.VaultName("terminating", 1)},
		{view.VaultView{Phase: "Bound", Claim: "default/x"}, nil, art.VaultName("bound", 1)},
		{view.VaultView{Phase: "Bound", Claim: "default/x"}, a, art.VaultName("bound", 1)},
		{view.VaultView{Phase: "Bound", Claim: "default/x"}, &view.ArchiveView{SettleIn: 1, Books: 1}, art.VaultName("bound", 3)},
	}
	for _, c := range cases {
		if got := VaultLook(c.v, c.a); got != c.want {
			t.Errorf("VaultLook(%+v) = %s, want %s", c.v, got, c.want)
		}
	}
}

func TestTheVaultStructureDrawsAsItsVault(t *testing.T) {
	r := NewRenderer(testLib())
	sc := storageScene()
	s := sc.Site.Structures[len(sc.Site.Structures)-2]
	names, _ := r.structureLook(s, sc)
	if len(names) == 0 || names[0] != art.VaultName("bound", 2) {
		t.Fatalf("vault look = %v", names)
	}
}

func TestSelectingAHutOrItsVaultDrawsTheChainToTheKeepAndTheMountLine(t *testing.T) {
	sc := storageScene()
	homes := storageHomes()
	if len(StorageLinks(sc, homes)) != 0 {
		t.Fatal("links show with nothing selected")
	}
	for _, sel := range []string{"default/archive-0", "vault-1"} {
		sc.Selected = sel
		links := StorageLinks(sc, homes)
		if len(links) != 1 {
			t.Fatalf("%s: links = %+v", sel, links)
		}
		l := links[0]
		keep, _ := KeepForNode(sc.Site, "lone-tower")
		if l.Red || l.Vault != FootprintCentre(sc.Site.Structures[3]) || l.Keep != FootprintCentre(keep) || l.Hut != (Vec{12.5, 12.5}) || !l.Sealed || !l.Mounted {
			t.Errorf("%s: link = %+v", sel, l)
		}
		if l.Label != "PVC books-archive-0 -> PV vault-1" {
			t.Errorf("label = %q", l.Label)
		}
	}
}

func TestAChainThatWouldCrossWaterShowsRed(t *testing.T) {
	sc := storageScene()
	sc.Selected = "vault-2"
	links := StorageLinks(sc, storageHomes())
	if len(links) != 1 || !links[0].Red || links[0].Mounted {
		t.Fatalf("an archivist left pending on another isle: %+v", links)
	}
	sc.Pods[1].Node = "lone-tower"
	links = StorageLinks(sc, storageHomes())
	if len(links) != 1 || !links[0].Red {
		t.Fatalf("a Keep on another isle than its Vault: %+v", links)
	}
}

func TestHutsHoldingADeedCarryItsTagUnlessTheirLinkShows(t *testing.T) {
	sc := storageScene()
	sc.Storage.Storage.Deeds[1] = view.DeedView{Name: "books-archive-1", Shelves: 3, Phase: "Pending"}
	marks := DeedMarks(sc)
	if len(marks) != 2 || marks[0].Pod != "archive-0" || !marks[0].Sealed || marks[1].Sealed || marks[1].Shelves != 3 {
		t.Fatalf("deed marks = %+v", marks)
	}
	sc.Selected = "default/archive-0"
	if marks := DeedMarks(sc); len(marks) != 1 || marks[0].Pod != "archive-1" {
		t.Fatalf("the selected hut still wears its tag: %+v", marks)
	}
}

func TestAVaultWaitsForDetachWhileItsOldHutStillStandsOnAnotherKeep(t *testing.T) {
	sc := storageScene()
	if len(DetachWaits(sc)) != 0 {
		t.Fatal("nothing waits yet")
	}
	sc.Pods[0].Terminating = true
	sc.Pods = append(sc.Pods, model.PodView{Namespace: "default", Name: "archive-0", Node: "stone-keep-2", Image: "archivist:1.0", State: model.HearthKindling, Claims: []string{"books-archive-0"}, UID: "new"})
	got := DetachWaits(sc)
	if got["vault-1"] != "lone-tower" {
		t.Fatalf("detach waits = %v", got)
	}
}

func TestLoosePagesScatterAfterAnEmptyDirLoss(t *testing.T) {
	sc := storageScene()
	sc.RealmTime = 100
	if len(LoosePageHuts(sc)) != 0 {
		t.Fatal("pages scatter with no loss")
	}
	sc.Storage.Archive.LostAt, sc.Storage.Archive.LostCause = 95, view.LossEmptyDir
	if got := LoosePageHuts(sc); strings.Join(got, ",") != "archive-0,archive-1" {
		t.Fatalf("loose pages at %v", got)
	}
	sc.Storage.Archive.LostCause = view.LossVolume
	if len(LoosePageHuts(sc)) != 0 {
		t.Fatal("a Vault's loss is not an emptyDir loss")
	}
	sc.Storage.Archive.LostCause, sc.RealmTime = view.LossEmptyDir, 95+loosePagesLife+1
	if len(LoosePageHuts(sc)) != 0 {
		t.Fatal("the pages settle after a while")
	}
}

func TestStorageDrawsWithoutPanicking(t *testing.T) {
	r := NewRenderer(testLib())
	sc := storageScene()
	sc.Selected = "vault-1"
	cam := render.NewCamera(480, 60).CenterOn(12, 14, 960, 540)
	dst := ebiten.NewImage(960, 540)
	r.Homes = storageHomes()
	items := r.storageItems(cam, sc)
	if len(items) < 2 {
		t.Fatalf("storage items = %d", len(items))
	}
	r.DrawStorageMarks(dst, cam, sc)
	if len(r.DrawnStorageMarks()) == 0 {
		t.Fatal("no tag drawn")
	}
	for _, m := range r.DrawnStorageMarks() {
		if m.Text == "" {
			t.Errorf("the %s tag over %s has no tip", m.Kind, m.Pod)
		}
	}
}
