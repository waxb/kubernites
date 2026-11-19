package lexicon

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedLexicon(t *testing.T) {
	lx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range lx.Entries {
		if !strings.HasPrefix(e.Docs, "https://") {
			t.Errorf("%s has no https docs link", e.K8s)
		}
		seen[e.K8s] = e.Name
	}
	for k8s, name := range map[string]string{"Pod": "Hut", "Node": "Keep", "cpu": "Food", "memory": "Housing", "ReplicaSet": "Shift", "CrashLoopBackOff": "Guttering hut",
		"HorizontalPodAutoscaler": "War Horn", "Target utilization": "The Horn's Pitch", "Stabilization window": "The Horn's Memory",
		"minReplicas / maxReplicas": "Muster Floor and Ceiling", "metrics-server": "Augury",
		"Static Pod": "Hut at the door", "Mirror Pod": "Mirror hut", "hostPort": "The Keep's Door",
		"Ephemeral storage": "Cellar", "etcd compaction": "Palimpsest", "etcd defragmentation": "Rebinding the Codex",
		"PriorityClass": "Rank", "Preemption": "Displacement"} {
		if seen[k8s] != name {
			t.Errorf("%s => %q, want %q", k8s, seen[k8s], name)
		}
	}
}

func TestParseRejectsBadLexicons(t *testing.T) {
	for name, doc := range map[string]string{
		"bad yaml":      "entries: [",
		"unknown field": "entries:\n  - {k8s: Pod, name: Hearth, color: red}\n",
		"missing name":  "entries:\n  - {k8s: Pod}\n",
	} {
		if _, err := parse([]byte(doc)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestFamiliarsAndIslesAreInTheLexicon(t *testing.T) {
	lx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	byK8s := map[string]Entry{}
	for _, e := range lx.Entries {
		byK8s[e.K8s] = e
	}
	for k8s, want := range map[string]Entry{
		"Sidecar container":               {Name: "Familiar", Docs: "https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/#pod-sidecar-containers"},
		"kubectl drain":                   {Name: "Drain", Docs: "https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/#use-kubectl-drain-to-remove-a-node-from-service"},
		"Zone":                            {Name: "Isle", Docs: "https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesiozone"},
		"Node unreachable":                {Name: "Storm", Docs: "https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/#taint-based-evictions"},
		"Instance type":                   {Name: "Keep size", Docs: "https://kubernetes.io/docs/reference/labels-annotations-taints/#nodekubernetesioinstance-type"},
		"Pod topology spread constraints": {Name: "Spread", Docs: "https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/"},
		"PodDisruptionBudget":             {Name: "Quorum", Docs: "https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets"},
	} {
		got, ok := byK8s[k8s]
		if !ok || got.Name != want.Name || got.Docs != want.Docs || got.Flavor == "" {
			t.Errorf("%s => %+v, want %+v", k8s, got, want)
		}
	}
	for _, label := range []string{"topology.kubernetes.io/zone", "node.kubernetes.io/instance-type", "node.kubernetes.io/unreachable", "restartPolicy: Always"} {
		found := false
		for _, e := range lx.Entries {
			found = found || strings.Contains(e.Flavor, label)
		}
		if !found {
			t.Errorf("no flavor names %s", label)
		}
	}
}
