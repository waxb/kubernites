package kubectltree_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/client/termview"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

type auditEvent struct {
	Stage      string `json:"stage"`
	Verb       string `json:"verb"`
	RequestURI string `json:"requestURI"`
	UserAgent  string `json:"userAgent"`
	Received   string `json:"requestReceivedTimestamp"`
}

type auditTrail struct {
	path string
	from int64
}

func (a *auditTrail) mark() {
	st, err := os.Stat(a.path)
	if err == nil {
		a.from = st.Size()
	}
}

func (a *auditTrail) since() []auditEvent {
	f, err := os.Open(a.path)
	if err != nil {
		return nil
	}
	defer f.Close()
	f.Seek(a.from, 0)
	var out []auditEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		var e auditEvent
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.Stage == "ResponseComplete" {
			out = append(out, e)
		}
	}
	return out
}

func (a *auditTrail) matching(verb, part string) []string {
	var out []string
	for _, e := range a.since() {
		if e.Verb == verb && strings.Contains(e.RequestURI, part) {
			out = append(out, e.Verb+" "+e.RequestURI)
		}
	}
	return out
}

type auditKind struct {
	reachKind
	trail *auditTrail
}

func newAuditKind(t *testing.T) auditKind {
	cfg, bin, log := os.Getenv("KUBERNITES_AUDIT_KUBECONFIG_A"), os.Getenv("KUBERNITES_AUDIT_KUBECTL"), os.Getenv("KUBERNITES_AUDIT_LOG_A")
	if cfg == "" || bin == "" || log == "" {
		t.Skip("KUBERNITES_AUDIT_KUBECONFIG_A, KUBERNITES_AUDIT_LOG_A and KUBERNITES_AUDIT_KUBECTL name a kind cluster this VM made with its API server's audit log, busybox:1.36 loaded")
	}
	raw, err := clientcmd.LoadFromFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	k := auditKind{reachKind: reachKind{parityKind: parityKind{t: t, config: cfg, context: raw.CurrentContext, bin: bin, home: t.TempDir()}}, trail: &auditTrail{path: log}}
	k.server = raw.Clusters[raw.Contexts[k.context].Cluster].Server
	k.user = raw.Contexts[k.context].AuthInfo
	return k
}

func (k auditKind) must(args ...string) string {
	k.t.Helper()
	out, errOut, code := k.binary(args...)
	if code != 0 {
		k.t.Fatalf("kubectl %v: %s", args, errOut)
	}
	return out
}

func (k auditKind) pods(manifest string, names ...string) {
	k.t.Helper()
	path := k.home + "/pods.yaml"
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		k.t.Fatal(err)
	}
	k.must("apply", "-f", path)
	for _, n := range names {
		k.must("wait", "-n", "audit3", "--for=condition=Ready", "pod/"+n, "--timeout=120s")
	}
}

const auditPods = `apiVersion: v1
kind: Namespace
metadata:
  name: audit3
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: deployer
  namespace: audit3
---
apiVersion: v1
kind: Pod
metadata:
  name: editor
  namespace: audit3
spec:
  terminationGracePeriodSeconds: 1
  containers:
  - name: editor
    image: busybox:1.36
    command: ["sleep", "3600"]
    stdin: true
    tty: true
---
apiVersion: v1
kind: Pod
metadata:
  name: talker
  namespace: audit3
spec:
  terminationGracePeriodSeconds: 1
  containers:
  - name: talker
    image: busybox:1.36
    command: ["nc", "-lk", "-p", "8080", "-e", "yes"]
    ports:
    - containerPort: 8080
`

