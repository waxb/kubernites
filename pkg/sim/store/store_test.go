package store

import (
	"errors"
	"strconv"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fixedNow() metav1.Time {
	return metav1.NewTime(time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC))
}

func defaultAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: namespace}}
}

func cluster(namespaces ...string) *Store {
	s := New(fixedNow)
	for _, ns := range append([]string{"default"}, namespaces...) {
		if _, err := s.Create(defaultAccount(ns)); err != nil {
			panic(err)
		}
	}
	return s
}

func newPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "lantern", Image: "lantern:1.0"}}},
	}
}

func TestCreateSetsMetadataAndTypeMeta(t *testing.T) {
	s := cluster()
	want := strconv.FormatUint(s.ResourceVersion()+1, 10)
	created, err := s.Create(newPod("a"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.GetUID() == "" || created.GetResourceVersion() != want {
		t.Fatalf("uid=%q rv=%q", created.GetUID(), created.GetResourceVersion())
	}
	if created.GetCreationTimestamp().Time.IsZero() {
		t.Fatal("creationTimestamp not set")
	}
	gvk := created.GetObjectKind().GroupVersionKind()
	if gvk.Kind != "Pod" || gvk.Version != "v1" {
		t.Fatalf("gvk = %v", gvk)
	}
}

func TestCreateDuplicateFails(t *testing.T) {
	s := cluster()
	if _, err := s.Create(newPod("a")); err != nil {
		t.Fatal(err)
	}
	_, err := s.Create(newPod("a"))
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateDoesNotAliasCallerObject(t *testing.T) {
	s := cluster()
	p := newPod("a")
	if _, err := s.Create(p); err != nil {
		t.Fatal(err)
	}
	if p.ResourceVersion != "" {
		t.Fatal("caller object was mutated")
	}
	p.Labels = map[string]string{"x": "y"}
	got, _ := Get[*corev1.Pod](s, "Pod", "default", "a")
	if got.Labels != nil {
		t.Fatal("store aliases caller object")
	}
}

func TestGetReturnsCopy(t *testing.T) {
	s := cluster()
	_, _ = s.Create(newPod("a"))
	got, ok := Get[*corev1.Pod](s, "Pod", "default", "a")
	if !ok {
		t.Fatal("not found")
	}
	got.Spec.NodeName = "mutated"
	again, _ := Get[*corev1.Pod](s, "Pod", "default", "a")
	if again.Spec.NodeName != "" {
		t.Fatal("get returned alias")
	}
}

func TestUpdateConflictAndGeneration(t *testing.T) {
	s := cluster()
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "lantern", Namespace: "default"}}
	created, _ := s.Create(rs)
	if created.GetGeneration() != 1 {
		t.Fatalf("generation = %d", created.GetGeneration())
	}
	cur := created.(*appsv1.ReplicaSet)
	next := cur.DeepCopy()
	replicas := int32(3)
	next.Spec.Replicas = &replicas
	updated, err := s.Update(next)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetGeneration() != 2 {
		t.Fatalf("generation after spec change = %d", updated.GetGeneration())
	}
	statusOnly := updated.(*appsv1.ReplicaSet).DeepCopy()
	statusOnly.Status.Replicas = 3
	afterStatus, err := s.Update(statusOnly)
	if err != nil {
		t.Fatal(err)
	}
	if afterStatus.GetGeneration() != 2 {
		t.Fatal("status update bumped generation")
	}
	if _, err := s.Update(next); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update err = %v", err)
	}
}

func TestUpdateMissingFails(t *testing.T) {
	s := cluster()
	if _, err := s.Update(newPod("ghost")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestDeleteAndWatchEvents(t *testing.T) {
	s := cluster()
	var seen []EventType
	s.Watch(func(e Event) { seen = append(seen, e.Type) })
	created, _ := s.Create(newPod("a"))
	next := created.(*corev1.Pod).DeepCopy()
	next.Spec.NodeName = "lone-tower"
	_, _ = s.Update(next)
	if err := s.Delete(KeyOf(created)); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(KeyOf(created)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete err = %v", err)
	}
	want := []EventType{Added, Modified, Deleted}
	if len(seen) != len(want) {
		t.Fatalf("seen = %v", seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("seen = %v", seen)
		}
	}
}

func TestListSortedAndFiltered(t *testing.T) {
	s := cluster("kube-system")
	for _, n := range []string{"c", "a", "b"} {
		_, _ = s.Create(newPod(n))
	}
	other := newPod("z")
	other.Namespace = "kube-system"
	_, _ = s.Create(other)
	pods := List[*corev1.Pod](s, "Pod", "default")
	if len(pods) != 3 || pods[0].Name != "a" || pods[2].Name != "c" {
		t.Fatalf("pods = %v", names(pods))
	}
	all := List[*corev1.Pod](s, "Pod", "")
	if len(all) != 4 {
		t.Fatalf("all = %d", len(all))
	}
}

func TestSnapshotRestoreRoundTrip(t *testing.T) {
	s := cluster()
	_, _ = s.Create(newPod("a"))
	_, _ = s.Create(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "lone-tower"}})
	snap := s.Snapshot()
	r := New(fixedNow)
	if err := r.Restore(snap.Objects, snap.ResourceVersion, snap.UIDSeq); err != nil {
		t.Fatal(err)
	}
	if _, ok := Get[*corev1.Node](r, "Node", "", "lone-tower"); !ok {
		t.Fatal("node missing after restore")
	}
	created, _ := r.Create(newPod("b"))
	if created.GetResourceVersion() != strconv.FormatUint(snap.ResourceVersion+1, 10) {
		t.Fatalf("rv after restore = %s", created.GetResourceVersion())
	}
}

