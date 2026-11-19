package store

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func priorityClass(name string, value int32, globalDefault bool, policy *corev1.PreemptionPolicy) *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		ObjectMeta:       metav1.ObjectMeta{Name: name},
		Value:            value,
		GlobalDefault:    globalDefault,
		PreemptionPolicy: policy,
	}
}

func policyOf(p corev1.PreemptionPolicy) *corev1.PreemptionPolicy {
	return &p
}

func rankedPod(name, class string) *corev1.Pod {
	p := newPod(name)
	p.Spec.PriorityClassName = class
	return p
}

func mustCreate(t *testing.T, s *Store, obj Object) Object {
	t.Helper()
	created, err := s.Create(obj)
	if err != nil {
		t.Fatalf("create %s: %v", obj.GetName(), err)
	}
	return created
}

func TestPriorityClassIsAClusterScopedSchedulingKind(t *testing.T) {
	obj, err := NewObject(KindPriorityClass)
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := KindOf(obj); err != nil || kind != KindPriorityClass {
		t.Fatalf("KindOf = %s, %v", kind, err)
	}
	gvk, ok := GVKFor(KindPriorityClass)
	if !ok || gvk.Group != "scheduling.k8s.io" || gvk.Version != "v1" || gvk.Kind != "PriorityClass" {
		t.Fatalf("gvk = %v", gvk)
	}
	if !IsClusterScoped(KindPriorityClass) {
		t.Fatal("PriorityClass must be cluster-scoped")
	}
	s := cluster()
	created := mustCreate(t, s, priorityClass("realm-high", 1000, false, nil))
	if created.GetObjectKind().GroupVersionKind() != gvk || created.GetNamespace() != "" {
		t.Fatalf("stored = %v in %q", created.GetObjectKind().GroupVersionKind(), created.GetNamespace())
	}
}

func TestPodPriorityIsResolvedFromItsClass(t *testing.T) {
	s := cluster()
	mustCreate(t, s, priorityClass("realm-high", 1000, false, policyOf(corev1.PreemptLowerPriority)))
	mustCreate(t, s, priorityClass("patient", 500, false, policyOf(corev1.PreemptNever)))
	high := mustCreate(t, s, rankedPod("high", "realm-high")).(*corev1.Pod)
	if high.Spec.Priority == nil || *high.Spec.Priority != 1000 || *high.Spec.PreemptionPolicy != corev1.PreemptLowerPriority {
		t.Fatalf("high spec = %+v", high.Spec)
	}
	patient := mustCreate(t, s, rankedPod("patient", "patient")).(*corev1.Pod)
	if *patient.Spec.Priority != 500 || *patient.Spec.PreemptionPolicy != corev1.PreemptNever {
		t.Fatalf("patient spec = %+v", patient.Spec)
	}
}

func TestPodWithoutClassGetsZeroOrTheGlobalDefault(t *testing.T) {
	s := cluster()
	plain := mustCreate(t, s, newPod("plain")).(*corev1.Pod)
	if plain.Spec.PriorityClassName != "" || *plain.Spec.Priority != 0 || *plain.Spec.PreemptionPolicy != corev1.PreemptLowerPriority {
		t.Fatalf("plain spec = %+v", plain.Spec)
	}
	mustCreate(t, s, priorityClass("everyone", 50, true, policyOf(corev1.PreemptNever)))
	later := mustCreate(t, s, newPod("later")).(*corev1.Pod)
	if later.Spec.PriorityClassName != "everyone" || *later.Spec.Priority != 50 || *later.Spec.PreemptionPolicy != corev1.PreemptNever {
		t.Fatalf("later spec = %+v", later.Spec)
	}
	if again := storedPod(t, s, "plain"); *again.Spec.Priority != 0 {
		t.Fatal("a new global default changed an existing pod")
	}
}

