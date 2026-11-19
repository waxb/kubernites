package kubectl

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVersionPrintsTheKustomizeVersion(t *testing.T) {
	f := newCluster(t)
	want := "Client Version: v1.37.0\nKustomize Version: v5.8.1\nServer Version: v1.37.0\n"
	if res := run(f, "version"); !res.OK || res.Output != want {
		t.Fatalf("version = %q", res.Output)
	}
}

func TestExplainOmitsTheCoreGroupAndListsFields(t *testing.T) {
	f := newCluster(t)
	out := run(f, "explain pods").Output
	if !strings.HasPrefix(out, "KIND:       Pod\nVERSION:    v1\n\nDESCRIPTION:\n    Pod is a collection of containers") {
		t.Fatalf("explain pods:\n%s", out)
	}
	for _, want := range []string{"\nFIELDS:\n  apiVersion\t<string>\n    APIVersion defines", "\n  kind\t<string>\n", "\n  metadata\t<ObjectMeta>\n", "\n  spec\t<PodSpec>\n", "\n  status\t<PodStatus>\n", "IN THE REALM:"} {
		if !strings.Contains(out, want) {
			t.Errorf("explain pods lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "GROUP:") {
		t.Errorf("explain pods prints a GROUP for the core group:\n%s", out)
	}
}

func TestExplainAFieldListsItsSubfields(t *testing.T) {
	f := newCluster(t)
	out := run(f, "explain pod.spec").Output
	if !strings.HasPrefix(out, "KIND:       Pod\nVERSION:    v1\n\nFIELD: spec <PodSpec>\n\n\nDESCRIPTION:\n") {
		t.Fatalf("explain pod.spec:\n%s", out)
	}
	for _, want := range []string{"\nFIELDS:\n", "\n  containers\t<[]Container> -required-\n", "\n  nodeName\t<string>\n", "\n  activeDeadlineSeconds\t<integer>\n", "\n  hostNetwork\t<boolean>\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("explain pod.spec lacks %q:\n%s", want, out)
		}
	}
	leaf := run(f, "explain pod.spec.nodeName").Output
	if strings.Contains(leaf, "FIELDS:") || !strings.Contains(leaf, "FIELD: nodeName <string>") {
		t.Fatalf("explain pod.spec.nodeName:\n%s", leaf)
	}
	apps := run(f, "explain deployment").Output
	if !strings.HasPrefix(apps, "GROUP:      apps\nKIND:       Deployment\nVERSION:    v1\n") || !strings.Contains(apps, "\n  spec\t<DeploymentSpec>\n") {
		t.Fatalf("explain deployment:\n%s", apps)
	}
}

func TestExplainWrapsDescriptionsLikeKubectl(t *testing.T) {
	f := newCluster(t)
	for _, line := range strings.Split(run(f, "explain pods").Output, "\n") {
		if len(line) > 4+76 && !strings.Contains(line, "https://") {
			t.Errorf("line longer than kubectl wraps: %q", line)
		}
	}
}

func TestTopWithoutAnyMetricsSaysTheyAreNotAvailableYet(t *testing.T) {
	f := newCluster(t)
	f.unlocks["augury"] = true
	f.runningPod(t, "lantern-abcde", 0)
	if res := run(f, "top nodes"); res.OK || res.Output != "error: metrics not available yet\n" {
		t.Fatalf("top nodes = %+v", res)
	}
	if res := run(f, "top pods"); res.OK || res.Output != "error: metrics not available yet\n" {
		t.Fatalf("top pods = %+v", res)
	}
	f.clk.Advance(180)
	if res := run(f, "top pods"); res.OK || !strings.HasPrefix(res.Output, "error: Metrics not available for pod default/lantern-abcde, age: 3m") {
		t.Fatalf("top pods on an old pod = %+v", res)
	}
}

func TestTopNodesListsUnmeasuredNodesLastAsUnknown(t *testing.T) {
	f := newCluster(t)
	f.unlocks["augury"] = true
	res := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("4Gi")}
	_, _ = f.st.Create(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "a-keep"}, Status: corev1.NodeStatus{Capacity: res, Allocatable: res}})
	f.usage["node:lone-tower"] = Usage{CPUMilli: 500, MemBytes: 1 << 30}
	out := run(f, "top nodes").Output
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "lone-tower") || !strings.HasPrefix(lines[2], "a-keep") || !strings.Contains(lines[2], "<unknown>") {
		t.Fatalf("top nodes:\n%s", out)
	}
}

func TestGrimoireLoreSpeaksOfBoxesNotPilgrims(t *testing.T) {
	for _, table := range []map[string]explanation{explanations, hpaExplanations, priorityExplanations} {
		for key, e := range table {
			if strings.Contains(strings.ToLower(e.lore+e.text), "pilgrim") {
				t.Errorf("%s still speaks of pilgrims: %q", key, e.lore)
			}
		}
	}
}

func TestTopPodsWithNoPodsSaysSo(t *testing.T) {
	f := newCluster(t)
	f.unlocks["augury"] = true
	if res := run(f, "top pods"); !res.OK || res.Output != "No resources found in default namespace.\n" {
		t.Fatalf("top pods = %+v", res)
	}
	if res := run(f, "top pods -A"); !res.OK || res.Output != "No resources found\n" {
		t.Fatalf("top pods -A = %+v", res)
	}
}
