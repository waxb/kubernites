package live

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	gapsKubeconfigEnv = "KUBERNITES_WATCH_KUBECONFIG"
	gapsKubectlEnv    = "KUBERNITES_WATCH_KUBECTL"
	gapsGatewayEnv    = "KUBERNITES_WATCH_GATEWAY_API"
	gapsReportEnv     = "KUBERNITES_WATCH_REPORT"
	gapsContext       = "kind-kubernites"
	gapsLister        = "hut-lister"
	gapsWait          = 2 * time.Minute
	gapsTick          = 200 * time.Millisecond
)

type gapsRun struct {
	t       *testing.T
	config  string
	kubectl string
}

func (k gapsRun) run(line string) (string, error) {
	k.t.Helper()
	args := append([]string{"--kubeconfig", k.config, "--context", gapsContext}, strings.Fields(line)...)
	cmd := exec.Command(k.kubectl, args...)
	cmd.Dir = filepath.Join("testdata", "watch")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+k.config)
	out, err := cmd.CombinedOutput()
	k.t.Logf("kubectl %s: %s", line, strings.TrimSpace(string(out)))
	return string(out), err
}

func (k gapsRun) must(line string) string {
	k.t.Helper()
	out, err := k.run(line)
	if err != nil {
		k.t.Fatalf("kubectl %s: %v", line, err)
	}
	return out
}

func (k gapsRun) listerConfig() string {
	k.t.Helper()
	token := strings.TrimSpace(k.must("-n listed create token hut-lister --duration=2h"))
	admin, err := loadKubeconfig([]string{k.config})
	if err != nil {
		k.t.Fatal(err)
	}
	cluster := admin.Clusters[admin.Contexts[gapsContext].Cluster]
	cfg := clientcmdapi.NewConfig()
	cfg.Clusters[gapsLister] = cluster
	cfg.AuthInfos[gapsLister] = &clientcmdapi.AuthInfo{Token: token}
	cfg.Contexts[gapsLister] = &clientcmdapi.Context{Cluster: gapsLister, AuthInfo: gapsLister, Namespace: "listed"}
	cfg.CurrentContext = gapsLister
	path := filepath.Join(k.t.TempDir(), "hut-lister.kubeconfig")
	if err := clientcmd.WriteToFile(*cfg, path); err != nil {
		k.t.Fatal(err)
	}
	return path
}

