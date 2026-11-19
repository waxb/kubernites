package kubectl

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func deploymentHeaders(wide bool) []string {
	h := []string{"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	if wide {
		h = append(h, "CONTAINERS", "IMAGES", "SELECTOR")
	}
	return h
}

func deploymentRow(c Cluster, o store.Object, wide bool) []string {
	d := o.(*appsv1.Deployment)
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	st := d.Status
	row := []string{d.Name, fmt.Sprintf("%d/%d", st.ReadyReplicas, desired), fmt.Sprint(st.UpdatedReplicas), fmt.Sprint(st.AvailableReplicas), age(c, d.CreationTimestamp)}
	if wide {
		names, images := []string{}, []string{}
		for _, ct := range d.Spec.Template.Spec.Containers {
			names, images = append(names, ct.Name), append(images, ct.Image)
		}
		row = append(row, strings.Join(names, ","), strings.Join(images, ","), selectorString(d.Spec.Selector))
	}
	return row
}

func selectorString(sel *metav1.LabelSelector) string {
	selector, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return "<invalid>"
	}
	return selector.String()
}
