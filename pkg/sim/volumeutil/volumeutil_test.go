package volumeutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func keep(name, zone string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{corev1.LabelTopologyZone: zone}}}
}

func TestFitsNodeFollowsTheVolumesNodeAffinity(t *testing.T) {
	pv := &corev1.PersistentVolume{Spec: corev1.PersistentVolumeSpec{NodeAffinity: ZoneAffinity("isle-home")}}
	if !FitsNode(pv, keep("tower", "isle-home")) || FitsNode(pv, keep("morrow", "isle-morrow")) {
		t.Fatal("zone affinity not honoured")
	}
	if ZoneOf(pv) != "isle-home" {
		t.Fatalf("ZoneOf = %q", ZoneOf(pv))
	}
	byName := &corev1.PersistentVolume{Spec: corev1.PersistentVolumeSpec{NodeAffinity: &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchFields: []corev1.NodeSelectorRequirement{{Key: "metadata.name", Operator: corev1.NodeSelectorOpIn, Values: []string{"tower"}}}}},
	}}}}
	if !FitsNode(byName, keep("tower", "a")) || FitsNode(byName, keep("morrow", "a")) {
		t.Fatal("matchFields on metadata.name not honoured")
	}
	if !FitsNode(&corev1.PersistentVolume{}, keep("any", "")) {
		t.Fatal("a volume without affinity fits every node")
	}
}
