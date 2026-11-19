package kubectl

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestDescribePodPrintsThePhaseAsStatus(t *testing.T) {
	f := newCluster(t)
	f.placed(t, "default", "porter-1", "lone-tower", func(p *corev1.Pod) {
		p.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "c", Image: "porter:1.0",
			State:                corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "Error"}},
			RestartCount:         1}}
	})
	out := run(f, "kubectl describe pod porter-1").Output
	if !strings.Contains(out, "\nStatus:           Running\n") || strings.Contains(out, "Status:           CrashLoopBackOff") || strings.Contains(out, "Status:           Error") {
		t.Errorf("describe prints the Pod's phase as Status, a container's reason only under State:\n%s", out)
	}
	if !strings.Contains(out, "Reason:       CrashLoopBackOff") {
		t.Errorf("the container's waiting reason stays under its State:\n%s", out)
	}
	if got := run(f, "kubectl get pod porter-1").Output; !strings.Contains(got, "CrashLoopBackOff") {
		t.Errorf("get keeps the STATUS column's reason:\n%s", got)
	}
}
