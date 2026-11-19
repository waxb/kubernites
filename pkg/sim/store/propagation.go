package store

import (
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func propagationFinalizers(current []string, policy metav1.DeletionPropagation) ([]string, bool) {
	next := []string{}
	for _, f := range current {
		if f != metav1.FinalizerOrphanDependents && f != metav1.FinalizerDeleteDependents {
			next = append(next, f)
		}
	}
	switch policy {
	case metav1.DeletePropagationOrphan:
		next = append(next, metav1.FinalizerOrphanDependents)
	case metav1.DeletePropagationForeground:
		next = append(next, metav1.FinalizerDeleteDependents)
	}
	return next, !sets.New(current...).Equal(sets.New(next...))
}

func (s *Store) DeleteWithPropagation(key Key, policy metav1.DeletionPropagation) error {
	current, exists := s.objects[key]
	if !exists {
		return fmt.Errorf("%s %q: %w", key.Kind, key.Name, ErrNotFound)
	}
	finalizers, changed := propagationFinalizers(current.GetFinalizers(), policy)
	if !changed {
		return s.Delete(key)
	}
	if len(finalizers) == 0 {
		s.remove(key, current)
		return nil
	}
	s.writeDeleting(key, current, finalizers)
	return nil
}

func (s *Store) SetPropagation(key Key, policy metav1.DeletionPropagation) error {
	current, exists := s.objects[key]
	if !exists {
		return fmt.Errorf("%s %q: %w", key.Kind, key.Name, ErrNotFound)
	}
	finalizers, changed := propagationFinalizers(current.GetFinalizers(), policy)
	if !changed {
		return nil
	}
	next := copyOf(current)
	next.SetFinalizers(finalizers)
	s.write(key, current, next)
	return nil
}

func (s *Store) writeDeleting(key Key, current Object, finalizers []string) {
	next := copyOf(current)
	next.SetFinalizers(finalizers)
	if next.GetDeletionTimestamp() == nil && next.GetGeneration() > 0 {
		next.SetGeneration(next.GetGeneration() + 1)
	}
	now := s.now()
	if ts := next.GetDeletionTimestamp(); ts == nil || ts.After(now.Time) {
		next.SetDeletionTimestamp(&now)
	}
	zero := int64(0)
	next.SetDeletionGracePeriodSeconds(&zero)
	s.write(key, current, next)
}

func (s *Store) write(key Key, current, next Object) {
	s.rv++
	next.SetResourceVersion(strconv.FormatUint(s.rv, 10))
	s.objects[key] = next
	s.putEntry(key, next)
	s.meterWrite(Modified, key.Kind, current, next)
	s.emit(Modified, next)
}
