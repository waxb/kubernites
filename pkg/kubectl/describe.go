package kubectl

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

const timeLayout = "Mon, 02 Jan 2006 15:04:05 -0700"

type block struct {
	lines []string
}

func (b *block) kv(indent int, width int, key, value string) {
	b.row(indent, width, key+":", value)
}

func (b *block) raw(line string) {
	b.lines = append(b.lines, line)
}

func (b *block) String() string {
	return strings.Join(b.lines, "\n")
}

func describe(c Cluster, inv Invocation) (string, string, error) {
	typeTok, names, err := SplitTypeName(inv.Args)
	if err != nil {
		return "", "", err
	}
	r, err := ResolveType(typeTok)
	if err != nil {
		return "", typeTok, err
	}
	objs, err := SelectObjects(c, r, inv, names)
	if err != nil {
		return "", r.Singular, err
	}
	if len(objs) == 0 {
		return noResources(r, inv), r.Singular, nil
	}
	parts := make([]string, 0, len(objs))
	for _, o := range objs {
		parts = append(parts, describeOne(c, o))
	}
	return strings.Join(parts, "\n\n\n"), r.Singular, nil
}

func describeOne(c Cluster, o store.Object) string {
	if out, ok := describeStorage(c, o); ok {
		return out
	}
	switch obj := o.(type) {
	case *corev1.Pod:
		return describePod(c, obj)
	case *corev1.Node:
		return describeNode(c, obj)
	case *appsv1.ReplicaSet:
		return describeReplicaSet(c, obj)
	case *appsv1.Deployment:
		return describeDeployment(c, obj)
	case *appsv1.DaemonSet:
		return describeDaemonSet(c, obj)
	case *corev1.Service:
		return describeService(c, obj)
	case *autoscalingv2.HorizontalPodAutoscaler:
		return describeHPA(c, obj)
	case *schedulingv1.PriorityClass:
		return describePriorityClass(c, obj)
	case *corev1.LimitRange:
		return describeLimitRange(obj)
	case *corev1.ServiceAccount:
		return describeServiceAccount(c, obj)
	}
	data, _ := yaml.Marshal(o)
	return string(data)
}

func describePod(c Cluster, p *corev1.Pod) string {
	b := &block{}
	head, ips := podHeader(c, p)
	b.fields(head)
	if len(ips) > 0 {
		b.raw("IPs:")
		b.fields(ips)
	}
	if len(p.Spec.InitContainers) > 0 {
		b.raw("Init Containers:")
		for _, ct := range p.Spec.InitContainers {
			describeContainer(b, ct, statusNamed(p.Status.InitContainerStatuses, ct.Name))
		}
	}
	b.raw("Containers:")
	for i, ct := range p.Spec.Containers {
		describeContainer(b, ct, statusAt(p, i))
	}
	if len(p.Status.Conditions) > 0 {
		width := len("Type")
		for _, cond := range p.Status.Conditions {
			width = max(width, len(cond.Type)+1)
		}
		b.raw("Conditions:")
		b.raw(fmt.Sprintf("  %-*sStatus", width+2, "Type"))
		for _, cond := range p.Status.Conditions {
			b.raw(fmt.Sprintf("  %-*s%s", width+2, cond.Type, cond.Status))
		}
	}
	b.kv(0, 29, "QoS Class", string(podutil.QoSClass(p)))
	b.kv(0, 29, "Node-Selectors", labelString(p.Spec.NodeSelector))
	b.kv(0, 29, "Tolerations", tolerationString(p.Spec.Tolerations))
	b.raw(eventsSection(c, p.UID))
	return b.String()
}

type field struct {
	key   string
	value string
}

func (b *block) fields(fs []field) {
	width := 0
	for _, f := range fs {
		width = max(width, len(f.key)+1)
	}
	for _, f := range fs {
		b.row(0, width+2, f.label(), f.value)
	}
}