func TestOnKindATokenRequestReachesTheServerOnlyAfterItsConfirm(t *testing.T) {
	k := newAuditKind(t)
	k.pods(auditPods)
	for _, yes := range []bool{false, true} {
		tree, gate := k.tree(write.Writes)
		k.trail.mark()
		var during []string
		confirms := 0
		tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
			return live.HoldStoneWrite(ctx, gate, k.server, h, func(a live.StoneAsk) bool {
				if a.Why == write.NeedsClass {
					return true
				}
				confirms++
				time.Sleep(time.Second)
				during = append(during, k.trail.matching("create", "/serviceaccounts/deployer/token")...)
				return yes
			})
		})
		res := run(t, tree, "create token deployer -n audit3 --duration=10m")
		time.Sleep(time.Second)
		after := k.trail.matching("create", "/serviceaccounts/deployer/token")
		t.Logf("CONFIRM %v: %d asks, code %d, token requests during the CONFIRM %v, after %v", yes, confirms, res.Code, during, after)
		if confirms != 1 || len(during) != 0 {
			t.Errorf("CONFIRM %v: %d CONFIRMs, and the API server saw %v before CONFIRM was answered", yes, confirms, during)
		}
		switch {
		case !yes && len(after) != 0:
			t.Errorf("with CONFIRM answered N the API server saw %v", after)
		case yes && (len(after) != 1 || strings.Contains(after[0], "dryRun")):
			t.Errorf("with CONFIRM answered Y the API server saw %v, not one token request", after)
		}
	}
}

func TestOnKindCtrlPCtrlQLeavesViAfterEscapeKeysWhileTheContainerReadsNoInput(t *testing.T) {
	k := newAuditKind(t)
	k.pods(auditPods, "editor")
	tree, _ := k.tree(write.Seamless, write.Workloads)
	var term *kubectltree.Terminal
	view := termview.New(100, 30, func(b []byte) { term.Type(b) })
	defer view.Close()
	term = kubectltree.NewTerminal(100, 30, view)
	job := tree.StartIn("exec -it editor -n audit3 -- vi /tmp/notes", nil, term)
	until(t, "the terminal is live", term.Live)
	text := func() string { return strings.Join(view.Text(), "\n") }
	for deadline := time.Now().Add(30 * time.Second); !strings.Contains(text(), "~"); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("vi never drew:\n%s", text())
		}
	}
	for _, keys := range [][]termview.Key{{{Name: "escape"}, {Rune: 'P'}}, {{Name: "escape"}, {Rune: ']'}}, {{Name: "escape"}, {Rune: 'O'}}} {
		for _, key := range keys {
			view.Key(key)
			time.Sleep(100 * time.Millisecond)
		}
	}
	view.Key(termview.Key{Name: "escape"})
	view.Type(":!sleep 600\r")
	time.Sleep(2 * time.Second)
	view.Type("these keys are never read\r")
	view.Key(termview.Key{Rune: 'p', Ctrl: true})
	view.Key(termview.Key{Rune: 'q', Ctrl: true})
	for deadline := time.Now().Add(20 * time.Second); !job.Done(); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("ctrl-p ctrl-q did not leave vi's shell escape within 20 s:\n%s", text())
		}
	}
	t.Logf("left: %+v", job.Result().Sessions)
}

func TestOnKindAStoppedForwardCarriesNothingMoreWhileThePodWritesWithoutEnd(t *testing.T) {
	k := newAuditKind(t)
	k.pods(auditPods, "talker")
	tree, _ := k.tree(write.Seamless, write.Workloads)
	res := run(t, tree, "port-forward talker -n audit3 :8080")
	fws := tree.Forwards()
	if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
		t.Fatalf("the forward did not open: %+v\n%s", fws, res.Output)
	}
	addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var got atomic.Int64
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		buf := make([]byte, 32<<10)
		for {
			n, err := conn.Read(buf)
			got.Add(int64(n))
			if err != nil {
				return
			}
		}
	}()
	for deadline := time.Now().Add(20 * time.Second); got.Load() < 1<<20; time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the talker sent only %d bytes", got.Load())
		}
	}
	tree.StopForward(fws[0].ID)
	time.Sleep(500 * time.Millisecond)
	before := got.Load()
	time.Sleep(2 * time.Second)
	after := got.Load()
	t.Logf("read %d bytes before STOP settled, %d in the 2 s after", before, after-before)
	if after > before {
		t.Errorf("after STOP the local client still read %d bytes from the pod", after-before)
	}
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Error("after STOP the local client's connection is still open")
	}
}

