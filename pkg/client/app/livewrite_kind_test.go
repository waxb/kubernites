package app

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/view"
)

type kindCluster struct {
	t          *testing.T
	kubeconfig string
	kubectl    string
}

func kindW1(t *testing.T) *kindCluster {
	t.Helper()
	path, kubectl := os.Getenv("KUBERNITES_WRITES_KUBECONFIG"), os.Getenv("KUBERNITES_WRITES_KUBECTL")
	if path == "" || kubectl == "" {
		t.Skip("KUBERNITES_WRITES_KUBECONFIG and KUBERNITES_WRITES_KUBECTL name a kind cluster made in this VM")
	}
	return &kindCluster{t: t, kubeconfig: path, kubectl: kubectl}
}

func (k *kindCluster) run(args ...string) string {
	k.t.Helper()
	out, err := exec.Command(k.kubectl, append([]string{"--kubeconfig", k.kubeconfig}, args...)...).CombinedOutput()
	if err != nil {
		k.t.Fatalf("kubectl %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (k *kindCluster) apply(manifest string) {
	k.t.Helper()
	cmd := exec.Command(k.kubectl, "--kubeconfig", k.kubeconfig, "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	if out, err := cmd.CombinedOutput(); err != nil {
		k.t.Fatalf("apply: %v\n%s", err, out)
	}
}

func (k *kindCluster) fresh(ns string) {
	_, _ = exec.Command(k.kubectl, "--kubeconfig", k.kubeconfig, "wait", "--for=delete", "namespace/"+ns, "--timeout=180s").CombinedOutput()
}

func (k *kindCluster) viewer(dir string) string {
	k.t.Helper()
	k.apply(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: viewer
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: viewer
roleRef: {apiGroup: rbac.authorization.k8s.io, kind: ClusterRole, name: view}
subjects:
- {kind: ServiceAccount, name: viewer, namespace: default}
`)
	token := strings.TrimSpace(k.run("-n", "default", "create", "token", "viewer", "--duration=1h"))
	raw, err := clientcmd.LoadFromFile(k.kubeconfig)
	if err != nil {
		k.t.Fatal(err)
	}
	ctx := raw.Contexts[raw.CurrentContext]
	cluster := raw.Clusters[ctx.Cluster]
	cfg := `apiVersion: v1
kind: Config
clusters:
- name: kind
  cluster:
    server: ` + cluster.Server + `
    certificate-authority-data: ` + encodeCA(cluster.CertificateAuthorityData) + `
contexts:
- name: kind-viewer
  context: {cluster: kind, user: viewer}
users:
- name: viewer
  user:
    token: ` + token + `
current-context: kind-viewer
`
	path := filepath.Join(dir, "viewer-kubeconfig")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		k.t.Fatal(err)
	}
	return path
}

func connectFile(t *testing.T, kubeconfig, level string, classes ...write.Class) *liveRig {
	t.Helper()
	dir := t.TempDir()
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: dir, Live: LiveSource{Files: []string{kubeconfig}}}, nil, nil)
	rig := &liveRig{g: g, dir: dir}
	rig.title(hud.Intent{Kind: hud.IntentLive})
	rig.title(hud.Intent{Kind: hud.IntentLiveList})
	if len(g.Picker.Contexts) == 0 {
		t.Fatalf("no contexts: %s", g.Picker.Err)
	}
	raw, _ := clientcmd.LoadFromFile(kubeconfig)
	for i, c := range g.Picker.Contexts {
		if c.Name == raw.CurrentContext {
			g.Picker.Chosen = i
		}
	}
	if len(classes) > 0 {
		c := g.Picker.Contexts[g.Picker.Chosen]
		g.choices.Keep(write.Remembered{Context: write.Context{Name: c.Name, Server: c.Server, User: c.User, Identity: c.Identity}, Level: write.Seamless, Warned: write.Seamless, Classes: classes})
		g.saveChoices()
	}
	rig.press(t, "live:choose")
	if level != hud.LevelReadOnly {
		rig.press(t, hud.LiveBoxID)
		for g.Picker.Step == hud.LiveStepWrites || g.Picker.Step == hud.LiveStepSeam {
			if level == hud.LevelSeamless && g.Picker.Step == hud.LiveStepWrites {
				rig.press(t, "live:seam")
			}
			rig.press(t, "live:allow")
		}
		if g.Picker.ChosenLevel() != level {
			rig.press(t, "live:level")
		}
	}
	if g.Picker.ChosenLevel() != level {
		t.Fatalf("the picker chose %s, want %s", g.Picker.ChosenLevel(), level)
	}
	rig.press(t, "live:confirm")
	for deadline := time.Now().Add(60 * time.Second); g.Scene != SceneRealm; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the cluster never opened: %q %s", g.Picker.Step, g.Picker.Err)
		}
		g.StepLive()
	}
	rig.s = g.Sess
	if w := rig.s.Writer; w != nil {
		w.sync = true
		rig.gate = w.gate
	}
	return rig
}

func (r *liveRig) waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(90 * time.Second); !ok(); time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("waited 90 s for %s", what)
		}
		r.frame()
	}
}

const kindApp = `apiVersion: v1
kind: Namespace
metadata:
  name: NS
  labels: {hook: "LABEL"}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: NS
spec:
  replicas: 2
  selector:
    matchLabels: {app: web}
  template:
    metadata:
      labels: {app: web}
    spec:
      terminationGracePeriodSeconds: 2
      containers:
      - name: web
        image: busybox:1.37
        command: ["sh", "-c", "sleep 3600"]
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: web
  namespace: NS
spec:
  minAvailable: 2
  selector:
    matchLabels: {app: web}
`

func (k *kindCluster) app(ns string, hooked bool) {
	k.t.Helper()
	k.fresh(ns)
	label := "no"
	if hooked {
		label = "yes"
	}
	k.apply(strings.NewReplacer("NS", ns, "LABEL", label).Replace(kindApp))
	k.run("-n", ns, "rollout", "status", "deployment/web", "--timeout=120s")
}

func TestOnKindReadOnlyRBACGreysEveryOrderWithTheServersReason(t *testing.T) {
	k := kindW1(t)
	k.app("rbac", false)
	defer k.run("delete", "namespace", "rbac", "--wait=false")
	defer k.run("delete", "clusterrolebinding", "viewer", "--ignore-not-found")
	rig := connectFile(t, k.viewer(t.TempDir()), hud.LevelWrites, write.AllClasses...)
	huts := func() []model.PodView {
		var out []model.PodView
		for _, p := range rig.s.Pods {
			if p.Namespace == "rbac" {
				out = append(out, p)
			}
		}
		return out
	}
	rig.waitFor(t, "the crew and its two huts in the realm", func() bool {
		for _, c := range rig.s.Crews {
			if c.Namespace == "rbac" {
				return len(huts()) >= 2
			}
		}
		return false
	})
	sels := []hud.Selection{}
	for _, sel := range rig.selections() {
		if sel.Kind != hud.SelectHearth {
			sels = append(sels, sel)
		}
	}
	for _, p := range huts() {
		sels = append(sels, hud.Selection{Kind: hud.SelectHearth, ID: p.Namespace + "/" + p.Name, Pod: p})
	}
	n, reasons := 0, map[string]bool{}
	for _, sel := range sels {
		if sel.Kind == hud.SelectCrew && !strings.HasPrefix(sel.ID, "rbac/") {
			continue
		}
		for _, o := range rig.select_(sel) {
			if o.Key == orderMore {
				continue
			}
			if sp, ok := rig.s.spec(o.Key); ok && sp.read {
				if o.Disabled {
					t.Errorf("%s %s: the view role may read it, but it is greyed: why %q tip %q", sel.ID, o.Key, o.Why, o.Tip)
				}
				continue
			}
			n++
			if !o.Disabled || o.Why != refusedWhy {
				t.Errorf("%s %s: disabled %v why %q tip %q", sel.ID, o.Key, o.Disabled, o.Why, o.Tip)
			}
			reasons[o.Tip] = true
		}
	}
	if n < 10 {
		t.Fatalf("only %d orders", n)
	}
	for r := range reasons {
		t.Logf("refusal: %s", r)
	}
}

func encodeCA(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func TestOnKindABudgetBlocksADrainAndIsShown(t *testing.T) {
	k := kindW1(t)
	k.app("budget", false)
	defer k.run("delete", "namespace", "budget", "--wait=false")
	defer k.run("uncordon", "kubernites-worker")
	rig := connectFile(t, k.kubeconfig, hud.LevelSeamless, write.AllClasses...)
	rig.s.Writer.sync = false
	var keep view.Structure
	rig.waitFor(t, "the worker Keep", func() bool {
		for _, st := range rig.s.Site.Structures {
			if st.Kind == view.StructKeep && st.Object.Name == "kubernites-worker" {
				keep = st
				return true
			}
		}
		return false
	})
	rig.select_(hud.Selection{Kind: "structure", ID: keep.ID, Structure: keep})
	rig.waitFor(t, "the access review", func() bool {
		for _, o := range rig.s.Hud.LiveOrders() {
			if o.Key == "drain" {
				return !o.Disabled
			}
		}
		return false
	})
	rig.goAll("drain", 1)
	deadline := time.Now().Add(90 * time.Second)
	for rig.s.Writer.drain == nil || !strings.Contains(rig.s.Writer.drain.blocked, "budget/web-") {
		if time.Now().After(deadline) {
			form, _ := rig.s.Hud.OrderOpen()
			var toasts []string
			for _, x := range rig.s.Hud.Toasts() {
				toasts = append(toasts, x.Text)
			}
			t.Fatalf("the drain never waited on the budget: drain %+v form %+v toasts %v run %+v", rig.s.Writer.drain, form, toasts, rig.s.Writer.run)
		}
		rig.frame()
		time.Sleep(50 * time.Millisecond)
	}
	card := rig.s.Hud.LiveCard()
	t.Logf("card: %s | %s", card.Draining, card.DrainWhy)
	if !strings.Contains(card.DrainWhy, "disruption budget") || !strings.Contains(card.DrainWhy, "budget/web: minAvailable 2") {
		t.Fatalf("the card does not show the blocking budget: %q", card.DrainWhy)
	}
	form, _ := rig.s.Hud.OrderOpen()
	t.Logf("scroll: %v", form.Progress)
	rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollStop})
	rig.waitFor(t, "STOP", func() bool { return rig.s.Writer.drain.over })
	if out := k.run("-n", "budget", "get", "pods", "-l", "app=web", "--no-headers"); strings.Count(out, "Running") != 2 {
		t.Fatalf("the budget's huts did not stay:\n%s", out)
	}
}

const sideEffectHook = `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: audit
webhooks:
- name: audit.example.com
  sideEffects: Some
  failurePolicy: Ignore
  admissionReviewVersions: ["v1"]
  timeoutSeconds: 2
  namespaceSelector:
    matchLabels: {hook: "yes"}
  rules:
  - apiGroups: ["apps"]
    apiVersions: ["v1"]
    operations: ["UPDATE"]
    resources: ["deployments", "deployments/scale"]
  clientConfig:
    url: https://127.0.0.1:9/audit
`

func TestOnKindAWebhookWithSideEffectsCannotBeDeclaredOnV137(t *testing.T) {
	k := kindW1(t)
	cmd := exec.Command(k.kubectl, "--kubeconfig", k.kubeconfig, "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(sideEffectHook)
	out, err := cmd.CombinedOutput()
	if err == nil {
		k.run("delete", "validatingwebhookconfiguration", "audit", "--ignore-not-found")
		t.Fatalf("the API server now takes sideEffects: Some, so the dry-run refusal can be produced on kind:\n%s", out)
	}
	if !strings.Contains(string(out), `Unsupported value: "Some": supported values: "None", "NoneOnDryRun"`) {
		t.Fatalf("refused for another reason:\n%s", out)
	}
	t.Logf("the API server: %s", strings.TrimSpace(string(out)))
}
