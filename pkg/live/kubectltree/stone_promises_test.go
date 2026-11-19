package kubectltree_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"k8s.io/client-go/rest"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func stoneTree(t *testing.T, path string, wrap func(*rest.Config) *rest.Config) *kubectltree.Tree {
	t.Helper()
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: wrap, Local: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	return tree
}

func TestDebugNeverFetchesAManifestFromTheWeb(t *testing.T) {
	f := newFake(t)
	var hits atomic.Int32
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/yaml")
		fmt.Fprint(w, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\n  namespace: shop\nspec:\n  securityContext: {}\n  containers:\n  - name: web\n    image: nginx:1.27\n")
	}))
	t.Cleanup(web.Close)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	ctl := run(t, tree, "get -f "+web.URL+"/pod.yaml")
	t.Logf("control get -f URL: hits=%d refused=%q", hits.Load(), ctl.Refused)
	res := run(t, tree, "debug -f "+web.URL+"/pod.yaml --image=busybox -n shop")
	t.Logf("debug -f URL: hits=%d refused=%q code=%d\n%s", hits.Load(), res.Refused, res.Code, res.Output)
	if hits.Load() != 0 {
		t.Fatalf("the Stone fetched a manifest from the web outside the gate (%d requests to %s)", hits.Load(), web.URL)
	}
}

func TestDebugOfALocalPodFileNeverEndsTheProcess(t *testing.T) {
	if os.Getenv("STONE_CHILD") == "1" {
		f := newFake(t)
		pod := filepath.Join(t.TempDir(), "pod.yaml")
		if err := os.WriteFile(pod, []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\n  namespace: shop\nspec:\n  containers:\n  - name: web\n    image: nginx:1.27\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
		res := run(t, tree, "debug -f "+pod+" --image=busybox -n shop")
		t.Logf("the Stone survived: code=%d refused=%q\n%s", res.Code, res.Refused, res.Output)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDebugOfALocalPodFileNeverEndsTheProcess$", "-test.v", "-test.count=1")
	cmd.Env = append(os.Environ(), "STONE_CHILD=1", "TMPDIR="+t.TempDir())
	out, err := cmd.CombinedOutput()
	text := string(out)
	if i := strings.Index(text, "goroutine "); i >= 0 && len(text) > i+1500 {
		text = text[:i+1500]
	}
	t.Logf("child exit: %v\n%s", err, text)
	if err != nil && strings.Contains(string(out), "panic:") {
		t.Fatalf("kubectl debug -f <local pod file> at READ ONLY ended the whole process")
	}
}

func TestShellCompletionIsRefusedBehindAnEmptyWord(t *testing.T) {
	f := newFake(t)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "kubectl-foo"), []byte("never executed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	logf := filepath.Join(t.TempDir(), "comp.log")
	t.Setenv("BASH_COMP_DEBUG_FILE", logf)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	ctl := run(t, tree, "__complete foo ''")
	t.Logf("control __complete: refused=%q", ctl.Refused)
	list := run(t, tree, "'' __complete f")
	t.Logf("'' __complete f: refused=%q code=%d\n%s", list.Refused, list.Code, list.Output)
	plug := run(t, tree, "'' __complete foo ''")
	t.Logf("'' __complete foo '': refused=%q code=%d\n%s", plug.Refused, plug.Code, plug.Output)
	b, _ := os.ReadFile(logf)
	t.Logf("cobra completion debug log:\n%s", b)
	if list.Refused == "" || plug.Refused == "" || strings.Contains(string(b), "kubectl_complete-foo") {
		t.Fatalf("shell completion ran in the Stone and looked up a plugin completion program on PATH")
	}
}

func TestConfigViewShowsNoKubeconfigCredential(t *testing.T) {
	f := newFake(t)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: KEPT-PLAIN-TOKEN\n- name: oidc-user\n  user:\n    auth-provider:\n      name: oidc\n      config:\n        client-id: kubernetes\n        client-secret: KEPT-CLIENT-SECRET\n        id-token: KEPT-ID-TOKEN\n        refresh-token: KEPT-REFRESH-TOKEN\n        idp-issuer-url: https://issuer.invalid\n- name: exec-user\n  user:\n    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: /nonexistent/never-run\n      args:\n      - --password=KEPT-EXEC-ARG\n      env:\n      - name: API_TOKEN\n        value: KEPT-EXEC-ENV\n      interactiveMode: Never\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	res := run(t, tree, "config view")
	t.Logf("config view (code %d, refused %q):\n%s", res.Code, res.Refused, res.Output)
	var leaked []string
	for _, s := range []string{"KEPT-PLAIN-TOKEN", "KEPT-CLIENT-SECRET", "KEPT-ID-TOKEN", "KEPT-REFRESH-TOKEN", "KEPT-EXEC-ARG", "KEPT-EXEC-ENV"} {
		if strings.Contains(res.Output, s) {
			leaked = append(leaked, s)
		}
	}
	if len(leaked) > 0 {
		t.Fatalf("config view printed kubeconfig credentials into the Stone: %v", leaked)
	}
}

func TestSecretValuesNeverComeBackThroughADryRunWrite(t *testing.T) {
	f := newFake(t)
	g, err := write.New(write.Choice{Context: write.Context{Name: "stone", Server: f.URL, User: "fake-user"}, Level: write.Writes, Classes: []write.Class{write.Secrets}})
	if err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), g.WrapConfig)
	got := run(t, tree, "get secret s -n shop -o json")
	t.Logf("control get secret: code=%d plain value shown=%v", got.Code, strings.Contains(got.Output, "aHVudGVyMg=="))
	res := run(t, tree, `patch secret s -n shop --type=merge -p '{"metadata":{"labels":{"seen":"yes"}}}' --dry-run=server -o json`)
	t.Logf("patch --dry-run=server: code=%d refused=%q server saw %v\n%s", res.Code, res.Refused, f.seen.all(), res.Output)
	if strings.Contains(res.Output, "aHVudGVyMg==") || strings.Contains(res.Output, "Y2VydA==") {
		t.Fatalf("a Secret's values reached the Stone unmasked through a dry-run write at WRITES, with no CONFIRM")
	}
}

