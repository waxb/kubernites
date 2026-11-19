package view

import (
	"slices"
	"strings"
)

type Bridge struct {
	Page    string
	Command string
	Verbs   []string
	YAML    string
	Rule    string
	Doc     string
	Bends   []string
}

var handDepot = Bridge{Page: "depot", Command: "kubectl apply -f parcels.yaml", Verbs: []string{"expose"},
	YAML: "apiVersion: v1\nkind: Service\nmetadata:\n  name: parcels\n  labels:\n    app: porters\nspec:\n  ports:\n  - name: http\n    port: 80\n    targetPort: 8080\n" +
		"---\napiVersion: discovery.k8s.io/v1\nkind: EndpointSlice\nmetadata:\n  name: parcels-by-hand\n  labels:\n    kubernetes.io/service-name: parcels\n" +
		"    endpointslice.kubernetes.io/managed-by: staff\naddressType: IPv4\nports:\n- name: http\n  port: 8080\nendpoints:",
	Rule: "A Service with no selector gets no EndpointSlice from the control plane. You write one yourself, labelled " +
		"kubernetes.io/service-name with the Service's name, and kube-proxy sends the Service's traffic to the addresses on it.",
	Doc: DocService + "#services-without-selectors", Bends: []string{LibertyEndpoints}}

func HandDepotBridge(addresses []string) Bridge {
	b := handDepot
	b.Verbs, b.Bends = slices.Clone(b.Verbs), slices.Clone(b.Bends)
	if len(addresses) == 0 {
		b.YAML += " []"
	}
	for _, a := range addresses {
		b.YAML += "\n- addresses:\n  - " + a
	}
	return b
}

const PodSlot = "<pod>"

const NodeSlot = "<node>"

func (b Bridge) Filled(pod, node string) Bridge {
	next := b
	fill := strings.NewReplacer(PodSlot, orSlot(pod, PodSlot), NodeSlot, orSlot(node, NodeSlot))
	next.Command, next.YAML = fill.Replace(b.Command), fill.Replace(b.YAML)
	return next
}

func orSlot(name, slot string) string {
	if strings.TrimSpace(name) == "" {
		return slot
	}
	return name
}

func (b Bridge) Matches(command string) bool {
	return b.MatchesWords(CommandWords(command))
}

func (b Bridge) MatchesWords(words []string) bool {
	for _, verb := range b.Verbs {
		pattern := strings.Fields(verb)
		if len(words) >= len(pattern) && slices.Equal(words[:len(pattern)], pattern) {
			return true
		}
	}
	return false
}

func (b Bridge) Reads() bool {
	return readVerbs[firstWord(b.Command)]
}

var readVerbs = map[string]bool{"get": true, "describe": true, "top": true, "logs": true, "cluster-info": true, "endpoint": true}

func firstWord(command string) string {
	words := CommandWords(command)
	if len(words) == 0 {
		return ""
	}
	if words[0] == "etcdctl" && len(words) > 1 {
		return words[1]
	}
	return words[0]
}

var kindAliases = map[string]string{
	"deploy": "deployment", "deployments": "deployment", "deployment.apps": "deployment",
	"rs": "replicaset", "replicasets": "replicaset", "replicaset.apps": "replicaset",
	"svc": "service", "services": "service", "po": "pod", "pods": "pod", "no": "node", "nodes": "node",
	"hpa": "horizontalpodautoscaler", "horizontalpodautoscalers": "horizontalpodautoscaler",
	"endpointslices": "endpointslice", "ds": "daemonset", "daemonsets": "daemonset", "sts": "statefulset", "statefulsets": "statefulset",
}

func CommandWords(command string) []string {
	out := []string{}
	for i, tok := range strings.Fields(command) {
		if i == 0 && tok == "kubectl" {
			continue
		}
		if strings.HasPrefix(tok, "-") || strings.Contains(tok, "=") && !strings.Contains(tok, "/") {
			continue
		}
		for _, part := range strings.SplitN(tok, "/", 2) {
			if alias, ok := kindAliases[part]; ok {
				part = alias
			}
			out = append(out, part)
		}
	}
	return out
}

const ChapterDoorYAML = "# /etc/kubernetes/manifests/porter.yaml on the Node\napiVersion: v1\nkind: Pod\nmetadata:\n  name: porter\n  labels:\n    app: porters\nspec:\n  hostNetwork: true\n  containers:\n  - name: porter\n    image: porter:1.0\n    ports:\n    - name: http\n      containerPort: 80\n      hostPort: 80\n    env:\n    - name: PORT\n      value: \"80\"\n    resources:\n      requests:\n        cpu: 100m\n      limits:\n        cpu: 300m"
