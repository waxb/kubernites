package model

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

type Horn struct {
	Name        string
	Target      string
	Min         int
	Max         int
	Current     int
	Desired     int
	Utilization int
	Goal        int
	Sounding    bool
	Message     string
	Measured    bool
}

func (c *Cache) HPAs() []*autoscalingv2.HorizontalPodAutoscaler {
	return list[*autoscalingv2.HorizontalPodAutoscaler](c, store.KindHorizontalPodAutoscaler)
}

func (c *Cache) Horns() []Horn {
	out := []Horn{}
	for _, h := range c.HPAs() {
		out = append(out, HornOf(h))
	}
	return out
}

func HornOf(h *autoscalingv2.HorizontalPodAutoscaler) Horn {
	v := Horn{Name: h.Name, Target: h.Spec.ScaleTargetRef.Name, Max: int(h.Spec.MaxReplicas), Min: 1,
		Current: int(h.Status.CurrentReplicas), Desired: int(h.Status.DesiredReplicas), Sounding: true}
	if h.Spec.MinReplicas != nil {
		v.Min = int(*h.Spec.MinReplicas)
	}
	for _, m := range h.Spec.Metrics {
		if m.Resource != nil && m.Resource.Target.AverageUtilization != nil {
			v.Goal = int(*m.Resource.Target.AverageUtilization)
		}
	}
	for _, m := range h.Status.CurrentMetrics {
		if m.Resource != nil && m.Resource.Current.AverageUtilization != nil {
			v.Utilization, v.Measured = int(*m.Resource.Current.AverageUtilization), true
		}
	}
	for _, cond := range h.Status.Conditions {
		if cond.Type == autoscalingv2.ScalingActive && cond.Status == corev1.ConditionFalse {
			v.Sounding, v.Message = false, cond.Message
		}
	}
	return v
}