func TestUnknownClassIsForbiddenWithTheRealMessage(t *testing.T) {
	s := cluster()
	_, err := s.Create(rankedPod("lost", "realm-high"))
	var forbidden *ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("err = %v", err)
	}
	if err.Error() != `pods "lost" is forbidden: no PriorityClass with name realm-high was found` {
		t.Fatalf("message = %q", err.Error())
	}
	if _, ok := Get[*corev1.Pod](s, KindPod, "default", "lost"); ok {
		t.Fatal("forbidden pod was stored")
	}
}

func TestExplicitPriorityMustMatchTheClass(t *testing.T) {
	s := cluster()
	mustCreate(t, s, priorityClass("realm-high", 1000, false, policyOf(corev1.PreemptLowerPriority)))
	wrong := rankedPod("wrong", "realm-high")
	five := int32(5)
	wrong.Spec.Priority = &five
	_, err := s.Create(wrong)
	want := `pods "wrong" is forbidden: the integer value of priority (5) must not be provided in pod spec; priority admission controller computed 1000 from the given PriorityClass name`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
	never := rankedPod("never", "realm-high")
	never.Spec.PreemptionPolicy = policyOf(corev1.PreemptNever)
	_, err = s.Create(never)
	want = `pods "never" is forbidden: the string value of PreemptionPolicy (Never) must not be provided in pod spec; priority admission controller computed PreemptLowerPriority from the given PriorityClass name`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
	right := rankedPod("right", "realm-high")
	thousand := int32(1000)
	right.Spec.Priority = &thousand
	mustCreate(t, s, right)
}

func TestPodUpdateKeepsItsAdmittedPriority(t *testing.T) {
	s := cluster()
	mustCreate(t, s, priorityClass("realm-high", 1000, false, policyOf(corev1.PreemptLowerPriority)))
	mustCreate(t, s, rankedPod("high", "realm-high"))
	if err := s.Delete(Key{Kind: KindPriorityClass, Name: "realm-high"}); err != nil {
		t.Fatal(err)
	}
	next := storedPod(t, s, "high")
	next.Spec.Priority, next.Spec.PreemptionPolicy = nil, nil
	next.Labels = map[string]string{"app": "lantern"}
	updated, err := s.Update(next)
	if err != nil {
		t.Fatalf("update after the class was deleted: %v", err)
	}
	spec := updated.(*corev1.Pod).Spec
	if *spec.Priority != 1000 || *spec.PreemptionPolicy != corev1.PreemptLowerPriority {
		t.Fatalf("spec = %+v", spec)
	}
}

func TestOnlyOneGlobalDefaultClass(t *testing.T) {
	s := cluster()
	mustCreate(t, s, priorityClass("first", 10, true, nil))
	_, err := s.Create(priorityClass("second", 20, true, nil))
	want := `priorityclasses.scheduling.k8s.io "second" is forbidden: PriorityClass first is already marked as default. Only one default can exist`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
	quiet := mustCreate(t, s, priorityClass("quiet", 30, false, nil)).(*schedulingv1.PriorityClass)
	promoted := quiet.DeepCopy()
	promoted.GlobalDefault = true
	if _, err := s.Update(promoted); err == nil {
		t.Fatal("a second default was allowed by update")
	}
	first, _ := Get[*schedulingv1.PriorityClass](s, KindPriorityClass, "", "first")
	first.Description = "the realm's default"
	if _, err := s.Update(first); err != nil {
		t.Fatalf("updating the default itself: %v", err)
	}
}

func TestLowestGlobalDefaultWinsWhenSeveralWereRestored(t *testing.T) {
	s := cluster()
	err := s.Restore([]Object{priorityClass("b-high", 90, true, nil), priorityClass("a-low", 40, true, nil), defaultAccount("default")}, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	pod := mustCreate(t, s, newPod("p")).(*corev1.Pod)
	if pod.Spec.PriorityClassName != "a-low" || *pod.Spec.Priority != 40 || pod.Spec.PreemptionPolicy != nil {
		t.Fatalf("spec = %+v", pod.Spec)
	}
}
