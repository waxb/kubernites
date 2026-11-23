package live

import (
	corev1 "k8s.io/api/core/v1"
	resourcehelper "k8s.io/component-helpers/resource"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	tierLabel        = "tier"
	controlPlaneTier = "control-plane"
	systemNamespace  = "kube-system"
)

var schedulerCounting = resourcehelper.PodResourcesOptions{UseStatusResources: true, InPlacePodLevelResourcesVerticalScalingEnabled: true}

func scheduledOn(p *corev1.Pod) bool {
	return p.Spec.NodeName != "" && p.Status.Phase != corev1.PodSucceeded && p.Status.Phase != corev1.PodFailed
}

func controlPlanePod(p *corev1.Pod) bool {
	return p.Namespace == systemNamespace && p.Labels[tierLabel] == controlPlaneTier
}

func nodeMeasures(s *Snapshot, u Usage) map[string]view.NodeMetrics {
	out := make(map[string]view.NodeMetrics, len(s.Nodes))
	for i := range s.Nodes {
		n := &s.Nodes[i]
		alloc := n.Status.Allocatable
		m := view.NodeMetrics{CPUAllocatableMilli: alloc.Cpu().MilliValue(), MemoryAllocatableBytes: alloc.Memory().Value(),
			PodsAllocatable: int(alloc.Pods().Value()), EphemeralAllocatableBytes: alloc.StorageEphemeral().Value()}
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeDiskPressure && c.Status == corev1.ConditionTrue {
				m.DiskPressure = true
			}
		}
		if got, ok := u.Nodes[n.Name]; ok && u.Available {
			m.CPUMilli, m.MemoryBytes, m.Measured = got.CPUMilli, got.MemBytes, true
		}
		out[n.Name] = m
	}
	for i := range s.Pods {
		p := &s.Pods[i]
		m, ok := out[p.Spec.NodeName]
		if !ok || !scheduledOn(p) {
			continue
		}
		req := resourcehelper.PodRequests(p, schedulerCounting)
		cpu, mem := req.Cpu().MilliValue(), req.Memory().Value()
		m.CPURequestedMilli += cpu
		m.MemoryRequestedBytes += mem
		m.EphemeralRequestedBytes += req.StorageEphemeral().Value()
		m.Pods++
		if controlPlanePod(p) {
			m.ControlPlaneCPUMilli += cpu
			m.ControlPlaneMemoryBytes += mem
		}
		out[p.Spec.NodeName] = m
	}
	return out
}

func podMeasures(u Usage) map[string]view.PodMetrics {
	out := make(map[string]view.PodMetrics, len(u.Pods))
	for key, got := range u.Pods {
		out[key] = view.PodMetrics{CPUMilli: got.CPUMilli, MemoryBytes: got.MemBytes}
	}
	return out
}

func (r *Realm) measure() {
	r.measured = view.MetricsSnapshot{Pods: podMeasures(r.usage), Nodes: nodeMeasures(&r.snap, r.usage), Augury: r.usage.Available}
	r.council = councilOf(&r.snap, r.usage)
}
