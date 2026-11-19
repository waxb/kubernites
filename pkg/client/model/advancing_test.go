package model

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func progressing(status corev1.ConditionStatus, reason string, at time.Time) *appsv1.Deployment {
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "porters"}}
	d.Status.Conditions = []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Status: status, Reason: reason, LastUpdateTime: metav1.NewTime(at)}}
	return d
}

func TestARolloutAdvancesWhileItsProgressingConditionIsFresh(t *testing.T) {
	now := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	within := 45 * time.Second
	cases := []struct {
		name string
		d    *appsv1.Deployment
		want bool
	}{
		{"progressed moments ago", progressing(corev1.ConditionTrue, "ReplicaSetUpdated", now.Add(-10*time.Second)), true},
		{"a new ReplicaSet just created", progressing(corev1.ConditionTrue, "NewReplicaSetCreated", now), true},
		{"no progress for a minute", progressing(corev1.ConditionTrue, "ReplicaSetUpdated", now.Add(-time.Minute)), false},
		{"finished", progressing(corev1.ConditionTrue, "NewReplicaSetAvailable", now), false},
		{"past its deadline", progressing(corev1.ConditionFalse, "ProgressDeadlineExceeded", now), false},
		{"no condition", &appsv1.Deployment{}, false},
	}
	for _, c := range cases {
		if got := Advancing(c.d, now, within); got != c.want {
			t.Errorf("%s: advancing %v, want %v", c.name, got, c.want)
		}
	}
}
