package kubectltree_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestAQuotedWebAddressIsNeverFetched(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	var hits atomic.Int32
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/yaml")
		fmt.Fprint(w, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\n  namespace: shop\nspec:\n  containers:\n  - name: web\n    image: nginx:1.27\n")
	}))
	t.Cleanup(web.Close)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	ctl := run(t, tree, "get -f "+web.URL+"/pod.yaml -o name")
	t.Logf("control: hits=%d refused=%q", hits.Load(), ctl.Refused)
	for _, line := range []string{
		`get -f '"` + web.URL + `/pod.yaml"' -o name`,
		`get --filename='"` + web.URL + `/pod.yaml"' -o name`,
	} {
		before := hits.Load()
		res := run(t, tree, line)
		t.Logf("%s: hits=%d refused=%q code=%d\n%s", line, hits.Load()-before, res.Refused, res.Code, res.Output)
	}
	if hits.Load() != 0 {
		t.Fatalf("at READ ONLY the Stone fetched a manifest from the web outside the gate (%d requests to %s)", hits.Load(), web.URL)
	}
}

func TestAQuotedDashNeverReadsTheProcessStdin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdin
	os.Stdin = pr
	t.Cleanup(func() { os.Stdin = saved; pr.Close() })
	go func() {
		fmt.Fprint(pw, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: read-from-the-process-stdin\n  namespace: shop\n")
		pw.Close()
	}()
	ctl := run(t, tree, "create -f - --dry-run=client -o name")
	t.Logf("control -f -: refused=%q", ctl.Refused)
	res := run(t, tree, `create -f '"-"' --dry-run=client -o name`)
	t.Logf(`create -f '"-"': refused=%q code=%d\n%s`, res.Refused, res.Code, res.Output)
	if strings.Contains(res.Output, "read-from-the-process-stdin") || res.Refused == "" {
		t.Fatalf("the Stone, which has no standard input, read the process's standard input")
	}
}

func TestNoLocalFileFlagWaitsOnAPipe(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe.yaml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	glob := filepath.Join(dir, "pip[e].yaml")
	var stuck []string
	for _, line := range []string{
		"patch deployment web -n shop --patch-file=" + fifo,
		"create configmap c -n shop --from-literal=a=b --dry-run=client -o go-template-file=" + fifo,
		"get pods -n shop -o jsonpath-file=" + fifo,
		"get pods -n shop -o custom-columns-file=" + fifo,
		"get -f " + glob,
		`apply -f '"` + fifo + `"'`,
		`create configmap c -n shop --from-file='"k=` + fifo + `"' --dry-run=client`,
	} {
		tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, f.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan kubectltree.Result, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			done <- tree.Run(ctx, line)
		}()
		select {
		case res := <-done:
			t.Logf("%s: code=%d refused=%q", line, res.Code, res.Refused)
		case <-time.After(4 * time.Second):
			stuck = append(stuck, line)
			go func() {
				if w, err := os.OpenFile(fifo, os.O_WRONLY, 0); err == nil {
					w.Close()
				}
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
			}
		}
		tree.Close()
	}
	if len(stuck) > 0 {
		t.Fatalf("these Stone commands waited on a local pipe, which a regular-files-only guard refuses:\n%s", strings.Join(stuck, "\n"))
	}
}

func TestEveryLocalFileFlagRefusesAPipeADeviceTheWebAndTheKubeconfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := newFake(t)
	connected := kubeconfig(t, f.URL, "stone", "shop")
	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".kube", "config"), []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe.yaml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(dir, "linked.yaml")
	if err := os.Symlink(connected, linked); err != nil {
		t.Fatal(err)
	}
	holder := filepath.Join(dir, "holder")
	if err := os.MkdirAll(holder, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(fifo, filepath.Join(holder, "inner.yaml")); err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{
		"a pipe":                         fifo,
		"a device":                       "/dev/zero",
		"a pattern matching a pipe":      filepath.Join(dir, "pip[e].yaml"),
		"a directory holding a pipe":     holder,
		"the web":                        "https://manifests.invalid/x.yaml",
		"the connected kubeconfig":       connected,
		"a link to the connected config": linked,
		"a file in ~/.kube":              filepath.Join(home, ".kube", "config"),
		"~/.kube":                        filepath.Join(home, ".kube"),
	}
	lines := []string{
		"apply -f {}",
		"create -f {} --dry-run=client",
		"create configmap c -n shop --from-file={} --dry-run=client",
		"create configmap c -n shop --from-file=k={} --dry-run=client",
		"create configmap c -n shop --from-env-file={} --dry-run=client",
		"create secret generic s -n shop --from-file={} --dry-run=client",
		"create secret generic s -n shop --from-env-file={} --dry-run=client",
		"create secret docker-registry s -n shop --from-file={} --dry-run=client",
		"create secret tls s -n shop --cert={} --key={} --dry-run=client",
		"patch deployment web -n shop --patch-file={}",
		"debug web-1 -n shop --image=busybox:1.36 --custom={}",
		"get pods -n shop -o go-template-file={}",
		"get pods -n shop -o templatefile --template={}",
		"get pods -n shop -o jsonpath-file={}",
		"get pods -n shop -o jsonpath-file --template={}",
		"get pods -n shop -o custom-columns-file={}",
		"create --raw /api/v1/namespaces/shop/configmaps -f {}",
		"replace --raw /api/v1/namespaces/shop/configmaps/c -f {}",
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{connected}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	for _, line := range lines {
		for what, p := range paths {
			typed := strings.ReplaceAll(line, "{}", p)
			done := make(chan kubectltree.Result, 1)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				done <- tree.Run(ctx, typed)
			}()
			select {
			case res := <-done:
				if res.Refused == "" {
					t.Errorf("%s with %s was not refused before kubectl read it (code %d):\n%s", line, what, res.Code, res.Output)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("%s with %s waited on a local read", line, what)
				if w, err := os.OpenFile(fifo, os.O_WRONLY, 0); err == nil {
					w.Close()
				}
				<-done
			}
		}
	}
}

func TestTheStoneChecksEveryFlagKubectlReadsALocalFileThrough(t *testing.T) {
	f := newFake(t)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	reads := map[string]bool{"filename": true, "from-file": true, "from-env-file": true, "patch-file": true, "custom": true, "cert": true, "key": true}
	for _, name := range kubectltree.FileFlags {
		reads[name] = true
	}
	for _, format := range []string{"go-template-file", "templatefile", "jsonpath-file", "custom-columns-file"} {
		if !kubectltree.TemplateFiles[format] {
			t.Errorf("the output format %s reads its template from a file and is not checked", format)
		}
	}
	seen := map[string]bool{}
	for _, fl := range kubectltree.TreeFlags(t, tree) {
		seen[fl.Name] = true
		if fl.Locked {
			continue
		}
		if (strings.HasPrefix(fl.Name, "file") || strings.Contains(fl.Name, "-file")) && !slices.Contains(kubectltree.FileFlags, fl.Name) {
			t.Errorf("%s --%s names a file and the Stone does not check it", fl.Command, fl.Name)
		}
		if reads[fl.Name] && !slices.Contains(kubectltree.FileFlags, fl.Name) {
			t.Errorf("%s --%s reads a local file and the Stone does not check it", fl.Command, fl.Name)
		}
	}
	for name := range reads {
		if !seen[name] {
			t.Errorf("--%s is not in the Stone's tree any more: read kubectl's source again for the flags that read local files", name)
		}
	}
}
