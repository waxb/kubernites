package kubectltree_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

type reachKind struct {
	parityKind
	server string
	user   string
}

func newReachKind(t *testing.T) reachKind {
	cfg, bin := os.Getenv("KUBERNITES_REACH_KUBECONFIG"), os.Getenv("KUBERNITES_REACH_KUBECTL")
	if cfg == "" || bin == "" {
		t.Skip("KUBERNITES_REACH_KUBECONFIG and KUBERNITES_REACH_KUBECTL name a kind cluster this VM made (busybox:1.36 loaded) and the kubectl binary of the same version")
	}
	raw, err := clientcmd.LoadFromFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	k := reachKind{parityKind: parityKind{t: t, config: cfg, context: raw.CurrentContext, bin: bin, home: t.TempDir()}}
	k.server = raw.Clusters[raw.Contexts[k.context].Cluster].Server
	k.user = raw.Contexts[k.context].AuthInfo
	if _, errOut, code := k.binary("apply", "-f", filepath.Join("testdata", "reach", "reach.yaml")); code != 0 {
		t.Fatalf("apply: %s", errOut)
	}
	k.binary("delete", "pod", "-n", "reach", "tmp-1", "shell-copy", "--ignore-not-found", "--wait=true")
	if names, _, _ := k.binary("get", "pods", "-n", "reach", "-o", "name"); strings.Contains(names, "node-debugger") {
		for _, n := range strings.Fields(names) {
			if strings.Contains(n, "node-debugger") {
				k.binary("delete", n, "-n", "reach", "--wait=true")
			}
		}
	}
	if _, errOut, code := k.binary("rollout", "status", "deployment/shell", "-n", "reach", "--timeout=120s"); code != 0 {
		t.Fatalf("rollout: %s", errOut)
	}
	for _, target := range [][]string{{"-l", "app=shell"}, {"tty-1"}} {
		if _, errOut, code := k.binary(append([]string{"wait", "-n", "reach", "--for=condition=Ready", "pod", "--timeout=120s"}, target...)...); code != 0 {
			t.Fatalf("wait: %s", errOut)
		}
	}
	return k
}

func (k reachKind) pod(label string) string {
	out, _, _ := k.binary("get", "pods", "-n", "reach", "-l", label, "-o", "jsonpath={range .items[*]}{.metadata.name} {.metadata.deletionTimestamp}{\"\\n\"}{end}")
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) == 1 {
			return f[0]
		}
	}
	return ""
}

func (k reachKind) tree(level write.Level, passed ...write.Class) (*kubectltree.Tree, *write.Gate) {
	k.t.Helper()
	raw, err := clientcmd.LoadFromFile(k.config)
	if err != nil {
		k.t.Fatal(err)
	}
	gate, err := write.New(write.Choice{Context: write.Context{Name: k.context, Server: k.server, User: k.user}, Level: level, Classes: passed})
	if err != nil {
		k.t.Fatal(err)
	}
	rc, err := clientcmd.NewNonInteractiveClientConfig(*raw, k.context, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		k.t.Fatal(err)
	}
	if _, err := gate.Client(rc); err != nil {
		k.t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{k.config}, Context: k.context, Wrap: gate.WrapConfig})
	if err != nil {
		k.t.Fatal(err)
	}
	k.t.Cleanup(func() { tree.Close() })
	return tree, gate
}

type liveShell struct {
	t    *testing.T
	out  *screen
	term *kubectltree.Terminal
	job  *kubectltree.Job
}

func open(t *testing.T, tree *kubectltree.Tree, line string, cols, rows int) liveShell {
	t.Helper()
	s := liveShell{t: t, out: &screen{}}
	s.term = kubectltree.NewTerminal(cols, rows, s.out)
	s.job = tree.StartIn(line, nil, s.term)
	return s
}

func (s liveShell) wait(what string, ok func(string) bool) {
	s.t.Helper()
	for deadline := time.Now().Add(90 * time.Second); !ok(s.out.String()); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) || s.job.Done() && !ok(s.out.String()) {
			chunks, _ := s.job.Take()
			s.t.Fatalf("never: %s\nscreen:\n%q\nstone: %v\nresult: %+v", what, s.out.String(), chunks, s.job.Result())
		}
	}
}

func (s liveShell) says(text string) {
	s.t.Helper()
	s.wait("the screen says "+text, func(out string) bool { return strings.Contains(out, text) })
}

func (s liveShell) typed(keys string) {
	s.term.Type([]byte(keys))
}

func (s liveShell) end() kubectltree.Result {
	s.t.Helper()
	until(s.t, "the session ended", s.job.Done)
	return s.job.Result()
}

