package kubectl

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func TestDescribeNodeShowsTheFloorAndDiskPressure(t *testing.T) {
	f := newCluster(t)
	node, _ := store.Get[*corev1.Node](f.st, store.KindNode, "", "lone-tower")
	next := node.DeepCopy()
	next.Status.Capacity = next.Status.Capacity.DeepCopy()
	next.Status.Allocatable = next.Status.Allocatable.DeepCopy()
	next.Status.Capacity[corev1.ResourceEphemeralStorage] = resource.MustParse("20Gi")
	next.Status.Allocatable[corev1.ResourceEphemeralStorage] = resource.MustParse("18Gi")
	stamp := f.clk.MetaNow()
	next.Status.Conditions = append([]corev1.NodeCondition{{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue, Reason: "KubeletHasDiskPressure",
		Message: "kubelet has disk pressure", LastHeartbeatTime: stamp, LastTransitionTime: stamp}}, next.Status.Conditions...)
	if _, err := f.st.Update(next); err != nil {
		t.Fatal(err)
	}
	p := f.runningPod(t, "scribe-abcde", 0)
	bound := p.DeepCopy()
	bound.Spec.Containers[0].Resources.Requests[corev1.ResourceEphemeralStorage] = resource.MustParse("1843Mi")
	bound.Spec.Containers[0].Resources.Limits = corev1.ResourceList{corev1.ResourceEphemeralStorage: resource.MustParse("3686Mi")}
	if _, err := f.st.Update(bound); err != nil {
		t.Fatal(err)
	}
	out := run(f, "describe node lone-tower").Output
	stampText := stamp.Format(timeLayout)
	for _, want := range []string{
		"LastHeartbeatTime", "LastTransitionTime", "----",
		"DiskPressure   True     " + stampText,
		"KubeletHasDiskPressure   kubelet has disk pressure",
		"  ephemeral-storage:  20Gi", "  ephemeral-storage:  18Gi",
		"ephemeral-storage   1843Mi (9%)   3686Mi (19%)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("describe node missing %q:\n%s", want, out)
		}
	}
}

func TestDescribeNodeCountsNoFloorPledgeAsZero(t *testing.T) {
	f := newCluster(t)
	f.runningPod(t, "lantern-abcde", 0)
	out := run(f, "describe node lone-tower").Output
	if !strings.Contains(out, "ephemeral-storage   0 (0%)      0 (0%)") {
		t.Fatalf("allocated resources lack the ephemeral-storage row:\n%s", out)
	}
}