func TestALocalReadThatBlocksEndsOnStopAndClose(t *testing.T) {
	f := newFake(t)
	fifo := filepath.Join(t.TempDir(), "pipe.yaml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, f.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	job := tree.Start("apply -f "+fifo, nil)
	time.Sleep(1500 * time.Millisecond)
	job.Stop()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !job.Done() {
		time.Sleep(50 * time.Millisecond)
	}
	jobStuck := !job.Done()
	closed := make(chan struct{})
	go func() {
		tree.Close()
		close(closed)
	}()
	closeStuck := false
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		closeStuck = true
	}
	w, werr := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if werr == nil {
		w.Close()
	}
	select {
	case <-closed:
	case <-time.After(30 * time.Second):
		t.Fatal("Close never returned, even after the pipe closed")
	}
	t.Logf("job stuck 3s after Stop=%v, Close stuck 2s=%v, writer open err=%v, result=%q", jobStuck, closeStuck, werr, job.Result().Output)
	if jobStuck || closeStuck {
		t.Fatalf("a typed command blocked on a local read ignores Stop, and Tree.Close (Realm.Stop) waits on it")
	}
}

func TestConfigViewRedactsEveryCredentialOfEveryUserInEveryForm(t *testing.T) {
	f := newFake(t)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\n- name: other\n  context:\n    cluster: fake\n    user: oidc-user\nusers:\n- name: fake-user\n  user:\n    token: KEPT-PLAIN-TOKEN\n    password: KEPT-PASSWORD\n    client-key-data: S0VQVC1LRVk=\n- name: oidc-user\n  user:\n    auth-provider:\n      name: oidc\n      config:\n        client-secret: KEPT-CLIENT-SECRET\n        id-token: KEPT-ID-TOKEN\n- name: exec-user\n  user:\n    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: /nonexistent/never-run\n      args:\n      - --password=KEPT-EXEC-ARG\n      env:\n      - name: API_TOKEN\n        value: KEPT-EXEC-ENV\n      interactiveMode: Never\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{
		"config view", "config view -o json", "config view --minify", "config view --context other --minify -o yaml",
		"config view -o jsonpath={.users[*].user}", "config view -o go-template={{range .users}}{{.user}}{{end}}",
		"config view --merge=false", "config view --raw", "config view --flatten",
	} {
		res := run(t, tree, line)
		for _, s := range []string{"KEPT-PLAIN-TOKEN", "KEPT-PASSWORD", "S0VQVC1LRVk=", "KEPT-CLIENT-SECRET", "KEPT-ID-TOKEN", "KEPT-EXEC-ARG", "KEPT-EXEC-ENV"} {
			if strings.Contains(res.Output, s) {
				t.Errorf("%s printed %s:\n%s", line, s, res.Output)
			}
		}
	}
	res := run(t, tree, "config view")
	for _, s := range []string{"exec-user", "oidc-user", "client-secret", "API_TOKEN", "/nonexistent/never-run"} {
		if !strings.Contains(res.Output, s) {
			t.Errorf("config view no longer shows %s:\n%s", s, res.Output)
		}
	}
}

