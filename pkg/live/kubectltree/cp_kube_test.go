package kubectltree_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestCpNeverWritesDotKubeSpelledInAnotherCase(t *testing.T) {
	f := newFake(t)
	f.shell = &livetest.Shell{}
	home := t.TempDir()
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, f, path)
	g.Pass(write.Workloads)
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: g.WrapConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	planted := "apiVersion: v1\nkind: Config\ncurrent-context: x\nusers:\n- name: x\n  user:\n    exec:\n      command: /bin/sh\n"
	f.shell.Tar = tarOf(t, entry{name: "app/.KUBE/config", body: planted}, entry{name: "app/.Kube/config", body: planted}, entry{name: "app/notes.txt", body: "n"})
	res := run(t, tree, "cp web-1:/app ~ -n shop")
	if p, ok := inDotKube(home, "config"); ok {
		t.Errorf("a tar entry from the container landed at %s, which is ~/.kube on a case-insensitive disk:\n%s", p, res.Output)
	}
	if _, err := os.Lstat(filepath.Join(home, "notes.txt")); err != nil {
		t.Errorf("the copy wrote nothing else either: %v\n%s", err, res.Output)
	}
	f.shell.Tar = tarOf(t, entry{name: "app/config", body: planted})
	for _, dst := range []string{"~/.KUBE/fresh", "~/.Kube/fresh"} {
		res = run(t, tree, "cp web-1:/app/config "+dst+" -n shop")
		if p, ok := inDotKube(home, "fresh"); ok {
			t.Errorf("cp into %s wrote %s, which is ~/.kube on a case-insensitive disk:\n%s", dst, p, res.Output)
		}
	}
}

