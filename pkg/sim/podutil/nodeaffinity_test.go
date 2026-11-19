package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func keepNode(name, zone string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{
		"kubernetes.io/hostname": name, "topology.kubernetes.io/zone": zone,
	}}}
}

func required(terms ...corev1.NodeSelectorTerm) *corev1.Affinity {
	return &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{NodeSelectorTerms: terms},
	}}
}

func nameIn(name string) corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{MatchFields: []corev1.NodeSelectorRequirement{{Key: "metadata.name", Operator: corev1.NodeSelectorOpIn, Values: []string{name}}}}
}

func zoneIn(zones ...string) corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "topology.kubernetes.io/zone", Operator: corev1.NodeSelectorOpIn, Values: zones}}}
}

func TestRequiredNodeAffinityMatchFieldsOnName(t *testing.T) {
	aff := required(nameIn("morrow"))
	if !MatchesRequiredNodeAffinity(nil, aff, keepNode("morrow", "a")) {
		t.Fatal("morrow should match its own name")
	}
	if MatchesRequiredNodeAffinity(nil, aff, keepNode("tower", "a")) {
		t.Fatal("tower matched a term naming morrow")
	}
}

func TestRequiredNodeAffinityTermsAreORedExpressionsANDed(t *testing.T) {
	aff := required(zoneIn("b"), nameIn("tower"))
	for _, c := range []struct {
		node *corev1.Node
		want bool
	}{{keepNode("tower", "a"), true}, {keepNode("morrow", "b"), true}, {keepNode("morrow", "a"), false}} {
		if got := MatchesRequiredNodeAffinity(nil, aff, c.node); got != c.want {
			t.Fatalf("%s in %s: got %v", c.node.Name, c.node.Labels["topology.kubernetes.io/zone"], got)
		}
	}
	both := corev1.NodeSelectorTerm{MatchExpressions: zoneIn("a").MatchExpressions, MatchFields: nameIn("tower").MatchFields}
	if MatchesRequiredNodeAffinity(nil, required(both), keepNode("morrow", "a")) {
		t.Fatal("a term must match both its expressions and its fields")
	}
}

func TestRequiredNodeAffinityNodeSelectorAndEmptyTerms(t *testing.T) {
	sel := map[string]string{"topology.kubernetes.io/zone": "a"}
	if MatchesRequiredNodeAffinity(sel, required(nameIn("morrow")), keepNode("morrow", "b")) {
		t.Fatal("nodeSelector must hold as well as the affinity")
	}
	if !MatchesRequiredNodeAffinity(sel, nil, keepNode("morrow", "a")) {
		t.Fatal("nodeSelector alone should match")
	}
	if MatchesRequiredNodeAffinity(nil, required(corev1.NodeSelectorTerm{}), keepNode("morrow", "a")) {
		t.Fatal("an empty term matches no objects")
	}
	bad := corev1.NodeSelectorTerm{MatchFields: []corev1.NodeSelectorRequirement{{Key: "metadata.name", Operator: corev1.NodeSelectorOpExists}}}
	if MatchesRequiredNodeAffinity(nil, required(bad), keepNode("morrow", "a")) {
		t.Fatal("an unsupported field operator must not match")
	}
}

func TestRequiredNodeAffinityOperators(t *testing.T) {
	node := keepNode("morrow", "a")
	node.Labels["cpu"] = "4"
	for op, want := range map[corev1.NodeSelectorOperator]bool{
		corev1.NodeSelectorOpNotIn: false, corev1.NodeSelectorOpExists: true, corev1.NodeSelectorOpDoesNotExist: false,
	} {
		term := corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "topology.kubernetes.io/zone", Operator: op}}}
		if op == corev1.NodeSelectorOpNotIn {
			term.MatchExpressions[0].Values = []string{"a"}
		}
		if got := MatchesRequiredNodeAffinity(nil, required(term), node); got != want {
			t.Fatalf("%s: got %v", op, got)
		}
	}
	gt := corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "cpu", Operator: corev1.NodeSelectorOpGt, Values: []string{"2"}}}}
	if !MatchesRequiredNodeAffinity(nil, required(gt), node) {
		t.Fatal("Gt 2 should match 4")
	}
}

func TestTargetNodeName(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Affinity: required(nameIn("morrow"))}}
	if got, ok := TargetNodeName(pod); !ok || got != "morrow" {
		t.Fatalf("target = %q %v", got, ok)
	}
	pod.Spec.NodeName = "tower"
	if got, _ := TargetNodeName(pod); got != "tower" {
		t.Fatalf("bound pod target = %q", got)
	}
	if _, ok := TargetNodeName(&corev1.Pod{}); ok {
		t.Fatal("no target expected")
	}
}
