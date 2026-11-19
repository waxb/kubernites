package model

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
)

type ContainerRequest struct {
	Name        string
	CPUMilli    int64
	MemoryBytes int64
}

func RequestsOf(p *corev1.Pod) []ContainerRequest {
	if len(p.Spec.Containers) == 0 {
		return nil
	}
	out := make([]ContainerRequest, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		cpu, mem := podutil.ContainerRequests(c)
		out = append(out, ContainerRequest{Name: c.Name, CPUMilli: cpu, MemoryBytes: mem})
	}
	return out
}

func (v PodView) RequestSum() (int64, int64) {
	var cpu, mem int64
	for _, r := range v.Requests {
		cpu += r.CPUMilli
		mem += r.MemoryBytes
	}
	return cpu, mem
}
