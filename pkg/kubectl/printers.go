package kubectl

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

type printer struct {
	headers func(wide bool) []string
	row     func(c Cluster, o store.Object, wide bool) []string
}

func printerFor(kind string) printer {
	switch kind {
	case "Pod":
		return printer{podHeaders, podRow}
	case "Node":
		return printer{nodeHeaders, nodeRow}
	case "ReplicaSet":
		return printer{rsHeaders, rsRow}
	case "Deployment":
		return printer{deploymentHeaders, deploymentRow}
	case "DaemonSet":
		return printer{daemonSetHeaders, daemonSetRow}
	case "ControllerRevision":
		return printer{func(bool) []string { return []string{"NAME", "CONTROLLER", "REVISION", "AGE"} }, revisionRow}
	case "Service":
		return printer{svcHeaders, svcRow}
	case "EndpointSlice":
		return printer{func(bool) []string { return []string{"NAME", "ADDRESSTYPE", "PORTS", "ENDPOINTS", "AGE"} }, sliceRow}
	case "Event":
		return printer{func(bool) []string { return []string{"LAST SEEN", "TYPE", "REASON", "OBJECT", "MESSAGE"} }, eventRow}
	case "ConfigMap":
		return printer{func(bool) []string { return []string{"NAME", "DATA", "AGE"} }, cmRow}
	case "ServiceAccount":
		return printer{func(bool) []string { return []string{"NAME", "AGE"} }, saRow}
	case "HorizontalPodAutoscaler":
		return printer{hpaHeaders, hpaRow}
	case "PriorityClass":
		return printer{priorityClassHeaders, priorityClassRow}
	case "LimitRange":
		return printer{limitRangeHeaders, limitRangeRow}
	}
	if p, ok := storagePrinter(kind); ok {
		return p
	}
	return printer{func(bool) []string { return []string{"NAME", "STATUS", "AGE"} }, nsRow}
}

func podHeaders(wide bool) []string {
	h := []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	if wide {
		h = append(h, "IP", "NODE", "NOMINATED NODE", "READINESS GATES")
	}
	return h
}

func podRow(c Cluster, o store.Object, wide bool) []string {
	p := o.(*corev1.Pod)
	ready, total := podutil.ReadyCount(p)
	row := []string{p.Name, fmt.Sprintf("%d/%d", ready, total), podutil.StatusReason(p), restartsColumn(c, p), age(c, p.CreationTimestamp)}
	if wide {
		row = append(row, orNone(p.Status.PodIP), orNone(p.Spec.NodeName), orNone(p.Status.NominatedNodeName), "<none>")
	}
	return row
}

func restartsColumn(c Cluster, p *corev1.Pod) string {
	restarts := podutil.RestartCount(p)
	last := podutil.LastRestart(p)
	if restarts == 0 || last.IsZero() {
		return fmt.Sprintf("%d", restarts)
	}
	return fmt.Sprintf("%d (%s ago)", restarts, age(c, last))
}

func orNone(s string) string {
	if s == "" {
		return "<none>"
	}
	return s
}

func nodeHeaders(wide bool) []string {
	h := []string{"NAME", "STATUS", "ROLES", "AGE", "VERSION"}
	if wide {
		h = append(h, "INTERNAL-IP", "EXTERNAL-IP", "OS-IMAGE", "KERNEL-VERSION", "CONTAINER-RUNTIME")
	}
	return h
}

func nodeStatus(n *corev1.Node) string {
	status := "Unknown"
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			status = "NotReady"
			if cond.Status == corev1.ConditionTrue {
				status = "Ready"
			}
		}
	}
	if n.Spec.Unschedulable {
		status += ",SchedulingDisabled"
	}
	return status
}

func nodeRoles(n *corev1.Node) string {
	roles := []string{}
	for k := range n.Labels {
		if role, ok := strings.CutPrefix(k, "node-role.kubernetes.io/"); ok {
			roles = append(roles, role)
		}
	}
	sort.Strings(roles)
	if len(roles) == 0 {
		return "<none>"
	}
	return strings.Join(roles, ",")
}

