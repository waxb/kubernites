package kubectl

import (
	"fmt"
	"strings"
)

type ResourceType struct {
	Kind       string
	Plural     string
	Singular   string
	short      string
	apiVersion string
	group      string
	Namespaced bool
}

var resourceTypes = []ResourceType{
	{Kind: "Pod", Plural: "pods", Singular: "pod", short: "po", apiVersion: "v1", Namespaced: true},
	{Kind: "Node", Plural: "nodes", Singular: "node", short: "no", apiVersion: "v1"},
	{Kind: "Deployment", Plural: "deployments", Singular: "deployment", short: "deploy", apiVersion: "apps/v1", group: "apps", Namespaced: true},
	{Kind: "DaemonSet", Plural: "daemonsets", Singular: "daemonset", short: "ds", apiVersion: "apps/v1", group: "apps", Namespaced: true},
	{Kind: "ControllerRevision", Plural: "controllerrevisions", Singular: "controllerrevision", apiVersion: "apps/v1", group: "apps", Namespaced: true},
	{Kind: "ReplicaSet", Plural: "replicasets", Singular: "replicaset", short: "rs", apiVersion: "apps/v1", group: "apps", Namespaced: true},
	{Kind: "Service", Plural: "services", Singular: "service", short: "svc", apiVersion: "v1", Namespaced: true},
	{Kind: "EndpointSlice", Plural: "endpointslices", Singular: "endpointslice", apiVersion: "discovery.k8s.io/v1", group: "discovery.k8s.io", Namespaced: true},
	{Kind: "Event", Plural: "events", Singular: "event", short: "ev", apiVersion: "v1", Namespaced: true},
	{Kind: "Namespace", Plural: "namespaces", Singular: "namespace", short: "ns", apiVersion: "v1"},
	{Kind: "ConfigMap", Plural: "configmaps", Singular: "configmap", short: "cm", apiVersion: "v1", Namespaced: true},
	{Kind: "ServiceAccount", Plural: "serviceaccounts", Singular: "serviceaccount", short: "sa", apiVersion: "v1", Namespaced: true},
	{Kind: "LimitRange", Plural: "limitranges", Singular: "limitrange", short: "limits", apiVersion: "v1", Namespaced: true},
	{Kind: "HorizontalPodAutoscaler", Plural: "horizontalpodautoscalers", Singular: "horizontalpodautoscaler", short: "hpa", apiVersion: "autoscaling/v2", group: "autoscaling", Namespaced: true},
	{Kind: "PriorityClass", Plural: "priorityclasses", Singular: "priorityclass", short: "pc", apiVersion: "scheduling.k8s.io/v1", group: "scheduling.k8s.io"},
}

func (r ResourceType) Qualified() string {
	if r.group == "" {
		return r.Singular
	}
	return r.Singular + "." + r.group
}

func ResolveType(token string) (ResourceType, error) {
	lower := strings.ToLower(token)
	for _, r := range resourceTypes {
		if lower == r.Plural || lower == r.Singular || (r.short != "" && lower == r.short) || lower == strings.ToLower(r.Kind) {
			return r, nil
		}
		if r.group != "" && (lower == r.Plural+"."+r.group || lower == r.Singular+"."+r.group) {
			return r, nil
		}
	}
	return ResourceType{}, fmt.Errorf("error: the server doesn't have a resource type %q", token)
}

func SplitTypeName(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("error: you must specify the type of resource to get. Use \"kubectl api-resources\" for a complete list of supported resources")
	}
	if kind, name, ok := strings.Cut(args[0], "/"); ok {
		return kind, append([]string{name}, args[1:]...), nil
	}
	return args[0], args[1:], nil
}

func NotFound(r ResourceType, name string) error {
	return fmt.Errorf("Error from server (NotFound): %s %q not found", qualifiedPlural(r), name)
}

func qualifiedPlural(r ResourceType) string {
	if r.group == "" {
		return r.Plural
	}
	return r.Plural + "." + r.group
}

func NamespaceFor(r ResourceType, inv Invocation) string {
	if !r.Namespaced {
		return ""
	}
	return inv.Namespace
}
