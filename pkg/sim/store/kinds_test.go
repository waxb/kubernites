package store

import (
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKindRegistryRoundTrip(t *testing.T) {
	for _, kind := range []string{KindPod, KindNode, KindService, KindEvent, KindNamespace, KindConfigMap, KindReplicaSet, KindEndpointSlice, KindHorizontalPodAutoscaler} {
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
	if _, err := NewObject("Widget"); err == nil {
		t.Fatal("unknown kind accepted")
	}
	if !IsClusterScoped(KindNode) || IsClusterScoped(KindPod) || IsClusterScoped(KindHorizontalPodAutoscaler) {
		t.Fatal("scope wrong")
	}
	if (Key{Kind: "Node", Name: "a"}).String() != "Node/a" || (Key{Kind: "Pod", Namespace: "d", Name: "a"}).String() != "Pod/d/a" {
		t.Fatal("key string wrong")
	}
}

func warHorn(maxReplicas int32) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "lantern", Namespace: "default"},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "lantern"},
			MaxReplicas:    maxReplicas,
		},
	}
}

func TestHorizontalPodAutoscalerStoredWithGVKAndGeneration(t *testing.T) {
	s := cluster()
	created, err := s.Create(warHorn(5))
	if err != nil {
		t.Fatal(err)
	}
	gvk := created.GetObjectKind().GroupVersionKind()
	if gvk.Group != "autoscaling" || gvk.Version != "v2" || gvk.Kind != KindHorizontalPodAutoscaler {
		t.Fatalf("gvk = %v", gvk)
	}
	statusOnly := created.(*autoscalingv2.HorizontalPodAutoscaler).DeepCopy()
	statusOnly.Status.CurrentReplicas = 3
	updated, err := s.Update(statusOnly)
	if err != nil || updated.GetGeneration() != 1 {
		t.Fatalf("status update generation = %d err=%v", updated.GetGeneration(), err)
	}
	specChange := updated.(*autoscalingv2.HorizontalPodAutoscaler).DeepCopy()
	specChange.Spec.MaxReplicas = 9
	bumped, err := s.Update(specChange)
	if err != nil || bumped.GetGeneration() != 2 {
		t.Fatalf("spec update generation = %d err=%v", bumped.GetGeneration(), err)
	}
	got, ok := Get[*autoscalingv2.HorizontalPodAutoscaler](s, KindHorizontalPodAutoscaler, "default", "lantern")
	if !ok || got.Spec.MaxReplicas != 9 || got.Status.CurrentReplicas != 3 {
		t.Fatalf("got = %+v", got)
	}
	if len(List[*autoscalingv2.HorizontalPodAutoscaler](s, KindHorizontalPodAutoscaler, "")) != 1 {
		t.Fatal("list missed the HPA")
	}
}

func TestOnlyKindsWhoseStrategiesTrackItCarryAGeneration(t *testing.T) {
	s := cluster()
	untracked := []Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "parcels", Namespace: "default"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "notes", Namespace: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "harbor"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "stone-keep-1"}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "seen", Namespace: "default"}},
	}
	for _, obj := range untracked {
		created, err := s.Create(obj)
		if err != nil || created.GetGeneration() != 0 {
			t.Fatalf("%T created with generation %d, %v", obj, created.GetGeneration(), err)
		}
		next := copyOf(created)
		next.SetLabels(map[string]string{"tier": "1"})
		updated, err := s.Update(next)
		if err != nil || updated.GetGeneration() != 0 {
			t.Fatalf("%T updated to generation %d, %v", obj, updated.GetGeneration(), err)
		}
	}
	pod, err := s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "porter", Namespace: "default"}})
	if err != nil || pod.GetGeneration() != 1 {
		t.Fatalf("pod generation = %d, %v", pod.GetGeneration(), err)
	}
}

func TestRestoreDropsGenerationsTheirKindsNeverTrack(t *testing.T) {
	s := cluster()
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "parcels", Namespace: "default", Generation: 3}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "porter", Namespace: "default", Generation: 2}}
	if err := s.Restore([]Object{svc, pod}, 9, 2); err != nil {
		t.Fatal(err)
	}
	if got, _ := Get[*corev1.Service](s, KindService, "default", "parcels"); got.Generation != 0 {
		t.Fatalf("service generation = %d", got.Generation)
	}
	if got, _ := Get[*corev1.Pod](s, KindPod, "default", "porter"); got.Generation != 2 {
		t.Fatalf("pod generation = %d", got.Generation)
	}
}

func TestServiceAccountsAreANamespacedCoreKind(t *testing.T) {
	obj, err := NewObject(KindServiceAccount)
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := KindOf(obj); err != nil || kind != "ServiceAccount" {
		t.Fatalf("KindOf = %q, %v", kind, err)
	}
	if gvk, ok := GVKFor(KindServiceAccount); !ok || gvk.Group != "" || gvk.Version != "v1" {
		t.Fatalf("gvk = %v", gvk)
	}
	if IsClusterScoped(KindServiceAccount) {
		t.Fatal("a ServiceAccount lives in a namespace")
	}
	s := cluster()
	created, err := s.Create(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "guild"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetGeneration() != 0 {
		t.Fatalf("generation = %d", created.GetGeneration())
	}
	if _, ok := Get[*corev1.ServiceAccount](s, KindServiceAccount, "guild", "default"); !ok {
		t.Fatal("not stored")
	}
}
