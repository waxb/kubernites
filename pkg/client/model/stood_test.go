package model

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nodeAt(created time.Time, status corev1.ConditionStatus, changed time.Time) *corev1.Node {
	n := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "stone-keep-1", CreationTimestamp: metav1.NewTime(created)}}
	n.Status.Conditions = []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status, LastTransitionTime: metav1.NewTime(changed)}}
	return n
}

func TestANodeStandsQuietSinceItsCreationOrItsLastReturnToReady(t *testing.T) {
	born := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	back := born.Add(time.Hour)
	if since, ok := StoodSince(nodeAt(born, corev1.ConditionTrue, born)); !ok || !since.Equal(born) {
		t.Errorf("a calm Node stands since its creation, got %v %v", since, ok)
	}
	if since, ok := StoodSince(nodeAt(born, corev1.ConditionTrue, back)); !ok || !since.Equal(back) {
		t.Errorf("a Node that came back stands since then, got %v %v", since, ok)
	}
	if _, ok := StoodSince(nodeAt(born, corev1.ConditionFalse, back)); ok {
		t.Error("a Node that is not Ready has no quiet age")
	}
	if _, ok := StoodSince(&corev1.Node{}); ok {
		t.Error("a Node without a timestamp has no quiet age")
	}
}
