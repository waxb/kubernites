package kubectltree_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

const fixManifest = `apiVersion: v1
kind: Namespace
metadata:
  name: fix
---
apiVersion: v1
kind: Secret
metadata:
  name: s
  namespace: fix
stringData:
  password: kind-hunter2
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shell
  namespace: fix
spec:
  replicas: 1
  selector:
    matchLabels:
      app: shell
  template:
    metadata:
      labels:
        app: shell
    spec:
      terminationGracePeriodSeconds: 1
      containers:
      - name: main
        image: busybox:1.36
        imagePullPolicy: Never
        command: ["sh", "-c", "mkdir -p /data && echo from-the-hut > /data/greeting.txt && echo two > /data/second.txt && exec sleep 3600"]
`

type fixKind struct {
	t      *testing.T
	a, b   string
	bin    string
	dir    string
	ctx    string
	worker string
}

func newFixKind(t *testing.T) fixKind {
	a, b, bin := os.Getenv("KUBERNITES_FIX_KUBECONFIG_A"), os.Getenv("KUBERNITES_FIX_KUBECONFIG_B"), os.Getenv("KUBERNITES_FIX_KUBECTL")
	if a == "" || b == "" || bin == "" {
		t.Skip("KUBERNITES_FIX_KUBECONFIG_A and _B name two kind clusters this VM made (A with a worker node, busybox:1.36 loaded in both) and KUBERNITES_FIX_KUBECTL a kubectl binary")
	}
	k := fixKind{t: t, a: a, b: b, bin: bin, dir: t.TempDir(), ctx: "fix"}
	k.kubectl(a, "apply", "-f", "-").stdin(fixManifest).must()
	k.kubectl(a, "rollout", "status", "deployment/shell", "-n", "fix", "--timeout=180s").must()
	nodes := k.kubectl(a, "get", "nodes", "-o", "name").must()
	for _, n := range strings.Fields(nodes) {
		if strings.HasSuffix(n, "-worker") {
			k.worker = strings.TrimPrefix(n, "node/")
		}
	}
	return k
}

type kubectlRun struct {
	k    fixKind
	args []string
	in   string
}

func (k fixKind) kubectl(config string, args ...string) *kubectlRun {
	return &kubectlRun{k: k, args: append([]string{"--kubeconfig", config}, args...)}
}

func (r *kubectlRun) stdin(s string) *kubectlRun {
	r.in = s
	return r
}

func (r *kubectlRun) out() (string, error) {
	cmd := exec.Command(r.k.bin, r.args...)
	cmd.Env = append(os.Environ(), "HOME="+r.k.dir, "KUBECONFIG=")
	cmd.Stdin = strings.NewReader(r.in)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *kubectlRun) must() string {
	r.k.t.Helper()
	out, err := r.out()
	if err != nil {
		r.k.t.Fatalf("kubectl %v: %v\n%s", r.args, err, out)
	}
	return out
}

func (k fixKind) pod() string {
	out := k.kubectl(k.a, "get", "pods", "-n", "fix", "-l", "app=shell", "--field-selector=status.phase=Running", "-o", "jsonpath={.items[0].metadata.name}").must()
	return strings.TrimSpace(out)
}

func (k fixKind) entry(config string) (*clientcmdapi.Cluster, *clientcmdapi.AuthInfo) {
	raw, err := clientcmd.LoadFromFile(config)
	if err != nil {
		k.t.Fatal(err)
	}
	c := raw.Contexts[raw.CurrentContext]
	return raw.Clusters[c.Cluster], raw.AuthInfos[c.AuthInfo]
}

func (k fixKind) write(path, config string, server string) {
	k.t.Helper()
	cluster, user := k.entry(config)
	cluster = cluster.DeepCopy()
	if server != "" {
		cluster.Server = server
	}
	cfg := clientcmdapi.NewConfig()
	cfg.Clusters["fix"] = cluster
	cfg.AuthInfos["fix"] = user
	cfg.Contexts[k.ctx] = &clientcmdapi.Context{Cluster: "fix", AuthInfo: "fix", Namespace: "fix"}
	cfg.CurrentContext = k.ctx
	if err := clientcmd.WriteToFile(*cfg, path); err != nil {
		k.t.Fatal(err)
	}
}

type tcpRecorder struct {
	ln    net.Listener
	conns atomic.Int32
}