func TestKindOfUnknownType(t *testing.T) {
	if _, err := KindOf(&corev1.PodList{}); err == nil {
		t.Fatal("expected error for unregistered type")
	}
}

func names(pods []*corev1.Pod) []string {
	out := make([]string, 0, len(pods))
	for _, p := range pods {
		out = append(out, p.Name)
	}
	return out
}

func TestRestoreNeverReusesExistingUIDs(t *testing.T) {
	s := cluster()
	for _, n := range []string{"a", "b", "c"} {
		_, _ = s.Create(newPod(n))
	}
	snap := s.Snapshot()
	r := cluster()
	if err := r.Restore(snap.Objects, snap.ResourceVersion, 0); err != nil {
		t.Fatal(err)
	}
	created, _ := r.Create(newPod("d"))
	for _, o := range snap.Objects {
		if o.GetUID() == created.GetUID() {
			t.Fatalf("uid %s reused after restore with stale uidSeq", created.GetUID())
		}
	}
}

func TestViewSharesTheStoredObjectsInOrder(t *testing.T) {
	s := cluster()
	for _, name := range []string{"c", "a", "b"} {
		_, _ = s.Create(newPod(name))
	}
	first, second := View[*corev1.Pod](s, "Pod", "default"), View[*corev1.Pod](s, "Pod", "")
	if len(first) != 3 || first[0].Name != "a" || first[2].Name != "c" {
		t.Fatalf("view order = %v", first)
	}
	if first[0] != second[0] {
		t.Fatal("view copied the stored object")
	}
	if listed := List[*corev1.Pod](s, "Pod", "default"); listed[0] == first[0] {
		t.Fatal("list aliases the stored object")
	}
	next := first[1].DeepCopy()
	next.Spec.NodeName = "keep"
	if _, err := s.Update(next); err != nil {
		t.Fatal(err)
	}
	if first[1].Spec.NodeName != "" || View[*corev1.Pod](s, "Pod", "default")[1].Spec.NodeName != "keep" {
		t.Fatal("an update rewrote an object a view already handed out")
	}
	if got := View[*corev1.Pod](s, "Pod", "kube-system"); len(got) != 0 {
		t.Fatalf("view leaked across namespaces: %v", got)
	}
}

func TestViewFollowsCreatesDeletesAndRestores(t *testing.T) {
	s := cluster()
	_, _ = s.Create(newPod("b"))
	_ = View[*corev1.Pod](s, "Pod", "")
	_, _ = s.Create(newPod("a"))
	if got := View[*corev1.Pod](s, "Pod", ""); len(got) != 2 || got[0].Name != "a" {
		t.Fatalf("view missed a create: %v", got)
	}
	if err := s.Delete(Key{Kind: "Pod", Namespace: "default", Name: "a"}); err != nil {
		t.Fatal(err)
	}
	if got := View[*corev1.Pod](s, "Pod", "default"); len(got) != 1 || got[0].Name != "b" {
		t.Fatalf("view missed a delete: %v", got)
	}
	if err := s.Restore([]Object{newPod("z")}, 10, 10); err != nil {
		t.Fatal(err)
	}
	if got := View[*corev1.Pod](s, "Pod", ""); len(got) != 1 || got[0].Name != "z" {
		t.Fatalf("view survived a restore: %v", got)
	}
}

func TestPeekSharesWhileGetCopies(t *testing.T) {
	s := cluster()
	_, _ = s.Create(newPod("a"))
	one, ok := Peek[*corev1.Pod](s, "Pod", "default", "a")
	two, _ := Peek[*corev1.Pod](s, "Pod", "default", "a")
	if !ok || one != two {
		t.Fatal("peek copied the stored object")
	}
	if got, _ := Get[*corev1.Pod](s, "Pod", "default", "a"); got == one {
		t.Fatal("get aliases the stored object")
	}
	if _, ok := Peek[*corev1.Pod](s, "Pod", "default", "missing"); ok {
		t.Fatal("peek found a missing object")
	}
	if _, ok := Peek[*corev1.Node](s, "Pod", "default", "a"); ok {
		t.Fatal("peek ignored the type")
	}
}
