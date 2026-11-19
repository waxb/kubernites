package volumeutil

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func vault(name, size, class string, modes ...corev1.PersistentVolumeAccessMode) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{"shelf": name}},
		Spec: corev1.PersistentVolumeSpec{
			Capacity:         corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)},
			AccessModes:      modes,
			StorageClassName: class,
		},
		Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeAvailable},
	}
}

func deed(name, size string, class *string, modes ...corev1.PersistentVolumeAccessMode) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID("uid-" + name)},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      modes,
			StorageClassName: class,
			Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)}},
		},
	}
}

func named(s string) *string { return &s }

func TestBoundToClaimChecksNamespaceNameAndUID(t *testing.T) {
	pvc := deed("books", "1Gi", nil)
	pv := vault("v1", "1Gi", "")
	if BoundToClaim(pv, pvc) {
		t.Fatal("a volume without a claimRef is bound to nothing")
	}
	pv.Spec.ClaimRef = &corev1.ObjectReference{Namespace: "default", Name: "books"}
	if !BoundToClaim(pv, pvc) {
		t.Fatal("a claimRef without a UID binds by name")
	}
	pv.Spec.ClaimRef.UID = "other"
	if BoundToClaim(pv, pvc) {
		t.Fatal("a claimRef with another UID belongs to an older claim of that name")
	}
}

func TestMatchesClaimNeedsSizeClassModesAndSelector(t *testing.T) {
	rwo := corev1.ReadWriteOnce
	pvc := deed("books", "2Gi", named("vault-wrights"), rwo)
	cases := []struct {
		name string
		pv   *corev1.PersistentVolume
		want bool
	}{
		{"fits", vault("v", "2Gi", "vault-wrights", rwo), true},
		{"larger fits", vault("v", "5Gi", "vault-wrights", rwo, corev1.ReadOnlyMany), true},
		{"too small", vault("v", "1Gi", "vault-wrights", rwo), false},
		{"other class", vault("v", "2Gi", "", rwo), false},
		{"missing mode", vault("v", "2Gi", "vault-wrights", corev1.ReadOnlyMany), false},
	}
	for _, c := range cases {
		if got := MatchesClaim(c.pv, pvc); got != c.want {
			t.Errorf("%s: MatchesClaim = %v, want %v", c.name, got, c.want)
		}
	}
	deleting := vault("v", "2Gi", "vault-wrights", rwo)
	deleting.DeletionTimestamp = &metav1.Time{Time: time.Unix(1, 0)}
	if MatchesClaim(deleting, pvc) {
		t.Error("a volume being deleted matches no claim")
	}
	pvc.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"shelf": "v2"}}
	if MatchesClaim(vault("v1", "2Gi", "vault-wrights", rwo), pvc) || !MatchesClaim(vault("v2", "2Gi", "vault-wrights", rwo), pvc) {
		t.Error("the claim's selector must pick the volume by its labels")
	}
}

func TestFindMatchingVolumePrefersTheBoundThenTheSmallest(t *testing.T) {
	pvc := deed("books", "1Gi", named(""))
	big, small, taken := vault("big", "8Gi", ""), vault("small", "2Gi", ""), vault("taken", "1Gi", "")
	taken.Spec.ClaimRef = &corev1.ObjectReference{Namespace: "default", Name: "someone-else"}
	pending := vault("pending", "1Gi", "")
	pending.Status.Phase = corev1.VolumePending
	if got := FindMatchingVolume(pvc, []*corev1.PersistentVolume{big, taken, pending, small}, nil, false); got != small {
		t.Fatalf("want the smallest available match, got %v", got)
	}
	bound := vault("bound", "9Gi", "")
	bound.Spec.ClaimRef = &corev1.ObjectReference{Namespace: "default", Name: "books"}
	if got := FindMatchingVolume(pvc, []*corev1.PersistentVolume{small, bound}, nil, false); got != bound {
		t.Fatalf("a volume already bound to the claim wins, got %v", got)
	}
	if got := FindMatchingVolume(pvc, []*corev1.PersistentVolume{small}, nil, true); got != nil {
		t.Fatalf("a delayed class binds only once a node is chosen, got %v", got)
	}
}

