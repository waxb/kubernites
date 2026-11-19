package kubectl

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/waxb/kubernites/pkg/sim/store"
)

const priorityClassKeyWidth = 19

func priorityClassHeaders(bool) []string {
	return []string{"NAME", "VALUE", "GLOBAL-DEFAULT", "AGE", "PREEMPTIONPOLICY"}
}

func priorityClassRow(c Cluster, o store.Object, _ bool) []string {
	pc := o.(*schedulingv1.PriorityClass)
	return []string{pc.Name, fmt.Sprint(pc.Value), fmt.Sprint(pc.GlobalDefault), age(c, pc.CreationTimestamp), policyString(pc.PreemptionPolicy)}
}

func policyString(p *corev1.PreemptionPolicy) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

func describePriorityClass(c Cluster, pc *schedulingv1.PriorityClass) string {
	b := &block{}
	b.kv(0, priorityClassKeyWidth, "Name", pc.Name)
	b.kv(0, priorityClassKeyWidth, "Value", fmt.Sprint(pc.Value))
	b.kv(0, priorityClassKeyWidth, "GlobalDefault", fmt.Sprint(pc.GlobalDefault))
	b.kv(0, priorityClassKeyWidth, "PreemptionPolicy", policyString(pc.PreemptionPolicy))
	b.kv(0, priorityClassKeyWidth, "Description", pc.Description)
	b.kv(0, priorityClassKeyWidth, "Annotations", labels.FormatLabels(pc.Annotations))
	if len(eventsFor(c, pc.UID)) == 0 {
		b.kv(0, priorityClassKeyWidth, "Events", "<none>")
		return b.String()
	}
	b.raw(eventsSection(c, pc.UID))
	return b.String()
}

var priorityExplanations = buildPriorityExplanations()

func priorityField(field, typ, text, lore string) explanation {
	return explanation{kind: "PriorityClass", version: "scheduling.k8s.io/v1", field: field, typ: typ, text: text, lore: lore}
}

func podField(field, typ, text, lore string) explanation {
	return explanation{kind: "Pod", version: "v1", field: field, typ: typ, text: text, lore: lore}
}

func buildPriorityExplanations() map[string]explanation {
	pc := schedulingv1.PriorityClass{}.SwaggerDoc()
	spec := corev1.PodSpec{}.SwaggerDoc()
	status := corev1.PodStatus{}.SwaggerDoc()
	return map[string]explanation{
		"priorityclass": {kind: "PriorityClass", version: "scheduling.k8s.io/v1", text: pc[""],
			lore: "A Rank: how much a hut matters. The Quartermaster places higher Ranks first, and when quarters run out it displaces (preempts) lower Ranks to place a higher one. Users may rank up to 1,000,000,000; above that the Citadel's own classes stand."},
		"priorityclass.value": priorityField("value", "integer", pc["value"],
			"The Rank's weight. Only the order matters: a hut of Rank 1000 outranks one of Rank 100."),
		"priorityclass.globaldefault": priorityField("globalDefault", "boolean", pc["globalDefault"],
			"The Rank given to huts that name none. Only one Rank may be the default, and it binds only huts raised after it."),
		"priorityclass.description": priorityField("description", "string", pc["description"],
			"A note for the realm's stewards: when this Rank should be used."),
		"priorityclass.preemptionpolicy": priorityField("preemptionPolicy", "string", pc["preemptionPolicy"],
			"Never: the hut waits at the head of the queue but never displaces anyone."),
		"pod.spec.priorityclassname": podField("priorityClassName", "string", spec["priorityClassName"],
			"The Rank the hut claims. The Chancery resolves it once, when the hut is written: change a crew's Rank and only the new workers of its rollout carry it."),
		"pod.spec.priority": podField("priority", "integer", spec["priority"],
			"Written by the Chancery (the Priority admission plugin) from the Rank. You may not choose it yourself."),
		"pod.spec.preemptionpolicy": podField("preemptionPolicy", "string", spec["preemptionPolicy"],
			"Copied from the Rank: whether this hut may displace lower Ranks."),
		"pod.status.nominatednodename": podField("nominatedNodeName", "string", status["nominatedNodeName"],
			"The Keep the Quartermaster cleared for this hut by displacing lower Ranks. It waits there while the displaced leave."),
	}
}