func TestTheStonesTerminalOnKind(t *testing.T) {
	k := newReachKind(t)
	tree, _ := k.tree(write.Seamless, write.Workloads)
	pod := k.pod("app=shell")

	sh := open(t, tree, "exec -it "+pod+" -n reach -- sh", 100, 30)
	sh.says("# ")
	sh.typed("echo $((6*7))\r")
	sh.says("42")
	sh.typed("stty size\r")
	sh.says("30 100")
	sh.term.Resize(120, 40)
	time.Sleep(300 * time.Millisecond)
	sh.typed("stty size\r")
	sh.says("40 120")
	sh.typed("top\r")
	sh.says("Mem:")
	sh.says("PID")
	sh.typed("q")
	sh.typed("vi /tmp/note\r")
	sh.says("\x1b[")
	time.Sleep(500 * time.Millisecond)
	sh.typed("ihello from vi")
	sh.says("hello from vi")
	sh.typed("\x1b")
	time.Sleep(300 * time.Millisecond)
	sh.typed(":wq\r")
	sh.says("\x1b[?1049l")
	sh.typed("cat /tmp/note\r")
	sh.says("hello from vi\r\n")
	sh.typed("exit 7\r")
	res := sh.end()
	if len(res.Sessions) != 1 || res.Sessions[0].Ended != kubectltree.EndedExited || res.Sessions[0].Code != 7 || !res.Sessions[0].TTY || res.Sessions[0].Container != "main" {
		t.Fatalf("the shell's session %+v\n%s", res.Sessions, res.Output)
	}
	t.Logf("exec -it: a shell, stty size 30 100 then 40 120 after the resize, top, vi; exit 7 ended it with code %d", res.Sessions[0].Code)

	at := open(t, tree, "attach -it tty-1 -n reach", 80, 24)
	until(t, "the attach is live", at.term.Live)
	at.typed("\r")
	at.says("# ")
	at.typed("echo attached-$((40+2))\r")
	at.says("attached-42")
	at.typed("\x10\x11")
	res = at.end()
	if len(res.Sessions) != 1 || res.Sessions[0].Ended != kubectltree.EndedLeft || res.Sessions[0].Code != 0 {
		t.Fatalf("attach's session %+v\n%s", res.Sessions, res.Output)
	}
	if phase, _, _ := k.binary("get", "pod", "tty-1", "-n", "reach", "-o", "jsonpath={.status.phase}"); phase != "Running" {
		t.Fatalf("leaving the attach stopped the container: %s", phase)
	}
	t.Logf("attach -it: echoed, left with ctrl-p ctrl-q, tty-1 still Running")

	helper := fmt.Sprintf("helper-%d", time.Now().Unix()%100000)
	dbg := open(t, tree, "debug -it "+pod+" -n reach --image=busybox:1.36 --image-pull-policy=Never --target=main -c "+helper+" -- sh", 80, 24)
	dbg.says("# ")
	dbg.typed("cat /proc/1/cmdline | tr '\\0' ' '; echo\r")
	dbg.says("sleep 3600")
	dbg.typed("exit\r")
	res = dbg.end()
	if len(res.Sessions) != 1 || res.Sessions[0].Ended != kubectltree.EndedExited || res.Sessions[0].Container != helper || res.Sessions[0].Channel != "attach" {
		t.Fatalf("debug's session %+v\n%s", res.Sessions, res.Output)
	}
	if names, _, _ := k.binary("get", "pod", pod, "-n", "reach", "-o", "jsonpath={.spec.ephemeralContainers[*].name}"); !strings.Contains(names, helper) {
		t.Fatalf("no ephemeral container: %q", names)
	}
	t.Logf("debug -it: an ephemeral container %s saw the target's sleep 3600", helper)

	node := open(t, tree, "debug node/kubernites-worker -it -n reach --image=busybox:1.36 --image-pull-policy=Never -- sh", 80, 24)
	node.says("# ")
	node.typed("cat /host/etc/hostname\r")
	node.says("kubernites-worker")
	node.typed("exit\r")
	res = node.end()
	if len(res.Sessions) != 1 || res.Sessions[0].Ended != kubectltree.EndedExited || !strings.HasPrefix(res.Sessions[0].Pod, "node-debugger-kubernites-worker-") {
		t.Fatalf("node debugging's session %+v\n%s", res.Sessions, res.Output)
	}
	k.binary("delete", "pod", res.Sessions[0].Pod, "-n", "reach", "--wait=false")
	t.Logf("debug node: %s read the Node's /etc/hostname", res.Sessions[0].Pod)

	cp := open(t, tree, "debug "+pod+" -it -n reach --copy-to=shell-copy --image=busybox:1.36 --image-pull-policy=Never -c copied -- sh", 80, 24)
	cp.says("# ")
	cp.typed("hostname\r")
	cp.says("shell-copy")
	cp.typed("exit\r")
	res = cp.end()
	if len(res.Sessions) != 1 || res.Sessions[0].Pod != "shell-copy" || res.Sessions[0].Ended != kubectltree.EndedExited {
		t.Fatalf("--copy-to's session %+v\n%s", res.Sessions, res.Output)
	}
	k.binary("delete", "pod", "shell-copy", "-n", "reach", "--wait=false")
	t.Logf("debug --copy-to: shell-copy answered its own hostname")

	runIt := open(t, tree, "run tmp-1 -it --rm --restart=Never -n reach --image=busybox:1.36 --image-pull-policy=Never -- sh", 80, 24)
	until(t, "run's attach is live", runIt.term.Live)
	runIt.typed("\r")
	runIt.says("# ")
	runIt.typed("echo ran-$((1+1))\r")
	runIt.says("ran-2")
	runIt.typed("exit\r")
	res = runIt.end()
	if len(res.Sessions) != 1 || res.Sessions[0].Ended != kubectltree.EndedExited || !strings.Contains(res.Output, `pod "tmp-1" deleted`) {
		t.Fatalf("run -it's session %+v\n%s", res.Sessions, res.Output)
	}
	t.Logf("run -it --rm: tmp-1 ran a shell and was deleted:\n%s", res.Output)
}