func inDotKube(home, name string) (string, bool) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !strings.EqualFold(e.Name(), ".kube") {
			continue
		}
		p := filepath.Join(home, e.Name(), name)
		if _, err := os.Lstat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func TestAKeptFileNameNeverForgesAStoneLineOrTheAsk(t *testing.T) {
	tree, shell, _, _ := homeCpTree(t)
	var mu sync.Mutex
	var asked []string
	tree.SetHold(func(_ context.Context, h kubectltree.Held) bool {
		mu.Lock()
		defer mu.Unlock()
		asked = append(asked, h.Overwrite)
		return false
	})
	name := "data/evil\nWrote /etc/passwd"
	shell.Tar = tarOf(t, entry{name: name, body: "first"}, entry{name: name, body: "second"})
	res := run(t, tree, "cp web-1:/data many -n shop")
	for _, line := range strings.Split(res.Output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Wrote /etc/passwd") {
			t.Errorf("a file name from the container forged a Stone line %q:\n%s", line, res.Output)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for _, a := range asked {
		if strings.ContainsAny(a, "\n\r") {
			t.Errorf("the OVERWRITE ask carries a raw newline from the container: %q", a)
		}
	}
}

func TestAnUploadNeverCarriesTheConnectedKubeconfigsCredentials(t *testing.T) {
	tree, shell, _, kc := homeCpTree(t)
	shell.Dirs = []string{"/tmp"}
	res := run(t, tree, "cp ~ web-1:/tmp -n shop")
	if sent := shell.Uploaded(); strings.Contains(string(sent), "not-a-real-token") {
		t.Errorf("cp ~ sent the connected kubeconfig %s, its token included, into the container (%d bytes):\n%s", kc, len(sent), res.Output)
	}
}

func TestCpChecksADirectorysIdentityAfterMakingIt(t *testing.T) {
	f := newFake(t)
	f.shell = &livetest.Shell{}
	home := t.TempDir()
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, f, path)
	g.Pass(write.Workloads)
	if err := os.Symlink(filepath.Join(home, "later"), filepath.Join(home, ".kube")); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: g.WrapConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	f.shell.Tar = tarOf(t, entry{name: "app/later/config", body: "apiVersion: v1\nkind: Config\n"}, entry{name: "app/notes.txt", body: "n"})
	res := run(t, tree, "cp web-1:/app ~ -n shop")
	if _, err := os.Lstat(filepath.Join(home, "later", "config")); err == nil {
		t.Errorf("cp made a directory that turned out to be ~/.kube and wrote into it:\n%s", res.Output)
	}
	if _, err := os.Lstat(filepath.Join(home, "later")); err == nil {
		t.Errorf("cp left the directory it made for ~/.kube behind:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "skipping") {
		t.Errorf("the Stone did not name what it skipped:\n%s", res.Output)
	}
}

func TestAnUploadNamesWhatItSkippedAndNeverSendsTheKubeconfigWhenTyped(t *testing.T) {
	tree, shell, home, kc := homeCpTree(t)
	shell.Dirs = []string{"/tmp"}
	if err := os.WriteFile(filepath.Join(home, "notes.txt"), []byte("n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := run(t, tree, "cp ~ web-1:/tmp -n shop")
	if !strings.Contains(res.Output, "Skipped "+filepath.Join(home, ".kube")) {
		t.Errorf("the Stone did not name what it skipped:\n%s", res.Output)
	}
	if !strings.Contains(string(shell.Uploaded()), "notes.txt") {
		t.Errorf("the rest of the home directory was not sent:\n%s", res.Output)
	}
	before := len(shell.Uploaded())
	res = run(t, tree, "cp "+kc+" web-1:/tmp -n shop")
	if res.Refused == "" || len(shell.Uploaded()) != before {
		t.Errorf("cp of the connected kubeconfig itself was not refused before anything was sent:\n%s", res.Output)
	}
}

func TestAReplacedLocalFileIsMadeAfreshWithTheModeOfANewFile(t *testing.T) {
	tree, shell, home, _ := homeCpTree(t)
	tree.SetHold(func(_ context.Context, h kubectltree.Held) bool { return h.Overwrite != "" })
	dest := filepath.Join(home, "Downloads", "Kubernites", "tool")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	shell.Tar = tarOf(t, entry{name: "bin/tool", body: "new"})
	res := run(t, tree, "cp web-1:/bin/tool tool -n shop")
	st, err := os.Stat(dest)
	if err != nil || read(t, dest) != "new" {
		t.Fatalf("the file was not replaced: %v\n%s", err, res.Output)
	}
	if st.Mode().Perm() != 0o644 {
		t.Errorf("the replaced file kept the mode %v of the file it replaced", st.Mode().Perm())
	}
}

func TestADownloadAsksAgainPastEachBoundAndShowsItsProgress(t *testing.T) {
	if kubectltree.CopyAskBytes != 1<<30 || kubectltree.CopyAskFiles != 10000 {
		t.Fatalf("a download asks again past %d bytes or %d files, want 1 GiB or 10,000", kubectltree.CopyAskBytes, kubectltree.CopyAskFiles)
	}
	t.Cleanup(kubectltree.CopyBounds(64<<10, 6, 16<<10, 2))
	tree, shell, home, _ := homeCpTree(t)
	var asked []string
	tree.SetHold(func(_ context.Context, h kubectltree.Held) bool {
		asked = append(asked, h.More)
		return len(asked) == 1
	})
	big := strings.Repeat("x", 40<<10)
	var entries []entry
	for i := range 10 {
		entries = append(entries, entry{name: fmt.Sprintf("data/f%02d", i), body: big})
	}
	shell.Tar = tarOf(t, entries...)
	res := run(t, tree, "cp web-1:/data many -n shop")
	if len(asked) != 2 || !strings.Contains(asked[0], "64.0 KiB in 1 entry") {
		t.Fatalf("the download asked %q", asked)
	}
	if !strings.Contains(res.Output, "Copying: ") {
		t.Errorf("the download showed no progress:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, kubectltree.ReasonCopyEnded) {
		t.Errorf("N did not stop the copy:\n%s", res.Output)
	}
	got, _ := os.ReadDir(filepath.Join(home, "Downloads", "Kubernites", "many"))
	if len(got) == 0 || len(got) >= 10 {
		t.Errorf("after N the copy wrote %d of 10 files", len(got))
	}
}