func recordInFront(t *testing.T, server string) (*tcpRecorder, string) {
	host := strings.TrimPrefix(server, "https://")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	r := &tcpRecorder{ln: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			r.conns.Add(1)
			go func() {
				up, err := net.Dial("tcp", host)
				if err != nil {
					c.Close()
					return
				}
				go io.Copy(up, c)
				io.Copy(c, up)
				c.Close()
				up.Close()
			}()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return r, "https://" + ln.Addr().String()
}

func (k fixKind) realm(path string, level write.Level, classes ...write.Class) (*live.Realm, *write.Gate) {
	k.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if level == write.ReadOnly {
		r, err := live.Connect(ctx, []string{path}, k.ctx, time.Now(), nil)
		if err != nil {
			k.t.Fatal(err)
		}
		k.t.Cleanup(r.Stop)
		return r, nil
	}
	listed, err := live.ListContexts([]string{path})
	if err != nil {
		k.t.Fatal(err)
	}
	var choice write.Context
	for _, c := range listed {
		if c.Name == k.ctx {
			choice = c.Remembered()
		}
	}
	g, err := write.New(write.Choice{Context: choice, Level: level, Classes: classes})
	if err != nil {
		k.t.Fatal(err)
	}
	r, err := live.ConnectGated(ctx, []string{path}, k.ctx, g, time.Now(), nil)
	if err != nil {
		k.t.Fatal(err)
	}
	k.t.Cleanup(r.Stop)
	return r, g
}

func TestOnKindTheStoneNeverReachesAClusterAKubeconfigRewriteNamesAtAnyLevel(t *testing.T) {
	k := newFixKind(t)
	bCluster, _ := k.entry(k.b)
	pod := k.pod()
	for _, level := range []write.Level{write.ReadOnly, write.Writes, write.Seamless} {
		t.Run(level.String(), func(t *testing.T) {
			k.t = t
			path := filepath.Join(t.TempDir(), "kubeconfig")
			k.write(path, k.a, "")
			realm, g := k.realm(path, level, write.AllClasses...)
			tree := realm.Tree()
			if tree == nil {
				t.Fatal("no tree")
			}
			asks := 0
			if g != nil {
				tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
					return live.HoldStoneWrite(ctx, g, g.Context().Server, h, func(live.StoneAsk) bool { asks++; return true })
				})
			}
			rec, front := recordInFront(t, bCluster.Server)
			k.write(path, k.b, front)
			note := "rewritten-" + strings.ToLower(level.String()[:4])
			lines := []string{
				"get pods -n fix -o name",
				"annotate namespace fix note=" + note + " --overwrite",
				"exec " + pod + " -n fix -- cat /data/greeting.txt",
				"logs " + pod + " -n fix",
				"auth whoami",
			}
			outs := map[string]kubectltree.Result{}
			for _, line := range lines {
				outs[line] = run(t, tree, line)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			tree.Reveal(ctx, "fix", "s", "password")
			tree.Apply(ctx, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: from-the-scriptorium\n  namespace: fix\ndata:\n  level: "+note+"\n", false)
			if n := rec.conns.Load(); n != 0 {
				t.Fatalf("cluster B, which the rewritten kubeconfig names, got %d connections", n)
			}
			if got := outs["get pods -n fix -o name"]; got.Code != 0 || !strings.Contains(got.Output, pod) {
				t.Fatalf("the read after the rewrite did not reach cluster A:\n%s", got.Output)
			}
			annotated := k.kubectl(k.a, "get", "namespace", "fix", "-o", "jsonpath={.metadata.annotations.note}").must()
			onB, _ := k.kubectl(k.b, "get", "namespace", "fix", "-o", "jsonpath={.metadata.annotations.note}").out()
			if strings.Contains(onB, note) {
				t.Fatalf("cluster B holds the annotation")
			}
			switch level {
			case write.ReadOnly:
				if annotated == note {
					t.Fatal("READ ONLY wrote to cluster A")
				}
			default:
				if annotated != note {
					t.Fatalf("the write did not land on cluster A (annotation %q, asks %d):\n%s", annotated, asks, outs[lines[1]].Output)
				}
			}
			t.Logf("%s: B saw 0 connections; A annotation %q; asks %d", level, annotated, asks)
		})
	}
}

func TestOnKindNoSecretValueComesBackInAnyAnswer(t *testing.T) {
	k := newFixKind(t)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	k.write(path, k.a, "")
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		realm, g := k.realm(path, level, write.Secrets)
		tree := realm.Tree()
		tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
			return live.HoldStoneWrite(ctx, g, g.Context().Server, h, func(live.StoneAsk) bool { return true })
		})
		for _, line := range []string{
			"get secret s -n fix -o yaml",
			"get secrets -n fix -o json",
			`patch secret s -n fix --type=merge -p {"metadata":{"labels":{"seen":"yes"}}} --dry-run=server -o json`,
			`patch secret s -n fix --type=merge -p {"metadata":{"labels":{"seen":"yes"}}} -o yaml`,
			"label secret s -n fix again=yes --overwrite -o json",
			"create secret generic made-" + strings.ToLower(level.String()[:4]) + " -n fix --from-literal=password=kind-made-value -o yaml",
			"create secret generic dry -n fix --from-literal=password=kind-made-value --dry-run=server -o json",
			"create secret generic local -n fix --from-literal=password=kind-made-value --dry-run=client -o yaml",
			"describe secret s -n fix",
		} {
			res := run(t, tree, line)
			for _, v := range []string{"kind-hunter2", "a2luZC1odW50ZXIy", "kind-made-value", "a2luZC1tYWRlLXZhbHVl"} {
				if strings.Contains(res.Output, v) {
					t.Errorf("%s at %s shows %s:\n%s", line, level, v, res.Output)
				}
			}
		}
	}
}

