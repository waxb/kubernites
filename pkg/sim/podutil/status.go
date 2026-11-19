package podutil

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const podInitializing = "PodInitializing"

type summary struct {
	reason          string
	ready           int
	total           int
	restarts        int32
	lastRestart     metav1.Time
	sidecarRestarts int32
	lastSidecar     metav1.Time
}

func summarize(pod *corev1.Pod) summary {
	s := summary{reason: string(pod.Status.Phase), total: len(pod.Spec.Containers)}
	if pod.Status.Reason != "" {
		s.reason = pod.Status.Reason
	}
	inits := map[string]*corev1.Container{}
	for i := range pod.Spec.InitContainers {
		inits[pod.Spec.InitContainers[i].Name] = &pod.Spec.InitContainers[i]
		if IsRestartableInitContainer(&pod.Spec.InitContainers[i]) {
			s.total++
		}
	}
	initializing := s.scanInits(pod, inits)
	if !initializing || initializedCondition(pod) {
		s.restarts, s.lastRestart = s.sidecarRestarts, s.lastSidecar
		s.scanApps(pod)
	}
	return s
}

func (s *summary) scanInits(pod *corev1.Pod, inits map[string]*corev1.Container) bool {
	for i, cs := range pod.Status.InitContainerStatuses {
		restartable := IsRestartableInitContainer(inits[cs.Name])
		s.restarts += cs.RestartCount
		s.lastRestart = later(s.lastRestart, cs.LastTerminationState)
		if restartable {
			s.sidecarRestarts += cs.RestartCount
			s.lastSidecar = later(s.lastSidecar, cs.LastTerminationState)
		}
		switch {
		case cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0:
			continue
		case restartable && cs.Started != nil && *cs.Started:
			if cs.Ready {
				s.ready++
			}
			continue
		case cs.State.Terminated != nil:
			s.reason = initTerminatedReason(cs.State.Terminated)
		case cs.State.Waiting != nil && cs.State.Waiting.Reason != "" && cs.State.Waiting.Reason != podInitializing:
			s.reason = "Init:" + cs.State.Waiting.Reason
		default:
			s.reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
		}
		return true
	}
	return false
}

func initTerminatedReason(t *corev1.ContainerStateTerminated) string {
	switch {
	case t.Reason != "":
		return "Init:" + t.Reason
	case t.Signal != 0:
		return fmt.Sprintf("Init:Signal:%d", t.Signal)
	}
	return fmt.Sprintf("Init:ExitCode:%d", t.ExitCode)
}

func (s *summary) scanApps(pod *corev1.Pod) {
	for _, cs := range pod.Status.ContainerStatuses {
		s.restarts += cs.RestartCount
		s.lastRestart = later(s.lastRestart, cs.LastTerminationState)
		if cs.Ready {
			s.ready++
		}
		switch {
		case cs.State.Waiting != nil && cs.State.Waiting.Reason != "":
			s.reason = cs.State.Waiting.Reason
		case cs.State.Terminated != nil && cs.State.Terminated.Reason != "":
			s.reason = cs.State.Terminated.Reason
		}
	}
}

func later(current metav1.Time, last corev1.ContainerState) metav1.Time {
	if t := last.Terminated; t != nil && t.FinishedAt.After(current.Time) {
		return t.FinishedAt
	}
	return current
}

func initializedCondition(pod *corev1.Pod) bool {
	c := GetCondition(pod.Status.Conditions, corev1.PodInitialized)
	return c != nil && c.Status == corev1.ConditionTrue
}

func ReadyCount(pod *corev1.Pod) (int, int) {
	s := summarize(pod)
	return s.ready, s.total
}

func RestartCount(pod *corev1.Pod) int32 {
	return summarize(pod).restarts
}

func LastRestart(pod *corev1.Pod) metav1.Time {
	return summarize(pod).lastRestart
}

func StatusReason(pod *corev1.Pod) string {
	reason := summarize(pod).reason
	if pod.Status.Reason != "" && IsTerminal(pod) {
		reason = pod.Status.Reason
	}
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}
	if reason == "" {
		return "Pending"
	}
	return reason
}

func ContainerStatusNamed(statuses []corev1.ContainerStatus, name string) (corev1.ContainerStatus, bool) {
	for _, cs := range statuses {
		if cs.Name == name {
			return cs, true
		}
	}
	return corev1.ContainerStatus{}, false
}

func ContainerRunning(statuses []corev1.ContainerStatus, name string) bool {
	cs, ok := ContainerStatusNamed(statuses, name)
	return ok && cs.State.Running != nil
}
