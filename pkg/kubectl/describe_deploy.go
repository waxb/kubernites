package kubectl

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/rolloututil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

type tabbed struct {
	buf bytes.Buffer
	w   *tabwriter.Writer
}

func newTabbed() *tabbed {
	t := &tabbed{}
	t.w = tabwriter.NewWriter(&t.buf, 0, 8, 2, ' ', 0)
	return t
}

func (t *tabbed) line(level int, format string, args ...any) {
	fmt.Fprintf(t.w, strings.Repeat("  ", level)+format, args...)
}

func (t *tabbed) String() string {
	_ = t.w.Flush()
	return strings.TrimRight(t.buf.String(), "\n")
}

func (t *tabbed) labels(prefix, title string, labels map[string]string) {
	keys := sortedKeys(labels)
	if len(keys) == 0 {
		t.line(0, "%s%s:\t<none>\n", prefix, title)
		return
	}
	for i, k := range keys {
		if i == 0 {
			t.line(0, "%s%s:\t%s=%s\n", prefix, title, k, labels[k])
			continue
		}
		t.line(0, "%s\t%s=%s\n", prefix, k, labels[k])
	}
}

func (t *tabbed) annotations(title string, annotations map[string]string) {
	WriteAnnotations(t.w, title, annotations)
}

func describeDeployment(c Cluster, d *appsv1.Deployment) string {
	t := newTabbed()
	t.line(0, "Name:\t%s\n", d.Name)
	t.line(0, "Namespace:\t%s\n", d.Namespace)
	t.line(0, "CreationTimestamp:\t%s\n", d.CreationTimestamp.Time.Format(time.RFC1123Z))
	t.labels("", "Labels", d.Labels)
	t.annotations("Annotations", d.Annotations)
	t.line(0, "Selector:\t%s\n", selectorString(d.Spec.Selector))
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	st := d.Status
	t.line(0, "Replicas:\t%d desired | %d updated | %d total | %d available | %d unavailable\n", desired, st.UpdatedReplicas, st.Replicas, st.AvailableReplicas, st.UnavailableReplicas)
	t.line(0, "StrategyType:\t%s\n", d.Spec.Strategy.Type)
	t.line(0, "MinReadySeconds:\t%d\n", d.Spec.MinReadySeconds)
	if ru := d.Spec.Strategy.RollingUpdate; ru != nil {
		t.line(0, "RollingUpdateStrategy:\t%s max unavailable, %s max surge\n", ru.MaxUnavailable.String(), ru.MaxSurge.String())
	}
	describePodTemplate(t, &d.Spec.Template)
	describeDeploymentConditions(t, d.Status.Conditions)
	oldRSs, newRS := deploymentSets(c, d)
	if len(oldRSs) > 0 || newRS != nil {
		t.line(0, "OldReplicaSets:\t%s\n", setsByLabels(oldRSs))
		t.line(0, "NewReplicaSet:\t%s\n", setsByLabels(compactSets(newRS)))
	}
	describeEventsTabbed(t, c, d)
	return t.String()
}

func describeDeploymentConditions(t *tabbed, conds []appsv1.DeploymentCondition) {
	if len(conds) == 0 {
		return
	}
	t.line(0, "Conditions:\n  Type\tStatus\tReason\n")
	t.line(1, "----\t------\t------\n")
	for _, cond := range conds {
		t.line(1, "%v \t%v\t%v\n", cond.Type, cond.Status, cond.Reason)
	}
}

func compactSets(rs *appsv1.ReplicaSet) []*appsv1.ReplicaSet {
	if rs == nil {
		return nil
	}
	return []*appsv1.ReplicaSet{rs}
}

func setsByLabels(rss []*appsv1.ReplicaSet) string {
	parts := make([]string, 0, len(rss))
	for _, rs := range rss {
		desired := int32(1)
		if rs.Spec.Replicas != nil {
			desired = *rs.Spec.Replicas
		}
		parts = append(parts, fmt.Sprintf("%s (%d/%d replicas created)", rs.Name, rs.Status.Replicas, desired))
	}
	if len(parts) == 0 {
		return "<none>"
	}
	return strings.Join(parts, ", ")
}

func deploymentSets(c Cluster, d *appsv1.Deployment) ([]*appsv1.ReplicaSet, *appsv1.ReplicaSet) {
	owned := OwnedSets(c, d)
	return rolloututil.FindOldReplicaSets(d, owned), rolloututil.FindNewReplicaSet(d, owned)
}