func nodeRow(c Cluster, o store.Object, wide bool) []string {
	n := o.(*corev1.Node)
	row := []string{n.Name, nodeStatus(n), nodeRoles(n), age(c, n.CreationTimestamp), n.Status.NodeInfo.KubeletVersion}
	if wide {
		ip := "<none>"
		for _, a := range n.Status.Addresses {
			if a.Type == corev1.NodeInternalIP {
				ip = a.Address
			}
		}
		info := n.Status.NodeInfo
		row = append(row, ip, "<none>", info.OSImage, info.KernelVersion, info.ContainerRuntimeVersion)
	}
	return row
}

func rsHeaders(wide bool) []string {
	h := []string{"NAME", "DESIRED", "CURRENT", "READY", "AGE"}
	if wide {
		h = append(h, "CONTAINERS", "IMAGES", "SELECTOR")
	}
	return h
}

func rsRow(c Cluster, o store.Object, wide bool) []string {
	rs := o.(*appsv1.ReplicaSet)
	desired := int32(1)
	if rs.Spec.Replicas != nil {
		desired = *rs.Spec.Replicas
	}
	row := []string{rs.Name, fmt.Sprint(desired), fmt.Sprint(rs.Status.Replicas), fmt.Sprint(rs.Status.ReadyReplicas), age(c, rs.CreationTimestamp)}
	if wide {
		names, images := []string{}, []string{}
		for _, ct := range rs.Spec.Template.Spec.Containers {
			names, images = append(names, ct.Name), append(images, ct.Image)
		}
		sel := "<none>"
		if rs.Spec.Selector != nil {
			sel = labelString(rs.Spec.Selector.MatchLabels)
		}
		row = append(row, strings.Join(names, ","), strings.Join(images, ","), sel)
	}
	return row
}

func svcHeaders(wide bool) []string {
	h := []string{"NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE"}
	if wide {
		h = append(h, "SELECTOR")
	}
	return h
}

func svcPorts(s *corev1.Service) string {
	parts := []string{}
	for _, p := range s.Spec.Ports {
		if p.NodePort > 0 {
			parts = append(parts, fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
	}
	if len(parts) == 0 {
		return "<none>"
	}
	return strings.Join(parts, ",")
}

func svcRow(c Cluster, o store.Object, wide bool) []string {
	s := o.(*corev1.Service)
	row := []string{s.Name, string(s.Spec.Type), orNone(s.Spec.ClusterIP), "<none>", svcPorts(s), age(c, s.CreationTimestamp)}
	if wide {
		row = append(row, labelString(s.Spec.Selector))
	}
	return row
}

func sliceRow(c Cluster, o store.Object, _ bool) []string {
	s := o.(*discoveryv1.EndpointSlice)
	ports := []string{}
	for _, p := range s.Ports {
		if p.Port != nil {
			ports = append(ports, fmt.Sprint(*p.Port))
		}
	}
	addrs := []string{}
	for _, e := range s.Endpoints {
		addrs = append(addrs, e.Addresses...)
	}
	eps := "<unset>"
	if len(addrs) > 0 {
		shown := addrs[:min(3, len(addrs))]
		eps = strings.Join(shown, ",")
		if len(addrs) > 3 {
			eps += fmt.Sprintf(" + %d more...", len(addrs)-3)
		}
	}
	return []string{s.Name, string(s.AddressType), orNone(strings.Join(ports, ",")), eps, age(c, s.CreationTimestamp)}
}

func eventRow(c Cluster, o store.Object, _ bool) []string {
	e := o.(*corev1.Event)
	obj := strings.ToLower(e.InvolvedObject.Kind) + "/" + e.InvolvedObject.Name
	return []string{age(c, eventLastSeen(e)), e.Type, e.Reason, obj, e.Message}
}

func cmRow(c Cluster, o store.Object, _ bool) []string {
	cm := o.(*corev1.ConfigMap)
	return []string{cm.Name, fmt.Sprint(len(cm.Data) + len(cm.BinaryData)), age(c, cm.CreationTimestamp)}
}

func saRow(c Cluster, o store.Object, _ bool) []string {
	sa := o.(*corev1.ServiceAccount)
	return []string{sa.Name, age(c, sa.CreationTimestamp)}
}

func nsRow(c Cluster, o store.Object, _ bool) []string {
	ns := o.(*corev1.Namespace)
	phase := string(ns.Status.Phase)
	if phase == "" {
		phase = "Active"
	}
	return []string{ns.Name, phase, age(c, ns.CreationTimestamp)}
}
