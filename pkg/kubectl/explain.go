package kubectl

import (
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type explanation struct {
	kind    string
	version string
	field   string
	typ     string
	text    string
	lore    string
}

var explanations = map[string]explanation{
	"pod": {kind: "Pod", version: "v1",
		text: "Pod is a collection of containers that can run on a host. This resource is created by clients and scheduled onto hosts.",
		lore: "A hut: one household of workers sharing a fire (IP), with its own cellar (volumes)."},
	"pod.spec": {kind: "Pod", version: "v1", field: "spec", typ: "PodSpec",
		text: "Specification of the desired behavior of the pod.",
		lore: "What the hut should be. The Reeve works day and night to make it so."},
	"pod.spec.containers": {kind: "Pod", version: "v1", field: "containers", typ: "[]Container",
		text: "List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod.",
		lore: "The workers who dwell at the hut, each trained at the Academy in the trade its image names."},
	"pod.spec.containers.resources": {kind: "Pod", version: "v1", field: "resources", typ: "ResourceRequirements",
		text: "Compute Resources required by this container. requests: minimum amount reserved by the scheduler. limits: maximum amount allowed.",
		lore: "Pledges (requests) are what the Quartermaster reserves. Bindings (limits) are the ceiling: Food Bindings chafe, Housing Bindings kill."},
	"pod.spec.initcontainers": {kind: "Pod", version: "v1", field: "initContainers", typ: "[]Container",
		text: corev1.PodSpec{}.SwaggerDoc()["initContainers"],
		lore: "Squires and Familiars who arrive before the hut's workers. A Familiar (restartPolicy: Always) is a sidecar: it shares the hut's fire and cellar, counts toward its Pledge, starts first, stays for the whole life of the hut, is relit on its own when it falls, and leaves only after the workers have gone. Squires run to completion, one after another, before the hut's workers start."},
	"pod.spec.initcontainers.restartpolicy": {kind: "Pod", version: "v1", field: "restartPolicy", typ: "string",
		text: corev1.Container{}.SwaggerDoc()["restartPolicy"],
		lore: "Always binds this init container as a Familiar (a sidecar container). Without it, it is a Squire: it runs to completion before the workers start."},
	"pod.spec.nodename": {kind: "Pod", version: "v1", field: "nodeName", typ: "string",
		text: "NodeName indicates in which node this pod is scheduled. It is set by the scheduler when the pod is bound.",
		lore: "The Keep the Quartermaster assigned."},
	"deployment": {kind: "Deployment", version: "apps/v1",
		text: "Deployment enables declarative updates for Pods and ReplicaSets.",
		lore: "A crew: it runs one shift per class. Change the crew's design and a rollout begins: a new shift takes over, hut by hut, while the old one stands down."},
	"deployment.spec.strategy": {kind: "Deployment", version: "apps/v1", field: "strategy", typ: "DeploymentStrategy",
		text: "The deployment strategy to use to replace existing pods with new ones. RollingUpdate (the default) replaces them gradually within maxSurge and maxUnavailable (25% each by default); Recreate stops every old pod before starting new ones.",
		lore: "The Vanguard (maxSurge) may ride ahead of the Banner, and the Allowed Gaps (maxUnavailable) may open in the line: never more."},
	"deployment.spec.revisionhistorylimit": {kind: "Deployment", version: "apps/v1", field: "revisionHistoryLimit", typ: "integer",
		text: "The number of old ReplicaSets to retain to allow rollback. Defaults to 10.",
		lore: "How many old shifts the crew keeps for kubectl rollout undo."},
	"deployment.spec.paused": {kind: "Deployment", version: "apps/v1", field: "paused", typ: "boolean",
		text: "Indicates that the deployment is paused.",
		lore: "A furled Banner: changes to its design wait until it is unfurled (kubectl rollout resume)."},
	"replicaset": {kind: "ReplicaSet", version: "apps/v1",
		text: "ReplicaSet ensures that a specified number of pod replicas are running at any given time.",
		lore: "A shift: the Stewards keep N workers burning and replace any that fall. They also adopt ownerless huts wearing the right badges."},
	"replicaset.spec.replicas": {kind: "ReplicaSet", version: "apps/v1", field: "replicas", typ: "integer",
		text: "Replicas is the number of desired pods. Defaults to 1.",
		lore: "The count of huts the shift keeps."},
	"replicaset.spec.selector": {kind: "ReplicaSet", version: "apps/v1", field: "selector", typ: "LabelSelector",
		text: "Selector is a label query over pods that should match the replica count. It must match the pod template's labels. It is immutable.",
		lore: "The badge it looks for. Once raised, it can never change."},
	"service": {kind: "Service", version: "v1",
		text: "Service is a named abstraction of software service consisting of local port that the proxy listens on, and the selector that determines which pods will answer requests sent through the proxy.",
		lore: "A depot: boxes come to it by name and are sent to any hut that is Fit for Duty and wears its badge."},
	"node": {kind: "Node", version: "v1",
		text: "Node is a worker node in Kubernetes. Each node has a unique identifier in the cache (i.e. in etcd).",
		lore: "A Keep raised on a plot of land. It counts only once its Reeve lights the candle (Ready)."},
	"event": {kind: "Event", version: "v1",
		text: "Event is a report of an event somewhere in the cluster. Events have a limited retention time.",
		lore: "Tidings: what the Stewards, the Reeves and the Quartermaster saw fit to report."},
}

func explain(_ Cluster, inv Invocation) (string, string, error) {
	if len(inv.Args) == 0 {
		return "", "", errors.New("error: You must specify the type of resource to explain")
	}
	path := strings.ToLower(inv.Args[0])
	head, tail, _ := strings.Cut(path, ".")
	r, err := ResolveType(head)
	if g, ok := gatewayKindOf(head); err != nil && ok {
		return explainGateway(g, tail)
	}
	if err != nil {
		return "", head, err
	}
	key := r.Singular
	if tail != "" {
		key += "." + tail
	}
	e, ok := explanations[key]
	if !ok {
		e, ok = hpaExplanations[key]
	}
	if !ok {
		e, ok = priorityExplanations[key]
	}
	if !ok {
		e, ok = limitRangeExplanations[key]
	}
	if !ok {
		segments := []string{}
		if tail != "" {
			segments = strings.Split(tail, ".")
		}
		if e, err = typeExplanation(r, segments); err != nil {
			return "", r.Plural, err
		}
	}
	return explainText(e, tail), r.Plural, nil
}

var gatewayExplanations = map[string]struct{ doc string }{
	"GatewayClass": {doc: "GatewayClass describes a class of Gateways available to the user for creating Gateway resources."},
	"Gateway":      {doc: "Gateway represents an instance of a service-traffic handling infrastructure by binding Listeners to a set of IP addresses."},
	"HTTPRoute":    {doc: "HTTPRoute provides a way to route HTTP requests. This includes the capability to match requests by hostname, path, header, or query param. Filters can be used to specify additional processing steps. Backends specify where matching requests should be routed."},
	"TCPRoute":     {doc: "TCPRoute provides a way to route TCP requests. When combined with a Gateway listener, it can be used to forward connections on the port specified by the listener to a set of backends specified by the TCPRoute."},
	"TLSRoute":     {doc: "The TLSRoute resource is similar to TCPRoute, but can be configured to match against TLS-specific metadata. This allows more flexibility in matching streams for a given TLS listener."},
}

func explainGateway(k gatewayKind, tail string) (string, string, error) {
	if tail != "" {
		field, _, _ := strings.Cut(tail, ".")
		return "", k.plural, fmt.Errorf("error: field %q is not described here: only the kind is", field)
	}
	e := explanation{kind: k.kind, version: k.apiVersion(), text: gatewayExplanations[k.kind].doc}
	return explainText(e, ""), k.plural, nil
}

func explainText(e explanation, tail string) string {
	var b strings.Builder
	if g := groupOf(e.version); g != "" {
		b.WriteString("GROUP:      " + g + "\n")
	}
	fmt.Fprintf(&b, "KIND:       %s\nVERSION:    %s\n\n", e.kind, versionOf(e.version))
	if e.field != "" {
		fmt.Fprintf(&b, "FIELD: %s <%s>\n\n\n", e.field, e.typ)
	}
	b.WriteString("DESCRIPTION:\n" + wrapIndented(e.text, explainWrap, explainTextPad) + "\n\n")
	if e.lore != "" {
		b.WriteString("IN THE REALM:\n" + wrapIndented(e.lore, explainWrap, explainTextPad) + "\n\n")
	}
	path := []string{}
	if tail != "" {
		path = strings.Split(tail, ".")
	}
	if fields := fieldList(e.kind, path); fields != "" {
		b.WriteString("FIELDS:\n" + fields)
	}
	return strings.TrimRight(b.String(), "\n")
}

func groupOf(apiVersion string) string {
	if g, _, ok := strings.Cut(apiVersion, "/"); ok {
		return g
	}
	return ""
}

func versionOf(apiVersion string) string {
	if _, v, ok := strings.Cut(apiVersion, "/"); ok {
		return v
	}
	return apiVersion
}
