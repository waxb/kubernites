package store

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentGenerationFollowsSpecAndAnnotationsButNotStatusWrites(t *testing.T) {
	s := cluster()
	one := int32(1)
	created, err := s.Create(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "porters", Namespace: "default"}, Spec: appsv1.DeploymentSpec{Replicas: &one}})
	if err != nil || created.GetObjectKind().GroupVersionKind().Kind != KindDeployment {
		t.Fatalf("create = %v %v", created, err)
	}
	status := created.(*appsv1.Deployment).DeepCopy()
	status.Annotations = map[string]string{"deployment.kubernetes.io/revision": "1"}
	status.Status.Replicas = 1
	written, err := s.UpdateStatus(status)
	if err != nil || written.GetGeneration() != 1 {
		t.Fatalf("status write generation = %d, %v", written.GetGeneration(), err)
	}
	annotated := written.(*appsv1.Deployment).DeepCopy()
	annotated.Annotations = map[string]string{"kubernetes.io/change-cause": "porters"}
	bumped, err := s.Update(annotated)
	if err != nil || bumped.GetGeneration() != 2 {
		t.Fatalf("annotation update generation = %d, %v", bumped.GetGeneration(), err)
	}
	two := int32(2)
	scaled := bumped.(*appsv1.Deployment).DeepCopy()
	scaled.Spec.Replicas = &two
	again, _ := s.Update(scaled)
	if again.GetGeneration() != 3 {
		t.Fatalf("spec update generation = %d", again.GetGeneration())
	}
	if obj, err := NewObject(KindDeployment); err != nil || obj == nil {
		t.Fatal("no Deployment constructor")
	}
}

func TestPodsAreAdmittedWithTheDefaultServiceAccount(t *testing.T) {
	s := cluster()
	if _, err := s.Create(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "clerk", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	plain, _ := s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"}})
	mirror, _ := s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "default", Annotations: map[string]string{corev1.MirrorPodAnnotationKey: "x"}}})
	named, _ := s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "default"}, Spec: corev1.PodSpec{ServiceAccountName: "clerk"}})
	if plain.(*corev1.Pod).Spec.ServiceAccountName != "default" || mirror.(*corev1.Pod).Spec.ServiceAccountName != "" || named.(*corev1.Pod).Spec.ServiceAccountName != "clerk" {
		t.Fatal("service account defaulting is wrong")
	}
}
