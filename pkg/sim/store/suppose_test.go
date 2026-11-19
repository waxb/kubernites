package store

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestASupposedChangeIsSeenInsideAndLeavesNoTrace(t *testing.T) {
	s := New(func() metav1.Time { return metav1.Time{} })
	var heard int
	s.Watch(func(Event) { heard++ })
	if _, err := s.Create(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		if _, err := s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}}); err != nil {
			t.Fatal(err)
		}
	}
	rv, version, events := s.ResourceVersion(), s.KindVersion(KindPod), heard
	changed := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "default", Labels: map[string]string{"app": "x"}}}
	inside := []string{}
	s.Suppose([]Key{{Kind: KindPod, Namespace: "default", Name: "a"}}, []Object{changed}, func() {
		for _, p := range View[*corev1.Pod](s, KindPod, "default") {
			inside = append(inside, p.Name+"="+p.Labels["app"])
		}
		if s.KindVersion(KindPod) == version {
			t.Error("inside a supposition the pods' version is the real one: a cache built from the real pods would answer")
		}
	})
	if len(inside) != 2 || inside[0] != "b=" || inside[1] != "c=x" {
		t.Fatalf("inside the supposition the pods read %v, want b and the changed c", inside)
	}
	after := []string{}
	for _, p := range View[*corev1.Pod](s, KindPod, "default") {
		after = append(after, p.Name+"="+p.Labels["app"])
	}
	if len(after) != 3 || after[0] != "a=" || after[2] != "c=" {
		t.Fatalf("after the supposition the pods read %v, want a, b and c as they were", after)
	}
	if s.ResourceVersion() != rv || heard != events {
		t.Fatalf("a supposition wrote: resource version %d to %d, %d events heard", rv, s.ResourceVersion(), heard-events)
	}
	if s.KindVersion(KindPod) == version {
		t.Fatal("after a supposition the pods' version is unchanged: a cache built inside it would answer for the real pods")
	}
}

func TestTheStoreSaysWhileItSupposes(t *testing.T) {
	s := New(func() metav1.Time { return metav1.Time{} })
	inside := false
	s.Suppose(nil, nil, func() {
		s.Suppose(nil, nil, func() {})
		inside = s.Supposing()
	})
	if !inside || s.Supposing() {
		t.Fatalf("supposing inside %v, after %v: a cache of real views must not keep what a supposition saw", inside, s.Supposing())
	}
}