func gapsWatch(t *testing.T, config, name string) *Realm {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	r, err := Connect(ctx, []string{config}, name, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	r.StartWatching()
	t.Cleanup(r.Stop)
	return r
}

func gapsUntil(t *testing.T, r *Realm, what string, ok func() bool) float64 {
	t.Helper()
	start := time.Now()
	for {
		r.Pump(gapsTick.Seconds())
		if ok() {
			return time.Since(start).Seconds()
		}
		if time.Since(start) > gapsWait {
			t.Fatalf("never: %s (freshness %+v)", what, r.Freshness())
		}
		time.Sleep(gapsTick)
	}
}

func gapsHold(r *Realm, d time.Duration) {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(gapsTick) {
		r.Pump(gapsTick.Seconds())
	}
}

func hasPod(r *Realm, ns, name string) bool {
	for _, p := range r.Snapshot().Pods {
		if p.Namespace == ns && p.Name == name {
			return true
		}
	}
	return false
}

func dragons(r *Realm) []string {
	var out []string
	for _, c := range r.Snapshot().Custom {
		if c.Kind == "Dragon" {
			out = append(out, c.Namespace+"/"+c.Name)
		}
	}
	slices.Sort(out)
	return out
}

func TestTheWatchGapsOnAKindCluster(t *testing.T) {
	config, kubectl := os.Getenv(gapsKubeconfigEnv), os.Getenv(gapsKubectlEnv)
	if config == "" || kubectl == "" {
		t.Skipf("set %s to the throwaway kind cluster's kubeconfig file and %s to a kubectl binary to run this test", gapsKubeconfigEnv, gapsKubectlEnv)
	}
	ctxs, err := ListContexts([]string{config})
	if err != nil || len(ctxs) != 1 || ctxs[0].Name != gapsContext {
		t.Fatalf("the kubeconfig must hold only %s: %+v %v", gapsContext, ctxs, err)
	}
	k := gapsRun{t: t, config: config, kubectl: kubectl}
	report := map[string]float64{}
	defer func() {
		if path := os.Getenv(gapsReportEnv); path != "" {
			data, _ := json.MarshalIndent(report, "", "  ")
			os.WriteFile(path, data, 0o644)
		}
	}()
	k.run("delete namespace listed --ignore-not-found --wait=true --timeout=120s")
	k.run("delete crd dragons.lair.kubernites.example --ignore-not-found --wait=true")
	k.must("apply -f rbac.yaml")
	if manifest := os.Getenv(gapsGatewayEnv); manifest != "" {
		k.must("apply --server-side -f " + manifest)
		k.must("wait --for condition=Established crd/gateways.gateway.networking.k8s.io --timeout=60s")
	}

	t.Run("a watch the credentials forbid is read by listing", func(t *testing.T) {
		k.t = t
		r := gapsWatch(t, k.listerConfig(), gapsLister)
		report["listing at"] = gapsUntil(t, r, "the pods read by listing", func() bool {
			l := r.Listing()
			return len(l) == 1 && l[0].List == "pods" && l[0].Every == ListEvery && !l[0].PastCap
		})
		gapsHold(r, StaleAfter+5*time.Second)
		if f := r.Freshness(); !f.Watching || !f.StaleSince.IsZero() {
			t.Fatalf("the realm is stale while the pods are listed: %+v", f)
		}
		k.must("apply -f listed.yaml")
		report["listed pod appears"] = gapsUntil(t, r, "the new pod", func() bool { return hasPod(r, "listed", "listed-in") })
		k.must("-n listed delete pod listed-in --wait=false")
		report["listed pod goes"] = gapsUntil(t, r, "the pod gone", func() bool { return !hasPod(r, "listed", "listed-in") })
		if l := r.Listing(); len(l) != 1 || l[0].List != "pods" {
			t.Errorf("after the lists the kinds read by listing are %+v", l)
		}
		if f := r.Freshness(); !f.StaleSince.IsZero() {
			t.Errorf("stale at the end: %+v", f)
		}
		t.Logf("listing at %.1fs, the pod appeared in %.1fs and went in %.1fs", report["listing at"], report["listed pod appears"], report["listed pod goes"])
	})

	t.Run("a CRD installed while watching", func(t *testing.T) {
		k.t = t
		r := gapsWatch(t, config, gapsContext)
		w := r.World()
		before := standingOf(w)
		gapsHold(r, 2*time.Second)
		k.must("apply -f dragons-crd.yaml")
		k.must("wait --for condition=Established crd/dragons.lair.kubernites.example --timeout=60s")
		report["dragons listed"] = gapsUntil(t, r, "the dragons read", func() bool {
			_, ok := r.Snapshot().Lists["dragons.lair.kubernites.example"]
			return ok
		})
		k.must("apply -f dragons.yaml")
		report["dragons appear"] = gapsUntil(t, r, "the dragons", func() bool { return len(dragons(r)) == 2 })
		report["their hall stands"] = gapsUntil(t, r, "the dragons' foreign hall", func() bool {
			return len(structuresFor(r.World(), "Dragon", "listed", "lair.kubernites.example")) == 1
		})
		if r.World() != w {
			t.Error("the dragons redrew the realm")
		}
		before.unmoved(t, r.World(), "after the dragons came")
		k.must("-n listed delete dragon dragon-a dragon-b --wait=false")
		report["dragons go"] = gapsUntil(t, r, "the dragons gone", func() bool { return len(dragons(r)) == 0 })
		report["their hall goes"] = gapsUntil(t, r, "the hall gone", func() bool {
			return len(structuresFor(r.World(), "Dragon", "listed", "lair.kubernites.example")) == 0
		})
		k.must("delete crd dragons.lair.kubernites.example --wait=true")
		report["their kind stops"] = gapsUntil(t, r, "the dragons no longer read", func() bool {
			_, ok := r.Snapshot().Lists["dragons.lair.kubernites.example"]
			return !ok
		})
		before.unmoved(t, r.World(), "after the dragons went")
		if f := r.Freshness(); !f.StaleSince.IsZero() {
			t.Errorf("stale at the end: %+v", f)
		}
	})

	t.Run("a Service, an Ingress and a Gateway created while watching", func(t *testing.T) {
		k.t = t
		if os.Getenv(gapsGatewayEnv) == "" {
			t.Skipf("set %s to the Gateway API standard-install.yaml", gapsGatewayEnv)
		}
		r := gapsWatch(t, config, gapsContext)
		w := r.World()
		gapsHold(r, 2*time.Second)
		before := standingOf(r.World())
		k.must("apply -f arrivals.yaml")
		for _, o := range [][2]string{{"Service", "basket"}, {"Ingress", "basket"}, {"Gateway", "front"}} {
			report[o[0]+" drawn"] = gapsUntil(t, r, o[0]+" drawn", func() bool { return len(structuresFor(r.World(), o[0], "listed", o[1])) > 0 })
		}
		if r.World() != w {
			t.Fatal("the new Service, Ingress and Gateway redrew the realm")
		}
		before.unmoved(t, r.World(), "after they came")
		if bad := overlapping(r.World()); len(bad) > 0 {
			t.Errorf("structures overlap: %v", bad)
		}
		middle := standingOf(r.World())
		k.must("delete -f arrivals.yaml --wait=false")
		for _, o := range [][2]string{{"Service", "basket"}, {"Ingress", "basket"}, {"Gateway", "front"}} {
			report[o[0]+" gone"] = gapsUntil(t, r, o[0]+" gone", func() bool { return len(structuresFor(r.World(), o[0], "listed", o[1])) == 0 })
		}
		middle.unmoved(t, r.World(), "after they went")
		if r.World() != w {
			t.Error("deleting them redrew the realm")
		}
	})
	k.t = t
	k.run("delete namespace listed --ignore-not-found --wait=false")
}
