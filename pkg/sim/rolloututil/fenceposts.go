package rolloututil

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var defaultRollingPercent = intstr.FromString("25%")

func IsRollingUpdate(d *appsv1.Deployment) bool {
	return d.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType
}

func RollingParams(d *appsv1.Deployment) (*intstr.IntOrString, *intstr.IntOrString) {
	ru := d.Spec.Strategy.RollingUpdate
	if ru == nil {
		return &defaultRollingPercent, &defaultRollingPercent
	}
	return ru.MaxSurge, ru.MaxUnavailable
}

func ResolveFenceposts(maxSurge, maxUnavailable *intstr.IntOrString, desired int32) (int32, int32, error) {
	surge, err := intstr.GetScaledValueFromIntOrPercent(intstr.ValueOrDefault(maxSurge, intstr.FromInt32(0)), int(desired), true)
	if err != nil {
		return 0, 0, err
	}
	unavailable, err := intstr.GetScaledValueFromIntOrPercent(intstr.ValueOrDefault(maxUnavailable, intstr.FromInt32(0)), int(desired), false)
	if err != nil {
		return 0, 0, err
	}
	if surge == 0 && unavailable == 0 {
		unavailable = 1
	}
	return int32(surge), int32(unavailable), nil
}

func MaxSurge(d *appsv1.Deployment) int32 {
	if !IsRollingUpdate(d) {
		return 0
	}
	surge, unavailable := RollingParams(d)
	maxSurge, _, _ := ResolveFenceposts(surge, unavailable, deploymentReplicas(d))
	return maxSurge
}

func MaxUnavailable(d *appsv1.Deployment) int32 {
	replicas := deploymentReplicas(d)
	if !IsRollingUpdate(d) || replicas == 0 {
		return 0
	}
	surge, unavailable := RollingParams(d)
	_, maxUnavailable, _ := ResolveFenceposts(surge, unavailable, replicas)
	return min(maxUnavailable, replicas)
}

func deploymentReplicas(d *appsv1.Deployment) int32 {
	if d.Spec.Replicas == nil {
		return 1
	}
	return *d.Spec.Replicas
}