func TestOnKindATokenRequestCarryingDryRunIsRefusedAtEveryLevelAndNeverReachesTheServer(t *testing.T) {
	k := newAuditKind(t)
	k.pods(auditPods)
	request := k.home + "/tr.json"
	if err := os.WriteFile(request, []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{"expirationSeconds":600}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		tree, gate := k.tree(level, write.AllClasses...)
		tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
			return live.HoldStoneWrite(ctx, gate, k.server, h, func(live.StoneAsk) bool { return true })
		})
		k.trail.mark()
		res := run(t, tree, "create --raw /api/v1/namespaces/audit3/serviceaccounts/deployer/token?dryRun=All -f "+request)
		time.Sleep(2 * time.Second)
		seen := k.trail.matching("create", "/serviceaccounts/deployer/token")
		t.Logf("%s: code %d, the API server saw %v\n%s", level, res.Code, seen, res.Output)
		if len(seen) != 0 || res.Code == 0 || !strings.Contains(res.Output, write.TokenDryRun) {
			t.Errorf("%s: a token request carrying dryRun was not refused with its reason before the server: %v", level, seen)
		}
	}
}

const lairCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: dragons.lair.kubernites.example
spec:
  group: lair.kubernites.example
  scope: Namespaced
  names: {plural: dragons, singular: dragon, kind: Dragon}
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        x-kubernetes-preserve-unknown-fields: true
`

const lairDragon = `apiVersion: lair.kubernites.example/v1
kind: Dragon
metadata:
  name: d1
  namespace: audit3
`

const lairAggregated = `apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: v2.lair.kubernites.example
spec:
  group: lair.kubernites.example
  version: v2
  groupPriorityMinimum: 1000
  versionPriority: 5
  insecureSkipTLSVerify: true
  service:
    namespace: audit3
    name: lair-server
    port: 443
`

func TestOnKindAGroupAggregatedAfterConnectGetsNoDryRunAtWrites(t *testing.T) {
	k := newAuditKind(t)
	k.pods(auditPods)
	write3 := func(name, text string) string {
		p := k.home + "/" + name
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	k.must("apply", "-f", write3("crd.yaml", lairCRD))
	k.must("wait", "--for=condition=Established", "crd/dragons.lair.kubernites.example", "--timeout=60s")
	k.must("apply", "-f", write3("dragon.yaml", lairDragon))
	defer k.binary("delete", "apiservice", "v2.lair.kubernites.example", "--ignore-not-found")
	defer k.binary("delete", "crd", "dragons.lair.kubernites.example", "--ignore-not-found")
	tree, gate := k.tree(write.Writes, write.AllClasses...)
	if err := gate.LearnLocalGroups(context.Background()); err != nil {
		t.Fatal(err)
	}
	asked := 0
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, gate, k.server, h, func(a live.StoneAsk) bool { asked++; return false })
	})
	k.trail.mark()
	run(t, tree, "label dragon d1 -n audit3 seen=before")
	time.Sleep(2 * time.Second)
	if before := k.trail.matching("patch", "/apis/lair.kubernites.example/"); len(before) != 1 || !strings.Contains(before[0], "dryRun=All") || asked != 1 {
		t.Fatalf("while the CRD serves the group, its dry run should reach the server once before CONFIRM: %v (asked %d)", before, asked)
	}
	k.must("apply", "-f", write3("apiservice.yaml", lairAggregated))
	k.trail.mark()
	asked = 0
	res := run(t, tree, "label dragon d1 -n audit3 seen=after")
	time.Sleep(2 * time.Second)
	after := k.trail.matching("patch", "/apis/lair.kubernites.example/")
	t.Logf("after the APIService named a service: code %d, asked %d, the server saw %v\n%s", res.Code, asked, after, res.Output)
	if len(after) != 0 || asked != 0 || !strings.Contains(res.Output, "aggregated API server") {
		t.Errorf("at WRITES a group whose APIService names a service still got a dry run or a change: %v (asked %d)", after, asked)
	}
}
