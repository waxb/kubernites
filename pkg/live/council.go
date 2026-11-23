package live

import (
	"cmp"
	"slices"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/waxb/kubernites/pkg/view"
)

const busiestShown = 5

var councilLists = []string{"nodes", "pods", "deployments", "statefulsets", "daemonsets", "services", "endpointslices", "events"}

func councilOf(s *Snapshot, u Usage) view.LiveCouncil {
	c := view.LiveCouncil{Nodes: len(s.Nodes)}
	for i := range s.Nodes {
		if nodeReady(s.Nodes[i]) == corev1.ConditionTrue {
			c.NodesReady++
		}
	}
	reasons := schedulingEvents(s.Events)
	for i := range s.Pods {
		p := &s.Pods[i]
		if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
			continue
		}
		c.PodsActive++
		if podIsReady(p) {
			c.PodsReady++
		}
		if p.Status.Phase == corev1.PodPending && p.Spec.NodeName == "" {
			c.Pending = append(c.Pending, view.LivePending{Namespace: p.Namespace, Name: p.Name, Reason: schedulerReason(p, reasons)})
		}
	}
	slices.SortFunc(c.Pending, func(a, b view.LivePending) int {
		return cmp.Or(cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Name, b.Name))
	})
	workloads(s, &c)
	services(s, &c)
	c.Busiest = busiestPods(u)
	for _, name := range councilLists {
		if slices.Contains(s.Denied, name) || slices.Contains(s.Missing, name) {
			c.Unread = append(c.Unread, name)
		}
	}
	slices.Sort(c.Unread)
	return c
}

func podIsReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func schedulingEvents(events []corev1.Event) map[string]*corev1.Event {
	out := map[string]*corev1.Event{}
	for i := range events {
		e := &events[i]
		if e.Reason != reasonFailedSched || e.InvolvedObject.Kind != "Pod" {
			continue
		}
		key := e.InvolvedObject.Namespace + "/" + e.InvolvedObject.Name
		if last, ok := out[key]; !ok || eventTime(last).Before(eventTime(e)) {
			out[key] = e
		}
	}
	return out
}

func schedulerReason(p *corev1.Pod, events map[string]*corev1.Event) string {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse && c.Message != "" {
			return c.Message
		}
	}
	if e, ok := events[p.Namespace+"/"+p.Name]; ok && (e.InvolvedObject.UID == "" || e.InvolvedObject.UID == p.UID) {
		return e.Message
	}
	return ""
}

func replicas(n *int32) int {
	if n == nil {
		return 1
	}
	return int(*n)
}

func workloads(s *Snapshot, c *view.LiveCouncil) {
	count := func(kind, ns, name string, ready, desired int) {
		c.Workloads++
		if ready >= desired {
			c.WorkloadsFull++
			return
		}
		c.Short = append(c.Short, view.LiveWorkload{Kind: kind, Namespace: ns, Name: name, Ready: ready, Desired: desired})
	}
	for _, d := range s.Deployments {
		count("Deployment", d.Namespace, d.Name, int(d.Status.ReadyReplicas), replicas(d.Spec.Replicas))
	}
	for _, st := range s.StatefulSets {
		count("StatefulSet", st.Namespace, st.Name, int(st.Status.ReadyReplicas), replicas(st.Spec.Replicas))
	}
	for _, d := range s.DaemonSets {
		count("DaemonSet", d.Namespace, d.Name, int(d.Status.NumberReady), int(d.Status.DesiredNumberScheduled))
	}
	slices.SortStableFunc(c.Short, func(a, b view.LiveWorkload) int {
		return cmp.Or(cmp.Compare(b.Desired-b.Ready, a.Desired-a.Ready), cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Name, b.Name))
	})
}

func services(s *Snapshot, c *view.LiveCouncil) {
	type endpoints struct{ all, ready int }
	seen := map[string]map[string]bool{}
	counts := map[string]endpoints{}
	for _, sl := range s.EndpointSlices {
		key := sl.Namespace + "/" + sl.Labels[discoveryv1.LabelServiceName]
		if seen[key] == nil {
			seen[key] = map[string]bool{}
		}
		for _, e := range sl.Endpoints {
			id := endpointID(e)
			if seen[key][id] {
				continue
			}
			seen[key][id] = true
			n := counts[key]
			n.all++
			if e.Conditions.Ready == nil || *e.Conditions.Ready {
				n.ready++
			}
			counts[key] = n
		}
	}
	for _, svc := range s.Services {
		if len(svc.Spec.Selector) == 0 || svc.Spec.Type == corev1.ServiceTypeExternalName {
			continue
		}
		c.Services++
		n := counts[svc.Namespace+"/"+svc.Name]
		if n.ready > 0 {
			c.ServicesServed++
			continue
		}
		c.Unserved = append(c.Unserved, view.LiveService{Namespace: svc.Namespace, Name: svc.Name, Endpoints: n.all})
	}
	slices.SortFunc(c.Unserved, func(a, b view.LiveService) int {
		return cmp.Or(cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Name, b.Name))
	})
}

func endpointID(e discoveryv1.Endpoint) string {
	if e.TargetRef != nil && e.TargetRef.UID != "" {
		return string(e.TargetRef.UID)
	}
	if len(e.Addresses) > 0 {
		return e.Addresses[0]
	}
	return ""
}

func busiestPods(u Usage) []view.LivePodUsage {
	if !u.Available {
		return nil
	}
	out := make([]view.LivePodUsage, 0, len(u.Pods))
	for key, got := range u.Pods {
		ns, name := splitKey(key)
		out = append(out, view.LivePodUsage{Namespace: ns, Name: name, CPUMilli: got.CPUMilli, MemoryBytes: got.MemBytes})
	}
	slices.SortFunc(out, func(a, b view.LivePodUsage) int {
		return cmp.Or(cmp.Compare(b.CPUMilli, a.CPUMilli), cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Name, b.Name))
	})
	return out[:min(len(out), busiestShown)]
}

func splitKey(key string) (string, string) {
	for i := range len(key) {
		if key[i] == '/' {
			return key[:i], key[i+1:]
		}
	}
	return "", key
}
