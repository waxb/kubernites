package kubectl

import (
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

const (
	kubeVersion      = "v1.37.0"
	kustomizeVersion = "v5.8.1"
)

func top(c Cluster, inv Invocation) (string, string, error) {
	if !c.MetricsAvailable() {
		return "", "", errors.New("error: Metrics API not available")
	}
	if len(inv.Args) == 0 {
		return "", "", errors.New("error: expected 'top pods' or 'top nodes'")
	}
	r, err := ResolveType(inv.Args[0])
	if err != nil {
		return "", inv.Args[0], err
	}
	switch r.Kind {
	case "Pod":
		out, err := topPods(c, inv)
		return out, "pods", err
	case "Node":
		out, err := topNodes(c)
		return out, "nodes", err
	}
	return "", r.Plural, fmt.Errorf("error: top only reads Omens of pods and nodes")
}

const metricsCreationDelay = 2 * time.Minute

var errMetricsNotYet = errors.New("error: metrics not available yet")

func topPods(c Cluster, inv Invocation) (string, error) {
	ns := inv.Namespace
	if inv.allNS {
		ns = ""
	}
	pods := []*corev1.Pod{}
	rows := [][]string{}
	selector := labels.Everything()
	if raw := inv.Flags["selector"]; raw != "" {
		parsed, err := labels.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("error: unable to parse requirement: %v", err)
		}
		selector = parsed
	}
	for _, p := range store.List[*corev1.Pod](c.Store(), store.KindPod, ns) {
		if podutil.IsTerminal(p) || !selector.Matches(labels.Set(p.Labels)) {
			continue
		}
		pods = append(pods, p)
		if u, ok := c.PodUsage(string(p.UID)); ok {
			rows = append(rows, podMetricsRow(p, u, inv.allNS))
		}
	}
	if len(rows) == 0 {
		return noPodMetrics(c, pods, inv)
	}
	headers := []string{"NAME", "CPU(cores)", "MEMORY(bytes)"}
	if inv.allNS {
		headers = append([]string{"NAMESPACE"}, headers...)
	}
	return renderTable(headers, rows), nil
}

func podMetricsRow(p *corev1.Pod, u Usage, allNS bool) []string {
	row := []string{p.Name, formatMilli(u.CPUMilli), formatMi(u.MemBytes)}
	if allNS {
		row = append([]string{p.Namespace}, row...)
	}
	return row
}

func noPodMetrics(c Cluster, pods []*corev1.Pod, inv Invocation) (string, error) {
	if len(pods) == 0 {
		if inv.allNS {
			return "No resources found", nil
		}
		return fmt.Sprintf("No resources found in %s namespace.", inv.Namespace), nil
	}
	for _, p := range pods {
		if age := c.Clock().Since(p.CreationTimestamp); age > metricsCreationDelay {
			return "", fmt.Errorf("error: Metrics not available for pod %s/%s, age: %s", p.Namespace, p.Name, age)
		}
	}
	return "", errMetricsNotYet
}

func topNodes(c Cluster) (string, error) {
	rows, missing := [][]string{}, [][]string{}
	for _, n := range store.List[*corev1.Node](c.Store(), store.KindNode, "") {
		u, ok := c.NodeUsage(n.Name)
		if !ok {
			missing = append(missing, []string{n.Name, "<unknown>", "<unknown>", "<unknown>", "<unknown>"})
			continue
		}
		cpu := n.Status.Allocatable.Cpu().MilliValue()
		mem := n.Status.Allocatable.Memory().Value()
		rows = append(rows, []string{n.Name, formatMilli(u.CPUMilli), fmt.Sprintf("%d%%", pct(int64(u.CPUMilli), cpu)), formatMi(u.MemBytes), fmt.Sprintf("%d%%", pct(int64(u.MemBytes), mem))})
	}
	if len(rows) == 0 {
		return "", errMetricsNotYet
	}
	return renderTable([]string{"NAME", "CPU(cores)", "CPU(%)", "MEMORY(bytes)", "MEMORY(%)"}, append(rows, missing...)), nil
}

func version(_ Cluster, _ Invocation) (string, string, error) {
	return "Client Version: " + kubeVersion + "\nKustomize Version: " + kustomizeVersion + "\nServer Version: " + kubeVersion, "", nil
}

func clusterInfo(_ Cluster, _ Invocation) (string, string, error) {
	return "Kubernetes control plane is running at https://172.18.0.2:6443\n" +
		"CoreDNS is running at https://172.18.0.2:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy\n\n" +
		"To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.", "", nil
}

func apiResources(_ Cluster, _ Invocation) (string, string, error) {
	rows := [][]string{}
	for _, r := range resourceTypes {
		rows = append(rows, []string{r.Plural, orEmpty(r.short), r.apiVersion, fmt.Sprint(r.Namespaced), r.Kind})
	}
	for _, k := range gatewayKinds {
		rows = append(rows, []string{k.plural, k.short, k.apiVersion(), fmt.Sprint(k.namespaced), k.kind})
	}
	return renderTable([]string{"NAME", "SHORTNAMES", "APIVERSION", "NAMESPACED", "KIND"}, rows), "", nil
}

func orEmpty(s string) string {
	return s
}

func help(_ Cluster, _ Invocation) (string, string, error) {
	lines := []string{
		"The Speaking Stone understands these commands:",
		"",
		"  kubectl get <type> [name] [-o wide|yaml|json|name] [-l k=v] [-L key] [--show-labels] [-A]",
		"  kubectl describe <type> <name>",
		"  kubectl logs <pod> [--previous] [--tail=N]",
		"  kubectl run <name> --image=<trade:class> [--labels=k=v,...]",
		"  kubectl create deployment <name> --image=<trade:class> [--replicas=N] [--port=P]",
		"  kubectl expose deployment|pod|rs|svc <name> [--name=<service>] [--port=P] [--target-port=P]",
		"  kubectl delete <type> <name> [--grace-period=N] [--force] [--cascade=background|foreground|orphan]",
		"  kubectl scale deployment|rs <name> --replicas=N",
		"  kubectl rollout status|history|undo|restart|pause|resume deployment/<name> [--revision=N] [--to-revision=N]",
		"  kubectl autoscale deployment|rs <name> --max=N [--min=N] [--cpu=70%|500m] [--name=<horn>]",
		"  kubectl patch hpa <name> -p '{\"spec\":{\"maxReplicas\":N}}'",
		"  kubectl patch <deployment|daemonset|statefulset|replicaset|pod> <name> -p '<patch>' [--type=strategic|merge|json]",
		"  kubectl label <type> <name> k=v [k-] [--overwrite]",
		"  kubectl set image deployment/<name> <container>=<image>",
		"  kubectl set resources deployment/<name> --requests=cpu=100m,memory=64Mi [--limits=...]",
		"  kubectl top pods|nodes",
		"  kubectl cordon|uncordon <node>",
		"  kubectl taint nodes <node> key=value:Effect | key:Effect- | key- [--overwrite]",
		"  kubectl explain <type>[.field]",
		"  kubectl version | cluster-info | api-resources",
		"",
		"The Codex answers etcdctl on the same Stone:",
		"  etcdctl endpoint status [--write-out=table] | alarm list | alarm disarm | defrag | compact <revision>",
		"",
		"Types: pods (po), nodes (no), deployments (deploy), replicasets (rs), services (svc), endpointslices, gateways (gtw), httproutes, events (ev), namespaces (ns), configmaps (cm), limitranges (limits), horizontalpodautoscalers (hpa)",
	}
	return strings.Join(lines, "\n"), "", nil
}