func TestTheTerminalAtWritesConfirmsOnceOnKind(t *testing.T) {
	k := newReachKind(t)
	tree, gate := k.tree(write.Writes)
	var asks []string
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return holdAll(ctx, gate, k.server, h, func(why string) { asks = append(asks, why) })
	})
	pod := k.pod("app=shell")
	sh := open(t, tree, "exec -it "+pod+" -n reach -- sh", 80, 24)
	sh.says("# ")
	sh.typed("exit\r")
	sh.end()
	want := "class workloads | connect exec pod reach/" + pod + " container main command sh"
	if strings.Join(asks, " | ") != want {
		t.Fatalf("the Stone asked %q, want %q", asks, want)
	}
	t.Logf("at WRITES, exec: %s", strings.Join(asks, " | "))
	asks = nil
	helper := fmt.Sprintf("confirmed-%d", time.Now().Unix()%100000)
	dbg := open(t, tree, "debug -it "+pod+" -n reach --image=busybox:1.36 --image-pull-policy=Never -c "+helper+" -- sh", 80, 24)
	dbg.says("# ")
	dbg.typed("exit\r")
	dbg.end()
	want = "write PATCH /api/v1/namespaces/reach/pods/" + pod + "/ephemeralcontainers previewed | connect attach pod reach/" + pod + " container " + helper + " command "
	if strings.Join(asks, " | ") != want {
		t.Fatalf("debug asked %q, want %q", asks, want)
	}
	t.Logf("at WRITES, debug: %s", strings.Join(asks, " | "))
}

func TestReadOnlyRBACRefusesEachDoorWithTheServersReason(t *testing.T) {
	k := newReachKind(t)
	token, errOut, code := k.binary("create", "token", "looker", "-n", "reach", "--duration=1h")
	if code != 0 {
		t.Fatal(errOut)
	}
	ca, _, _ := k.binary("config", "view", "--raw", "--minify", "-o", "jsonpath={.clusters[0].cluster.certificate-authority-data}")
	cfg := filepath.Join(t.TempDir(), "looker")
	body := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: looker\nclusters:\n- name: kind\n  cluster:\n    server: %s\n    certificate-authority-data: %s\ncontexts:\n- name: looker\n  context:\n    cluster: kind\n    user: looker\n    namespace: reach\nusers:\n- name: looker\n  user:\n    token: %s\n", k.server, strings.TrimSpace(ca), strings.TrimSpace(token))
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	looker := k
	looker.config, looker.context, looker.user = cfg, "looker", "looker"
	tree, _ := looker.tree(write.Seamless, write.Workloads)
	pod := k.pod("app=shell")
	for _, line := range []string{
		"exec -it " + pod + " -- sh",
		"attach -it tty-1",
		"debug -it " + pod + " --image=busybox:1.36 --image-pull-policy=Never -- sh",
		"get --raw /api/v1/namespaces/reach/pods/" + pod + "/proxy/",
	} {
		s := open(t, tree, line, 80, 24)
		res := s.end()
		if res.Code == 0 || !strings.Contains(res.Output, "forbidden") || !strings.Contains(res.Output, "system:serviceaccount:reach:looker") {
			t.Fatalf("%s at read-only RBAC answered %d:\n%s", line, res.Code, res.Output)
		}
		t.Logf("%s: %s", line, strings.TrimSpace(res.Output))
	}
}

