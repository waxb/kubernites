package live_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestTheManagedFixtureHasTheStagingClustersShape(t *testing.T) {
	s := livetest.Managed()
	count := func(name string, got, want int) {
		t.Helper()
		if got != want {
			t.Errorf("%s: %d, want %d", name, got, want)
		}
	}
	count("nodes", len(s.Nodes), 3)
	pools, zones := map[string]bool{}, map[string]bool{}
	for _, n := range s.Nodes {
		pools[n.Labels["cloud.google.com/gke-nodepool"]], zones[n.Labels[live.LabelZone]] = true, true
		if n.Labels[live.LabelInstanceType] != "e2-standard-2" {
			t.Errorf("node %s is %s", n.Name, n.Labels[live.LabelInstanceType])
		}
		if _, cp := n.Labels["node-role.kubernetes.io/control-plane"]; cp {
			t.Errorf("node %s is a control-plane node", n.Name)
		}
	}
	count("node pools", len(pools), 1)
	count("zones", len(zones), 1)
	count("pods", len(s.Pods), 58)
	count("namespaces", len(s.Namespaces), 9)
	owners, system := map[string]int{}, 0
	for _, p := range s.Pods {
		if ref := metav1.GetControllerOf(&p); ref != nil {
			owners[ref.Kind]++
		}
		if p.Namespace == "kube-system" {
			system++
		}
	}
	count("pods in kube-system", system, 29)
	count("ReplicaSet pods", owners["ReplicaSet"], 29)
	count("DaemonSet pods", owners["DaemonSet"], 21)
	count("StatefulSet pods", owners["StatefulSet"], 8)
	count("deployments", len(s.Deployments), 27)
	count("statefulsets", len(s.StatefulSets), 9)
	count("daemonsets", len(s.DaemonSets), 39)
	idle := 0
	for _, d := range s.DaemonSets {
		if d.Status.DesiredNumberScheduled == 0 {
			idle++
		}
	}
	count("daemonsets that schedule no pod", idle, 32)
	count("jobs", len(s.Jobs), 3)
	count("services", len(s.Services), 34)
	types := map[corev1.ServiceType]int{}
	for _, svc := range s.Services {
		types[svc.Spec.Type]++
	}
	count("ClusterIP services", types[corev1.ServiceTypeClusterIP], 32)
	count("LoadBalancer services", types[corev1.ServiceTypeLoadBalancer], 1)
	count("NodePort services", types[corev1.ServiceTypeNodePort], 1)
	count("ingresses", len(s.Ingresses), 7)
	for _, ing := range s.Ingresses {
		if ing.Spec.IngressClassName == nil || *ing.Spec.IngressClassName != "nginx" {
			t.Errorf("ingress %s is not an nginx ingress", ing.Name)
		}
	}
	count("pvcs", len(s.PVCs), 14)
	count("pvs", len(s.PVs), 14)
	count("storageclasses", len(s.StorageClasses), 4)
	count("hpas", len(s.HPAs), 0)
	groups := map[string]bool{}
	for _, c := range s.CRDs {
		groups[c.Group] = true
	}
	for _, g := range []string{"cert-manager.io", "external-secrets.io", "cilium.io", "networking.gke.io"} {
		if !groups[g] {
			t.Errorf("no CRD from %s", g)
		}
	}
	if len(s.Custom) == 0 {
		t.Error("no custom resources")
	}
	if !reflect.DeepEqual(s, livetest.Managed()) {
		t.Error("the fixture is not the same twice")
	}
}

func TestTheManagedFixtureOpensAsARealm(t *testing.T) {
	r, err := live.OpenRecorded(livetest.Managed())
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	if len(w.Isles) != 1 || len(w.Keeps) != 3 {
		t.Fatalf("isles %d, keeps %d", len(w.Isles), len(w.Keeps))
	}
}

func TestTheRecordedManagedFixtureIsCurrent(t *testing.T) {
	data, err := json.Marshal(livetest.Managed())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "managed-snapshot.json")
	if os.Getenv("KUBERNITES_RECORD_MANAGED") != "" {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("%s is stale (%v): KUBERNITES_RECORD_MANAGED=1 go test -run TestTheRecordedManagedFixtureIsCurrent ./pkg/live", path, err)
	}
}

func TestTheRecordedStagingFixtureIsCurrent(t *testing.T) {
	data, err := json.Marshal(livetest.Staging())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "gke-staging-snapshot.json")
	if os.Getenv("KUBERNITES_RECORD_MANAGED") != "" {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("%s is stale (%v): KUBERNITES_RECORD_MANAGED=1 go test -run TestTheRecordedStagingFixtureIsCurrent ./pkg/live", path, err)
	}
}
