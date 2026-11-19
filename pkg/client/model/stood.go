package model

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

func (c *Cache) NodeStood() map[string]time.Time {
	out := map[string]time.Time{}
	for _, n := range c.Nodes() {
		if since, ok := StoodSince(n); ok {
			out[n.Name] = since
		}
	}
	return out
}

func StoodSince(n *corev1.Node) (time.Time, bool) {
	since := n.CreationTimestamp.Time
	ready := false
	for _, cond := range n.Status.Conditions {
		if cond.Type != corev1.NodeReady {
			continue
		}
		ready = cond.Status == corev1.ConditionTrue
		if t := cond.LastTransitionTime.Time; ready && t.After(since) {
			since = t
		}
	}
	return since, ready && !since.IsZero()
}