func TestAtReadOnlyNoDoorLeavesOnKind(t *testing.T) {
	k := newReachKind(t)
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{k.config}, Context: k.context, Wrap: live.ReadOnlyConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	pod := k.pod("app=shell")
	before, _, _ := k.binary("get", "pods", "-A", "-o", "name")
	for _, line := range []string{
		"exec -it " + pod + " -n reach -- sh",
		"exec " + pod + " -n reach -- touch /tmp/read-only-was-here",
		"attach -it tty-1 -n reach",
		"debug -it " + pod + " -n reach --image=busybox:1.36 --image-pull-policy=Never -c sneaky -- sh",
		"debug node/kubernites-worker -it -n reach --image=busybox:1.36 --image-pull-policy=Never -- sh",
		"run sneaky -it --rm --restart=Never -n reach --image=busybox:1.36 --image-pull-policy=Never -- sh",
		"get --raw /api/v1/namespaces/reach/pods/" + pod + "/proxy/",
	} {
		s := open(t, tree, line, 80, 24)
		res := s.end()
		if res.Code == 0 || !strings.Contains(res.Output, "read-only") || len(res.Sessions) > 0 && res.Sessions[0].Opened {
			t.Fatalf("%s at READ ONLY answered %d with %+v:\n%s", line, res.Code, res.Sessions, res.Output)
		}
	}
	if out, _, _ := k.binary("exec", pod, "-n", "reach", "--", "ls", "/tmp"); strings.Contains(out, "read-only-was-here") {
		t.Fatal("an exec ran at READ ONLY")
	}
	if names, _, _ := k.binary("get", "pod", pod, "-n", "reach", "-o", "jsonpath={.spec.ephemeralContainers[*].name}"); strings.Contains(names, "sneaky") {
		t.Fatal("an ephemeral container was added at READ ONLY")
	}
	if after, _, _ := k.binary("get", "pods", "-A", "-o", "name"); after != before {
		t.Fatalf("Pods changed at READ ONLY:\n%s\nthen\n%s", before, after)
	}
}

func TestPortForwardOnKindAnswersOn127001AndNothingElse(t *testing.T) {
	k := newReachKind(t)
	tree, _ := k.tree(write.Seamless, write.Workloads)
	pod := k.pod("app=shell")
	for _, line := range []string{"port-forward " + pod + " -n reach :8080", "port-forward deployment/shell -n reach --address 0.0.0.0 :8080"} {
		res := run(t, tree, line)
		fws := tree.Forwards()
		if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
			t.Fatalf("%s answered %d, forwards %+v:\n%s", line, res.Code, fws, res.Output)
		}
		addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
		if !strings.HasPrefix(addr, "127.0.0.1:") {
			t.Fatalf("%s listens on %s", line, addr)
		}
		_, port, _ := strings.Cut(addr, ":")
		got := httpGet(t, "http://127.0.0.1:"+port+"/greeting.txt")
		if strings.TrimSpace(got) != "from-the-hut" {
			t.Fatalf("%s answered %q through 127.0.0.1", line, got)
		}
		others := otherAddresses(t)
		for _, host := range others {
			refused(t, host+":"+port)
		}
		tree.StopForward(fws[0].ID)
		refused(t, "127.0.0.1:"+port)
		t.Logf("%s: 127.0.0.1:%s answered %q; refused on %v; %s", line, port, strings.TrimSpace(got), others, strings.TrimSpace(res.Output))
	}
}

