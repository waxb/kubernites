package store

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/limitranger"
)

func resources(pairs ...string) corev1.ResourceList {
	out := corev1.ResourceList{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out[corev1.ResourceName(pairs[i])] = resource.MustParse(pairs[i+1])
	}
	return out
}

func workerDefaults(namespace string) *corev1.LimitRange {
	return &corev1.LimitRange{ObjectMeta: metav1.ObjectMeta{Name: "resource-defaults", Namespace: namespace},
		Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{Type: corev1.LimitTypeContainer,
			DefaultRequest: resources("cpu", "200m", "memory", "64Mi"), Default: resources("cpu", "500m", "memory", "1Gi")}}}}
}

func withRange(t *testing.T) *Store {
	t.Helper()
	s := cluster("kube-system")
	if _, err := s.Create(workerDefaults("default")); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLimitRangeIsStoredAsACoreKind(t *testing.T) {
	s := withRange(t)
	lr, ok := Get[*corev1.LimitRange](s, KindLimitRange, "default", "resource-defaults")
	if !ok || lr.APIVersion != "v1" || lr.Kind != KindLimitRange || lr.Generation != 0 {
		t.Fatalf("stored = %+v", lr)
	}
	next := lr.DeepCopy()
	next.Spec.Limits[0].Default = resources("cpu", "1")
	updated, err := s.Update(next)
	if err != nil || updated.GetGeneration() != 0 {
		t.Fatalf("update generation %d err %v", updated.GetGeneration(), err)
	}
	obj, err := NewObject(KindLimitRange)
	if kind, _ := KindOf(obj); err != nil || kind != KindLimitRange {
		t.Fatalf("registry: %v %v", kind, err)
	}
}

func TestNewPodsTakeTheDefaultsAndExistingPodsKeepTheirSpec(t *testing.T) {
	s := cluster()
	if _, err := s.Create(newPod("before")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(workerDefaults("default")); err != nil {
		t.Fatal(err)
	}
	after, err := s.Create(newPod("after"))
	if err != nil {
		t.Fatal(err)
	}
	res := after.(*corev1.Pod).Spec.Containers[0].Resources
	if res.Requests.Cpu().String() != "200m" || res.Limits.Memory().String() != "1Gi" || after.GetAnnotations()[limitranger.Annotation] == "" {
		t.Fatalf("defaulted = %+v annotations %v", res, after.GetAnnotations())
	}
	before := storedPod(t, s, "before")
	if len(before.Spec.Containers[0].Resources.Requests) != 0 {
		t.Fatalf("an existing pod changed: %+v", before.Spec.Containers[0].Resources)
	}
	before.Status.Phase = corev1.PodRunning
	if _, err := s.UpdateStatus(before); err != nil {
		t.Fatal(err)
	}
	if len(storedPod(t, s, "before").Spec.Containers[0].Resources.Requests) != 0 {
		t.Fatal("an update applied the defaults: LimitRanger skips pod updates")
	}
}

func TestOtherNamespacesAreUntouched(t *testing.T) {
	s := withRange(t)
	p := newPod("proxy")
	p.Namespace = "kube-system"
	created, err := s.Create(p)
	if err != nil || len(created.(*corev1.Pod).Spec.Containers[0].Resources.Requests) != 0 {
		t.Fatalf("kube-system pod = %+v err %v", created, err)
	}
}

func TestALimitOnlyContainerRequestsItsLimit(t *testing.T) {
	s := withRange(t)
	p := newPod("door")
	p.Spec.Containers[0].Resources = corev1.ResourceRequirements{Limits: resources("cpu", "300m")}
	created, err := s.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	res := created.(*corev1.Pod).Spec.Containers[0].Resources
	if res.Requests.Cpu().String() != "300m" || res.Requests.Memory().String() != "64Mi" {
		t.Fatalf("requests = %v", res.Requests)
	}
}

func TestARequestAboveTheDefaultLimitIsInvalid(t *testing.T) {
	s := withRange(t)
	p := newPod("greedy")
	p.Spec.Containers[0].Resources = corev1.ResourceRequirements{Requests: resources("cpu", "700m")}
	_, err := s.Create(p)
	var invalid *InvalidError
	if !errors.As(err, &invalid) || err.Error() != `Pod "greedy" is invalid: spec.containers[0].resources.requests: Invalid value: "700m": must be less than or equal to cpu limit of 500m` {
		t.Fatalf("err = %v", err)
	}
	two := &InvalidError{Kind: KindPod, Name: "x", Causes: []string{"a", "b"}}
	if two.Error() != `Pod "x" is invalid: [a, b]` {
		t.Fatalf("aggregate = %q", two.Error())
	}
}

func TestAMinimumBreachIsForbiddenLikeTheAdmissionPlugin(t *testing.T) {
	s := cluster()
	lr := workerDefaults("default")
	lr.Spec.Limits[0].Min = resources("memory", "128Mi")
	if _, err := s.Create(lr); err != nil {
		t.Fatal(err)
	}
	_, err := s.Create(newPod("small"))
	var forbidden *ForbiddenError
	if !errors.As(err, &forbidden) || err.Error() != `pods "small" is forbidden: minimum memory usage per Container is 128Mi, but request is 64Mi` {
		t.Fatalf("err = %v", err)
	}
}
