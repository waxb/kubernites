package kubectltree_test

import (
	"archive/tar"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestADownloadOfManyDirectoriesAsksAgain(t *testing.T) {
	t.Cleanup(kubectltree.CopyBounds(1<<30, 6, 1<<30, 1000))
	tree, shell, home, _ := homeCpTree(t)
	asked := 0
	tree.SetHold(func(_ context.Context, h kubectltree.Held) bool {
		if h.More != "" {
			asked++
		}
		return false
	})
	var entries []entry
	for i := range 40 {
		entries = append(entries, entry{name: fmt.Sprintf("data/d%02d/", i), flag: tar.TypeDir})
	}
	shell.Tar = tarOf(t, entries...)
	run(t, tree, "cp web-1:/data many -n shop")
	made, _ := os.ReadDir(filepath.Join(home, "Downloads", "Kubernites", "many"))
	if asked == 0 && len(made) > 6 {
		t.Fatalf("with the bound at 6 entries, the download made %d directories on this computer and never asked to copy on", len(made))
	}
}

func TestAFileCutShortByNIsRemovedAndNamedAsNotCopied(t *testing.T) {
	t.Cleanup(kubectltree.CopyBounds(64<<10, 1000, 1<<30, 1000))
	tree, shell, home, _ := homeCpTree(t)
	tree.SetHold(func(_ context.Context, h kubectltree.Held) bool { return false })
	shell.Tar = tarOf(t, entry{name: "data/whole", body: strings.Repeat("x", 40<<10)}, entry{name: "data/cut", body: strings.Repeat("y", 40<<10)})
	res := run(t, tree, "cp web-1:/data many -n shop")
	dir := filepath.Join(home, "Downloads", "Kubernites", "many")
	if _, err := os.Stat(filepath.Join(dir, "whole")); err != nil {
		t.Fatalf("the file copied whole before N is gone: %v\n%s", err, res.Output)
	}
	if _, err := os.Stat(filepath.Join(dir, "cut")); err == nil {
		t.Errorf("the file N cut short was kept on this computer\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "Not copied: "+filepath.Join(dir, "cut")) {
		t.Errorf("the Stone did not name the file N cut short as not copied:\n%s", res.Output)
	}
	if strings.Contains(res.Output, "Wrote") && strings.Contains(res.Output, filepath.Join(dir, "cut")+"\n") {
		t.Errorf("the Stone lists the cut file as written:\n%s", res.Output)
	}
}