func TestFindMatchingVolumeHonoursTheChosenNode(t *testing.T) {
	pvc := deed("books", "1Gi", named(""))
	home, away := vault("home", "1Gi", ""), vault("away", "1Gi", "")
	home.Spec.NodeAffinity, away.Spec.NodeAffinity = ZoneAffinity("isle-home"), ZoneAffinity("isle-away")
	if got := FindMatchingVolume(pvc, []*corev1.PersistentVolume{away, home}, keep("tower", "isle-home"), true); got != home {
		t.Fatalf("want the volume on the node's isle, got %v", got)
	}
	away.Spec.ClaimRef = &corev1.ObjectReference{Namespace: "default", Name: "books"}
	if got := FindMatchingVolume(pvc, []*corev1.PersistentVolume{away, home}, keep("tower", "isle-home"), true); got != nil {
		t.Fatalf("a claim bound to a volume on another isle cannot use this node, got %v", got)
	}
}

func TestZonesAndClaimNames(t *testing.T) {
	if ZoneOf(&corev1.PersistentVolume{}) != "" || NodeZone(keep("tower", "isle-home")) != "isle-home" {
		t.Fatal("zones")
	}
	pod := &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{
		{Name: "scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "books", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "books-archive-0"}}},
	}}}
	if got := ClaimNames(pod); len(got) != 1 || got[0] != "books-archive-0" {
		t.Fatalf("ClaimNames = %v", got)
	}
}

func TestNodeSelectorRequirementsFollowTheirOperators(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "tower", Labels: map[string]string{"tier": "3"}}}
	term := func(op corev1.NodeSelectorOperator, values ...string) corev1.NodeSelectorTerm {
		return corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "tier", Operator: op, Values: values}}}
	}
	cases := []struct {
		term corev1.NodeSelectorTerm
		want bool
	}{
		{term(corev1.NodeSelectorOpGt, "2"), true},
		{term(corev1.NodeSelectorOpLt, "2"), false},
		{term(corev1.NodeSelectorOpNotIn, "3"), false},
		{term(corev1.NodeSelectorOpExists), true},
		{term(corev1.NodeSelectorOpDoesNotExist), false},
		{term("Near", "3"), false},
		{corev1.NodeSelectorTerm{}, false},
	}
	for i, c := range cases {
		if got := termMatches(c.term, node); got != c.want {
			t.Errorf("case %d: termMatches = %v, want %v", i, got, c.want)
		}
	}
}

func TestStorageClassesAndModes(t *testing.T) {
	wait := storagev1.VolumeBindingWaitForFirstConsumer
	if DelaysBinding(nil) || !DelaysBinding(&storagev1.StorageClass{VolumeBindingMode: &wait}) {
		t.Fatal("WaitForFirstConsumer delays binding, and no class does not")
	}
	if !ModesCover([]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany}, []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}) {
		t.Fatal("modes")
	}
	pvc := deed("books", "1Gi", nil)
	if HasClass(pvc) || ClassOf(pvc) != "" {
		t.Fatal("a claim without a class names none")
	}
	pvc.Annotations = map[string]string{corev1.BetaStorageClassAnnotation: "old-wrights"}
	if !HasClass(pvc) || ClassOf(pvc) != "old-wrights" {
		t.Fatal("the beta annotation still names the class")
	}
	pvc.Spec.StorageClassName = named("vault-wrights")
	if ClassOf(pvc) != "vault-wrights" {
		t.Fatal("the spec field wins over the beta annotation")
	}
}

func TestTheNewestDefaultClassWins(t *testing.T) {
	class := func(name string, at int64, def string) *storagev1.StorageClass {
		return &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.Unix(at, 0),
			Annotations: map[string]string{AnnDefaultClass: def}}}
	}
	old, newer, plain := class("old", 10, "true"), class("newer", 20, "true"), class("plain", 30, "false")
	if got := DefaultClass([]*storagev1.StorageClass{old, plain, newer}); got != newer {
		t.Fatalf("want the newest default class, got %v", got)
	}
	a, b := class("b-wrights", 20, "true"), class("a-wrights", 20, "true")
	if got := DefaultClass([]*storagev1.StorageClass{a, b}); got != b {
		t.Fatalf("a tie goes to the first name, got %v", got)
	}
	beta := &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"storageclass.beta.kubernetes.io/is-default-class": "true"}}}
	if !IsDefaultClass(beta) || DefaultClass([]*storagev1.StorageClass{plain}) != nil {
		t.Fatal("default class annotations")
	}
}
