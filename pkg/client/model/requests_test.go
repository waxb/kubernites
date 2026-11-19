package model

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func withContainers(p *corev1.Pod, cs ...corev1.Container) *corev1.Pod {
	next := p.DeepCopy()
	next.Spec.Containers = cs
	return next
}

func container(name, cpu, mem string) corev1.Container {
	c := corev1.Container{Name: name, Image: "porter:1.0"}
	if cpu == "" && mem == "" {
		return c
	}
	c.Resources.Requests = corev1.ResourceList{}
	if cpu != "" {
		c.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if mem != "" {
		c.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(mem)
	}
	return c
}

func TestPodViewCarriesEachContainersRequests(t *testing.T) {
	p := withContainers(pod("porter-a", "tower", "porter:1.0"), container("porter", "100m", "64Mi"), container("scribe", "50m", "32Mi"))
	v := ViewOf(p)
	if len(v.Requests) != 2 || v.Requests[0] != (ContainerRequest{Name: "porter", CPUMilli: 100, MemoryBytes: 64 << 20}) {
		t.Fatalf("requests = %+v", v.Requests)
	}
	if cpu, mem := v.RequestSum(); cpu != 150 || mem != 96<<20 {
		t.Errorf("sum = %dm %d", cpu, mem)
	}
}

func TestPodViewRequestsFallBackToLimitsAsTheAPIDefaults(t *testing.T) {
	c := container("porter", "", "")
	c.Resources.Limits = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m")}
	v := ViewOf(withContainers(pod("porter-a", "tower", "porter:1.0"), c))
	if cpu, mem := v.RequestSum(); cpu != 200 || mem != 0 {
		t.Errorf("a limit alone is the request: %dm %d", cpu, mem)
	}
}

func TestBestEffortPodsRequestNothing(t *testing.T) {
	v := ViewOf(pod("porter-a", "tower", "porter:1.0"))
	if cpu, mem := v.RequestSum(); cpu != 0 || mem != 0 || v.QoS != QoSBestEffort {
		t.Errorf("best effort = %dm %d %s", cpu, mem, v.QoS)
	}
}

func TestInitContainersDoNotJoinTheHutsPiles(t *testing.T) {
	p := withContainers(pod("porter-a", "tower", "porter:1.0"), container("porter", "100m", "64Mi"))
	p.Spec.InitContainers = []corev1.Container{container("warmup", "900m", "1Gi")}
	if cpu, mem := ViewOf(p).RequestSum(); cpu != 100 || mem != 64<<20 {
		t.Errorf("only spec.containers count: %dm %d", cpu, mem)
	}
}
