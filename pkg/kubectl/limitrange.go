package kubectl

import (
	"bytes"
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func limitRangeHeaders(bool) []string {
	return []string{"NAME", "CREATED AT"}
}

func limitRangeRow(_ Cluster, o store.Object, _ bool) []string {
	lr := o.(*corev1.LimitRange)
	return []string{lr.Name, lr.CreationTimestamp.UTC().Format(time.RFC3339)}
}

func describeLimitRange(lr *corev1.LimitRange) string {
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "Name:\t%s\n", lr.Name)
	fmt.Fprintf(w, "Namespace:\t%s\n", lr.Namespace)
	fmt.Fprintf(w, "Type\tResource\tMin\tMax\tDefault Request\tDefault Limit\tMax Limit/Request Ratio\n")
	fmt.Fprintf(w, "----\t--------\t---\t---\t---------------\t-------------\t-----------------------\n")
	for _, item := range lr.Spec.Limits {
		for _, name := range limitResources(item) {
			fmt.Fprintf(w, "%s\t%v\t%v\t%v\t%v\t%v\t%v\n", item.Type, name, quantityOr(item.Min, name), quantityOr(item.Max, name),
				quantityOr(item.DefaultRequest, name), quantityOr(item.Default, name), quantityOr(item.MaxLimitRequestRatio, name))
		}
	}
	_ = w.Flush()
	return buf.String()
}

func limitResources(item corev1.LimitRangeItem) []corev1.ResourceName {
	set := map[corev1.ResourceName]bool{}
	for _, list := range []corev1.ResourceList{item.Max, item.Min, item.Default, item.DefaultRequest, item.MaxLimitRequestRatio} {
		for name := range list {
			set[name] = true
		}
	}
	out := make([]corev1.ResourceName, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func quantityOr(list corev1.ResourceList, name corev1.ResourceName) string {
	if q, ok := list[name]; ok {
		return q.String()
	}
	return "-"
}

func limitField(field, typ, text, lore string) explanation {
	return explanation{kind: "LimitRange", version: "v1", field: field, typ: typ, text: text, lore: lore}
}

var limitRangeExplanations = buildLimitRangeExplanations()

func buildLimitRangeExplanations() map[string]explanation {
	lr := corev1.LimitRange{}.SwaggerDoc()
	spec := corev1.LimitRangeSpec{}.SwaggerDoc()
	item := corev1.LimitRangeItem{}.SwaggerDoc()
	return map[string]explanation{
		"limitrange": {kind: "LimitRange", version: "v1", text: lr[""],
			lore: "The realm's default pledges, added to a namespace by its founding charter. A worker that pledges nothing is given them when the Chancery admits it (the LimitRanger admission plugin); workers already standing keep what they have."},
		"limitrange.spec": limitField("spec", "LimitRangeSpec", lr["spec"], "The rules of the namespace, one item per kind of object."),
		"limitrange.spec.limits": limitField("limits", "[]LimitRangeItem", spec["limits"],
			"Each item binds one kind: Container (each worker in a hut), Pod (the whole hut) or PersistentVolumeClaim."),
		"limitrange.spec.limits.type": limitField("type", "string", item["type"], "Container, Pod or PersistentVolumeClaim."),
		"limitrange.spec.limits.defaultrequest": limitField("defaultRequest", "map[string]Quantity", item["defaultRequest"],
			"The pledge a container is given when it names none. A container that names only a binding (limit) pledges that binding instead."),
		"limitrange.spec.limits.default": limitField("default", "map[string]Quantity", item["default"],
			"The binding (limit) a container is given when it names none."),
		"limitrange.spec.limits.min": limitField("min", "map[string]Quantity", item["min"],
			"The smallest pledge and binding allowed: a hut asking less is refused at the Chancery's door."),
		"limitrange.spec.limits.max": limitField("max", "map[string]Quantity", item["max"],
			"The largest binding allowed: a hut whose binding is higher, or missing, is refused."),
		"limitrange.spec.limits.maxlimitrequestratio": limitField("maxLimitRequestRatio", "map[string]Quantity", item["maxLimitRequestRatio"],
			"How far a binding may exceed its pledge: limit divided by request."),
	}
}
