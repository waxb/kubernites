package store

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func finalized(name string, finalizers ...string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Finalizers: finalizers}}
}

func recordEvents(s *Store) *[]EventType {
	seen := []EventType{}
	s.Watch(func(e Event) { seen = append(seen, e.Type) })
	return &seen
}

func TestDeleteWithFinalizersMarksTheObjectDeleting(t *testing.T) {
	s := New(fixedNow)
	s.Create(finalized("scroll", "example.com/seal"))
	events := recordEvents(s)
	if err := s.Delete(Key{Kind: KindConfigMap, Namespace: "default", Name: "scroll"}); err != nil {
		t.Fatal(err)
	}
	cm, ok := Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll")
	if !ok || cm.DeletionTimestamp == nil || cm.DeletionGracePeriodSeconds == nil || *cm.DeletionGracePeriodSeconds != 0 {
		t.Fatalf("object = %+v, want it kept with deletionTimestamp and grace 0", cm)
	}
	if len(*events) != 1 || (*events)[0] != Modified {
		t.Fatalf("events = %v, want one MODIFIED", *events)
	}
	if err := s.Delete(Key{Kind: KindConfigMap, Namespace: "default", Name: "scroll"}); err != nil || len(*events) != 1 {
		t.Fatalf("a second delete: err = %v, events = %v", err, *events)
	}
}

func TestRemovingTheLastFinalizerDeletesTheObject(t *testing.T) {
	s := New(fixedNow)
	s.Create(finalized("scroll", "a.io/one", "b.io/two"))
	s.Delete(Key{Kind: KindConfigMap, Namespace: "default", Name: "scroll"})
	cm, _ := Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll")
	cm.Finalizers = []string{"b.io/two"}
	if _, err := s.Update(cm); err != nil {
		t.Fatal(err)
	}
	if _, ok := Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll"); !ok {
		t.Fatal("deleted while a finalizer remained")
	}
	events := recordEvents(s)
	cm, _ = Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll")
	cm.Finalizers = nil
	if _, err := s.Update(cm); err != nil {
		t.Fatal(err)
	}
	if _, ok := Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll"); ok {
		t.Fatal("kept after its last finalizer was removed")
	}
	if len(*events) != 1 || (*events)[0] != Deleted {
		t.Fatalf("events = %v, want one DELETED", *events)
	}
}

func TestNoNewFinalizersOnADeletingObject(t *testing.T) {
	s := New(fixedNow)
	s.Create(finalized("scroll", "a.io/one"))
	s.Delete(Key{Kind: KindConfigMap, Namespace: "default", Name: "scroll"})
	cm, _ := Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll")
	cm.Finalizers = append(cm.Finalizers, "x.io/late")
	_, err := s.Update(cm)
	want := `The ConfigMap "scroll" is invalid: metadata.finalizers: Forbidden: no new finalizers can be added if the object is being deleted, found new finalizers []string{"x.io/late"}`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v\nwant %s", err, want)
	}
}

func TestAnUpdateCannotUndoADeletion(t *testing.T) {
	s := New(fixedNow)
	s.Create(finalized("scroll", "a.io/one"))
	s.Delete(Key{Kind: KindConfigMap, Namespace: "default", Name: "scroll"})
	cm, _ := Get[*corev1.ConfigMap](s, KindConfigMap, "default", "scroll")
	cm.DeletionTimestamp, cm.DeletionGracePeriodSeconds = nil, nil
	cm.Data = map[string]string{"k": "v"}
	updated, err := s.Update(cm)
	if err != nil || updated.GetDeletionTimestamp() == nil {
		t.Fatalf("err = %v, deletionTimestamp cleared by an update", err)
	}
}

func TestMarkingAGenerationTrackedObjectDeletingBumpsItsGeneration(t *testing.T) {
	s := New(fixedNow)
	s.Create(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "muniment", Namespace: "default", Finalizers: []string{"a.io/one"}}})
	s.Delete(Key{Kind: KindStatefulSet, Namespace: "default", Name: "muniment"})
	sts, _ := Get[*appsv1.StatefulSet](s, KindStatefulSet, "default", "muniment")
	if sts.Generation != 2 {
		t.Fatalf("generation = %d, want 2", sts.Generation)
	}
}

func TestStorageObjectsAreProtectedAtCreation(t *testing.T) {
	s := New(fixedNow)
	pvc, _ := s.Create(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "books", Namespace: "default"}})
	pv, _ := s.Create(&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "vault-1"}})
	if got := strings.Join(pvc.GetFinalizers(), ","); got != "kubernetes.io/pvc-protection" {
		t.Fatalf("claim finalizers = %q", got)
	}
	if got := strings.Join(pv.GetFinalizers(), ","); got != "kubernetes.io/pv-protection" {
		t.Fatalf("volume finalizers = %q", got)
	}
}
