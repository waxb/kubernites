package app

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/view"
)

type auditEvent struct {
	Stage      string    `json:"stage"`
	Verb       string    `json:"verb"`
	RequestURI string    `json:"requestURI"`
	UserAgent  string    `json:"userAgent"`
	Received   time.Time `json:"requestReceivedTimestamp"`
}

func auditEvents(t *testing.T, path string) []auditEvent {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
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

type auditClusters struct {
	t                *testing.T
	a, b             string
	logA, logB       string
	kubectl          string
	worker, drainsNS string
}

func newAuditClusters(t *testing.T) *auditClusters {
	k := &auditClusters{t: t, a: os.Getenv("KUBERNITES_AUDIT_KUBECONFIG_A"), b: os.Getenv("KUBERNITES_AUDIT_KUBECONFIG_B"),
		logA: os.Getenv("KUBERNITES_AUDIT_LOG_A"), logB: os.Getenv("KUBERNITES_AUDIT_LOG_B"), kubectl: os.Getenv("KUBERNITES_AUDIT_KUBECTL"), worker: os.Getenv("KUBERNITES_AUDIT_WORKER")}
	if k.a == "" || k.b == "" || k.logA == "" || k.logB == "" || k.kubectl == "" || k.worker == "" {
		t.Skip("KUBERNITES_AUDIT_KUBECONFIG_A/_B, KUBERNITES_AUDIT_LOG_A/_B, KUBERNITES_AUDIT_WORKER and KUBERNITES_AUDIT_KUBECTL name two kind clusters this VM made with their API servers' audit logs, busybox:1.36 loaded")
	}
	return k
}

func (k *auditClusters) kubectlOn(kubeconfig string, args ...string) string {
	k.t.Helper()
	out, err := exec.Command(k.kubectl, append([]string{"--kubeconfig", kubeconfig}, args...)...).CombinedOutput()
	if err != nil {
		k.t.Fatalf("kubectl %v: %v\n%s", args, err, out)
	}
	return string(out)
}

const heldDrain = `apiVersion: v1
kind: Namespace
metadata:
  name: held
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: held
  namespace: held
spec:
  replicas: 2
  selector:
    matchLabels: {app: held}
  template:
    metadata:
      labels: {app: held}
    spec:
      nodeSelector: {kubernetes.io/hostname: WORKER}
      terminationGracePeriodSeconds: 1
      containers:
      - name: held
        image: busybox:1.36
        command: ["sleep", "3600"]
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: held
  namespace: held
spec:
  minAvailable: 2
  selector:
    matchLabels: {app: held}
`

func evictionsAfter(events []auditEvent, at time.Time) []string {
	var out []string
	for _, e := range events {
		if e.Verb == "create" && strings.HasSuffix(e.RequestURI, "/eviction") && e.Received.After(at) {
			out = append(out, e.Received.Format(time.RFC3339Nano)+" "+e.RequestURI)
		}
	}
	return out
}

func TestOnKindLeavingTheRealmStopsADrainABudgetBlocksAndNoEvictionFollows(t *testing.T) {
	k := newAuditClusters(t)
	path := filepath.Join(t.TempDir(), "held.yaml")
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(heldDrain, "WORKER", k.worker)), 0o600); err != nil {
		t.Fatal(err)
	}
	k.kubectlOn(k.a, "apply", "-f", path)
	k.kubectlOn(k.a, "rollout", "status", "deployment/held", "-n", "held", "--timeout=120s")
	defer k.kubectlOn(k.a, "uncordon", k.worker)
	for _, level := range []string{hud.LevelWrites, hud.LevelSeamless} {
		rig := connectFile(t, k.a, level, write.AllClasses...)
		rig.s.Writer.sync = false
		rig.s.Writer.every = time.Second
		var keep view.Structure
		rig.waitFor(t, "the worker Keep", func() bool {
			for _, st := range rig.s.Site.Structures {
				if st.Kind == view.StructKeep && st.Object.Name == k.worker {
					keep = st
					return true
				}
			}
			return false
		})
		rig.select_(hud.Selection{Kind: "structure", ID: keep.ID, Structure: keep})
		rig.waitFor(t, "DRAIN offered", func() bool {
			for _, o := range rig.s.Hud.LiveOrders() {
				if o.Key == "drain" {
					return !o.Disabled
				}
			}
			return false
		})
		rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: "drain"})
		rig.waitFor(t, "the drain to wait on the budget", func() bool {
			if form, open := rig.s.Hud.OrderOpen(); open && form.Go != "" && !form.Locked || open && strings.HasPrefix(form.Go, "CONFIRM") {
				rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: rig.s.Hud.OrderValues()})
			}
			return rig.s.Writer.drain != nil && strings.Contains(rig.s.Writer.drain.blocked, "held/held-")
		})
		time.Sleep(2500 * time.Millisecond)
		left := rig.s
		ledger := left.Writer.ledger.Path()
		if err := rig.g.RealmIntent(hud.Intent{Kind: hud.IntentTitle}); err != nil {
			t.Fatal(err)
		}
		leftAt := time.Now()
		before := evictionsAfter(auditEvents(t, k.logA), leftAt.Add(-time.Hour))
		other := connectFile(t, k.b, hud.LevelReadOnly)
		time.Sleep(6 * time.Second)
		after := evictionsAfter(auditEvents(t, k.logA), leftAt)
		data, _ := os.ReadFile(ledger)
		t.Logf("%s: %d evictions reached the API server before leaving, %d after; ledger:\n%s", level, len(before), len(after), data)
		if len(after) > 0 {
			t.Errorf("%s: after leaving the realm the API server saw evictions: %v", level, after)
		}
		if dr := left.Writer.drain; dr == nil || !dr.over {
			t.Errorf("%s: the left session's drain still runs", level)
		}
		if lines := strings.Count(string(data), `"order":"evict"`); lines == 0 {
			t.Errorf("%s: the ledger has no eviction line", level)
		}
		if other.s.Writer != nil {
			t.Errorf("%s: the next connection opened with a writer", level)
		}
		rig.g.RealmIntent(hud.Intent{Kind: hud.IntentTitle})
		again := connectFile(t, k.a, level, write.AllClasses...)
		if again.s.Writer == nil || again.s.Writer.drain != nil {
			t.Errorf("%s: reconnecting finds a drain running: %+v", level, again.s.Writer)
		}
		again.g.RealmIntent(hud.Intent{Kind: hud.IntentTitle})
		k.kubectlOn(k.a, "uncordon", k.worker)
	}
}