func OwnedSets(c Cluster, d *appsv1.Deployment) []*appsv1.ReplicaSet {
	out := []*appsv1.ReplicaSet{}
	for _, rs := range store.List[*appsv1.ReplicaSet](c.Store(), store.KindReplicaSet, d.Namespace) {
		if ref := podutil.ControllerRef(rs); ref != nil && ref.UID == d.UID {
			out = append(out, rs)
		}
	}
	return out
}

func describePodTemplate(t *tabbed, tmpl *corev1.PodTemplateSpec) {
	t.line(0, "Pod Template:\n")
	t.labels("  ", "Labels", tmpl.Labels)
	if len(tmpl.Annotations) > 0 {
		t.annotations("  Annotations", tmpl.Annotations)
	}
	if tmpl.Spec.ServiceAccountName != "" {
		t.line(1, "Service Account:\t%s\n", tmpl.Spec.ServiceAccountName)
	}
	if len(tmpl.Spec.InitContainers) > 0 {
		describeTemplateContainersTabbed(t, "Init Containers", tmpl.Spec.InitContainers)
	}
	describeTemplateContainersTabbed(t, "Containers", tmpl.Spec.Containers)
	t.line(0, "  Volumes:\t<none>\n")
	if tmpl.Spec.PriorityClassName != "" {
		t.line(1, "Priority Class Name:\t%s\n", tmpl.Spec.PriorityClassName)
	}
	t.labels("  ", "Node-Selectors", tmpl.Spec.NodeSelector)
	tols := strings.Split(tolerationString(tmpl.Spec.Tolerations), "\n")
	for i, tol := range tols {
		lead := "  Tolerations:"
		if i > 0 {
			lead = ""
		}
		t.line(0, "%s\t%s\n", lead, strings.TrimSpace(tol))
	}
}

func describeTemplateContainersTabbed(t *tabbed, label string, containers []corev1.Container) {
	t.line(0, "  %s:\n", label)
	for _, ct := range containers {
		t.line(1, " %s:\n", ct.Name)
		t.line(2, "Image:\t%s\n", ct.Image)
		t.line(2, "Port:\t%s\n", portsOrNone(ct.Ports, func(p corev1.ContainerPort) int32 { return p.ContainerPort }))
		t.line(2, "Host Port:\t%s\n", portsOrNone(ct.Ports, func(p corev1.ContainerPort) int32 { return p.HostPort }))
		describeResourcesTabbed(t, "Limits", ct.Resources.Limits)
		describeResourcesTabbed(t, "Requests", ct.Resources.Requests)
		for _, l := range containerProbes(ct) {
			t.line(2, "%s:\t%s\n", l.key, describeProbe(l.probe))
		}
		t.line(2, "Environment:\t<none>\n")
		t.line(2, "Mounts:\t<none>\n")
	}
}

func portsOrNone(ports []corev1.ContainerPort, pick func(corev1.ContainerPort) int32) string {
	parts := []string{}
	for _, p := range ports {
		protocol := p.Protocol
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		s := fmt.Sprintf("%d/%s", pick(p), protocol)
		if p.Name != "" {
			s += fmt.Sprintf(" (%s)", p.Name)
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "<none>"
	}
	return strings.Join(parts, ", ")
}

func describeResourcesTabbed(t *tabbed, title string, rl corev1.ResourceList) {
	if len(rl) == 0 {
		return
	}
	t.line(2, "%s:\n", title)
	names := make([]string, 0, len(rl))
	for name := range rl {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		q := rl[corev1.ResourceName(name)]
		t.line(3, "%s:\t%s\n", name, q.String())
	}
}

func describeEventsTabbed(t *tabbed, c Cluster, obj store.Object) {
	events := eventsFor(c, obj.GetUID())
	if len(events) == 0 {
		t.line(0, "Events:\t<none>\n")
		return
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].LastTimestamp.Before(&events[j].LastTimestamp) })
	t.line(0, "Events:\n  Type\tReason\tAge\tFrom\tMessage\n")
	t.line(1, "----\t------\t----\t----\t-------\n")
	for _, e := range events {
		when := age(c, e.LastTimestamp)
		if e.Count > 1 {
			when = fmt.Sprintf("%s (x%d over %s)", when, e.Count, age(c, e.FirstTimestamp))
		}
		t.line(1, "%v\t%v\t%s\t%v\t%v\n", e.Type, e.Reason, when, e.Source.Component, strings.TrimSpace(e.Message))
	}
}
