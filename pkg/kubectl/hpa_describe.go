package kubectl

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func describeHPA(c Cluster, hpa *autoscalingv2.HorizontalPodAutoscaler) string {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 8, 2, ' ', 0)
	ref := hpa.Spec.ScaleTargetRef
	fmt.Fprintf(w, "Name:\t%s\n", hpa.Name)
	fmt.Fprintf(w, "Namespace:\t%s\n", hpa.Namespace)
	writeLabels(w, "Labels", hpa.Labels)
	WriteAnnotations(w, "Annotations", hpa.Annotations)
	fmt.Fprintf(w, "CreationTimestamp:\t%s\n", hpa.CreationTimestamp.Time.Format(timeLayout))
	fmt.Fprintf(w, "Reference:\t%s/%s\n", ref.Kind, ref.Name)
	fmt.Fprintf(w, "Metrics:\t( current / target )\n")
	for i, m := range hpa.Spec.Metrics {
		WriteHPAMetric(w, m, resourceStatus(hpa.Status.CurrentMetrics, i))
	}
	minReplicas := "<unset>"
	if hpa.Spec.MinReplicas != nil {
		minReplicas = fmt.Sprint(*hpa.Spec.MinReplicas)
	}
	fmt.Fprintf(w, "Min replicas:\t%s\n", minReplicas)
	fmt.Fprintf(w, "Max replicas:\t%d\n", hpa.Spec.MaxReplicas)
	if b := hpa.Spec.Behavior; b != nil {
		fmt.Fprintf(w, "Behavior:\n")
		WriteDirection(w, "Scale Up", b.ScaleUp)
		WriteDirection(w, "Scale Down", b.ScaleDown)
	}
	fmt.Fprintf(w, "%s pods:\t%d current / %d desired\n", ref.Kind, hpa.Status.CurrentReplicas, hpa.Status.DesiredReplicas)
	writeHPAConditions(w, hpa.Status.Conditions)
	writeEvents(c, w, hpa.UID)
	_ = w.Flush()
	return strings.TrimSuffix(buf.String(), "\n")
}

func writeLabels(w io.Writer, title string, labels map[string]string) {
	fmt.Fprintf(w, "%s:\t", title)
	if len(labels) == 0 {
		fmt.Fprintln(w, "<none>")
		return
	}
	for i, key := range sortedKeys(labels) {
		if i != 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprintf(w, "%s=%s\n", key, labels[key])
	}
}

func WriteHPAMetric(w io.Writer, m autoscalingv2.MetricSpec, status *autoscalingv2.ResourceMetricStatus) {
	if m.Type != autoscalingv2.ResourceMetricSourceType || m.Resource == nil {
		fmt.Fprintf(w, "  <unknown metric type %q>\n", string(m.Type))
		return
	}
	fmt.Fprintf(w, "  resource %s on pods", m.Resource.Name)
	current := "<unknown>"
	if avg := m.Resource.Target.AverageValue; avg != nil {
		if status != nil && status.Current.AverageValue != nil {
			current = status.Current.AverageValue.String()
		}
		fmt.Fprintf(w, ":\t%s / %s\n", current, avg.String())
		return
	}
	if status != nil && status.Current.AverageUtilization != nil {
		current = fmt.Sprintf("%d%% (%s)", *status.Current.AverageUtilization, status.Current.AverageValue.String())
	}
	target := "<auto>"
	if u := m.Resource.Target.AverageUtilization; u != nil {
		target = fmt.Sprintf("%d%%", *u)
	}
	fmt.Fprintf(w, "  (as a percentage of request):\t%s / %s\n", current, target)
}

func WriteDirection(w io.Writer, direction string, rules *autoscalingv2.HPAScalingRules) {
	if rules == nil {
		return
	}
	fmt.Fprintf(w, "  %s:\n", direction)
	if rules.StabilizationWindowSeconds != nil {
		fmt.Fprintf(w, "    Stabilization Window: %d seconds\n", *rules.StabilizationWindowSeconds)
	}
	if len(rules.Policies) == 0 {
		return
	}
	selectPolicy := autoscalingv2.MaxChangePolicySelect
	if rules.SelectPolicy != nil {
		selectPolicy = *rules.SelectPolicy
	}
	fmt.Fprintf(w, "    Select Policy: %s\n", selectPolicy)
	fmt.Fprintf(w, "    Policies:\n")
	for _, p := range rules.Policies {
		fmt.Fprintf(w, "      - Type: %s\tValue: %d\tPeriod: %d seconds\n", p.Type, p.Value, p.PeriodSeconds)
	}
}

func writeHPAConditions(w io.Writer, conds []autoscalingv2.HorizontalPodAutoscalerCondition) {
	if len(conds) == 0 {
		return
	}
	fmt.Fprintf(w, "Conditions:\n")
	fmt.Fprintf(w, "  Type\tStatus\tReason\tMessage\n")
	fmt.Fprintf(w, "  ----\t------\t------\t-------\n")
	for _, cond := range conds {
		fmt.Fprintf(w, "  %v\t%v\t%v\t%v\n", cond.Type, cond.Status, cond.Reason, cond.Message)
	}
}

func eventsFor(c Cluster, uid types.UID) []*corev1.Event {
	events := []*corev1.Event{}
	for _, e := range store.List[*corev1.Event](c.Store(), store.KindEvent, "") {
		if e.InvolvedObject.UID == uid {
			events = append(events, e)
		}
	}
	return events
}

func writeEvents(c Cluster, w *tabwriter.Writer, uid types.UID) {
	events := eventsFor(c, uid)
	if len(events) == 0 {
		fmt.Fprintf(w, "Events:\t<none>\n")
		return
	}
	_ = w.Flush()
	sort.SliceStable(events, func(i, j int) bool {
		a, b := eventLastSeen(events[i]), eventLastSeen(events[j])
		return a.Before(&b)
	})
	fmt.Fprintf(w, "Events:\n  Type\tReason\tAge\tFrom\tMessage\n")
	fmt.Fprintf(w, "  ----\t------\t----\t----\t-------\n")
	for _, e := range events {
		first := age(c, eventFirstSeen(e))
		interval := first
		switch {
		case e.Series != nil:
			interval = fmt.Sprintf("%s (x%d over %s)", age(c, eventLastSeen(e)), e.Series.Count, first)
		case e.Count > 1:
			interval = fmt.Sprintf("%s (x%d over %s)", age(c, e.LastTimestamp), e.Count, first)
		}
		source := e.Source.Component
		if source == "" {
			source = e.ReportingController
		}
		fmt.Fprintf(w, "  %v\t%v\t%s\t%v\t%v\n", e.Type, e.Reason, interval, source, strings.TrimSpace(e.Message))
	}
}
