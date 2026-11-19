package store

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDaemonSetStoredWithGVKAndGeneration(t *testing.T) {
	obj, err := NewObject(KindDaemonSet)
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := KindOf(obj); err != nil || kind != KindDaemonSet {
		t.Fatalf("KindOf = %s, %v", kind, err)
	}
	s := New(fixedNow)
	created, err := s.Create(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "warden", Namespace: "default"}})
	if err != nil {
		t.Fatal(err)
	}
	gvk := created.GetObjectKind().GroupVersionKind()
	if gvk.Group != "apps" || gvk.Version != "v1" || gvk.Kind != KindDaemonSet || created.GetGeneration() != 1 {
		t.Fatalf("gvk = %v generation = %d", gvk, created.GetGeneration())
	}
	status := created.(*appsv1.DaemonSet).DeepCopy()
	status.Status.NumberReady = 2
	updated, _ := s.Update(status)
	if updated.GetGeneration() != 1 {
		t.Fatalf("status update bumped generation to %d", updated.GetGeneration())
	}
	spec := updated.(*appsv1.DaemonSet).DeepCopy()
	spec.Spec.MinReadySeconds = 5
	updated, _ = s.Update(spec)
	if updated.GetGeneration() != 2 {
		t.Fatalf("spec update generation = %d", updated.GetGeneration())
	}
}

func TestDaemonSetTemplateGenerationAnnotationFollowsTheTemplate(t *testing.T) {
	s := New(fixedNow)
	created, _ := s.Create(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "warden", Namespace: "default"}})
	if got := created.GetAnnotations()[appsv1.DeprecatedTemplateGeneration]; got != "1" {
		t.Fatalf("created template generation = %q", got)
	}
	minReady := created.(*appsv1.DaemonSet).DeepCopy()
	minReady.Spec.MinReadySeconds = 3
	minReady.Annotations = nil
	updated, _ := s.Update(minReady)
	if got := updated.GetAnnotations()[appsv1.DeprecatedTemplateGeneration]; got != "1" {
		t.Fatalf("a spec change outside the template moved it to %q", got)
	}
	image := updated.(*appsv1.DaemonSet).DeepCopy()
	image.Spec.Template.Labels = map[string]string{"app": "warden"}
	image.Annotations[appsv1.DeprecatedTemplateGeneration] = "9"
	updated, _ = s.Update(image)
	if got := updated.GetAnnotations()[appsv1.DeprecatedTemplateGeneration]; got != "2" {
		t.Fatalf("template change: template generation = %q, want 2", got)
	}
}
