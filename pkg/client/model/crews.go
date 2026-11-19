package model

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
)

const (
	replicaSetKind = "ReplicaSet"
	deploymentKind = "Deployment"
)

type crewBook struct {
	sets     map[string]*appsv1.ReplicaSet
	services []*corev1.Service
}

func (c *Cache) crewBook() crewBook {
	b := crewBook{sets: map[string]*appsv1.ReplicaSet{}}
	for _, rs := range c.ReplicaSets() {
		b.sets[rs.Namespace+"/"+rs.Name] = rs
	}
	for _, svc := range c.Services() {
		if svc.Name != apiService && len(svc.Spec.Selector) > 0 {
			b.services = append(b.services, svc)
		}
	}
	return b
}

func (b crewBook) crewOf(p *corev1.Pod) string {
	ref := podutil.ControllerRef(p)
	if ref == nil {
		return ""
	}
	if ref.Kind == replicaSetKind {
		if rs, ok := b.sets[p.Namespace+"/"+ref.Name]; ok {
			if up := podutil.ControllerRef(rs); up != nil && up.Kind == deploymentKind {
				return up.Name
			}
		}
	}
	return ref.Name
}

func (b crewBook) depotOf(p *corev1.Pod) string {
	for _, svc := range b.services {
		if svc.Namespace == p.Namespace && selects(svc.Spec.Selector, p.Labels) {
			return svc.Name
		}
	}
	ref := podutil.ControllerRef(p)
	if ref == nil || ref.Kind != replicaSetKind {
		return ""
	}
	rs, ok := b.sets[p.Namespace+"/"+ref.Name]
	if !ok {
		return ""
	}
	for _, svc := range b.services {
		if svc.Namespace == p.Namespace && selects(svc.Spec.Selector, rs.Spec.Template.Labels) {
			return svc.Name
		}
	}
	return ""
}

func selects(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if got, ok := labels[k]; !ok || got != v {
			return false
		}
	}
	return true
}
