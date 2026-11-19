package kubectl

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

func describeNode(c Cluster, n *corev1.Node) string {
	b := &block{}
	b.kv(0, 20, "Name", n.Name)
	b.kv(0, 20, "Roles", nodeRoles(n))
	b.kvLines(0, 20, "Labels", labelLines(n.Labels))
	b.kvLines(0, 20, "Annotations", annotationLines(n.Annotations))
	b.kv(0, 20, "CreationTimestamp", n.CreationTimestamp.Format(timeLayout))
	taints := []string{}
	for _, t := range n.Spec.Taints {
		taints = append(taints, t.ToString())
	}
	if len(taints) == 0 {
		taints = []string{"<none>"}
	}
	b.kv(0, 20, "Taints", strings.Join(taints, "\n"+strings.Repeat(" ", 20)))
	b.kv(0, 20, "Unschedulable", fmt.Sprint(n.Spec.Unschedulable))
	describeNodeConditions(b, n.Status.Conditions)
	b.raw("Addresses:")
	for _, a := range n.Status.Addresses {
		b.kv(2, 13, string(a.Type), a.Address)
	}
	resourceBlock(b, "Capacity", n.Status.Capacity)
	resourceBlock(b, "Allocatable", n.Status.Allocatable)
	b.raw("System Info:")
	info := n.Status.NodeInfo
	b.kv(2, 28, "Kernel Version", info.KernelVersion)
	b.kv(2, 28, "OS Image", info.OSImage)
	b.kv(2, 28, "Container Runtime Version", info.ContainerRuntimeVersion)
	b.kv(2, 28, "Kubelet Version", info.KubeletVersion)
	if n.Spec.PodCIDR != "" {
		b.kv(0, 20, "PodCIDR", n.Spec.PodCIDR)
	}
	if len(n.Spec.PodCIDRs) > 0 {
		b.kv(0, 20, "PodCIDRs", strings.Join(n.Spec.PodCIDRs, ","))
	}
	podsOnNode(c, b, n)
	b.raw(eventsSection(c, n.UID))
	return b.String()
}

func describeNodeConditions(b *block, conds []corev1.NodeCondition) {
	if len(conds) == 0 {
		return
	}
	b.raw("Conditions:")
	rows := [][]string{{"----", "------", "-----------------", "------------------", "------", "-------"}}
	for _, cond := range conds {
		rows = append(rows, []string{string(cond.Type), string(cond.Status), cond.LastHeartbeatTime.Format(timeLayout),
			cond.LastTransitionTime.Format(timeLayout), cond.Reason, cond.Message})
	}
	headers := []string{"Type", "Status", "LastHeartbeatTime", "LastTransitionTime", "Reason", "Message"}
	b.raw(indent(strings.TrimRight(renderTable(headers, rows), "\n"), "  "))
}

func resourceBlock(b *block, label string, rl corev1.ResourceList) {
	b.raw(label + ":")
	for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceEphemeralStorage, corev1.ResourceMemory, corev1.ResourcePods} {
		if q, ok := rl[name]; ok {
			b.kv(2, 20, string(name), q.String())
		}
	}
}

func podsOnNode(c Cluster, b *block, n *corev1.Node) {
	pods := []*corev1.Pod{}
	for _, p := range store.List[*corev1.Pod](c.Store(), store.KindPod, "") {
		if p.Spec.NodeName == n.Name && !podutil.IsTerminal(p) {
			pods = append(pods, p)
		}
	}
	b.raw(fmt.Sprintf("Non-terminated Pods:          (%d in total)", len(pods)))
	rows := [][]string{{"---------", "----", "------------", "----------", "---------------", "-------------", "---"}}
	var cpuReq, memReq, cpuLim, memLim int64
	allocCPU := n.Status.Allocatable.Cpu().MilliValue()
	allocMem := n.Status.Allocatable.Memory().Value()
	for _, p := range pods {
		cr, mr := podutil.Requests(p)
		cl, ml := podutil.Limits(p)
		cpuReq, memReq, cpuLim, memLim = cpuReq+cr, memReq+mr, cpuLim+cl, memLim+ml
		rows = append(rows, []string{p.Namespace, p.Name, milliPct(cr, allocCPU), milliPct(cl, allocCPU), bytesPct(mr, allocMem), bytesPct(ml, allocMem), age(c, p.CreationTimestamp)})
	}
	table := renderTable([]string{"Namespace", "Name", "CPU Requests", "CPU Limits", "Memory Requests", "Memory Limits", "Age"}, rows)
	b.raw(indent(strings.TrimRight(table, "\n"), "  "))
	b.raw("Allocated resources:")
	b.raw("  (Total limits may be over 100 percent, i.e., overcommitted.)")
	floorReq, floorLim := floorPledges(pods)
	allocFloor := n.Status.Allocatable.StorageEphemeral().Value()
	alloc := renderTable([]string{"Resource", "Requests", "Limits"}, [][]string{
		{"--------", "--------", "------"},
		{"cpu", milliPct(cpuReq, allocCPU), milliPct(cpuLim, allocCPU)},
		{"memory", bytesPct(memReq, allocMem), bytesPct(memLim, allocMem)},
		{"ephemeral-storage", bytesPct(floorReq, allocFloor), bytesPct(floorLim, allocFloor)},
	})
	b.raw(indent(strings.TrimRight(alloc, "\n"), "  "))
}