func podHeader(c Cluster, p *corev1.Pod) ([]field, []field) {
	out := []field{{"Name", p.Name}, {"Namespace", p.Namespace}}
	if p.Spec.Priority != nil {
		out = append(out, field{"Priority", fmt.Sprint(*p.Spec.Priority)})
	}
	if p.Spec.PriorityClassName != "" {
		out = append(out, field{"Priority Class Name", p.Spec.PriorityClassName})
	}
	if p.Spec.ServiceAccountName != "" {
		out = append(out, field{"Service Account", p.Spec.ServiceAccountName})
	}
	out = append(out, field{"Node", nodeWithIP(p)})
	if p.Status.StartTime != nil {
		out = append(out, field{"Start Time", p.Status.StartTime.Format(timeLayout)})
	}
	out = append(out, fieldLines("Labels", labelLines(p.Labels))...)
	out = append(out, fieldLines("Annotations", annotationLines(p.Annotations))...)
	out = append(out, podStatusFields(c, p)...)
	if p.Status.Reason != "" {
		out = append(out, field{"Reason", p.Status.Reason}, field{"Message", p.Status.Message})
	}
	if sc := p.Spec.SecurityContext; sc != nil && sc.SeccompProfile != nil {
		out = append(out, field{"SeccompProfile", string(sc.SeccompProfile.Type)})
	}
	out = append(out, field{"IP", p.Status.PodIP})
	tail := &out
	var ips []field
	if len(p.Status.PodIPs) == 0 {
		out = append(out, field{"IPs", "<none>"})
	} else {
		for _, ip := range p.Status.PodIPs {
			ips = append(ips, field{"  IP", ip.IP})
		}
		tail = &ips
	}
	if ref := podutil.ControllerRef(p); ref != nil {
		*tail = append(*tail, field{"Controlled By", ref.Kind + "/" + ref.Name})
	}
	if p.Status.NominatedNodeName != "" {
		*tail = append(*tail, field{"NominatedNodeName", p.Status.NominatedNodeName})
	}
	return out, ips
}

func podStatusFields(c Cluster, p *corev1.Pod) []field {
	if ts := p.DeletionTimestamp; ts != nil && p.Status.Phase != corev1.PodFailed && p.Status.Phase != corev1.PodSucceeded {
		out := []field{{"Status", "Terminating (lasts " + age(c, *ts) + ")"}}
		if g := p.DeletionGracePeriodSeconds; g != nil {
			out = append(out, field{"Termination Grace Period", fmt.Sprintf("%ds", *g)})
		}
		return out
	}
	return []field{{"Status", string(p.Status.Phase)}}
}

func nodeWithIP(p *corev1.Pod) string {
	if p.Spec.NodeName == "" {
		return "<none>"
	}
	if p.Status.HostIP == "" {
		return p.Spec.NodeName
	}
	return p.Spec.NodeName + "/" + p.Status.HostIP
}

func statusAt(p *corev1.Pod, i int) *corev1.ContainerStatus {
	if i < len(p.Status.ContainerStatuses) {
		cs := p.Status.ContainerStatuses[i]
		return &cs
	}
	return nil
}

func statusNamed(statuses []corev1.ContainerStatus, name string) *corev1.ContainerStatus {
	if cs, ok := podutil.ContainerStatusNamed(statuses, name); ok {
		return &cs
	}
	return nil
}

func describeContainer(b *block, ct corev1.Container, cs *corev1.ContainerStatus) {
	b.raw("  " + ct.Name + ":")
	b.kv(4, 16, "Image", ct.Image)
	describePorts(b, ct.Ports)
	if cs != nil {
		describeState(b, "State", cs.State)
		if cs.LastTerminationState.Terminated != nil {
			describeState(b, "Last State", cs.LastTerminationState)
		}
		b.kv(4, 16, "Ready", boolWord(cs.Ready))
		b.kv(4, 16, "Restart Count", fmt.Sprint(cs.RestartCount))
	}
	describeResources(b, "Limits", ct.Resources.Limits)
	describeResources(b, "Requests", ct.Resources.Requests)
	for _, l := range containerProbes(ct) {
		b.kv(4, 16, l.key, describeProbe(l.probe))
	}
	b.kv(4, 16, "Environment", "<none>")
}

