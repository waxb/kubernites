package live

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/sim/store"
)

const kindKubeconfigEnv = "KUBERNITES_KIND_KUBECONFIG"

func kindRealm(t *testing.T) *Realm {
	t.Helper()
	path := os.Getenv(kindKubeconfigEnv)
	if path == "" {
		t.Skipf("set %s to the throwaway kind cluster's kubeconfig file to run this test", kindKubeconfigEnv)
	}
	ctxs, err := ListContexts([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(ctxs) != 1 || ctxs[0].Name != "kind-kubernites" {
		t.Fatalf("the kind kubeconfig must hold only kind-kubernites, holds %+v", ctxs)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	r, err := Connect(ctx, []string{path}, "kind-kubernites", time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestKindClusterOpensAsARealm(t *testing.T) {
	r := kindRealm(t)
	s := r.Snapshot()
	if len(s.Nodes) != 3 || len(s.Isles()) != 3 {
		t.Fatalf("nodes %d, isles %v", len(s.Nodes), s.Isles())
	}
	w := r.World()
	if len(w.Isles) != 3 || !w.Isles[0].Home || len(w.Keeps) != 3 {
		t.Fatalf("isles %+v keeps %+v", w.Isles, w.Keeps)
	}
	if w.Keeps[0].Variant != "tower" || !w.Keeps[0].Control {
		t.Errorf("the control plane is not the Lone Tower: %+v", w.Keeps[0])
	}
	if len(r.mirror.List(store.KindPod, "")) != len(s.Pods) || len(s.Pods) < 15 {
		t.Errorf("mirror holds %d pods of %d", len(r.mirror.List(store.KindPod, "")), len(s.Pods))
	}
	stories := map[string]string{}
	for _, c := range w.Crews {
		stories[c.Namespace+"/"+c.Name] = c.Story
	}
	for key, want := range map[string]string{"shop/web": "crew", "shop/ledger": "order", "kube-system/kube-proxy": "shrine", "batch/tally": "ritual", "batch/hermit": "lone hut"} {
		if stories[key] != want {
			t.Errorf("%s is %q, want %q (crews %v)", key, stories[key], want, stories)
		}
	}
	if out := r.Kubectl("kubectl get secrets -A").Output; !strings.Contains(out, "never read") {
		t.Errorf("get secrets: %s", out)
	}
	if out := r.Kubectl("kubectl get pods -n shop").Output; !strings.Contains(out, "web-") || !strings.Contains(out, "crasher-") {
		t.Errorf("get pods:\n%s", out)
	}
	crasher := ""
	for _, p := range s.Pods {
		if strings.HasPrefix(p.Name, "crasher-") {
			crasher = p.Name
		}
	}
	if out := r.Kubectl("kubectl logs -n shop " + crasher).Output; !strings.Contains(out, "opening the ledger") {
		t.Errorf("logs of the crasher:\n%s", out)
	}
	if out := r.Kubectl("kubectl events -n shop").Output; !strings.Contains(out, "BackOff") {
		t.Errorf("events:\n%s", out)
	}
	if out := r.Kubectl("kubectl cluster-info").Output; !strings.Contains(out, "control plane is running at https://127.0.0.1:") || !strings.Contains(out, "CoreDNS is running at") {
		t.Errorf("cluster-info:\n%s", out)
	}
	if out := r.Kubectl("kubectl version").Output; !strings.Contains(out, "Server Version: "+s.Version) || !strings.HasPrefix(s.Version, "v1.37") {
		t.Errorf("version (%s):\n%s", s.Version, out)
	}
	if out := r.Kubectl("kubectl top pods").Output; !strings.Contains(out, "no Augur") {
		t.Errorf("top without metrics-server:\n%s", out)
	}
	for _, cmd := range []string{"kubectl delete pod -n shop " + crasher, "kubectl scale deployment web -n shop --replicas=9", "kubectl cordon kubernites-worker"} {
		if out := r.Kubectl(cmd).Output; strings.TrimSpace(out) != ReadOnlyAnswer {
			t.Errorf("%s answered %q", cmd, out)
		}
	}
	oom := false
	for _, p := range s.Pods {
		for _, cs := range p.Status.ContainerStatuses {
			if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
				oom = true
			}
		}
	}
	if !oom {
		t.Error("no worker of the sample was banished (OOMKilled)")
	}
	if s.Identity.Username != "kubernetes-admin" {
		t.Errorf("the cluster says you are %+v", s.Identity)
	}
	kinds := map[string]int{}
	for _, st := range w.Site.Structures {
		kinds[st.Kind]++
	}
	if kinds[KindHarborGate] != 3 || kinds[KindSignpost] != 2 || w.Site.NoHarbor || len(w.Site.Havens) != 1 {
		t.Errorf("harbor: %d gates, %d signposts, %d havens", kinds[KindHarborGate], kinds[KindSignpost], len(w.Site.Havens))
	}
	if kinds[KindForeign] != 3 || len(s.Custom) != 4 || len(s.CRDs) != 2 {
		t.Errorf("foreign halls %d, custom resources %d, CRDs %d", kinds[KindForeign], len(s.Custom), len(s.CRDs))
	}
	if !slices.Contains(w.Idle, "watch/gpu-driver") || !slices.Contains(w.Idle, "watch/arm-agent") || kinds[KindShrine] != 9 {
		t.Errorf("idle %v, shrines %d", w.Idle, kinds[KindShrine])
	}
	if dir := os.Getenv("KUBERNITES_RECORD_SNAPSHOT"); dir != "" {
		recordSnapshot(t, s, filepath.Join(dir, "kind-snapshot.json"))
	}
}

func recordSnapshot(t *testing.T, s Snapshot, path string) {
	t.Helper()
	data, err := json.Marshal(s.WithoutProxyEnvironment())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "never-shown-in-the-realm") {
		t.Fatal("the sample Secret's value reached the snapshot")
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (s Snapshot) Isles() []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range s.Nodes {
		by, key := isleKey(n)
		if !seen[by+"/"+key] {
			seen[by+"/"+key] = true
			out = append(out, by+"/"+key)
		}
	}
	return out
}
