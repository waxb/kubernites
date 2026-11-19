package kubectl

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/waxb/kubernites/pkg/sim/store"
)

var metaFieldNames = []string{"metadata.name", "metadata.namespace"}

var kindFieldNames = map[string][]string{
	"Pod":   {"spec.nodeName", "spec.restartPolicy", "spec.schedulerName", "spec.serviceAccountName", "status.phase", "status.podIP"},
	"Node":  {"spec.unschedulable"},
	"Event": {"involvedObject.kind", "involvedObject.name", "involvedObject.namespace", "reason", "type"},
}

func knownFields(kind string) []string {
	return slices.Concat(metaFieldNames, kindFieldNames[kind])
}

func objectFields(obj store.Object) fields.Set {
	out := fields.Set{"metadata.name": obj.GetName(), "metadata.namespace": obj.GetNamespace()}
	switch o := obj.(type) {
	case *corev1.Pod:
		out["spec.nodeName"], out["status.phase"] = o.Spec.NodeName, string(o.Status.Phase)
		out["spec.restartPolicy"], out["spec.schedulerName"] = string(o.Spec.RestartPolicy), o.Spec.SchedulerName
		out["spec.serviceAccountName"], out["status.podIP"] = o.Spec.ServiceAccountName, o.Status.PodIP
	case *corev1.Node:
		out["spec.unschedulable"] = strconv.FormatBool(o.Spec.Unschedulable)
	case *corev1.Event:
		out["involvedObject.kind"], out["involvedObject.name"] = o.InvolvedObject.Kind, o.InvolvedObject.Name
		out["involvedObject.namespace"], out["reason"], out["type"] = o.InvolvedObject.Namespace, o.Reason, o.Type
	}
	return out
}

func fieldSelector(kind, raw string) (fields.Selector, error) {
	if raw == "" {
		return fields.Everything(), nil
	}
	sel, err := fields.ParseSelector(raw)
	if err != nil {
		return nil, fmt.Errorf("error: unable to read the field selector %q: %v", raw, err)
	}
	known := knownFields(kind)
	for _, req := range sel.Requirements() {
		if !slices.Contains(known, req.Field) {
			return nil, fmt.Errorf("Error from server (BadRequest): unable to filter by the field %q: try %s", req.Field, strings.Join(known, ", "))
		}
	}
	return sel, nil
}
