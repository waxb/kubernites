package app

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestANewArchivistWalksBackToItsVault(t *testing.T) {
	deeds := []view.DeedView{{Name: "books-archive-0", Phase: "Bound", Volume: "vault-1"}, {Name: "books", Phase: "Pending"}}
	before := map[string]bool{"archive-0": true}
	pods := []model.PodView{
		{Name: "archive-0", Image: "archivist:1.0", Claims: []string{"books-archive-0"}},
		{Name: "archive-1", Image: "archivist:1.0", Claims: []string{"books"}},
		{Name: "porters-a", Image: "porter:1.0"},
	}
	if got := ArchivistReturns(before, pods, deeds); len(got) != 0 {
		t.Fatalf("an archivist that never left walked: %v", got)
	}
	got := ArchivistReturns(map[string]bool{}, pods, deeds)
	if len(got) != 1 || got[0] != (bookWalk{pod: "archive-0", vault: "vault-1"}) {
		t.Fatalf("returns = %+v", got)
	}
}

func TestTheGuildRisesOverEveryVaultItDigs(t *testing.T) {
	vaults := []view.VaultView{{Name: "vault-1"}, {Name: "pvc-abc", Dynamic: true}, {Name: "pvc-def", Dynamic: true}}
	if got := GuildDug(map[string]bool{"pvc-abc": true}, vaults); !slices.Equal(got, []string{"pvc-def"}) {
		t.Fatalf("dug = %v", got)
	}
}
