package store

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestControllerRevisionStoredWithGVKAndNoGeneration(t *testing.T) {
	obj, err := NewObject(KindControllerRevision)
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := KindOf(obj); err != nil || kind != KindControllerRevision {
		t.Fatalf("KindOf = %s, %v", kind, err)
	}
	s := New(fixedNow)
	created, err := s.Create(&appsv1.ControllerRevision{ObjectMeta: metav1.ObjectMeta{Name: "warden-5d8f", Namespace: "default"},
		Data: runtime.RawExtension{Raw: []byte(`{"spec":{}}`)}, Revision: 1})
	if err != nil {
		t.Fatal(err)
	}
	gvk := created.GetObjectKind().GroupVersionKind()
	if gvk.Group != "apps" || gvk.Version != "v1" || gvk.Kind != KindControllerRevision || created.GetGeneration() != 0 || IsClusterScoped(KindControllerRevision) {
		t.Fatalf("gvk = %v generation = %d", gvk, created.GetGeneration())
	}
}
