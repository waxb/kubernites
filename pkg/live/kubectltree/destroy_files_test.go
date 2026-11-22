package kubectltree_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

const onePod = "apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\n  namespace: shop\n"
const otherPod = "apiVersion: v1\nkind: Pod\nmetadata:\n  name: shell-1\n  namespace: shop\n"

func TestEveryCommandThatDestroysSeveralObjectsAsksTheDestroyingWarningBeforeTheFirstDelete(t *testing.T) {
	dir := t.TempDir()
	write1 := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	single := write1("single/pod.yaml", onePod)
	multi := write1("multi.yaml", onePod+"---\n"+otherPod)
	list := write1("list.json", `{"apiVersion":"v1","kind":"List","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"web-1","namespace":"shop"}},{"apiVersion":"v1","kind":"Pod","metadata":{"name":"shell-1","namespace":"shop"}}]}`)
	write1("tree/a.yaml", onePod)
	write1("tree/deeper/b.yaml", otherPod)
	tree := filepath.Join(dir, "tree")
	pair := filepath.Join(dir, "pair")
	write1("pair/a.yaml", onePod)
	write1("pair/b.yaml", otherPod)
	for _, c := range []struct {
		line    string
		destroy bool
	}{
		{"delete -f " + multi + " --wait=false", true},
		{"delete -f " + list + " --wait=false", true},
		{"delete -f " + pair + " --wait=false", true},
		{"delete -R -f " + tree + " --wait=false", true},
		{"delete -f " + single + " -f " + filepath.Join(pair, "b.yaml") + " --wait=false", true},
		{"replace --force -f " + multi + " --timeout=1s", true},
		{"apply --force -f " + multi, true},
		{"drain keep-1 --disable-eviction --ignore-daemonsets --timeout=1s", true},
		{"drain keep-1 --force --ignore-daemonsets --timeout=1s", true},
		{"delete -f " + single + " --wait=false", false},
		{"delete -f " + tree + " --wait=false", false},
		{"replace -f " + multi, false},
		{"replace --force -f " + single + " --timeout=1s", false},
	} {
		f := newFake(t)
		path := kubeconfig(t, f.URL, "stone", "shop")
		g := gateAt(t, write.Seamless, f, path)
		tr := newTree(t, path, g.WrapConfig)
		asked, deletesBefore := false, -1
		tr.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
			return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
				if a.Why == write.NeedsClass && a.Class == write.Destroy && !asked {
					asked = true
					deletesBefore = 0
					for _, w := range f.seen.writes() {
						if strings.HasPrefix(w, "DELETE ") || strings.Contains(w, "/eviction") {
							deletesBefore++
						}
					}
				}
				return true
			})
		})
		res := run(t, tr, c.line)
		switch {
		case c.destroy && !asked:
			t.Errorf("%s sent %v with no destroying warning\n%s", c.line, f.seen.writes(), res.Output)
		case c.destroy && deletesBefore != 0:
			t.Errorf("%s asked the destroying warning only after %d deletes or evictions", c.line, deletesBefore)
		case !c.destroy && asked:
			t.Errorf("%s destroys at most one object and asked the destroying warning", c.line)
		}
	}
}
