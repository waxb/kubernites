package kubectltree_test

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

type entry struct {
	name, body, link string
	flag             byte
}

func tarOf(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.body)), Typeflag: e.flag, Linkname: e.link}
		if e.flag == 0 {
			h.Typeflag = tar.TypeReg
		}
		if e.flag != tar.TypeReg && e.flag != 0 {
			h.Size = 0
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			io.WriteString(tw, e.body)
		}
	}
	tw.Close()
	return buf.Bytes()
}

func cpTree(t *testing.T) (*kubectltree.Tree, *livetest.Shell, string) {
	t.Helper()
	f := newFake(t)
	f.shell = &livetest.Shell{}
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, f, path)
	g.Pass(write.Workloads)
	home := t.TempDir()
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: g.WrapConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	return tree, f.shell, home
}

func read(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func everyFile(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && p != root {
			rel, _ := filepath.Rel(root, p)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

func TestCpCopiesBothWaysWithItsRelativeLocalPathsUnderDownloadsKubernites(t *testing.T) {
	tree, shell, home := cpTree(t)
	cwd := t.TempDir()
	t.Chdir(cwd)
	work := filepath.Join(home, "Downloads", "Kubernites")
	if tree.LocalDir() != work || tree.LocalPath("hosts") != filepath.Join(work, "hosts") || tree.LocalPath("~/a/b") != filepath.Join(home, "a", "b") || tree.LocalPath("/x/../y") != "/y" {
		t.Fatalf("local paths: %s %s %s", tree.LocalDir(), tree.LocalPath("hosts"), tree.LocalPath("~/a/b"))
	}
	shell.Tar = tarOf(t, entry{name: "etc/hosts", body: "127.0.0.1 localhost\n"})
	res := run(t, tree, "cp web-1:/etc/hosts hosts -n shop")
	if res.Code != 0 || read(t, filepath.Join(work, "hosts")) != "127.0.0.1 localhost\n" || !strings.Contains(res.Output, "Wrote "+filepath.Join(work, "hosts")+"\n") {
		t.Fatalf("cp answered %d:\n%s", res.Code, res.Output)
	}
	shell.Tar = tarOf(t, entry{name: "tmp/data/", flag: tar.TypeDir}, entry{name: "tmp/data/a.txt", body: "a"}, entry{name: "tmp/data/sub/b.txt", body: "b"})
	res = run(t, tree, "cp shop/web-1:/tmp/data ~/out")
	if res.Code != 0 || read(t, filepath.Join(home, "out", "a.txt")) != "a" || read(t, filepath.Join(home, "out", "sub", "b.txt")) != "b" || !strings.Contains(res.Output, "Wrote 2 files under "+filepath.Join(home, "out")) {
		t.Fatalf("cp of a directory answered %d:\n%s", res.Code, res.Output)
	}
	os.MkdirAll(filepath.Join(work, "up"), 0o755)
	os.WriteFile(filepath.Join(work, "up", "x.txt"), []byte("to the hut"), 0o644)
	shell.Dirs = []string{"/tmp"}
	res = run(t, tree, "cp up web-1:/tmp -n shop")
	if res.Code != 0 || !strings.Contains(res.Output, "Read 1 file under "+filepath.Join(work, "up")) {
		t.Fatalf("cp to the Pod answered %d:\n%s", res.Code, res.Output)
	}
	tr := tar.NewReader(bytes.NewReader(shell.Uploaded()))
	h, err := tr.Next()
	body, _ := io.ReadAll(tr)
	if err != nil || h.Name != "up/x.txt" || string(body) != "to the hut" {
		t.Fatalf("the container got %+v %q", h, body)
	}
	_, urls, _, _ := shell.Seen()
	joined := strings.Join(urls, "\n")
	for _, want := range []string{"command=tar&command=cf&command=-&command=%2Fetc%2Fhosts", "command=test&command=-d&command=%2Ftmp", "command=tar&command=-xmf&command=-&command=-C&command=%2Ftmp"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the cluster saw\n%s\nwant %s", joined, want)
		}
	}
	if left := everyFile(t, cwd); len(left) != 0 {
		t.Fatalf("cp wrote into the working directory: %v", left)
	}
	if len(res.Sessions) != 2 || res.Sessions[1].Verb != "cp" || res.Sessions[1].Container != "web" || strings.Join(res.Sessions[1].Command, " ") != "tar -xmf - -C /tmp" || res.Sessions[1].Ended != kubectltree.EndedExited {
		t.Fatalf("sessions %+v", res.Sessions)
	}
}

func TestCpNeverReadsOrWritesOutsideThePathsTyped(t *testing.T) {
	tree, shell, _ := cpTree(t)
	home := tree.LocalDir()
	outside := filepath.Join(home, "outside")
	os.MkdirAll(outside, 0o755)
	os.MkdirAll(filepath.Join(home, "dest"), 0o755)
	os.Symlink(outside, filepath.Join(home, "dest", "pre"))
	shell.Tar = tarOf(t,
		entry{name: "tmp/data/ok.txt", body: "fine"},
		entry{name: "tmp/data/../../escape.txt", body: "escaped"},
		entry{name: "tmp/data/../dest-sibling.txt", body: "escaped"},
		entry{name: "tmp/data/link", flag: tar.TypeSymlink, link: outside},
		entry{name: "tmp/data/link/through.txt", body: "inside a directory named link"},
		entry{name: "tmp/data/hard", flag: tar.TypeLink, link: "/etc/passwd"},
		entry{name: "tmp/data/pre/owned.txt", body: "through a local link"},
	)
	res := run(t, tree, "cp web-1:/tmp/data dest -n shop")
	if res.Code != 0 {
		t.Fatalf("cp answered %d:\n%s", res.Code, res.Output)
	}
	if got := everyFile(t, outside); len(got) != 0 {
		t.Fatalf("cp wrote through a link: %v", got)
	}
	for _, p := range []string{filepath.Join(home, "escape.txt"), filepath.Join(home, "dest-sibling.txt"), filepath.Join(filepath.Dir(home), "escape.txt")} {
		if _, err := os.Lstat(p); err == nil {
			t.Fatalf("cp wrote %s", p)
		}
	}
	if read(t, filepath.Join(home, "dest", "ok.txt")) != "fine" || read(t, filepath.Join(home, "dest", "link", "through.txt")) != "inside a directory named link" {
		t.Fatalf("the files inside are missing:\n%s", res.Output)
	}
	if st, err := os.Lstat(filepath.Join(home, "dest", "link")); err != nil || st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("a link from the container was made")
	}
	for _, warn := range []string{"is outside target destination, skipping", "skipping symlink", "would be written through a link, skipping", "not a regular file"} {
		if !strings.Contains(res.Output, warn) {
			t.Errorf("no %q in:\n%s", warn, res.Output)
		}
	}
	shell.Tar = tarOf(t, entry{name: "etc/passwd", body: "root:x:0:0"})
	if res := run(t, tree, "cp web-1:/tmp/data dest2 -n shop"); res.Code == 0 || !strings.Contains(res.Output, "tar contents corrupted") {
		t.Fatalf("a tar that does not start with the path answered %d:\n%s", res.Code, res.Output)
	}

	secret := filepath.Join(home, "secret.txt")
	os.WriteFile(secret, []byte("TOP-SECRET"), 0o600)
	os.MkdirAll(filepath.Join(home, "up"), 0o755)
	os.WriteFile(filepath.Join(home, "up", "x.txt"), []byte("plain"), 0o644)
	os.Symlink(secret, filepath.Join(home, "up", "secret-link"))
	shell.Dirs = []string{"/tmp"}
	if res := run(t, tree, "cp up web-1:/tmp -n shop"); res.Code != 0 {
		t.Fatalf("cp to the Pod answered %d:\n%s", res.Code, res.Output)
	}
	sent := shell.Uploaded()
	if bytes.Contains(sent, []byte("TOP-SECRET")) {
		t.Fatal("cp read the file a link points to, outside the path typed")
	}
	tr := tar.NewReader(bytes.NewReader(sent))
	names := map[string]string{}
	for h, err := tr.Next(); err == nil; h, err = tr.Next() {
		names[h.Name] = h.Linkname
	}
	if _, ok := names["up/x.txt"]; !ok || names["up/secret-link"] != secret {
		t.Fatalf("the tar holds %v", names)
	}
}

func TestCpLocksRetriesAndNeedsTheWorkloadsWarning(t *testing.T) {
	tree, _, _ := cpTree(t)
	if res := run(t, tree, "cp web-1:/etc/hosts hosts --retries=3"); res.Code == 0 || !strings.Contains(res.Output, kubectltree.ReasonRetries) {
		t.Fatalf("--retries answered %d:\n%s", res.Code, res.Output)
	}
	gated, f, asks := gatedTree(t, write.Writes, false)
	f.shell = &livetest.Shell{}
	if res := run(t, gated, "cp web-1:/etc/hosts hosts"); res.Code == 0 || len(*asks) != 1 || (*asks)[0].Class != write.Workloads {
		t.Fatalf("cp at WRITES asked %+v and answered %d", *asks, res.Code)
	}
}