func httpGet(t *testing.T, u string) string {
	t.Helper()
	c := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := c.Get(u)
	if err != nil {
		t.Fatalf("GET %s: %v", u, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data)
}

func TestCpBothWaysOnKind(t *testing.T) {
	k := newReachKind(t)
	raw, err := clientcmd.LoadFromFile(k.config)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := write.New(write.Choice{Context: write.Context{Name: k.context, Server: k.server, User: k.user}, Level: write.Seamless, Classes: []write.Class{write.Workloads}})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := clientcmd.NewNonInteractiveClientConfig(*raw, k.context, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Client(rc); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{k.config}, Context: k.context, Wrap: gate.WrapConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	pod := k.pod("app=shell")
	work := tree.LocalDir()
	res := run(t, tree, "cp "+pod+":/data/greeting.txt greeting.txt -n reach")
	if res.Code != 0 || read(t, filepath.Join(work, "greeting.txt")) != "from-the-hut\n" || !strings.Contains(res.Output, "Wrote "+filepath.Join(work, "greeting.txt")) {
		t.Fatalf("cp from the Pod answered %d:\n%s", res.Code, res.Output)
	}
	t.Logf("from the Pod:\n%s", strings.TrimSpace(res.Output))
	res = run(t, tree, "cp reach/"+pod+":/data data-copy")
	if res.Code != 0 || read(t, filepath.Join(work, "data-copy", "greeting.txt")) != "from-the-hut\n" {
		t.Fatalf("cp of a directory answered %d:\n%s", res.Code, res.Output)
	}
	t.Logf("a directory from the Pod:\n%s", strings.TrimSpace(res.Output))
	os.MkdirAll(filepath.Join(work, "up", "inner"), 0o755)
	os.WriteFile(filepath.Join(work, "up", "inner", "note.txt"), []byte("to the hut\n"), 0o644)
	res = run(t, tree, "cp up "+pod+":/tmp -n reach")
	if res.Code != 0 || !strings.Contains(res.Output, "Read 1 file under "+filepath.Join(work, "up")) {
		t.Fatalf("cp to the Pod answered %d:\n%s", res.Code, res.Output)
	}
	if out, errOut, _ := k.binary("exec", pod, "-n", "reach", "--", "cat", "/tmp/up/inner/note.txt"); out != "to the hut\n" {
		t.Fatalf("the Pod holds %q (%s)", out, errOut)
	}
	t.Logf("to the Pod:\n%s", strings.TrimSpace(res.Output))
}

func TestRecordTheReachScenesOnKind(t *testing.T) {
	dir := os.Getenv("KUBERNITES_REACH_RECORD")
	if dir == "" {
		t.Skip("KUBERNITES_REACH_RECORD names the directory for the reach scenes' fixtures, recorded on a kind cluster this VM made")
	}
	k := newReachKind(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	realm, err := live.Connect(ctx, []string{k.config}, k.context, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := json.MarshalIndent(realm.Snapshot().WithoutProxyEnvironment(), "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reach-kind-snapshot.json"), snap, 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &livetest.Recording{Context: k.context, Sessions: map[string]string{}}
	gate, err := write.New(write.Choice{Context: write.Context{Name: k.context, Server: k.server, User: k.user}, Level: write.Seamless, Classes: []write.Class{write.Workloads}})
	if err != nil {
		t.Fatal(err)
	}
	wrap := func(c *rest.Config) *rest.Config {
		c = gate.WrapConfig(c)
		c.AcceptContentTypes, c.ContentType = "application/json", "application/json"
		c.Wrap(func(rt http.RoundTripper) http.RoundTripper { return &livetest.Recorder{Next: rt, Rec: rec} })
		return c
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{k.config}, Context: k.context, Wrap: wrap, Local: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	pod := k.pod("app=shell")
	sh := open(t, tree, "exec -it "+pod+" -n reach -- sh", 154, 10)
	sh.says("# ")
	sh.typed("ls -l /data\r")
	sh.says("greeting.txt")
	sh.typed("echo the hut answers $((6*7))\r")
	sh.says("the hut answers 42")
	time.Sleep(300 * time.Millisecond)
	rec.Sessions["sh"] = sh.out.String()
	sh.typed("exit\r")
	sh.end()
	top := open(t, tree, "exec -it "+pod+" -n reach -- top", 154, 27)
	top.says("PID")
	time.Sleep(1500 * time.Millisecond)
	rec.Sessions["top"] = top.out.String()
	top.typed("q")
	top.end()
	tr, _ := kubectltree.New(kubectltree.Config{Kubeconfig: []string{k.config}, Context: k.context, Wrap: wrap, Local: t.TempDir()})
	defer tr.Close()
	if res := run(t, tr, "cp "+pod+":/data/greeting.txt greeting.txt -n reach"); res.Code != 0 {
		t.Fatalf("cp: %s", res.Output)
	}
	raw, _, _ := k.binary("exec", pod, "-n", "reach", "--", "tar", "cf", "-", "/data/greeting.txt")
	rec.Sessions["tar"] = raw
	data, err := json.MarshalIndent(rec, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reach-kind-replay.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("recorded %d exchanges and the sessions %d, %d and %d bytes for %s", len(rec.Exchanges), len(rec.Sessions["sh"]), len(rec.Sessions["top"]), len(rec.Sessions["tar"]), pod)
}
