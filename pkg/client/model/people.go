package model

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const (
	ContainerWaiting    = "Waiting"
	ContainerRunning    = "Running"
	ContainerTerminated = "Terminated"
)

type Kubernite struct {
	Name       string
	Image      string
	Profession string
	Sprite     string
	Tier       int
	Ready      bool
	Restarts   int32
	Reason     string
	LastExit   string
	State      ContainerState
	Last       ContainerState
	Started    bool
	Readiness  bool
	Liveness   bool
	Startup    bool
	Resources  Resources
}

type ContainerState struct {
	Phase    string
	Reason   string
	ExitCode int32
	Started  float64
	Finished float64
}

func PeopleOf(p *corev1.Pod) []Kubernite {
	out := make([]Kubernite, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		profession, sprite := soulstone.ProfessionOf(c.Image)
		k := Kubernite{Name: c.Name, Image: c.Image, Profession: profession, Sprite: sprite, Tier: TierOf(c.Image),
			Readiness: c.ReadinessProbe != nil, Liveness: c.LivenessProbe != nil, Startup: c.StartupProbe != nil, Resources: resourcesOf(c)}
		if cs, ok := podutil.ContainerStatusNamed(p.Status.ContainerStatuses, c.Name); ok {
			k.Ready, k.Restarts, k.Reason, k.LastExit = cs.Ready, cs.RestartCount, stateReason(cs.State), exitReason(cs)
			k.State, k.Last, k.Started = containerState(cs.State), containerState(cs.LastTerminationState), cs.Started != nil && *cs.Started
		}
		out = append(out, k)
	}
	return out
}

func resourcesOf(c corev1.Container) Resources {
	cpu, mem := podutil.ContainerRequests(c)
	cpuLimit, memLimit := podutil.ContainerLimits(c)
	return Resources{CPURequest: cpu, MemRequest: mem, CPULimit: cpuLimit, MemLimit: memLimit}
}

func unixOf(t metav1.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / 1e9
}

func containerState(s corev1.ContainerState) ContainerState {
	switch {
	case s.Waiting != nil:
		return ContainerState{Phase: ContainerWaiting, Reason: s.Waiting.Reason}
	case s.Running != nil:
		return ContainerState{Phase: ContainerRunning, Started: unixOf(s.Running.StartedAt)}
	case s.Terminated != nil:
		t := s.Terminated
		return ContainerState{Phase: ContainerTerminated, Reason: t.Reason, ExitCode: t.ExitCode, Started: unixOf(t.StartedAt), Finished: unixOf(t.FinishedAt)}
	}
	return ContainerState{}
}

func (p PodView) Worker() (Kubernite, bool) {
	if p.Focus == "" {
		return Kubernite{}, false
	}
	for _, k := range p.People {
		if k.Name == p.Focus {
			return k, true
		}
	}
	return Kubernite{}, false
}

func (p PodView) Alone() bool {
	return len(p.People) == 1 && len(p.Familiars) == 0
}

func stateReason(s corev1.ContainerState) string {
	switch {
	case s.Waiting != nil:
		return s.Waiting.Reason
	case s.Terminated != nil:
		return s.Terminated.Reason
	}
	return ""
}

func exitReason(cs corev1.ContainerStatus) string {
	if t := cs.LastTerminationState.Terminated; t != nil {
		return t.Reason
	}
	return ""
}

func (p PodView) Persons() int {
	return max(len(p.People), 1)
}

func (p PodView) Person(i int) PodView {
	if len(p.People) == 0 {
		p.Focus = ""
		return p
	}
	if i < 0 || i >= len(p.People) {
		return p.Focused("")
	}
	k := p.People[i]
	p.Focus, p.Image, p.Profession, p.Sprite, p.Tier = k.Name, k.Image, k.Profession, k.Sprite, k.Tier
	p.Ready, p.Restarts, p.Reason, p.LastExit = k.Ready, k.Restarts, k.Reason, k.LastExit
	return p
}

func (p PodView) Focused(container string) PodView {
	for i, k := range p.People {
		if k.Name == container && container != "" {
			return p.Person(i)
		}
	}
	p.Focus = ""
	return p
}