func TestOnKindEachDrainConfirmReleasesExactlyTheEvictionItShows(t *testing.T) {
	k := newFixKind(t)
	if k.worker == "" {
		t.Skip("cluster A has no worker node")
	}
	k.kubectl(k.a, "create", "deployment", "two", "-n", "fix", "--image=busybox:1.36", "--replicas=2", "--", "sleep", "3600").out()
	k.kubectl(k.a, "patch", "deployment", "two", "-n", "fix", "--type=merge", "-p", `{"spec":{"template":{"spec":{"nodeName":"`+k.worker+`","terminationGracePeriodSeconds":1,"containers":[{"name":"busybox","image":"busybox:1.36","imagePullPolicy":"Never","command":["sleep","3600"]}]}}}}`).must()
	k.kubectl(k.a, "rollout", "status", "deployment/two", "-n", "fix", "--timeout=180s").must()
	t.Cleanup(func() {
		k.kubectl(k.a, "uncordon", k.worker).out()
		k.kubectl(k.a, "delete", "deployment", "two", "-n", "fix", "--wait=false").out()
	})
	path := filepath.Join(t.TempDir(), "kubeconfig")
	k.write(path, k.a, "")
	realm, g := k.realm(path, write.Writes)
	tree := realm.Tree()
	var mu sync.Mutex
	inflight, most := 0, 0
	var shown, sent []string
	g.Observe(func(p write.Passage) {
		if p.Write && strings.HasSuffix(p.Path, "/eviction") {
			mu.Lock()
			sent = append(sent, p.Path)
			mu.Unlock()
		}
	})
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, g.Context().Server, h, func(a live.StoneAsk) bool {
			mu.Lock()
			inflight++
			most = max(most, inflight)
			if a.Confirm() && strings.HasSuffix(a.Order.Request.Path, "/eviction") {
				shown = append(shown, a.Order.Request.Path)
			}
			mu.Unlock()
			time.Sleep(300 * time.Millisecond)
			mu.Lock()
			inflight--
			mu.Unlock()
			return true
		})
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res := tree.Run(ctx, "drain "+k.worker+" --ignore-daemonsets --delete-emptydir-data --timeout=150s")
	mu.Lock()
	defer mu.Unlock()
	t.Logf("drain answered %d; shown %v; sent %v\n%s", res.Code, shown, sent, res.Output)
	if most != 1 {
		t.Fatalf("%d CONFIRMs waited at once", most)
	}
	if len(shown) < 2 || !slices.Equal(shown, sent) {
		t.Fatalf("the CONFIRMs showed %v, the evictions sent were %v", shown, sent)
	}
}