func TestOnKindAKubeconfigRewrittenBeforeConnectReachesNeitherCluster(t *testing.T) {
	k := newAuditClusters(t)
	rawA, err := clientcmd.LoadFromFile(k.a)
	if err != nil {
		t.Fatal(err)
	}
	rawB, err := clientcmd.LoadFromFile(k.b)
	if err != nil {
		t.Fatal(err)
	}
	ca, cb := rawA.Contexts[rawA.CurrentContext], rawB.Contexts[rawB.CurrentContext]
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := clientcmd.WriteToFile(*rawA, path); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: dir, Live: LiveSource{Files: []string{path}}}, nil, nil)
	listed, err := live.ListContexts([]string{path})
	if err != nil || len(listed) != 1 {
		t.Fatalf("%v %v", listed, err)
	}
	g.choices.Keep(write.Remembered{Context: listed[0].Remembered(), Level: write.Seamless, Warned: write.Seamless, Classes: write.AllClasses})
	g.saveChoices()
	rig := &liveRig{g: g, dir: dir}
	rig.title(hud.Intent{Kind: hud.IntentLive})
	rig.title(hud.Intent{Kind: hud.IntentLiveList})
	rig.press(t, "live:row:0")
	rig.press(t, "live:choose")
	rig.press(t, hud.LiveBoxID)
	if got := g.Picker.ChosenLevel(); got != hud.LevelSeamless {
		t.Fatalf("the remembered level is %s", got)
	}
	rewritten := rawA.DeepCopy()
	rewritten.Clusters[ca.Cluster].CertificateAuthorityData = rawB.Clusters[cb.Cluster].CertificateAuthorityData
	rewritten.AuthInfos[ca.AuthInfo] = rawB.AuthInfos[cb.AuthInfo].DeepCopy()
	if err := clientcmd.WriteToFile(*rewritten, path); err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	rig.press(t, "live:confirm")
	for deadline := time.Now().Add(60 * time.Second); g.Opening != nil; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the opening never ended")
		}
		g.StepLive()
	}
	time.Sleep(2 * time.Second)
	if g.Scene == SceneRealm {
		t.Fatal("the rewritten kubeconfig connected")
	}
	t.Logf("the picker says: %s", g.Picker.Err)
	if !strings.Contains(g.Picker.Err, "credentials") || !strings.Contains(g.Picker.Err, "certificate authority") || !strings.Contains(g.Picker.Err, "READ ONLY") {
		t.Errorf("the picker does not say what changed: %q", g.Picker.Err)
	}
	for name, log := range map[string]string{"the first cluster": k.logA, "the second cluster": k.logB} {
		for _, e := range auditEvents(t, log) {
			if e.Received.After(at) && strings.Contains(strings.ToLower(e.UserAgent), "kubernites") {
				t.Errorf("%s's API server saw %s %s from Kubernites after CONNECT", name, e.Verb, e.RequestURI)
			}
		}
	}
}