func TestDebugTakesNamesOnlyAndNeverAFileTheWebOrStandardInput(t *testing.T) {
	f := newFake(t)
	pod := filepath.Join(t.TempDir(), "pod.yaml")
	if err := os.WriteFile(pod, []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\n  namespace: shop\nspec:\n  containers:\n  - name: web\n    image: nginx:1.27\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := write.New(write.Choice{Context: write.Context{Name: "stone", Server: f.URL, User: "fake-user"}, Level: write.Seamless, Classes: write.AllClasses})
	if err != nil {
		t.Fatal(err)
	}
	for _, wrap := range []func(*rest.Config) *rest.Config{live.ReadOnlyConfig, g.WrapConfig} {
		tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), wrap)
		for _, line := range []string{"debug -f " + pod + " --image=busybox -n shop", "debug -f - --image=busybox", "debug --filename=https://example.invalid/pod.yaml --image=busybox", "debug -f=http://example.invalid/p.yaml --image=busybox"} {
			before := len(f.seen.writes())
			res := run(t, tree, line)
			if res.Refused == "" || res.Code == 0 {
				t.Errorf("%s answered %d, refused %q:\n%s", line, res.Code, res.Refused, res.Output)
			}
			if got := f.seen.writes()[before:]; len(got) != 0 {
				t.Errorf("%s sent %v", line, got)
			}
		}
	}
}

func TestShellCompletionIsRefusedHoweverItIsReached(t *testing.T) {
	f := newFake(t)
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	for _, line := range []string{"__complete get ''", "'' __complete foo ''", "'' '' __completeNoDesc f", "-n shop __complete get ''", "--namespace=shop '' __complete foo", "kubectl '' __complete foo"} {
		if res := run(t, tree, line); res.Refused == "" || res.Code == 0 {
			t.Errorf("%s answered %d, refused %q:\n%s", line, res.Code, res.Refused, res.Output)
		}
	}
	if res := run(t, tree, "get pods -o name"); res.Code != 0 {
		t.Fatalf("a plain read was refused:\n%s", res.Output)
	}
}

func TestNoLocalReadCanWaitForeverAndCloseNeverWaitsForACommand(t *testing.T) {
	f := newFake(t)
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe.yaml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(t.TempDir(), "manifests")
	os.MkdirAll(nested, 0o755)
	syscall.Mkfifo(filepath.Join(nested, "inner.yaml"), 0o600)
	os.Symlink("/dev/stdin", filepath.Join(dir, "stdin.yaml"))
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	for _, line := range []string{
		"apply -f " + fifo, "apply -f /dev/stdin", "apply -f " + filepath.Join(dir, "stdin.yaml"), "apply -R -f " + filepath.Dir(nested),
		"create configmap c --from-file=k=" + fifo + " --dry-run=client", "create configmap c --from-file=" + fifo + " --dry-run=client",
		"create configmap c --from-env-file=" + fifo + " --dry-run=client", "create secret tls t --cert=" + fifo + " --key=" + fifo + " --dry-run=client",
	} {
		done := make(chan kubectltree.Result, 1)
		go func() { done <- run(t, tree, line) }()
		select {
		case res := <-done:
			if res.Refused == "" {
				t.Errorf("%s answered %d without a refusal:\n%s", line, res.Code, res.Output)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s waited on a local read", line)
		}
	}

	g := gateAt(t, write.Writes, f, kubeconfig(t, f.URL, "stone", "shop"))
	stuck := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), g.WrapConfig)
	asked := make(chan struct{})
	never := make(chan struct{})
	t.Cleanup(func() { close(never) })
	job := stuck.Start("delete pod web-1 --wait=false", func(context.Context, kubectltree.Held) bool {
		close(asked)
		<-never
		return false
	})
	<-asked
	closed := make(chan struct{})
	go func() {
		stuck.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close waited for a command that does not end")
	}
	_ = job
}