func floorPledges(pods []*corev1.Pod) (int64, int64) {
	var requests, limits int64
	for _, p := range pods {
		requests += podutil.EphemeralRequest(p)
		if limit, ok := podutil.EphemeralLimit(p); ok {
			limits += limit
		}
	}
	return requests, limits
}

func pct(v, total int64) int64 {
	if total <= 0 {
		return 0
	}
	return v * 100 / total
}

func milliPct(v, total int64) string {
	q := resource.NewMilliQuantity(v, resource.DecimalSI)
	return fmt.Sprintf("%s (%d%%)", q.String(), pct(v, total))
}

func bytesPct(v, total int64) string {
	q := resource.NewQuantity(v, resource.BinarySI)
	return fmt.Sprintf("%s (%d%%)", q.String(), pct(v, total))
}

func describeReplicaSet(c Cluster, rs *appsv1.ReplicaSet) string {
	b := &block{}
	b.kv(0, 14, "Name", rs.Name)
	b.kv(0, 14, "Namespace", rs.Namespace)
	sel := "<none>"
	if rs.Spec.Selector != nil {
		sel = labelString(rs.Spec.Selector.MatchLabels)
	}
	b.kv(0, 14, "Selector", sel)
	b.kvLines(0, 14, "Labels", labelLines(rs.Labels))
	b.kvLines(0, 14, "Annotations", annotationLines(rs.Annotations))
	desired := int32(1)
	if rs.Spec.Replicas != nil {
		desired = *rs.Spec.Replicas
	}
	b.kv(0, 14, "Replicas", fmt.Sprintf("%d current / %d desired", rs.Status.Replicas, desired))
	running, waiting, succeeded, failed := podCounts(c, rs)
	b.kv(0, 14, "Pods Status", fmt.Sprintf("%d Running / %d Waiting / %d Succeeded / %d Failed", running, waiting, succeeded, failed))
	b.raw("Pod Template:")
	b.kvLines(2, 9, "Labels", labelLines(rs.Spec.Template.Labels))
	if inits := rs.Spec.Template.Spec.InitContainers; len(inits) > 0 {
		b.raw("  Init Containers:")
		describeTemplateContainers(b, inits)
	}
	b.raw("  Containers:")
	describeTemplateContainers(b, rs.Spec.Template.Spec.Containers)
	if class := rs.Spec.Template.Spec.PriorityClassName; class != "" {
		b.kv(2, 22, "Priority Class Name", class)
	}
	b.raw(eventsSection(c, rs.UID))
	return b.String()
}

func describeTemplateContainers(b *block, containers []corev1.Container) {
	for _, ct := range containers {
		b.raw("   " + ct.Name + ":")
		b.kv(4, 16, "Image", ct.Image)
		describeResources(b, "Limits", ct.Resources.Limits)
		describeResources(b, "Requests", ct.Resources.Requests)
	}
}

func podCounts(c Cluster, rs *appsv1.ReplicaSet) (int, int, int, int) {
	var running, waiting, succeeded, failed int
	for _, p := range store.List[*corev1.Pod](c.Store(), store.KindPod, rs.Namespace) {
		ref := podutil.ControllerRef(p)
		if ref == nil || ref.UID != rs.UID {
			continue
		}
		switch p.Status.Phase {
		case corev1.PodRunning:
			running++
		case corev1.PodSucceeded:
			succeeded++
		case corev1.PodFailed:
			failed++
		default:
			waiting++
		}
	}
	return running, waiting, succeeded, failed
}

func describeService(c Cluster, s *corev1.Service) string {
	b := &block{}
	b.kv(0, 19, "Name", s.Name)
	b.kv(0, 19, "Namespace", s.Namespace)
	b.kvLines(0, 19, "Labels", labelLines(s.Labels))
	b.kvLines(0, 19, "Annotations", annotationLines(s.Annotations))
	b.kv(0, 19, "Selector", labelString(s.Spec.Selector))
	b.kv(0, 19, "Type", string(s.Spec.Type))
	b.kv(0, 19, "IP", orNone(s.Spec.ClusterIP))
	for _, p := range s.Spec.Ports {
		b.kv(0, 19, "Port", fmt.Sprintf("%s  %d/%s", orUnset(p.Name), p.Port, p.Protocol))
		b.kv(0, 19, "TargetPort", fmt.Sprintf("%s/%s", p.TargetPort.String(), p.Protocol))
	}
	b.kv(0, 19, "Endpoints", serviceEndpoints(c, s))
	b.kv(0, 19, "Session Affinity", orNone(string(s.Spec.SessionAffinity)))
	b.raw(eventsSection(c, s.UID))
	return b.String()
}

func orUnset(s string) string {
	if s == "" {
		return "<unset>"
	}
	return s
}

func serviceEndpoints(c Cluster, s *corev1.Service) string {
	addrs := []string{}
	for _, sl := range store.List[*discoveryv1.EndpointSlice](c.Store(), store.KindEndpointSlice, s.Namespace) {
		if sl.Labels[discoveryv1.LabelServiceName] != s.Name {
			continue
		}
		port := int32(0)
		if len(sl.Ports) > 0 && sl.Ports[0].Port != nil {
			port = *sl.Ports[0].Port
		}
		for _, e := range sl.Endpoints {
			if e.Conditions.Ready == nil || *e.Conditions.Ready {
				for _, a := range e.Addresses {
					addrs = append(addrs, fmt.Sprintf("%s:%d", a, port))
				}
			}
		}
	}
	if len(addrs) == 0 {
		return "<none>"
	}
	return strings.Join(addrs, ",")
}