func describePorts(b *block, ports []corev1.ContainerPort) {
	own, host := make([]string, 0, len(ports)), make([]string, 0, len(ports))
	for _, p := range ports {
		own = append(own, namedPort(p.ContainerPort, p))
		host = append(host, namedPort(p.HostPort, p))
	}
	portLabel, hostLabel := "Port", "Host Port"
	if len(ports) > 1 {
		portLabel, hostLabel = "Ports", "Host Ports"
	}
	b.kv(4, 16, portLabel, orNone(strings.Join(own, ", ")))
	b.kv(4, 16, hostLabel, orNone(strings.Join(host, ", ")))
}

func namedPort(port int32, p corev1.ContainerPort) string {
	out := fmt.Sprintf("%d/%s", port, p.Protocol)
	if p.Name != "" {
		out += " (" + p.Name + ")"
	}
	return out
}

func describeState(b *block, label string, s corev1.ContainerState) {
	switch {
	case s.Running != nil:
		b.kv(4, 16, label, "Running")
		b.kv(6, 14, "Started", s.Running.StartedAt.Format(timeLayout))
	case s.Waiting != nil:
		b.kv(4, 16, label, "Waiting")
		b.kv(6, 14, "Reason", s.Waiting.Reason)
		if s.Waiting.Message != "" {
			b.kv(6, 14, "Message", s.Waiting.Message)
		}
	case s.Terminated != nil:
		b.kv(4, 16, label, "Terminated")
		b.kv(6, 14, "Reason", s.Terminated.Reason)
		b.kv(6, 14, "Exit Code", fmt.Sprint(s.Terminated.ExitCode))
		b.kv(6, 14, "Finished", s.Terminated.FinishedAt.Format(timeLayout))
	default:
		b.kv(4, 16, label, "Waiting")
	}
}

func describeResources(b *block, label string, rl corev1.ResourceList) {
	if len(rl) == 0 {
		return
	}
	b.raw("    " + label + ":")
	names := make([]string, 0, len(rl))
	for n := range rl {
		names = append(names, string(n))
	}
	sort.Strings(names)
	for _, n := range names {
		q := rl[corev1.ResourceName(n)]
		b.raw(fmt.Sprintf("      %s:  %s", n, q.String()))
	}
}

func boolWord(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

func tolerationString(tols []corev1.Toleration) string {
	if len(tols) == 0 {
		return "<none>"
	}
	parts := []string{}
	for _, t := range tols {
		s := t.Key
		if t.Value != "" {
			s += "=" + t.Value
		}
		if t.Effect != "" {
			s += ":" + string(t.Effect)
		}
		if t.Operator == corev1.TolerationOpExists {
			s += " op=Exists"
		}
		if t.TolerationSeconds != nil {
			s += fmt.Sprintf(" for %ds", *t.TolerationSeconds)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n"+strings.Repeat(" ", 29))
}

func eventsSection(c Cluster, uid types.UID) string {
	events := eventsFor(c, uid)
	if len(events) == 0 {
		return "Events:                      <none>"
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].FirstTimestamp.Before(&events[j].FirstTimestamp) })
	rows := [][]string{{"----", "------", "----", "----", "-------"}}
	for _, e := range events {
		when := age(c, e.LastTimestamp)
		if e.Count > 1 {
			when = fmt.Sprintf("%s (x%d over %s)", when, e.Count, age(c, e.FirstTimestamp))
		}
		rows = append(rows, []string{e.Type, e.Reason, when, e.Source.Component, e.Message})
	}
	table := renderTable([]string{"Type", "Reason", "Age", "From", "Message"}, rows)
	return "Events:\n" + indent(strings.TrimRight(table, "\n"), "  ")
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
