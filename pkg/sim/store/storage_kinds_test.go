package store

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStorageKindsRoundTrip(t *testing.T) {
	for _, kind := range []string{KindPersistentVolume, KindPersistentVolumeClaim, KindStorageClass, KindStatefulSet} {
		obj, err := NewObject(kind)
		if err != nil {
			t.Fatalf("NewObject(%s): %v", kind, err)
		}
		got, err := KindOf(obj)
		if err != nil || got != kind {
			t.Fatalf("KindOf(NewObject(%s)) = %s, %v", kind, got, err)
		}
		if _, ok := GVKFor(kind); !ok {
			t.Fatalf("no GVK for %s", kind)
		}
	}
	if !IsClusterScoped(KindPersistentVolume) || !IsClusterScoped(KindStorageClass) {
		t.Fatal("PersistentVolume and StorageClass are cluster-scoped")
	}
	if IsClusterScoped(KindPersistentVolumeClaim) || IsClusterScoped(KindStatefulSet) {
		t.Fatal("PersistentVolumeClaim and StatefulSet are namespaced")
	}
}

func TestStorageKindsStoredWithGVK(t *testing.T) {
	s := New(fixedNow)
	objs := []Object{
		&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "vault-1"}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "books", Namespace: "default"}},
		&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "vault-wrights"}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "archive", Namespace: "default"}},
	}
	for _, obj := range objs {
		created, err := s.Create(obj)
		if err != nil {
			t.Fatal(err)
		}
		if created.GetObjectKind().GroupVersionKind().Kind == "" {
			t.Fatalf("%T stored without a GVK", obj)
		}
	}
	sts, _ := Get[*appsv1.StatefulSet](s, KindStatefulSet, "default", "archive")
	if sts.Generation != 1 {
		t.Fatalf("StatefulSet generation = %d, want 1", sts.Generation)
	}
	pv, _ := Get[*corev1.PersistentVolume](s, KindPersistentVolume, "", "vault-1")
	if pv.Generation != 0 {
		t.Fatalf("PersistentVolume generation = %d, want 0 (untracked)", pv.Generation)
	}
}

func TestStatefulSetSpecChangeBumpsGeneration(t *testing.T) {
	s := New(fixedNow)
	created, _ := s.Create(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "archive", Namespace: "default"}})
	next := created.(*appsv1.StatefulSet).DeepCopy()
	two := int32(2)
	next.Spec.Replicas = &two
	updated, err := s.Update(next)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetGeneration() != 2 {
		t.Fatalf("generation = %d, want 2", updated.GetGeneration())
	}
}

func TestDefaultStorageClassAdmission(t *testing.T) {
	s := New(fixedNow)
	s.Create(&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "vault-wrights", Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"}}})
	made, _ := s.Create(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "books", Namespace: "default"}})
	if got := made.(*corev1.PersistentVolumeClaim).Spec.StorageClassName; got == nil || *got != "vault-wrights" {
		t.Fatalf("class = %v, want the default", got)
	}
	empty := ""
	manual, _ := s.Create(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "manual", Namespace: "default"}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: &empty}})
	if got := manual.(*corev1.PersistentVolumeClaim).Spec.StorageClassName; *got != "" {
		t.Fatalf("an explicit empty class was replaced by %q", *got)
	}
}
