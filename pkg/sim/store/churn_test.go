package store

import (
	"fmt"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestViewStaysSortedUnderChurn(t *testing.T) {
	s := cluster("a", "b", "d")
	names := []string{}
	for i := 0; i < 60; i++ {
		name := fmt.Sprintf("p-%02d", (i*37)%60)
		s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: []string{"a", "b"}[i%2]}})
		if i%3 == 0 {
			View[*corev1.Pod](s, KindPod, "")
		}
		if i%7 == 0 {
			s.Delete(Key{Kind: KindPod, Namespace: []string{"a", "b"}[i%2], Name: name})
			continue
		}
		names = append(names, []string{"a", "b"}[i%2]+"/"+name)
	}
	sort.Strings(names)
	got := []string{}
	for _, p := range View[*corev1.Pod](s, KindPod, "") {
		got = append(got, p.Namespace+"/"+p.Name)
	}
	if fmt.Sprint(got) != fmt.Sprint(names) {
		t.Fatalf("view = %v\nwant %v", got, names)
	}
}

func TestAHeldViewIsNotChangedByLaterWrites(t *testing.T) {
	s := cluster("a", "b", "d")
	s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "d"}})
	s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "d", Namespace: "d"}})
	held := s.keysOf(KindPod)
	s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "d"}})
	s.Delete(Key{Kind: KindPod, Namespace: "d", Name: "d"})
	if len(held) != 2 || held[0].Name != "b" || held[1].Name != "d" {
		t.Fatalf("held keys changed to %v", held)
	}
}

func TestKindVersionMovesOnlyWithItsKind(t *testing.T) {
	s := cluster("a", "b", "d")
	before := s.KindVersion(KindPod)
	s.Create(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "scroll", Namespace: "d"}})
	if s.KindVersion(KindPod) != before {
		t.Fatal("a ConfigMap write moved the Pod version")
	}
	made, _ := s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "d"}})
	created := s.KindVersion(KindPod)
	if created == before {
		t.Fatal("creating a Pod did not move the Pod version")
	}
	s.UpdateStatus(made)
	updated := s.KindVersion(KindPod)
	s.Delete(Key{Kind: KindPod, Namespace: "d", Name: "p"})
	if !(updated > created && s.KindVersion(KindPod) > updated) {
		t.Fatal("updates and deletes must move the Pod version")
	}
	snap := s.Snapshot()
	versionBefore := s.KindVersion(KindPod)
	s.Restore(snap.Objects, snap.ResourceVersion, snap.UIDSeq)
	if s.KindVersion(KindPod) == versionBefore {
		t.Fatal("a restore must move every kind's version")
	}
}
