package store

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	PVCProtectionFinalizer = "kubernetes.io/pvc-protection"
	PVProtectionFinalizer  = "kubernetes.io/pv-protection"
)

func (s *Store) markAsDeleting(key Key, current Object) {
	next := copyOf(current)
	if next.GetDeletionTimestamp() == nil && next.GetGeneration() > 0 {
		next.SetGeneration(next.GetGeneration() + 1)
	}
	now := s.now()
	if ts := next.GetDeletionTimestamp(); ts == nil || ts.After(now.Time) {
		next.SetDeletionTimestamp(&now)
	}
	zero := int64(0)
	next.SetDeletionGracePeriodSeconds(&zero)
	s.rv++
	next.SetResourceVersion(strconv.FormatUint(s.rv, 10))
	s.objects[key] = next
	s.putEntry(key, next)
	s.meterWrite(Modified, key.Kind, current, next)
	s.emit(Modified, next)
}

func alreadyDeleting(current Object) bool {
	grace := current.GetDeletionGracePeriodSeconds()
	return current.GetDeletionTimestamp() != nil && len(current.GetFinalizers()) > 0 && grace != nil && *grace == 0
}

func keepDeletion(stored, current Object) error {
	if current.GetDeletionTimestamp() == nil {
		return nil
	}
	extra := sets.List(sets.New(stored.GetFinalizers()...).Difference(sets.New(current.GetFinalizers()...)))
	if len(extra) > 0 {
		kind, _ := KindOf(current)
		err := field.Forbidden(field.NewPath("metadata", "finalizers"), fmt.Sprintf("no new finalizers can be added if the object is being deleted, found new finalizers %#v", extra))
		return fmt.Errorf("The %s %q is invalid: %s", kind, current.GetName(), err.Error())
	}
	stored.SetDeletionTimestamp(current.GetDeletionTimestamp())
	stored.SetDeletionGracePeriodSeconds(current.GetDeletionGracePeriodSeconds())
	return nil
}

func shouldDeleteDuringUpdate(stored, current Object) bool {
	if len(stored.GetFinalizers()) > 0 || current.GetDeletionTimestamp() == nil {
		return false
	}
	grace := current.GetDeletionGracePeriodSeconds()
	return grace == nil || *grace == 0
}

func protect(obj Object, old Object) {
	if old != nil {
		return
	}
	switch obj.(type) {
	case *corev1.PersistentVolumeClaim:
		addFinalizer(obj, PVCProtectionFinalizer)
	case *corev1.PersistentVolume:
		addFinalizer(obj, PVProtectionFinalizer)
	}
}

func addFinalizer(obj metav1.Object, name string) {
	for _, f := range obj.GetFinalizers() {
		if f == name {
			return
		}
	}
	obj.SetFinalizers(append(append([]string(nil), obj.GetFinalizers()...), name))
}