func TestOnKindADrainRecordsItsCordonAndHowItEnded(t *testing.T) {
	k := newAuditClusters(t)
	path := filepath.Join(t.TempDir(), "held.yaml")
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(heldDrain, "WORKER", k.worker)), 0o600); err != nil {
		t.Fatal(err)
	}
	k.kubectlOn(k.a, "apply", "-f", path)
	k.kubectlOn(k.a, "rollout", "status", "deployment/held", "-n", "held", "--timeout=120s")
	defer k.kubectlOn(k.a, "uncordon", k.worker)
	for _, level := range []string{hud.LevelWrites, hud.LevelSeamless} {
		k.kubectlOn(k.a, "uncordon", k.worker)
		rig := connectFile(t, k.a, level, write.AllClasses...)
		rig.s.Writer.sync = false
		rig.s.Writer.every = time.Second
		var keep view.Structure
		rig.waitFor(t, "the worker Keep", func() bool {
			for _, st := range rig.s.Site.Structures {
				if st.Kind == view.StructKeep && st.Object.Name == k.worker {
					keep = st
					return true
				}
			}
			return false
		})
		rig.select_(hud.Selection{Kind: "structure", ID: keep.ID, Structure: keep})
		rig.waitFor(t, "DRAIN offered", func() bool {
			for _, o := range rig.s.Hud.LiveOrders() {
				if o.Key == "drain" {
					return !o.Disabled
				}
			}
			return false
		})
		started := time.Now()
		rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: "drain"})
		rig.waitFor(t, "the drain to wait on the budget", func() bool {
			if form, open := rig.s.Hud.OrderOpen(); open && form.Go != "" && !form.Locked || open && strings.HasPrefix(form.Go, "CONFIRM") {
				rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: rig.s.Hud.OrderValues()})
			}
			return rig.s.Writer.drain != nil && strings.Contains(rig.s.Writer.drain.blocked, "held/held-")
		})
		rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollStop})
		rig.waitFor(t, "STOP to end the drain", func() bool { return rig.s.Writer.drain != nil && rig.s.Writer.drain.over })
		var cordons []string
		for _, e := range auditEvents(t, k.logA) {
			if e.Verb == "patch" && strings.HasSuffix(strings.Split(e.RequestURI, "?")[0], "/nodes/"+k.worker) && strings.Contains(e.RequestURI, "fieldManager=kubernites") && !strings.Contains(e.RequestURI, "dryRun") && e.Received.After(started) {
				cordons = append(cordons, e.RequestURI)
			}
		}
		data, _ := os.ReadFile(rig.s.Writer.ledger.Path())
		t.Logf("%s: cordons the API server saw %v; ledger:\n%s", level, cordons, data)
		if len(cordons) != 1 {
			t.Errorf("%s: the API server saw %d cordons, want 1", level, len(cordons))
		}
		if !strings.Contains(string(data), `"order":"cordon","object":"node/`+k.worker+`"`) || !strings.Contains(string(data), "drain stopped by STOP") {
			t.Errorf("%s: the ledger lacks the drain's cordon or how it ended", level)
		}
		rig.g.RealmIntent(hud.Intent{Kind: hud.IntentTitle})
	}
}
