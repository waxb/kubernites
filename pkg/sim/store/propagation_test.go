package store

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestForegroundDeletionWaitsOnItsFinalizer(t *testing.T) {
	st := New(fixedNow)
	_, _ = st.Create(finalized("crew", "keep.io/other"))
	if err := st.DeleteWithPropagation(Key{Kind: KindConfigMap, Namespace: "default", Name: "crew"}, metav1.DeletePropagationForeground); err != nil {
		t.Fatal(err)
	}
	cm, ok := st.Get(Key{Kind: KindConfigMap, Namespace: "default", Name: "crew"})
	if !ok || cm.GetDeletionTimestamp() == nil {
		t.Fatal("a foreground delete leaves the owner Terminating")
	}
	if got := cm.GetFinalizers(); len(got) != 2 || got[0] != "keep.io/other" || got[1] != metav1.FinalizerDeleteDependents {
		t.Fatalf("finalizers = %v", got)
	}
}

func TestOrphanDeletionAddsTheOrphanFinalizer(t *testing.T) {
	st := New(fixedNow)
	_, _ = st.Create(finalized("crew"))
	key := Key{Kind: KindConfigMap, Namespace: "default", Name: "crew"}
	if err := st.DeleteWithPropagation(key, metav1.DeletePropagationOrphan); err != nil {
		t.Fatal(err)
	}
	cm, ok := st.Get(key)
	if !ok || cm.GetDeletionTimestamp() == nil || len(cm.GetFinalizers()) != 1 || cm.GetFinalizers()[0] != metav1.FinalizerOrphanDependents {
		t.Fatalf("orphan delete = %v %+v", ok, cm)
	}
	if err := st.DeleteWithPropagation(key, metav1.DeletePropagationBackground); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Get(key); ok {
		t.Fatal("a background delete drops the orphan finalizer, and with none left the object goes")
	}
}

func TestBackgroundDeletionRemovesAtOnce(t *testing.T) {
	st := New(fixedNow)
	_, _ = st.Create(finalized("crew"))
	key := Key{Kind: KindConfigMap, Namespace: "default", Name: "crew"}
	if err := st.DeleteWithPropagation(key, metav1.DeletePropagationBackground); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Get(key); ok {
		t.Fatal("background deletion without finalizers removes the object")
	}
	if err := st.DeleteWithPropagation(key, metav1.DeletePropagationBackground); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}
