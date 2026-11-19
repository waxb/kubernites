package model

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var workerEpoch = time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)

func at(seconds int) metav1.Time {
	return metav1.NewTime(workerEpoch.Add(time.Duration(seconds) * time.Second))
}

func unix(seconds int) float64 {
	return float64(workerEpoch.Unix() + int64(seconds))
}

func TestWorkerReadsItsOwnContainerStatus(t *testing.T) {
	p := household()
	p.Spec.RestartPolicy = corev1.RestartPolicyAlways
	p.Spec.Containers[0].ReadinessProbe = &corev1.Probe{}
	p.Spec.Containers[0].LivenessProbe = &corev1.Probe{}
	p.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
	}
	started := true
	p.Status.ContainerStatuses[1].State.Running.StartedAt = at(30)
	p.Status.ContainerStatuses[1].Started = &started
	p.Status.ContainerStatuses[1].RestartCount = 2
	p.Status.ContainerStatuses[1].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1, StartedAt: at(10), FinishedAt: at(25)}}
	p.Status.ContainerStatuses[0].LastTerminationState.Terminated.ExitCode = 137
	p.Status.ContainerStatuses[0].LastTerminationState.Terminated.FinishedAt = at(40)

	v := ViewOf(p)
	if v.RestartPolicy != "Always" {
		t.Errorf("restart policy = %q", v.RestartPolicy)
	}
	porter, ok := v.Focused("porter").Worker()
	if !ok {
		t.Fatal("the porter is a worker of the hut")
	}
	if porter.State != (ContainerState{Phase: ContainerRunning, Started: unix(30)}) || !porter.Started {
		t.Errorf("porter state = %+v started %v", porter.State, porter.Started)
	}
	if porter.Last != (ContainerState{Phase: ContainerTerminated, Reason: "Error", ExitCode: 1, Started: unix(10), Finished: unix(25)}) {
		t.Errorf("porter last state = %+v", porter.Last)
	}
	if !porter.Readiness || !porter.Liveness || porter.Startup {
		t.Errorf("porter probes = %+v", porter)
	}
	if porter.Resources != (Resources{CPURequest: 200, MemRequest: 64 << 20, CPULimit: 500, MemLimit: 128 << 20}) {
		t.Errorf("porter resources = %+v", porter.Resources)
	}
	packer, _ := v.Focused("packer").Worker()
	if packer.State != (ContainerState{Phase: ContainerWaiting, Reason: "CrashLoopBackOff"}) {
		t.Errorf("packer state = %+v", packer.State)
	}
	if packer.Last.Phase != ContainerTerminated || packer.Last.Reason != "OOMKilled" || packer.Last.ExitCode != 137 || packer.Last.Finished != unix(40) {
		t.Errorf("packer last state = %+v", packer.Last)
	}
	if packer.Resources.CPULimit != 0 || packer.Resources.MemLimit != 0 || packer.Readiness {
		t.Errorf("packer declares nothing: %+v", packer)
	}
	if _, ok := v.Worker(); ok {
		t.Error("an unfocused hut view names no worker")
	}
}

func TestWorkerWithoutStatusIsWaitingForItsFirstStart(t *testing.T) {
	p := pod("fresh", "", "porter:1.0")
	p.Status.ContainerStatuses = nil
	w, ok := ViewOf(p).Person(0).Worker()
	if !ok || w.State.Phase != "" || w.Started {
		t.Fatalf("a container with no status yet has no state: %+v", w)
	}
}

func TestWorkerAloneOnlyWithoutFamiliarsOrOtherWorkers(t *testing.T) {
	if ViewOf(household()).Person(0).Alone() {
		t.Error("a hut of two workers and a raven: the hut's food is not the porter's")
	}
	if !ViewOf(pod("bare", "n1", "porter:1.0")).Person(0).Alone() {
		t.Error("a hut of one worker: the hut's food is its worker's")
	}
}

func TestHutKnowsWhenItWasReadiedAndDeleted(t *testing.T) {
	p := pod("p", "n1", "porter:1.0")
	p.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: at(50)}}
	gone := at(90)
	p.DeletionTimestamp = &gone
	v := ViewOf(p)
	if v.ReadySince != unix(50) || v.DeletedAt != unix(90) {
		t.Errorf("ready since %v deleted at %v", v.ReadySince, v.DeletedAt)
	}
}

func TestFamiliarsKeepTheirContainerNames(t *testing.T) {
	v := ViewOf(household())
	if len(v.PetNames) != 1 || v.PetNames[0] != "raven" || v.Familiars[0] != "raven" {
		t.Errorf("familiars %v named %v", v.Familiars, v.PetNames)
	}
}