func TestOnKindARefusedSendLeavesNoTicketBehind(t *testing.T) {
	k := newFixKind(t)
	k.kubectl(k.a, "run", "keep-me", "-n", "fix", "--image=busybox:1.36", "--image-pull-policy=Never", "--", "sleep", "3600").out()
	t.Cleanup(func() { k.kubectl(k.a, "delete", "pod", "keep-me", "-n", "fix", "--wait=false").out() })
	path := filepath.Join(t.TempDir(), "kubeconfig")
	k.write(path, k.a, "")
	realm, g := k.realm(path, write.Writes)
	var stale atomic.Bool
	g.StaleWhen(func() string {
		if stale.Load() {
			return "watch lost"
		}
		return ""
	})
	tree := realm.Tree()
	asks, answer := 0, true
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, g.Context().Server, h, func(a live.StoneAsk) bool {
			asks++
			if a.Confirm() && answer {
				stale.Store(true)
			}
			return answer
		})
	})
	first := run(t, tree, "delete pod keep-me -n fix --wait=false")
	stale.Store(false)
	before := asks
	answer = false
	second := run(t, tree, "delete pod keep-me -n fix --wait=false")
	if first.Code == 0 || asks == before || second.Code == 0 {
		t.Fatalf("first %d, asks %d then %d, second %d:\n%s\n%s", first.Code, before, asks, second.Code, first.Output, second.Output)
	}
	if out := k.kubectl(k.a, "get", "pod", "keep-me", "-n", "fix", "-o", "jsonpath={.metadata.deletionTimestamp}").must(); out != "" {
		t.Fatalf("the Pod was deleted: %s", out)
	}
}

func TestOnKindDebugWithAWebAddressIsRefusedAtEveryLevel(t *testing.T) {
	k := newFixKind(t)
	var hits atomic.Int32
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		fmt.Fprint(w, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n")
	}))
	t.Cleanup(web.Close)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	k.write(path, k.a, "")
	pod := k.pod()
	for _, level := range []write.Level{write.ReadOnly, write.Writes, write.Seamless} {
		realm, _ := k.realm(path, level, write.AllClasses...)
		for _, line := range []string{"debug -f " + web.URL + "/pod.yaml --image=busybox:1.36 -n fix", "debug " + pod + " -n fix -f " + web.URL + "/p.yaml --image=busybox:1.36", "debug -f - -n fix --image=busybox:1.36"} {
			res := run(t, realm.Tree(), line)
			if res.Refused == "" || res.Code == 0 {
				t.Errorf("%s at %s: %d %q\n%s", line, level, res.Code, res.Refused, res.Output)
			}
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the web server was fetched %d times", n)
	}
}

func TestOnKindCpLandsUnderDownloadsKubernitesAndNeverWritesTheKubeconfig(t *testing.T) {
	k := newFixKind(t)
	home := t.TempDir()
	kc := filepath.Join(home, ".kube", "config")
	os.MkdirAll(filepath.Dir(kc), 0o700)
	k.write(kc, k.a, "")
	before := read(t, kc)
	cluster, _ := k.entry(kc)
	g, err := write.New(write.Choice{Context: write.Context{Name: k.ctx, Server: cluster.Server, User: "fix"}, Level: write.Seamless, Classes: []write.Class{write.Workloads}})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(rc); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kc}, Context: k.ctx, Wrap: live.PinnedTo(rc, g.WrapConfig), Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	pod := k.pod()
	res := run(t, tree, "cp "+pod+":/data data -n fix")
	got := filepath.Join(home, "Downloads", "Kubernites", "data", "greeting.txt")
	if res.Code != 0 || read(t, got) != "from-the-hut\n" || !strings.Contains(res.Output, got) {
		t.Fatalf("cp into ~/Downloads/Kubernites answered %d:\n%s", res.Code, res.Output)
	}
	for _, dest := range []string{"~/.kube", "~/.kube/config", kc, "~/.kube/other"} {
		if res := run(t, tree, "cp "+pod+":/data/greeting.txt "+dest+" -n fix"); res.Code == 0 || res.Refused == "" {
			t.Errorf("cp into %s answered %d:\n%s", dest, res.Code, res.Output)
		}
	}
	if read(t, kc) != before {
		t.Fatal("cp changed the kubeconfig")
	}
	os.WriteFile(got, []byte("MINE"), 0o644)
	asked := 0
	tree.SetHold(func(context.Context, kubectltree.Held) bool { asked++; return false })
	res = run(t, tree, "cp "+pod+":/data data -n fix")
	if read(t, got) != "MINE" || asked != 1 || !strings.Contains(res.Output, "Kept "+got) {
		t.Fatalf("an existing file was overwritten or not asked about (asked %d):\n%s", asked, res.Output)
	}
	if read(t, filepath.Join(home, "Downloads", "Kubernites", "data", "second.txt")) != "two\n" {
		t.Fatalf("a file that did not exist was not copied:\n%s", res.Output)
	}
}
