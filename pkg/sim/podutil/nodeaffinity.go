package podutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

var labelOperators = map[corev1.NodeSelectorOperator]selection.Operator{
	corev1.NodeSelectorOpIn:           selection.In,
	corev1.NodeSelectorOpNotIn:        selection.NotIn,
	corev1.NodeSelectorOpExists:       selection.Exists,
	corev1.NodeSelectorOpDoesNotExist: selection.DoesNotExist,
	corev1.NodeSelectorOpGt:           selection.GreaterThan,
	corev1.NodeSelectorOpLt:           selection.LessThan,
}

func MatchesRequiredNodeAffinity(nodeSelector map[string]string, affinity *corev1.Affinity, node *corev1.Node) bool {
	if len(nodeSelector) > 0 && !labels.SelectorFromSet(nodeSelector).Matches(labels.Set(node.Labels)) {
		return false
	}
	required := RequiredNodeSelector(affinity)
	if required == nil {
		return true
	}
	for i := range required.NodeSelectorTerms {
		if termMatches(&required.NodeSelectorTerms[i], node) {
			return true
		}
	}
	return false
}

func RequiredNodeSelector(affinity *corev1.Affinity) *corev1.NodeSelector {
	if affinity == nil || affinity.NodeAffinity == nil {
		return nil
	}
	return affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
}

func termMatches(term *corev1.NodeSelectorTerm, node *corev1.Node) bool {
	if len(term.MatchExpressions) == 0 && len(term.MatchFields) == 0 {
		return false
	}
	if len(term.MatchExpressions) > 0 {
		sel, ok := expressionSelector(term.MatchExpressions)
		if !ok || !sel.Matches(labels.Set(node.Labels)) {
			return false
		}
	}
	if len(term.MatchFields) > 0 {
		sel, ok := fieldSelector(term.MatchFields)
		if !ok || (node.Name != "" && !sel.Matches(fields.Set{metav1.ObjectNameField: node.Name})) {
			return false
		}
	}
	return true
}

func expressionSelector(reqs []corev1.NodeSelectorRequirement) (labels.Selector, bool) {
	selector := labels.NewSelector()
	for _, expr := range reqs {
		op, known := labelOperators[expr.Operator]
		if !known {
			return nil, false
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, false
		}
		selector = selector.Add(*r)
	}
	return selector, true
}

func fieldSelector(reqs []corev1.NodeSelectorRequirement) (fields.Selector, bool) {
	selectors := make([]fields.Selector, 0, len(reqs))
	for _, expr := range reqs {
		if len(expr.Values) != 1 {
			return nil, false
		}
		switch expr.Operator {
		case corev1.NodeSelectorOpIn:
			selectors = append(selectors, fields.OneTermEqualSelector(expr.Key, expr.Values[0]))
		case corev1.NodeSelectorOpNotIn:
			selectors = append(selectors, fields.OneTermNotEqualSelector(expr.Key, expr.Values[0]))
		default:
			return nil, false
		}
	}
	return fields.AndSelectors(selectors...), true
}

func TargetNodeName(pod *corev1.Pod) (string, bool) {
	if pod.Spec.NodeName != "" {
		return pod.Spec.NodeName, true
	}
	required := RequiredNodeSelector(pod.Spec.Affinity)
	if required == nil {
		return "", false
	}
	for _, term := range required.NodeSelectorTerms {
		for _, exp := range term.MatchFields {
			if exp.Key == metav1.ObjectNameField && exp.Operator == corev1.NodeSelectorOpIn {
				return firstOnly(exp.Values)
			}
		}
	}
	return "", false
}

func firstOnly(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}
