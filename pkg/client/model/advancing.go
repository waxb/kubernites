package model

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
)

const newReplicaSetAvailable = "NewReplicaSetAvailable"

func Advancing(d *appsv1.Deployment, now time.Time, within time.Duration) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing {
			return c.Status == corev1.ConditionTrue && c.Reason != newReplicaSetAvailable && now.Sub(c.LastUpdateTime.Time) <= within
		}
	}
	return false
}

func (c *Cache) AdvancingOwners(now time.Time, within time.Duration) map[string]bool {
	rolling := map[string]bool{}
	for _, d := range c.Deployments() {
		if Advancing(d, now, within) {
			rolling[d.Namespace+"/"+d.Name] = true
		}
	}
	out := map[string]bool{}
	for _, rs := range c.ReplicaSets() {
		if ref := podutil.ControllerRef(rs); ref != nil && ref.Kind == "Deployment" && rolling[rs.Namespace+"/"+ref.Name] {
			out[rs.Namespace+"/ReplicaSet/"+rs.Name] = true
		}
	}
	return out
}

func (p PodView) OwnerKey() string {
	return p.Namespace + "/" + p.Owner
}
