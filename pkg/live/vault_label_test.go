package live_test

import (
	"testing"

	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAVaultReadsAsTheClaimItHoldsWithKubernetesNames(t *testing.T) {
	for _, fixture := range []string{"testdata/kind-snapshot.json", "testdata/managed-snapshot.json"} {
		w := build(t, livetest.Snapshot(t, fixture))
		vaults := structuresOf(w, view.StructVault)
		if len(vaults) == 0 {
			t.Fatalf("%s: no vault", fixture)
		}
		for _, v := range vaults {
			want := "PersistentVolumeClaim " + v.Object.Name
			if got := lexicon.TrueNames(v.Label); v.Object.Kind != "PersistentVolumeClaim" || got != want {
				t.Errorf("%s: the vault for claim %s/%s reads %q with Kubernetes names, want %q", fixture, v.Object.Namespace, v.Object.Name, got, want)
			}
		}
	}
}
