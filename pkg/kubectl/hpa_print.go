package kubectl

import (
	"fmt"
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2"

	"github.com/waxb/kubernites/pkg/sim/store"
)

const maxShownMetrics = 2

func hpaHeaders(bool) []string {
	return []string{"NAME", "REFERENCE", "TARGETS", "MINPODS", "MAXPODS", "REPLICAS", "AGE"}
}

func hpaRow(c Cluster, o store.Object, _ bool) []string {
	hpa := o.(*autoscalingv2.HorizontalPodAutoscaler)
	minPods := "<unset>"
	if hpa.Spec.MinReplicas != nil {
		minPods = fmt.Sprint(*hpa.Spec.MinReplicas)
	}
	ref := hpa.Spec.ScaleTargetRef
	return []string{
		hpa.Name, ref.Kind + "/" + ref.Name, FormatHPAMetrics(hpa.Spec.Metrics, hpa.Status.CurrentMetrics),
		minPods, fmt.Sprint(hpa.Spec.MaxReplicas), fmt.Sprint(hpa.Status.CurrentReplicas), age(c, hpa.CreationTimestamp),
	}
}

func resourceStatus(statuses []autoscalingv2.MetricStatus, i int) *autoscalingv2.ResourceMetricStatus {
	if len(statuses) > i {
		return statuses[i].Resource
	}
	return nil
}

func formatResourceMetric(spec *autoscalingv2.ResourceMetricSource, status *autoscalingv2.ResourceMetricStatus) string {
	current := "<unknown>"
	if spec.Target.AverageValue != nil {
		if status != nil && status.Current.AverageValue != nil {
			current = status.Current.AverageValue.String()
		}
		return fmt.Sprintf("%s: %s/%s", spec.Name, current, spec.Target.AverageValue.String())
	}
	if status != nil && status.Current.AverageUtilization != nil {
		current = fmt.Sprintf("%d%%", *status.Current.AverageUtilization)
	}
	target := "<auto>"
	if spec.Target.AverageUtilization != nil {
		target = fmt.Sprintf("%d%%", *spec.Target.AverageUtilization)
	}
	return fmt.Sprintf("%s: %s/%s", spec.Name, current, target)
}

func FormatHPAMetrics(specs []autoscalingv2.MetricSpec, statuses []autoscalingv2.MetricStatus) string {
	if len(specs) == 0 {
		return "<none>"
	}
	list := []string{}
	for i, spec := range specs {
		if spec.Type != autoscalingv2.ResourceMetricSourceType || spec.Resource == nil {
			list = append(list, "<unknown type>")
			continue
		}
		list = append(list, formatResourceMetric(spec.Resource, resourceStatus(statuses, i)))
	}
	if len(list) > maxShownMetrics {
		return fmt.Sprintf("%s + %d more...", strings.Join(list[:maxShownMetrics], ", "), len(list)-maxShownMetrics)
	}
	return strings.Join(list, ", ")
}
